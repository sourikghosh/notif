package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"net/http"
	"notif/implementation/email"
	"notif/pkg"
	"notif/pkg/config"
	"sync"

	"github.com/avast/retry-go"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Service interface {
	SendEmailRequest(ctx context.Context, e email.Entity) (*nats.PubAck, error)
	RecvEmailRequest(ctx context.Context, wg *sync.WaitGroup)
}

type messageSvc struct {
	js          nats.JetStreamContext
	emailSvc    email.Service
	log         *zap.SugaredLogger
	tracer      trace.Tracer
	propagators propagation.TextMapPropagator
}

func NewMessageService(
	l *zap.SugaredLogger, jetStream nats.JetStreamContext,
	e email.Service, t trace.Tracer, p propagation.TextMapPropagator) Service {
	return &messageSvc{
		log:         l,
		js:          jetStream,
		emailSvc:    e,
		tracer:      t,
		propagators: p,
	}
}

func (s *messageSvc) SendEmailRequest(ctx context.Context, e email.Entity) (*nats.PubAck, error) {
	// starting span for publishing the msg
	spanCtx, span := s.tracer.Start(ctx, "message.svc-publish")
	defer span.End()

	// extracting traceID for logging purpose
	traceID := span.SpanContext().TraceID().String()

	// marshalling email struct to send as msg data
	eBytes, err := json.Marshal(e)
	if err != nil {
		s.errLogWithSpanAttributes("marshiling failed", traceID, err, span)

		return nil, pkg.NotifErr{
			Code: http.StatusBadRequest,
			Err:  err,
		}
	}

	// prepare nats msg with data and headers and
	// injecting the current traceID into the msg headers
	header := make(nats.Header)
	s.propagators.Inject(spanCtx, propagation.HeaderCarrier(header))
	m := &nats.Msg{
		Subject: fmt.Sprintf("%s.send", config.StreamName),
		Header:  header,
		Data:    eBytes,
	}

	// publishing the msg
	pub, err := s.js.PublishMsg(m)
	if err != nil {
		s.errLogWithSpanAttributes("publishing failed", traceID, err, span)
		return nil, err
	}

	return pub, nil
}

func (s *messageSvc) RecvEmailRequest(ctx context.Context, wg *sync.WaitGroup) {
	// for the subscriber
	wg.Add(1)
	// preparing args for new consumer
	subj := fmt.Sprintf("%s.send", config.StreamName)
	durableName := fmt.Sprintf("%s_pullSub", config.StreamName)

	// creating a new pull based consumer
	sub, err := s.js.PullSubscribe(subj, durableName, nats.PullMaxWaiting(128))
	if err != nil {
		s.log.Errorf("subcribing to stream: %s failed with err: %v", config.StreamName, err)
		wg.Done()

		return
	}

	s.log.Infof("subscriber added to stream : %s of name: %s", config.StreamName, durableName)

	// iterate over till ctx is not done
	for {
		select {
		case <-ctx.Done():
			wg.Done()
			return

		default:
		}

		// fecthing msgs in batch till context deadline or timeout
		msgs, err := sub.Fetch(config.NatsBatchSize, nats.MaxWait(config.NatsSubMaxWait))
		if err != nil && !errors.Is(err, nats.ErrTimeout) {
			s.log.Errorf("failed to fetch msg in batch:%v", err)
			continue
		}

		// range over the batch of msgs and sends them using go-smtp
		s.eventProcessor(ctx, msgs)
	}
}

func (s *messageSvc) eventProcessor(ctx context.Context, msgs []*nats.Msg) {
	for i := range msgs {
		// Extracts the trace from msg header and creates a span for processing.
		var span trace.Span
		spanCtx := s.propagators.Extract(ctx, propagation.HeaderCarrier(msgs[i].Header))
		spanCtx, span = s.tracer.Start(spanCtx, "message.svc-subscribe.eventProcessor")

		// extracting traceID for logging purpose
		traceID := span.SpanContext().TraceID().String()

		if err := msgs[i].Ack(); err != nil {
			s.errLogWithSpanAttributes("ack failed", traceID, err, span)
			continue
		}

		var e email.Entity
		err := json.Unmarshal(msgs[i].Data, &e)
		if err != nil {
			s.errLogWithSpanAttributes("unmarshalling msgData failed", traceID, err, span)
			continue
		}

		// Send email service retry logic with maxAttempt and delay between each attempt
		err = retry.Do(func() error {
			return s.emailSvc.SendEmail(spanCtx, e)
		}, retry.Attempts(config.SmtpRetryAttempts),
			retry.Delay(config.SmtpRetryDelay),
			retry.Context(spanCtx),
		)

		if err != nil {
			s.errLogWithSpanAttributes("sending email failed", traceID, err, span)
			continue
		}

		s.log.Info("successfully send email", zap.String("traceID", traceID))
		span.End()
	}
}

func (s *messageSvc) errLogWithSpanAttributes(msg, traceID string, err error, span trace.Span) {
	s.log.Errorf(msg+"err: %v", err, zap.String("traceID", traceID))
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
