// Sending Email Using Smtp in Golang
package email

import (
	"context"
	"net/smtp"

	"notif/pkg/config"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Service interface {
	SendEmail(ctx context.Context, e Entity) error
}

type service struct {
	log    *zap.SugaredLogger
	cfg    *config.NotifConfig
	tracer trace.Tracer
}

func NewEmailService(logger *zap.SugaredLogger, config *config.NotifConfig, t trace.Tracer) Service {
	return &service{
		log:    logger,
		cfg:    config,
		tracer: t,
	}
}

func (s *service) SendEmail(ctx context.Context, e Entity) error {
	_, span := s.tracer.Start(ctx, "sendEmail-func")
	defer span.End()

	auth := smtp.PlainAuth("", s.cfg.EmailSmtpUserName, s.cfg.EmailSmtpPassword, s.cfg.EmailSmtpHost)
	smtpAddr := s.cfg.EmailSmtpHost + ":" + s.cfg.EmailSmtpPORT

	body := []byte(e.Body)
	err := smtp.SendMail(smtpAddr, auth, s.cfg.EmailSmtpUserName, []string{e.ToList[0].EmailAddr}, body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return nil
}
