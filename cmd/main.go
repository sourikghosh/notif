package main

import (
	"context"
	"log"

	"notif/implementation/email"
	"notif/implementation/message"
	"notif/pkg/config"

	logger "notif/pkg/log"
	natshelper "notif/pkg/nats"

	"github.com/nats-io/nats.go"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("failed to load config: %v", err.Error())
	}

	zapLogger := logger.NewLogger(cfg)

	opts := natshelper.SetupConnOptions(zapLogger)
	natsConn, err := nats.Connect(nats.DefaultURL, opts...)
	if err != nil {
		zapLogger.Fatalf("nats connection failed: %v", err.Error())
	}

	js, err := natsConn.JetStream()
	if err != nil {
		zapLogger.Fatalf("nats-js connection failed: %v", err.Error())
	}

	if err := natshelper.CreateStream(js, zapLogger); err != nil {
		zapLogger.Fatalf("nats-js stream creation failed: %v", err.Error())
	}

	emailSvc := email.NewEmailService(zapLogger, cfg)
	svc := message.NewServer(zapLogger, js, emailSvc)

	svc.RecvEmailRequest(context.Background())
}
