package config

import "time"

const (
	Development string = "dev"
	Production  string = "prod"
	StreamName  string = "NOTIFS"
)

var (
	NatsTotalWait           = 10 * time.Second
	NatsReconnectDelay      = time.Second
	NatsBatchSize           = 2
	SmtpRetryAttempts  uint = 3
	SmtpRetryDelay          = 2 * time.Second
)
