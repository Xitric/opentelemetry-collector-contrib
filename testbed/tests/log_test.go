// Copyright 2019, OpenTelemetry Authors
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

// Package tests contains test cases. To run the tests go to tests directory and run:
// RUN_TESTBED=1 go test -v

package tests

import (
	"testing"

	"go.opentelemetry.io/collector/testbed/testbed"
	scenarios "go.opentelemetry.io/collector/testbed/tests"

	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/datareceivers"
	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/datasenders"
)

func TestLog10kDPS(t *testing.T) {
	flw := datasenders.NewFluentBitFileLogWriter(testbed.DefaultHost, testbed.GetAvailablePort(t))
	tests := []struct {
		name         string
		sender       testbed.DataSender
		receiver     testbed.DataReceiver
		resourceSpec testbed.ResourceSpec
		extensions   map[string]string
	}{
		{
			name:     "OTLP",
			sender:   testbed.NewOTLPLogsDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t)),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 82,
			},
		},
		{
			name:     "filelog",
			sender:   datasenders.NewFileLogWriter(),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 85,
			},
		},
		{
			name:     "filelog checkpoints",
			sender:   datasenders.NewFileLogWriter(),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 85,
			},
			extensions: datasenders.NewLocalFileStorageExtension(),
		},
		{
			name:     "kubernetes containers",
			sender:   datasenders.NewKubernetesContainerWriter(),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "k8s CRI-Containerd",
			sender:   datasenders.NewKubernetesCRIContainerdWriter(),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "k8s CRI-Containerd no attr ops",
			sender:   datasenders.NewKubernetesCRIContainerdNoAttributesOpsWriter(),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "CRI-Containerd",
			sender:   datasenders.NewCRIContainerdWriter(),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "syslog-udp",
			sender:   datasenders.NewSyslogWriter("udp", testbed.DefaultHost, testbed.GetAvailablePort(t), 1),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "syslog-tcp-batch-1",
			sender:   datasenders.NewTCPUDPWriter("tcp", testbed.DefaultHost, testbed.GetAvailablePort(t), 1),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "syslog-tcp-batch-100",
			sender:   datasenders.NewTCPUDPWriter("tcp", testbed.DefaultHost, testbed.GetAvailablePort(t), 100),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "FluentBitToOTLP",
			sender:   flw,
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 155,
			},
			extensions: flw.Extensions(),
		},
		{
			name:     "FluentForward-SplunkHEC",
			sender:   datasenders.NewFluentLogsForwarder(t, testbed.GetAvailablePort(t)),
			receiver: datareceivers.NewSplunkHECDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "tcp-batch-1",
			sender:   datasenders.NewTCPUDPWriter("tcp", testbed.DefaultHost, testbed.GetAvailablePort(t), 1),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "tcp-batch-100",
			sender:   datasenders.NewTCPUDPWriter("tcp", testbed.DefaultHost, testbed.GetAvailablePort(t), 100),
			receiver: testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 150,
			},
		},
	}

	processors := map[string]string{
		"batch": `
  batch:
`,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scenarios.Scenario10kItemsPerSecond(
				t,
				test.sender,
				test.receiver,
				test.resourceSpec,
				contribPerfResultsSummary,
				processors,
				test.extensions,
			)
		})
	}
}
