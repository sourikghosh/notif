package message

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"notif/implementation/email"
	"notif/pkg"
	"notif/pkg/config"
	"time"

	"github.com/avast/retry-go"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Service interface {
	SendEmailRequest(ctx context.Context, e email.Entity) (*nats.PubAck, error)
	RecvEmailRequest(ctx context.Context)
}

type srv struct {
	js          nats.JetStreamContext
	emailSvc    email.Service
	log         *zap.SugaredLogger
	tracer      trace.Tracer
	propagators propagation.TextMapPropagator
}

func NewServer(l *zap.SugaredLogger, jetStream nats.JetStreamContext, e email.Service, t trace.Tracer, p propagation.TextMapPropagator) Service {
	return &srv{
		log:         l,
		js:          jetStream,
		emailSvc:    e,
		tracer:      t,
		propagators: p,
	}
}

func (s *srv) SendEmailRequest(ctx context.Context, e email.Entity) (*nats.PubAck, error) {
	spanCtx, span := s.tracer.Start(ctx, "message.svc-sendEmailReq")
	defer span.End()

	span.AddEvent("marshalling notif-event")
	eBytes, err := json.Marshal(e)
	if err != nil {
		s.log.Errorf("marshiling failed: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, pkg.NotifErr{
			Code: http.StatusBadRequest,
			Err:  err,
		}
	}

	// prepare nats msg with data and headers
	header := make(nats.Header)
	s.propagators.Inject(spanCtx, propagation.HeaderCarrier(header))
	m := &nats.Msg{
		Subject: fmt.Sprintf("%s.send", config.StreamName),
		Header:  header,
		Data:    eBytes,
	}

	span.AddEvent("msg published into stream")
	pub, err := s.js.PublishMsg(m)
	if err != nil {
		s.log.Errorf("publishing failed: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

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
		var span trace.Span
		spanCtx := s.propagators.Extract(ctx, propagation.HeaderCarrier(msgs[i].Header))
		spanCtx, span = s.tracer.Start(spanCtx, "message.svc-RecvEmailRequest.processMsg")

		if err := msgs[i].Ack(); err != nil {
			s.log.Errorf("ack failed with err: %v", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return err
		}

		var e email.Entity
		err := json.Unmarshal(msgs[i].Data, &e)
		if err != nil {
			s.log.Errorf("unmarshalling msgData failed err: %v", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return err
		}

		if err := retry.Do(func() error {
			return s.emailSvc.SendEmail(spanCtx, e)
		},
			retry.Attempts(config.SmtpRetryAttempts),
			retry.Delay(config.SmtpRetryDelay),
			retry.Context(spanCtx),
		); err != nil {
			s.log.Errorf("sending email failed err: %v", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return err
		}

		fmt.Println("successfully send email")
		span.End()
	}

	return nil
}
