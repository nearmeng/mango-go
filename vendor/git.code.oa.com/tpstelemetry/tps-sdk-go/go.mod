module git.code.oa.com/tpstelemetry/tps-sdk-go

go 1.14

require (
	git.code.oa.com/tpstelemetry/cgroups v0.1.1
	git.code.oa.com/tpstelemetry/tpstelemetry-protocol v0.0.0-20210614022014-4c430301b7b3
	github.com/golang/protobuf v1.4.3
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/guillermo/go.procmeminfo v0.0.0-20131127224636-be4355a9fb0e
	github.com/hanjm/etcd v0.0.0-20200824100457-c52182889b11
	github.com/json-iterator/go v1.1.10
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/exporters/otlp v0.19.0
	go.opentelemetry.io/otel/exporters/stdout v0.19.0
	go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/trace v0.19.0
	go.uber.org/automaxprocs v1.3.0
	go.uber.org/zap v1.16.0
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/mod v0.4.1 // indirect
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b // indirect
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)

replace (
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/exporters/otlp => go.opentelemetry.io/otel/exporters/otlp v0.19.0
	go.opentelemetry.io/otel/exporters/stdout => go.opentelemetry.io/otel/exporters/stdout v0.19.0
	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v0.19.0
)
