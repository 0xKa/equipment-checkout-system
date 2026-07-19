package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultAppEnv   = "development"
	defaultHTTPHost = "localhost"
	defaultPort     = "8080"
)

type Config struct {
	AppEnv   string
	HTTPHost string
	HTTPPort string
}

func (c Config) HTTPAddress() string {
	return fmt.Sprintf("%s:%s", c.HTTPHost, c.HTTPPort)
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	appEnv := getEnv("APP_ENV", defaultAppEnv)
	if appEnv == "" {
		return Config{}, fmt.Errorf("APP_ENV must not be empty")
	}

	host := getEnv("HTTP_HOST", defaultHTTPHost)
	if host == "" {
		return Config{}, fmt.Errorf("HTTP_HOST must not be empty")
	}

	portValue := getEnv("HTTP_PORT", defaultPort)
	port, err := strconv.Atoi(portValue)
	if err != nil || port <= 0 || port > 65535 {
		return Config{}, fmt.Errorf("HTTP_PORT must be an integer between 1 and 65535")
	}

	return Config{
		AppEnv:   appEnv,
		HTTPHost: host,
		HTTPPort: portValue,
	}, nil
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return strings.TrimSpace(value)
}
