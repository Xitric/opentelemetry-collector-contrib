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

package datasenders

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/humioexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/testbed/testbed"
	"go.uber.org/zap"
)

// TODO: Try to intercept the consume traces call, and log from there to see if that even works?
// The problem is actually that we skip all spans that lack a service name. We have to disable that feature for performance testing, lol
type HumioDataSender struct {
	testbed.DataSenderBase
	t      consumer.Traces
	logger *zap.Logger
	// client humioexporter.ExporterClient
}

func NewHumioDataSender(port int) *HumioDataSender {
	// factory := humioexporter.NewFactory()
	// cfg := factory.CreateDefaultConfig().(*humioexporter.Config)
	// cfg.HTTPClientSettings = confighttp.HTTPClientSettings{
	// 	Endpoint: "https://cloud.humio.com/",
	// 	TLSSetting: configtls.TLSClientSetting{
	// 		InsecureSkipVerify: true,
	// 	},
	// }
	// cfg.Logs = humioexporter.LogsConfig{
	// 	IngestToken: "052c2230-f26a-40bf-8b6e-d8375b936af3",
	// }
	// cfg.Sanitize()

	// client, err := humioexporter.NewHumioClient(cfg, zap.L())
	// if err != nil {
	// 	panic("Shit")
	// }

	return &HumioDataSender{
		DataSenderBase: testbed.DataSenderBase{
			Port: port,
			Host: testbed.DefaultHost,
		},
		logger: zap.L(),
		// client: client,
	}
}

func (hs *HumioDataSender) Start() error {
	factory := humioexporter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*humioexporter.Config)
	cfg.HTTPClientSettings = confighttp.HTTPClientSettings{
		Endpoint: fmt.Sprintf("http://%s/", hs.GetEndpoint()),
	}
	cfg.Traces = humioexporter.TracesConfig{
		IngestToken: "token",
	}
	cfg.DisableCompression = true

	param := component.ExporterCreateParams{
		Logger: hs.logger,
	}
	humio, err := factory.CreateTracesExporter(context.Background(), param, cfg)
	if err != nil {
		return err
	}

	hs.t = humio
	return nil
}

func (hs *HumioDataSender) GenConfigYAMLStr() string {
	// This creates a config for a receiver. But since we are using a mock
	// receiver, this is actually unused
	return fmt.Sprintf(`
  humio:
    endpoint: "%s"`, hs.GetEndpoint())
}

func (hs *HumioDataSender) ProtocolName() string {
	return "humio"
}

func (hs *HumioDataSender) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	hs.logger.Warn("Oh yeah, send that stuffz!")
	// err := hs.client.SendUnstructuredEvents(context.Background(), []*humioexporter.HumioUnstructuredEvents{
	// 	{
	// 		Fields: map[string]string{
	// 			"source": "load_test",
	// 		},
	// 		Messages: []string{
	// 			"ConsumeTraces received traces to send",
	// 		},
	// 	},
	// })
	// if err != nil {
	// 	panic(err.Error())
	// }
	// hs.client.SendUnstructuredEvents(context.Background(), []*humioexporter.HumioUnstructuredEvents{
	// 	{
	// 		Fields: map[string]string{
	// 			"source": "load_test",
	// 		},
	// 		Messages: []string{
	// 			fmt.Sprintf("Consumer: %s", reflect.TypeOf(hs.t)),
	// 		},
	// 	},
	// })
	err := hs.t.ConsumeTraces(ctx, td)
	if err != nil {
		hs.logger.Error(err.Error())
	}
	return err
}
