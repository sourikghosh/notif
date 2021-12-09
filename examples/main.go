package main

import (
	"fmt"
	"log"
	"notif/implementation/email"
	"notif/implementation/message"
	"notif/pkg/config"
	natshelper "notif/pkg/nats"
	"strings"

	logger "notif/pkg/log"

	"github.com/matcornic/hermes/v2"
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

	toList := make([]email.NameAddr, 1)

	toList[0].EmailAddr = cfg.Emailtest
	toList[0].UserName = "Official Sourik"

	for i := 0; i < 4; i++ {
		e := email.Entity{
			FromName: "Bhalo checle",
			ToList:   toList,
			Subject:  "test mail via smtp with js" + fmt.Sprintf("%d", i),
		}

		e.Body = prepareEmail()
		e.Body = BuildMessage(e, *cfg)

		emailSvc := email.NewEmailService(zapLogger, cfg)
		svc := message.NewServer(zapLogger, js, emailSvc)

		pub, err := svc.SendEmailRequest(e)
		if err != nil {
			zapLogger.Errorf(err.Error())
			break
		}

		zapLogger.Infof("published: %v", pub)
	}

}

func BuildMessage(e email.Entity, cfg config.NotifConfig) string {
	tolist := make([]string, len(e.ToList))
	for i := range e.ToList {
		tolist = append(tolist, e.ToList[i].EmailAddr)
	}

	msg := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	msg += fmt.Sprintf("From: %s <%s>\r\n", e.FromName, cfg.EmailSmtpUserName)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(tolist, ";"))
	msg += fmt.Sprintf("Subject: %s\r\n", e.Subject)
	msg += fmt.Sprintf("\r\n%s\r\n", e.Body)

	return msg
}

func prepareEmail() string {
	h := hermes.Hermes{
		Theme: new(hermes.Default),
		Product: hermes.Product{
			Copyright: "Copyright © 2021 Notif. All rights reserved.",
			Name:      "Notif",
			Logo:      "https://storage.googleapis.com/gopherizeme.appspot.com/gophers/8f42d0b66299ee3f295d8299da93b8c60fba2bd8.png",
		},
	}

	email := hermes.Email{
		Body: hermes.Body{
			Greeting: "Hey",
			Name:     "Heyy.....",
			Intros: []string{
				"This is a test msg to check if things work.",
			},
			Outros: []string{
				"Need help, or have questions? Just reply to this email, we'd love to help.",
			},
			Signature: "Thanks",
		},
	}

	bodyStr, err := h.GenerateHTML(email)
	if err != nil {
		return ""
	}

	return bodyStr
}
