package config

import "time"

const (
	Development string = "dev"
	Production  string = "prod"
	StreamName  string = "NOTIFS"
)

var (
	NatsTotalWait              = 10 * time.Second
	NatsReconnectDelay         = time.Second
	NatsBatchSize              = 5
	SmtpRetryAttempts     uint = 3
	SmtpRetryDelay             = 2 * time.Second
	HttpTimeOut                = 5 * time.Second
	ServerShutdownTimeOut      = 10 * time.Second
)
