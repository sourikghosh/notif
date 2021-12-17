package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
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
)

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	// added 1 for nats connection exiting
	wg.Add(1)

	// fetchs configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		fmt.Printf("failed to load config: %s", err.Error())
		os.Exit(1)
	}

	// setting up logger with the config
	zapLogger := logger.NewLogger(cfg)

	// Creating the exporter exporter
	exp, err := jaeger.New(jaeger.WithAgentEndpoint())
	if err != nil {
		zapLogger.Fatalf("jaeger exported creation failed: %s", err.Error())
	}

	// Defining resource attributes
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("notif-svc"),
		semconv.ServiceVersionKey.String("1.0.0"),
		attribute.Int64("ID", 1),
	)

	// Create the trace provider with the exporter and resources
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource),
	)

	// b3 propagator initilizes
	propagator := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader))
	tracer := provider.Tracer("notifSvc")

	// setting few basic nats opts and connecting to nats
	opts := natshelper.SetupConnOptions(zapLogger, &wg)
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
	svc := message.NewMessageService(zapLogger, js, emailSvc, tracer, propagator)
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
	go func(svc message.Service, ctx context.Context, conn *nats.Conn, wg *sync.WaitGroup) {
		// for the subscriber
		wg.Add(1)
		svc.RecvEmailRequest(ctx, wg)

		zapLogger.Info("subscriber returned")
		// closing the connection because subscriber returned
		conn.Close()
	}(svc, ctx, natsConn, &wg)

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
	defer cancel()

	// gracefully shutdown http server
	if err := server.Shutdown(ctxWithTimeOut); err != nil {
		cancel()
		zapLogger.Warnf("Server forced to shutdown: %s", err.Error())
	}

	// gracefully shutdown provider
	if err := provider.Shutdown(ctxWithTimeOut); err != nil {
		zapLogger.Warn(err)
	}

	// cancel the ctx to stop the pullSubscriber close the nats connection
	cancelCtx()

	// wait till the nats connection is closed and pullSubscriber returned
	wg.Wait()

	zapLogger.Info("application exited")
}
