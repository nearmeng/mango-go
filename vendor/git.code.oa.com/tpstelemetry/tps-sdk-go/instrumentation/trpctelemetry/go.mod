module git.code.oa.com/tpstelemetry/tps-sdk-go/instrumentation/trpctelemetry

go 1.14

require (
	git.code.oa.com/tpstelemetry/tps-sdk-go v0.4.16
	git.code.oa.com/trpc-go/trpc-database/kafka v0.1.8
	git.code.oa.com/trpc-go/trpc-go v0.4.2
	github.com/Shopify/sarama v1.29.0
	github.com/golang/protobuf v1.4.3
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hanjm/etcd v0.0.0-20200824100457-c52182889b11
	github.com/json-iterator/go v1.1.10
	github.com/modern-go/reflect2 v1.0.1
	github.com/mozillazg/go-pinyin v0.18.0
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.10.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/exporters/otlp v0.19.0
	go.opentelemetry.io/otel/metric v0.19.0
	go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/trace v0.19.0
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace git.code.oa.com/tpstelemetry/tps-sdk-go => ../../

replace (
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/exporters/otlp => go.opentelemetry.io/otel/exporters/otlp v0.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlpgrpc => go.opentelemetry.io/otel/exporters/otlp/otlpgrpc v0.19.0
	go.opentelemetry.io/otel/exporters/stdout => go.opentelemetry.io/otel/exporters/stdout v0.19.0
	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v0.19.0
)
