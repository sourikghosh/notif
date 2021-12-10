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

	logger "notif/pkg/log"
	natshelper "notif/pkg/nats"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func main() {
	// fetching config
	cfg, err := config.LoadConfig(".")
	if err != nil {
		fmt.Printf("failed to load config: %v", err.Error())
		os.Exit(1)
	}

	// setting up logger with the config
	zapLogger := logger.NewLogger(cfg)

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

	emailSvc := email.NewEmailService(zapLogger, cfg)
	svc := message.NewServer(zapLogger, js, emailSvc)
	end := endpoints.MakeEndpoints(svc)
	h := httpTransport.NewHTTPService(end)
	// svc.RecvEmailRequest(context.Background())

	// creating server with timeout and assigning the routes
	server := &http.Server{
		Addr:              ":" + cfg.PORT,
		ReadHeaderTimeout: config.HttpTimeOut,
		ReadTimeout:       config.HttpTimeOut,
		WriteTimeout:      config.HttpTimeOut,
		IdleTimeout:       config.HttpTimeOut,
		Handler:           h,
	}

	// start subscribing for notif events
	go func(svc message.Service) {
		svc.RecvEmailRequest(context.Background())
	}(svc)

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

	ctx, cancel := context.WithTimeout(context.Background(), config.ServerShutdownTimeOut)
	defer cleanUp(cancel, zapLogger)

	if err := server.Shutdown(ctx); err != nil {
		cleanUp(cancel, zapLogger)
		zapLogger.Fatalf("Server forced to shutdown: %s", err.Error())
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
