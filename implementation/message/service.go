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
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	jaegerConfig "github.com/uber/jaeger-client-go/config"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Service interface {
	SendEmailRequest(ctx context.Context, e email.Entity) (*nats.PubAck, error)
	RecvEmailRequest(ctx context.Context)
}

type srv struct {
	js       nats.JetStreamContext
	emailSvc email.Service
	log      *zap.SugaredLogger
	tracer   trace.Tracer
}

func NewServer(l *zap.SugaredLogger, jetStream nats.JetStreamContext, e email.Service, t trace.Tracer) Service {
	return &srv{
		log:      l,
		js:       jetStream,
		emailSvc: e,
		tracer:   t,
	}
}

func (s *srv) SendEmailRequest(ctx context.Context, e email.Entity) (*nats.PubAck, error) {
	spanCtx, span := s.tracer.Start(ctx, "message.svc-sendEmailReq")
	defer span.End()

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

	header := make(nats.Header)
	childSpan, _ := opentracing.StartSpanFromContext(spanCtx, "publish-msg")
	childSpan.Tracer().Inject(
		childSpan.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(header),
	)

	m := &nats.Msg{
		Subject: fmt.Sprintf("%s.send", config.StreamName),
		Header:  header,
		Data:    eBytes,
	}

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
		cfg := &jaegerConfig.Configuration{
			ServiceName: "notif-svc",

			// "const" sampler is a binary sampling strategy: 0=never sample, 1=always sample.
			Sampler: &jaegerConfig.SamplerConfig{
				Type:  "const",
				Param: 1,
			},

			// Log the emitted spans to stdout.
			Reporter: &jaegerConfig.ReporterConfig{
				LogSpans:           true,
				LocalAgentHostPort: "localhost:6831",
			},
		}

		tracer, closer, err := cfg.NewTracer()
		if err != nil {
			panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
		}
		defer closer.Close()

		if err := msgs[i].Ack(); err != nil {
			s.log.Errorf("ack failed with err: %v", err)
			return err
		}

		spanCtx, _ := tracer.Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(msgs[i].Header),
		)

		span := tracer.StartSpan("message.svc-RecvEmailSend", ext.RPCServerOption(spanCtx))
		sCtx := opentracing.ContextWithSpan(context.Background(), span)
		var e email.Entity
		err = json.Unmarshal(msgs[i].Data, &e)
		if err != nil {
			s.log.Errorf("unmarshalling msgData failed err: %v", err)
			return err
		}

		if err := retry.Do(func() error {
			return s.emailSvc.SendEmail(sCtx, e)
		},
			retry.Attempts(config.SmtpRetryAttempts),
			retry.Delay(config.SmtpRetryDelay),
			retry.Context(sCtx),
		); err != nil {
			s.log.Errorf("sending email failed err: %v", err)
			return err
		}

		fmt.Println("successfully send email")
		span.Finish()
	}

	return nil
}
