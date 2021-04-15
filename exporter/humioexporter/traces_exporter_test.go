// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package humioexporter

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	"go.uber.org/zap"
)

func createSpanID(stringVal string) [8]byte {
	var id [8]byte
	b, _ := hex.DecodeString(stringVal)
	copy(id[:], b)
	return id
}

func createTraceID(stringVal string) [16]byte {
	var id [16]byte
	b, _ := hex.DecodeString(stringVal)
	copy(id[:], b)
	return id
}

// Implement a mock of the client interface for testing
type clientMock struct {
	response func() error
}

func (m *clientMock) sendUnstructuredEvents(ctx context.Context, evts []*HumioUnstructuredEvents) error {
	return m.response()
}

func (m *clientMock) sendStructuredEvents(ctx context.Context, evts []*HumioStructuredEvents) error {
	return m.response()
}

func TestPushTraceData(t *testing.T) {
	// Arrange
	testCases := []struct {
		desc     string
		client   exporterClient
		wantErr  bool
		wantPerm bool
	}{
		{
			desc: "Valid request",
			client: &clientMock{
				response: func() error {
					return nil
				},
			},
			wantErr:  false,
			wantPerm: false,
		},
		{
			desc: "Forwards transient errors",
			client: &clientMock{
				response: func() error {
					return errors.New("Error")
				},
			},
			wantErr:  true,
			wantPerm: false,
		},
		{
			desc: "Forwards permanent errors",
			client: &clientMock{
				response: func() error {
					return consumererror.Permanent(errors.New("Error"))
				},
			},
			wantErr:  true,
			wantPerm: true,
		},
	}

	// Act
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			exp := newTracesExporter(&Config{}, zap.NewNop(), tC.client)
			err := exp.pushTraceData(context.Background(), pdata.NewTraces())

			// Assert
			if (err != nil) != tC.wantErr {
				t.Errorf("pushTraceData() error = %v, wantErr %v", err, tC.wantErr)
			}

			if consumererror.IsPermanent(err) != tC.wantPerm {
				t.Errorf("pushTraceData() permanent = %v, wantPerm %v",
					consumererror.IsPermanent(err), tC.wantPerm)
			}
		})
	}
}

func TestPushTraceData_PermanentOnCompleteFailure(t *testing.T) {
	// Arrange
	// We do not export spans with missing service names, so this span should
	// fail exporting
	traces := pdata.NewTraces()
	traces.ResourceSpans().Resize(1)
	traces.ResourceSpans().At(0).InstrumentationLibrarySpans().Resize(1)
	traces.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)

	exp := newTracesExporter(&Config{}, zap.NewNop(), &clientMock{})

	// Act
	err := exp.pushTraceData(context.Background(), traces)

	// Assert
	require.Error(t, err)
	assert.True(t, consumererror.IsPermanent(err))
}

func TestPushTraceData_TransientOnPartialFailure(t *testing.T) {
	// Arrange
	// Prepare a valid span with a service name...
	traces := pdata.NewTraces()
	traces.ResourceSpans().Resize(2)
	traces.ResourceSpans().At(0).Resource().Attributes().InsertString(conventions.AttributeServiceName, "service1")
	traces.ResourceSpans().At(0).InstrumentationLibrarySpans().Resize(1)
	traces.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().Resize(1)

	// ...and one without (partial failure)
	traces.ResourceSpans().At(1).InstrumentationLibrarySpans().Resize(1)
	traces.ResourceSpans().At(1).InstrumentationLibrarySpans().At(0).Spans().Resize(1)

	exp := newTracesExporter(&Config{}, zap.NewNop(), &clientMock{
		func() error { return nil },
	})

	// Act
	err := exp.pushTraceData(context.Background(), traces)

	// Assert
	require.Error(t, err)
	assert.False(t, consumererror.IsPermanent(err))

	tErr := consumererror.Traces{}
	if ok := consumererror.AsTraces(err, &tErr); !ok {
		assert.Fail(t, "PushTraceData did not return a Traces error")
	}
	assert.Equal(t, 1, tErr.GetTraces().ResourceSpans().Len())
}

func TestTracesToHumioEvents_OrganizedByTags(t *testing.T) {
	// Arrange
	traces := pdata.NewTraces()

	// Three spans for the same service-A across two different resources, as
	// well a span from a separate service-B
	res1 := pdata.NewResourceSpans()
	res1.Resource().Attributes().InsertString(conventions.AttributeServiceName, "service-A")
	res1.InstrumentationLibrarySpans().Resize(1)
	res1.InstrumentationLibrarySpans().At(0).Spans().Resize(2)
	traces.ResourceSpans().Append(res1)

	res2 := pdata.NewResourceSpans()
	res2.Resource().Attributes().InsertString(conventions.AttributeServiceName, "service-A")
	res2.InstrumentationLibrarySpans().Resize(1)
	res2.InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	traces.ResourceSpans().Append(res2)

	res3 := pdata.NewResourceSpans()
	res3.Resource().Attributes().InsertString(conventions.AttributeServiceName, "service-B")
	res3.InstrumentationLibrarySpans().Resize(1)
	res3.InstrumentationLibrarySpans().At(0).Spans().Resize(1)
	traces.ResourceSpans().Append(res3)

	// Organize by service name
	exp := newTracesExporter(&Config{
		Tag: TagServiceName,
	}, zap.NewNop(), &clientMock{})

	// Act
	actual, err := exp.tracesToHumioEvents(traces)

	// Assert
	require.NoError(t, err)
	assert.Len(t, actual, 2)
	for _, group := range actual {
		assert.Contains(t, group.Tags, string(TagServiceName))

		if group.Tags[string(TagServiceName)] == "service-A" {
			assert.Len(t, group.Events, 3)
		} else {
			assert.Len(t, group.Events, 1)
		}
	}
}

func TestSpanToHumioEvent(t *testing.T) {
	// Arrange
	span := pdata.NewSpan()
	span.SetTraceID(pdata.NewTraceID(createTraceID("10")))
	span.SetSpanID(pdata.NewSpanID(createSpanID("20")))
	span.SetName("span")
	span.SetKind(pdata.SpanKindSERVER)
	span.SetStartTimestamp(pdata.TimestampFromTime(
		time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC),
	))
	span.SetEndTimestamp(pdata.TimestampFromTime(
		time.Date(2020, 1, 1, 12, 0, 16, 0, time.UTC),
	))
	span.Status().SetCode(pdata.StatusCodeOk)
	span.Status().SetMessage("done")
	span.Attributes().InsertString("key", "val")

	inst := pdata.NewInstrumentationLibrary()
	inst.SetName("otel-test")
	inst.SetVersion("1.0.0")

	res := pdata.NewResource()
	res.Attributes().InsertString("service.name", "myapp")

	expected := &HumioStructuredEvent{
		Timestamp: time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC),
		AsUnix:    true,
		Attributes: &HumioSpan{
			TraceID:           "10000000000000000000000000000000",
			SpanID:            "2000000000000000",
			ParentSpanID:      "",
			Name:              "span",
			Kind:              "SPAN_KIND_SERVER",
			Start:             time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC).UnixNano(),
			End:               time.Date(2020, 1, 1, 12, 0, 16, 0, time.UTC).UnixNano(),
			StatusCode:        "STATUS_CODE_OK",
			StatusDescription: "done",
			ServiceName:       "myapp",
			Links:             []*HumioLink{},
			Attributes: map[string]interface{}{
				"key":                  "val",
				"otel.library.name":    "otel-test",
				"otel.library.version": "1.0.0",
			},
		},
	}

	exp := newTracesExporter(&Config{
		Traces: TracesConfig{
			UnixTimestamps: true,
		},
	}, zap.NewNop(), &clientMock{})

	// Act
	actual := exp.spanToHumioEvent(span, inst, res)

	// Assert
	assert.Equal(t, expected, actual)
}

func TestSpanToHumioEventNoInstrumentation(t *testing.T) {
	// Arrange
	span := pdata.NewSpan()
	inst := pdata.NewInstrumentationLibrary()
	res := pdata.NewResource()

	exp := newTracesExporter(&Config{
		Traces: TracesConfig{
			UnixTimestamps: true,
		},
	}, zap.NewNop(), &clientMock{})

	// Act
	actual := exp.spanToHumioEvent(span, inst, res)

	// Assert
	require.IsType(t, &HumioSpan{}, actual.Attributes)
	assert.Empty(t, actual.Attributes.(*HumioSpan).Attributes)
}

func TestToHumioLinks(t *testing.T) {
	// Arrange
	slice := pdata.NewSpanLinkSlice()
	link1 := pdata.NewSpanLink()
	link1.SetTraceID(pdata.NewTraceID(createTraceID("11")))
	link1.SetSpanID(pdata.NewSpanID(createSpanID("22")))
	link1.SetTraceState("state1")
	slice.Append(link1)

	link2 := pdata.NewSpanLink()
	link2.SetTraceID(pdata.NewTraceID(createTraceID("33")))
	link2.SetSpanID(pdata.NewSpanID(createSpanID("44")))
	slice.Append(link2)

	expected := []*HumioLink{
		{
			TraceID:    "11000000000000000000000000000000",
			SpanID:     "2200000000000000",
			TraceState: "state1",
		},
		{
			TraceID:    "33000000000000000000000000000000",
			SpanID:     "4400000000000000",
			TraceState: "",
		},
	}

	// Act
	actual := toHumioLinks(slice)

	// Assert
	assert.Equal(t, expected, actual)
}

func TestToHumioAttributes(t *testing.T) {
	// Arrange
	testCases := []struct {
		desc     string
		attr     func() pdata.AttributeMap
		expected interface{}
	}{
		{
			desc: "Simple types",
			attr: func() pdata.AttributeMap {
				attrMap := pdata.NewAttributeMap()
				attrMap.InsertString("string", "val")
				attrMap.InsertInt("integer", 42)
				attrMap.InsertDouble("double", 4.2)
				attrMap.InsertBool("bool", false)
				return attrMap
			},
			expected: map[string]interface{}{
				"string":  "val",
				"integer": int64(42),
				"double":  4.2,
				"bool":    false,
			},
		},
		{
			desc: "Nil element",
			attr: func() pdata.AttributeMap {
				attrMap := pdata.NewAttributeMap()
				attrMap.InsertNull("key")
				return attrMap
			},
			expected: map[string]interface{}{
				"key": nil,
			},
		},
		{
			desc: "Array element",
			attr: func() pdata.AttributeMap {
				attrMap := pdata.NewAttributeMap()
				arr := pdata.NewAttributeValueArray()
				arr.ArrayVal().Append(pdata.NewAttributeValueString("a"))
				arr.ArrayVal().Append(pdata.NewAttributeValueString("b"))
				arr.ArrayVal().Append(pdata.NewAttributeValueInt(4))
				attrMap.Insert("array", arr)
				return attrMap
			},
			expected: map[string]interface{}{
				"array": []interface{}{
					"a", "b", int64(4),
				},
			},
		},
		{
			desc: "Nested map",
			attr: func() pdata.AttributeMap {
				attrMap := pdata.NewAttributeMap()
				nested := pdata.NewAttributeValueMap()
				nested.MapVal().InsertString("key", "val")
				attrMap.Insert("nested", nested)
				attrMap.InsertBool("active", true)
				return attrMap
			},
			expected: map[string]interface{}{
				"nested": map[string]interface{}{
					"key": "val",
				},
				"active": true,
			},
		},
	}

	// Act
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			actual := toHumioAttributes(tC.attr())

			assert.Equal(t, tC.expected, actual)
		})
	}
}

func TestToHumioAttributesShaded(t *testing.T) {
	// Arrange
	attrMapA := pdata.NewAttributeMap()
	attrMapA.InsertString("string", "val")
	attrMapA.InsertInt("integer", 42)

	attrMapB := pdata.NewAttributeMap()
	attrMapB.InsertInt("integer", 0)
	attrMapB.InsertString("key", "val")

	expected := map[string]interface{}{
		"string":  "val",
		"integer": int64(0),
		"key":     "val",
	}

	// Act
	actual := toHumioAttributes(attrMapA, attrMapB)

	// Assert
	assert.Equal(t, expected, actual)
}

func TestShutdown(t *testing.T) {
	// Arrange
	exp := newTracesExporter(&Config{}, zap.NewNop(), &clientMock{})

	// Act
	err := exp.shutdown(context.Background())

	// Assert
	require.NoError(t, err)
}
