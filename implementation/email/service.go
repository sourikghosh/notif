// Sending Email Using Smtp in Golang
package email

import (
	"context"
	"net/smtp"

	"notif/pkg/config"

	"go.uber.org/zap"
)

type Service interface {
	SendEmail(ctx context.Context, e Entity) error
}

type service struct {
	log *zap.SugaredLogger
	cfg *config.NotifConfig
}

func NewEmailService(logger *zap.SugaredLogger, config *config.NotifConfig) Service {
	return &service{
		log: logger,
		cfg: config,
	}
}

func (s *service) SendEmail(ctx context.Context, e Entity) error {
	auth := smtp.PlainAuth("", s.cfg.EmailSmtpUserName, s.cfg.EmailSmtpPassword, s.cfg.EmailSmtpHost)
	smtpAddr := s.cfg.EmailSmtpHost + ":" + s.cfg.EmailSmtpPORT

	body := []byte(e.Body)
	return smtp.SendMail(smtpAddr, auth, s.cfg.EmailSmtpUserName, []string{e.ToList[0].EmailAddr}, body)
}
