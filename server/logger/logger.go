package logger

import (
	"log/slog"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
)

func New(appEnv string) (*zap.Logger, error) {
	if strings.EqualFold(appEnv, "development") {
		cfg := zap.NewDevelopmentConfig()

		cfg.EncoderConfig.EncodeTime =
			zapcore.TimeEncoderOfLayout("15:04:05.000")
		cfg.EncoderConfig.EncodeLevel =
			zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeCaller =
			zapcore.ShortCallerEncoder
		cfg.EncoderConfig.ConsoleSeparator = " | "

		return cfg.Build()
	}

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.MessageKey = "message"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return cfg.Build()
}

func AsSlog(log *zap.Logger) *slog.Logger {
	return slog.New(zapslog.NewHandler(
		log.Core(),
		zapslog.WithCaller(true),
	))
}
