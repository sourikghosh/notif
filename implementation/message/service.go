package message

import (
	"context"
	"encoding/json"
	"fmt"
	"notif/implementation/email"
	"notif/pkg/config"
	"time"

	"github.com/avast/retry-go"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type Service interface {
	SendEmailRequest(e email.Entity) (*nats.PubAck, error)
	RecvEmailRequest(ctx context.Context)
}

type srv struct {
	js       nats.JetStreamContext
	emailSvc email.Service
	log      *zap.SugaredLogger
}

func NewServer(logger *zap.SugaredLogger, jetStream nats.JetStreamContext, emailSvc email.Service) Service {
	return &srv{
		log:      logger,
		js:       jetStream,
		emailSvc: emailSvc,
	}
}

func (s *srv) SendEmailRequest(e email.Entity) (*nats.PubAck, error) {
	eBytes, err := json.Marshal(e)
	if err != nil {
		s.log.Errorf("marshiling failed: %v", err)
		return nil, err
	}

	pub, err := s.js.Publish(fmt.Sprintf("%s.send", config.StreamName), eBytes)
	if err != nil {
		s.log.Errorf("publishing failed: %v", err)
		return nil, err
	}

	return pub, nil
}

func (s *srv) RecvEmailRequest(ctx context.Context) {
	sub, err := s.js.PullSubscribe(fmt.Sprintf("%s.send", config.StreamName), fmt.Sprintf("%s_pullSub", config.StreamName), nats.PullMaxWaiting(128))
	if err != nil {
		s.log.Errorf("subcribing to stream: %s failed with err: %v", config.StreamName, err)
		return
	}

	for {
		select {
		case <-ctx.Done():
		default:
		}

		msgs, err := sub.Fetch(config.NatsBatchSize, nats.MaxWait(30*time.Second))
		if err != nil && err != nats.ErrTimeout {
			s.log.Errorf("failed to fetch msg in batch:%v", err)
		}

		// range over the batch of msgs and sends them using go-smtp
		s.processMsg(ctx, msgs)
	}
}

func (s *srv) processMsg(ctx context.Context, msgs []*nats.Msg) error {
	for i := range msgs {
		if err := msgs[i].Ack(); err != nil {
			s.log.Errorf("ack failed with err: %v", err)
			return err
		}

		var e email.Entity
		err := json.Unmarshal(msgs[i].Data, &e)
		if err != nil {
			s.log.Errorf("unmarshalling msgData failed err: %v", err)
			return err
		}

		if err := retry.Do(func() error {
			return s.emailSvc.SendEmail(ctx, e)
		},
			retry.Attempts(config.SmtpRetryAttempts),
			retry.Delay(config.SmtpRetryDelay),
			retry.Context(ctx),
		); err != nil {
			s.log.Errorf("sending email failed err: %v", err)
			return err
		}

		fmt.Println("successfully send email")
	}

	return nil
}
