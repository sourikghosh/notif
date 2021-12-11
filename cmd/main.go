package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"notif/implementation/email"
	"notif/implementation/message"
	"notif/pkg/config"
	"notif/transport/endpoints"
	httpTransport "notif/transport/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	logger "notif/pkg/log"
	natshelper "notif/pkg/nats"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	// fetching config
	cfg, err := config.LoadConfig(".")
	if err != nil {
		fmt.Printf("failed to load config: %s", err.Error())
		os.Exit(1)
	}

	// setting up logger with the config
	zapLogger := logger.NewLogger(cfg)

	// Create the exporter
	exp, err := jaeger.New(jaeger.WithAgentEndpoint())
	if err != nil {
		zapLogger.Fatalf("jaeger exported creation failed: %s", err.Error())
	}

	// Define resource attributes
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("notif-svc"),
		semconv.ServiceVersionKey.String("1.0.0"),
		attribute.Int64("ID", 1),
	)

	// Create the trace provider with the exporter and resources
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp), // Always be sure to batch in production.
		sdktrace.WithResource(resource),
	)

	propagator := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader))
	// propagator.Extract()
	// propagator.Inject(ctx,)
	tracer := provider.Tracer("notifSvc")

	// setting few basic nats opts
	opts := natshelper.SetupConnOptions(zapLogger)

	// connecting to nats with the following opts
	natsConn, err := nats.Connect(nats.DefaultURL, opts...)
	if err != nil {
		zapLogger.Fatalf("nats connection failed: %v", err.Error())
	}

	// creating jetStream from natsConn
	js, err := natsConn.JetStream()
	if err != nil {
		zapLogger.Fatalf("nats-js connection failed: %v", err.Error())
	}

	// creating the notification stream for event processing
	if err := natshelper.CreateStream(js, zapLogger); err != nil {
		zapLogger.Fatalf("nats-js stream creation failed: %v", err.Error())
	}

	emailSvc := email.NewEmailService(zapLogger, cfg, tracer)
	svc := message.NewServer(zapLogger, js, emailSvc, tracer, propagator)
	end := endpoints.MakeEndpoints(svc, tracer)
	h := httpTransport.NewHTTPService(end, tracer)

	// creating server with timeout and assigning the routes
	server := &http.Server{
		Addr:         ":" + cfg.PORT,
		ReadTimeout:  config.HttpTimeOut,
		WriteTimeout: config.HttpTimeOut,
		IdleTimeout:  config.HttpTimeOut,
		Handler: otelhttp.NewHandler(
			h,
			"http.server",
			otelhttp.WithPropagators(propagator),
		),
	}

	// start subscribing for notif events
	go func(svc message.Service, ctx context.Context) {
		svc.RecvEmailRequest(ctx)
	}(svc, ctx)

	// start listening and serving http server
	go func() {
		zapLogger.Infof("🚀 HTTP server running on port %v\n", cfg.PORT)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zapLogger.Errorf("Err occurred:%v", err.Error())
		}
	}()

	// listening for system events to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zapLogger.Infof("Signal received to Shutdown server...")

	ctxWithTimeOut, cancel := context.WithTimeout(ctx, config.ServerShutdownTimeOut)
	defer cleanUp(cancel, zapLogger)

	if err := server.Shutdown(ctxWithTimeOut); err != nil {
		cleanUp(cancel, zapLogger)
		zapLogger.Warnf("Server forced to shutdown: %s", err.Error())
	}

	if err := provider.Shutdown(ctx); err != nil {
		zapLogger.Warn(err)
	}
}

// cleanUp cleans or rather releases all resources
func cleanUp(cancel context.CancelFunc, log *zap.SugaredLogger) {
	var successBool bool
	cancel()

	if successBool {
		log.Infof("Server exited successfully")
	}
}
