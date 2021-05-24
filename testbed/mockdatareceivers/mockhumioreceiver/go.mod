module github.com/open-telemetry/opentelemetry-collector-contrib/testbed/mockdatareceivers/mockhumioreceiver

go 1.15

require (
	github.com/gorilla/mux v1.8.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/humioexporter v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/collector v0.26.0
	go.uber.org/zap v1.16.0
)

replace github.com/open-telemetry/opentelemetry-collector-contrib/exporter/humioexporter => ../../../exporter/humioexporter
