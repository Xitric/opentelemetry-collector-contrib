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

package datareceivers

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/mockdatareceivers/mockhumioreceiver"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/testbed/testbed"
	"go.uber.org/zap"
)

type HumioDataReceiver struct {
	testbed.DataReceiverBase
	component.TracesReceiver
}

func NewHumioDataReceiver(port int) *HumioDataReceiver {
	return &HumioDataReceiver{
		DataReceiverBase: testbed.DataReceiverBase{
			Port: port,
		},
	}
}

func (hr *HumioDataReceiver) Start(tc consumer.Traces, _ consumer.Metrics, _ consumer.Logs) error {
	factory := mockhumioreceiver.NewFactory()
	cfg := factory.CreateDefaultConfig().(*mockhumioreceiver.Config)
	cfg.HTTPServerSettings = confighttp.HTTPServerSettings{
		Endpoint: fmt.Sprintf("localhost:%d", hr.Port),
	}

	param := component.ReceiverCreateParams{
		Logger: zap.L(),
	}

	humio, err := factory.CreateTracesReceiver(context.Background(), param, cfg, tc)
	if err != nil {
		return err
	}

	hr.TracesReceiver = humio
	return hr.TracesReceiver.Start(context.Background(), hr)
}

func (hr *HumioDataReceiver) Stop() error {
	if hr.TracesReceiver != nil {
		return hr.TracesReceiver.Shutdown(context.Background())
	}
	return nil
}

func (hr *HumioDataReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  humio:
    endpoint: http://localhost:%d/
    disable_compression: true
    traces:
      ingest_token: token`, hr.Port)
}

func (hr *HumioDataReceiver) ProtocolName() string {
	return "humio"
}
