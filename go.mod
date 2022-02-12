module notif

go 1.16

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/gin-gonic/gin v1.7.7
	github.com/go-playground/validator/v10 v10.9.0
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/matcornic/hermes/v2 v2.1.0
	github.com/nats-io/nats-server/v2 v2.7.2 // indirect
	github.com/nats-io/nats.go v1.13.1-0.20220121202836-972a071d373d
	github.com/spf13/viper v1.9.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.28.0
	go.opentelemetry.io/contrib/propagators/b3 v1.3.0
	go.opentelemetry.io/otel v1.3.0
	go.opentelemetry.io/otel/exporters/jaeger v1.3.0
	go.opentelemetry.io/otel/sdk v1.3.0
	go.opentelemetry.io/otel/trace v1.3.0
	go.uber.org/zap v1.17.0
)
