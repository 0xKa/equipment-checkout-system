package config

import (
	"errors"
	"fmt"
	"net/url"
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
	defaultOIDCAudience                  = "equipment-api"
	defaultOIDCHTTPTimeout               = "5s"
	defaultOIDCClockSkew                 = "30s"
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
	APIDocsEnabled          bool
	OIDCIssuerURL           string
	OIDCJWKSURL             string
	OIDCAudience            string
	OIDCHTTPTimeout         time.Duration
	OIDCClockSkew           time.Duration
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

	apiDocsEnabled, err := parseExactBool(
		"API_DOCS_ENABLED",
		getEnv("API_DOCS_ENABLED", "false"),
	)
	if err != nil {
		return Config{}, err
	}

	oidcIssuerURL, err := parseHTTPURL(
		"OIDC_ISSUER_URL",
		getEnv("OIDC_ISSUER_URL", ""),
		false,
	)
	if err != nil {
		return Config{}, err
	}

	oidcJWKSURL, err := parseHTTPURL(
		"OIDC_JWKS_URL",
		getEnv("OIDC_JWKS_URL", ""),
		true,
	)
	if err != nil {
		return Config{}, err
	}

	oidcAudience := getEnv("OIDC_AUDIENCE", defaultOIDCAudience)
	if oidcAudience == "" {
		return Config{}, fmt.Errorf("OIDC_AUDIENCE must not be empty")
	}

	oidcHTTPTimeout, err := parsePositiveDuration(
		"OIDC_HTTP_TIMEOUT",
		getEnv("OIDC_HTTP_TIMEOUT", defaultOIDCHTTPTimeout),
	)
	if err != nil {
		return Config{}, err
	}

	oidcClockSkew, err := time.ParseDuration(
		getEnv("OIDC_CLOCK_SKEW", defaultOIDCClockSkew),
	)
	if err != nil || oidcClockSkew < 0 {
		return Config{}, fmt.Errorf("OIDC_CLOCK_SKEW must be a nonnegative duration")
	}

	return Config{
		AppEnv:                  appEnv,
		HTTPHost:                host,
		HTTPPort:                portValue,
		DatabaseURL:             databaseURL,
		DBMaxConnections:        maxConnections,
		DBMinConnections:        minConnections,
		DBMaxConnectionLifetime: maxConnectionLifetime,
		APIDocsEnabled:          apiDocsEnabled,
		OIDCIssuerURL:           oidcIssuerURL,
		OIDCJWKSURL:             oidcJWKSURL,
		OIDCAudience:            oidcAudience,
		OIDCHTTPTimeout:         oidcHTTPTimeout,
		OIDCClockSkew:           oidcClockSkew,
	}, nil
}

func parseHTTPURL(name, value string, allowQuery bool) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", name)
	}

	parsed, err := url.Parse(value)
	if err != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" ||
		parsed.User != nil ||
		parsed.Fragment != "" ||
		(!allowQuery && parsed.RawQuery != "") {
		return "", fmt.Errorf("%s must be an absolute HTTP(S) URL without credentials, a fragment, or an unsupported query", name)
	}

	return value, nil
}

func parsePositiveDuration(name, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return duration, nil
}

func parseExactBool(name, value string) (bool, error) {
	switch value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be true or false", name)
	}
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
