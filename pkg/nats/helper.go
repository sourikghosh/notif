package natshelper

import (
	"fmt"
	"notif/pkg/config"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func SetupConnOptions(log *zap.SugaredLogger, wg *sync.WaitGroup) []nats.Option {
	opts := make([]nats.Option, 0)
	// Buffering Messages During Reconnect Attempts
	opts = append(opts, nats.ReconnectBufSize(5*1024*1024))
	// Set reconnect interval
	opts = append(opts, nats.ReconnectWait(config.NatsReconnectDelay))
	// Set max reconnects attempts
	opts = append(opts, nats.MaxReconnects(int(config.NatsReconnectTotalWait/config.NatsReconnectDelay)))

	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Infof("Reconnected [%s]", nc.ConnectedUrl())
	}))

	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		// done when nats is closed
		wg.Done()
		log.Infof("Exiting: %v", nc.LastError())
	}))

	return opts
}

func CreateStream(js nats.JetStreamContext, log *zap.SugaredLogger) (err error) {
	stream, _ := js.StreamInfo(config.StreamName)
	if stream == nil {
		subj := fmt.Sprintf("%s.*", config.StreamName)
		log.Debugf("creating stream %q and subjects %q", config.StreamName, subj)

		if _, err = js.AddStream(&nats.StreamConfig{
			Name:        config.StreamName,
			Description: "notification stream",
			Subjects:    []string{subj},
			Retention:   nats.WorkQueuePolicy,
			Discard:     nats.DiscardOld,
			MaxAge:      24 * time.Hour,
			Storage:     nats.FileStorage,
		}); err != nil {
			return
		}
	}

	return nil
}
