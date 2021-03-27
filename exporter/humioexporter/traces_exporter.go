// Copyright 2021, OpenTelemetry Authors
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
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

type humioTracesExporter struct {
	config *Config
	logger *zap.Logger
	client *humioClient
	wg     sync.WaitGroup
}

func newTracesExporter(config *Config, logger *zap.Logger, client *humioClient) *humioTracesExporter {
	return &humioTracesExporter{
		config: config,
		logger: logger,
		client: client,
	}
}

func (e *humioTracesExporter) pushTraceData(ctx context.Context, td pdata.Traces) error {
	e.wg.Add(1)
	defer e.wg.Done()

	// TODO: Transform to Humio event structure

	return e.client.sendStructuredEvents(ctx, nil)
}

func (e *humioTracesExporter) start(context.Context, component.Host) error {
	// TODO: Make test request to ensure Humio connectivity? (Fail fast)
	return nil
}

func (e *humioTracesExporter) shutdown(context.Context) error {
	e.wg.Wait()
	return nil
}

func (e *humioTracesExporter) tracesToHumioEvents() {

}
