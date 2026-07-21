package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultAppEnv                        = "development"
	defaultHTTPHost                      = "localhost"
	defaultPort                          = "8080"
	defaultDBMaxConnections              = "10"
	defaultDBMinConnections              = "1"
	defaultDBMaxConnectionLifetime       = "30m"
	maxSupportedDBConnections      int64 = 100
)

type Config struct {
	AppEnv                  string
	HTTPHost                string
	HTTPPort                string
	DatabaseURL             string
	DBMaxConnections        int32
	DBMinConnections        int32
	DBMaxConnectionLifetime time.Duration
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

	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL must not be empty")
	}

	maxConnections, err := parseConnectionCount(
		"DB_MAX_CONNECTIONS",
		getEnv("DB_MAX_CONNECTIONS", defaultDBMaxConnections),
		1,
		maxSupportedDBConnections,
	)
	if err != nil {
		return Config{}, err
	}

	minConnections, err := parseConnectionCount(
		"DB_MIN_CONNECTIONS",
		getEnv("DB_MIN_CONNECTIONS", defaultDBMinConnections),
		1,
		int64(maxConnections),
	)
	if err != nil {
		return Config{}, err
	}

	maxConnectionLifetimeValue := getEnv(
		"DB_MAX_CONNECTION_LIFETIME",
		defaultDBMaxConnectionLifetime,
	)
	maxConnectionLifetime, err := time.ParseDuration(maxConnectionLifetimeValue)
	if err != nil || maxConnectionLifetime <= 0 {
		return Config{}, fmt.Errorf("DB_MAX_CONNECTION_LIFETIME must be a positive duration")
	}

	return Config{
		AppEnv:                  appEnv,
		HTTPHost:                host,
		HTTPPort:                portValue,
		DatabaseURL:             databaseURL,
		DBMaxConnections:        maxConnections,
		DBMinConnections:        minConnections,
		DBMaxConnectionLifetime: maxConnectionLifetime,
	}, nil
}

func parseConnectionCount(name, value string, minimum, maximum int64) (int32, error) {
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", name, minimum, maximum)
	}

	return int32(parsed), nil
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return strings.TrimSpace(value)
}
