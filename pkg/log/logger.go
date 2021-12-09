package log

import (
	"notif/pkg/config"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(cfg *config.NotifConfig) *zap.SugaredLogger {
	logLevel := zaplogLevel(cfg)

	logWriter := zapcore.AddSync(os.Stderr)

	var encoderCfg zapcore.EncoderConfig
	if cfg.Mode == config.Development {
		encoderCfg = zap.NewDevelopmentEncoderConfig()
	} else {
		encoderCfg = zap.NewProductionEncoderConfig()
	}

	var encoder zapcore.Encoder
	encoderCfg.LevelKey = "LEVEL"
	encoderCfg.CallerKey = "CALLER"
	encoderCfg.TimeKey = "TIME"
	encoderCfg.NameKey = "NAME"
	encoderCfg.MessageKey = "MESSAGE"

	if cfg.Encoding == "" && cfg.Mode == config.Development {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(encoder, logWriter, zap.NewAtomicLevelAt(logLevel))
	newLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	logSugar := newLogger.Sugar()
	logSugar.Sync()

	return logSugar
}

func zaplogLevel(cfg *config.NotifConfig) (logLevel zapcore.Level) {
	switch cfg.LogLevel {
	case "debug":
		logLevel = zapcore.DebugLevel

	case "info":
		logLevel = zapcore.InfoLevel

	case "warn":
		logLevel = zapcore.WarnLevel

	case "error":
		logLevel = zapcore.ErrorLevel

	default:
		logLevel = zapcore.DebugLevel
	}

	return logLevel
}
