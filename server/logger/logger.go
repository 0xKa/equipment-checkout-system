package logger

import (
	"strings"

	"go.uber.org/zap"
)

func New(appEnv string) (*zap.Logger, error) {

	if strings.EqualFold(appEnv, "development") {
		return zap.NewDevelopment()
	}

	return zap.NewProduction()
}
