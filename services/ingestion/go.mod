module github.com/activeprospect/multi-tenant-billing/ingestion

go 1.22

require (
	github.com/IBM/sarama v1.43.1
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.5.1
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0
	go.opentelemetry.io/otel/propagation v1.24.0
	go.opentelemetry.io/otel/sdk v1.24.0
	go.opentelemetry.io/otel/trace v1.24.0
	google.golang.org/grpc v1.62.1
)
