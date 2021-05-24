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

package mockhumioreceiver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/humioexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.uber.org/zap"
)

type humioTracesReceiver struct {
	sync.Mutex
	cfg            *Config
	logger         *zap.Logger
	tracesConsumer consumer.Traces
	server         *http.Server
}

func NewTracesReceiver(
	cfg *Config,
	logger *zap.Logger,
	nextConsumer consumer.Traces,
) (component.TracesReceiver, error) {
	if nextConsumer == nil {
		return nil, errors.New("missing required consumer for traces")
	}

	return &humioTracesReceiver{
		cfg:            cfg,
		logger:         logger,
		tracesConsumer: nextConsumer,
	}, nil
}

func (r *humioTracesReceiver) Start(ctx context.Context, host component.Host) error {
	r.Lock()
	defer r.Unlock()

	listener, err := r.cfg.HTTPServerSettings.ToListener()
	if err != nil {
		return err
	}

	mx := mux.NewRouter()
	mx.NewRoute().HandlerFunc(r.handleRequest)

	r.server = r.cfg.HTTPServerSettings.ToServer(mx)

	go func() {
		if errHTTP := r.server.Serve(listener); errHTTP != http.ErrServerClosed {
			host.ReportFatalError(errHTTP)
		}
	}()

	return nil
}

func (r *humioTracesReceiver) Shutdown(ctx context.Context) error {
	r.Lock()
	defer r.Unlock()

	return r.server.Close()
}

func (r *humioTracesReceiver) handleRequest(resp http.ResponseWriter, req *http.Request) {
	ctx := obsreport.ReceiverContext(req.Context(), r.cfg.ID().String(), "http")

	decoder := json.NewDecoder(req.Body)
	var events []*humioexporter.HumioStructuredEvents

	// Read through the array of structured events one group at a time
	for decoder.More() {
		var group []*humioexporter.HumioStructuredEvents
		err := decoder.Decode(&group)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			return
		}

		events = append(events, group...)
	}

	// Then record the spans we received
	spans := 0
	for _, group := range events {
		spans += len(group.Events)
	}

	td := pdata.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ils := rs.InstrumentationLibrarySpans().AppendEmpty()
	ils.Spans().Resize(spans)

	err := r.tracesConsumer.ConsumeTraces(ctx, td)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
	}
}
