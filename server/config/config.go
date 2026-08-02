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
	defaultAppEnv                            = "development"
	defaultHTTPHost                          = "localhost"
	defaultPort                              = "8080"
	defaultDBMaxConnections                  = "10"
	defaultDBMinConnections                  = "1"
	defaultDBMaxConnectionLifetime           = "30m"
	defaultOIDCAudience                      = "equipment-api"
	defaultOIDCHTTPTimeout                   = "5s"
	defaultOIDCClockSkew                     = "30s"
	defaultKeycloakRealm                     = "equipment"
	defaultKeycloakApplicationClientID       = "equipment-api"
	defaultKeycloakAdminTimeout              = "5s"
	maxSupportedDBConnections          int64 = 100
)

type Config struct {
	AppEnv         string
	HTTP           HTTPConfig
	Database       DatabaseConfig
	OIDC           OIDCConfig
	KeycloakAdmin  KeycloakAdminConfig
	APIDocsEnabled bool
}

type HTTPConfig struct {
	Host string
	Port string
}

func (c HTTPConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

type DatabaseConfig struct {
	URL                   string
	MaxConnections        int32
	MinConnections        int32
	MaxConnectionLifetime time.Duration
}

type OIDCConfig struct {
	IssuerURL   string
	JWKSURL     string
	Audience    string
	HTTPTimeout time.Duration
	ClockSkew   time.Duration
}

type KeycloakAdminConfig struct {
	BaseURL             string
	Realm               string
	ServiceClientID     string
	ServiceClientSecret string
	ApplicationClientID string
	Timeout             time.Duration
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	appEnv := getEnv("APP_ENV", defaultAppEnv)
	if appEnv == "" {
		return Config{}, fmt.Errorf("APP_ENV must not be empty")
	}

	httpConfig, err := loadHTTPConfig()
	if err != nil {
		return Config{}, err
	}

	databaseConfig, err := loadDatabaseConfig()
	if err != nil {
		return Config{}, err
	}

	apiDocsEnabled, err := parseExactBool(
		"API_DOCS_ENABLED",
		getEnv("API_DOCS_ENABLED", "false"),
	)
	if err != nil {
		return Config{}, err
	}

	oidcConfig, err := loadOIDCConfig()
	if err != nil {
		return Config{}, err
	}

	keycloakAdminConfig, err := loadKeycloakAdminConfig()
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppEnv:         appEnv,
		HTTP:           httpConfig,
		Database:       databaseConfig,
		APIDocsEnabled: apiDocsEnabled,
		OIDC:           oidcConfig,
		KeycloakAdmin:  keycloakAdminConfig,
	}, nil
}

func loadKeycloakAdminConfig() (KeycloakAdminConfig, error) {
	baseURL, err := parseHTTPURL(
		"KEYCLOAK_ADMIN_URL",
		getEnv("KEYCLOAK_ADMIN_URL", ""),
		false,
	)
	if err != nil {
		return KeycloakAdminConfig{}, err
	}

	realm := getEnv("KEYCLOAK_REALM", defaultKeycloakRealm)
	if realm == "" {
		return KeycloakAdminConfig{}, fmt.Errorf("KEYCLOAK_REALM must not be empty")
	}
	serviceClientID := getEnv("KEYCLOAK_USER_SYNC_CLIENT_ID", "")
	if serviceClientID == "" {
		return KeycloakAdminConfig{}, fmt.Errorf("KEYCLOAK_USER_SYNC_CLIENT_ID must not be empty")
	}
	serviceClientSecret := getEnv("KEYCLOAK_USER_SYNC_CLIENT_SECRET", "")
	if serviceClientSecret == "" {
		return KeycloakAdminConfig{}, fmt.Errorf("KEYCLOAK_USER_SYNC_CLIENT_SECRET must not be empty")
	}
	applicationClientID := getEnv(
		"KEYCLOAK_APPLICATION_CLIENT_ID",
		defaultKeycloakApplicationClientID,
	)
	if applicationClientID == "" {
		return KeycloakAdminConfig{}, fmt.Errorf("KEYCLOAK_APPLICATION_CLIENT_ID must not be empty")
	}
	timeout, err := parsePositiveDuration(
		"KEYCLOAK_ADMIN_TIMEOUT",
		getEnv("KEYCLOAK_ADMIN_TIMEOUT", defaultKeycloakAdminTimeout),
	)
	if err != nil {
		return KeycloakAdminConfig{}, err
	}

	return KeycloakAdminConfig{
		BaseURL:             baseURL,
		Realm:               realm,
		ServiceClientID:     serviceClientID,
		ServiceClientSecret: serviceClientSecret,
		ApplicationClientID: applicationClientID,
		Timeout:             timeout,
	}, nil
}

func loadHTTPConfig() (HTTPConfig, error) {
	host := getEnv("HTTP_HOST", defaultHTTPHost)
	if host == "" {
		return HTTPConfig{}, fmt.Errorf("HTTP_HOST must not be empty")
	}

	portValue := getEnv("HTTP_PORT", defaultPort)
	port, err := strconv.Atoi(portValue)
	if err != nil || port <= 0 || port > 65535 {
		return HTTPConfig{}, fmt.Errorf("HTTP_PORT must be an integer between 1 and 65535")
	}

	return HTTPConfig{
		Host: host,
		Port: portValue,
	}, nil
}

func loadDatabaseConfig() (DatabaseConfig, error) {
	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		return DatabaseConfig{}, fmt.Errorf("DATABASE_URL must not be empty")
	}

	maxConnections, err := parseConnectionCount(
		"DB_MAX_CONNECTIONS",
		getEnv("DB_MAX_CONNECTIONS", defaultDBMaxConnections),
		1,
		maxSupportedDBConnections,
	)
	if err != nil {
		return DatabaseConfig{}, err
	}

	minConnections, err := parseConnectionCount(
		"DB_MIN_CONNECTIONS",
		getEnv("DB_MIN_CONNECTIONS", defaultDBMinConnections),
		1,
		int64(maxConnections),
	)
	if err != nil {
		return DatabaseConfig{}, err
	}

	maxConnectionLifetimeValue := getEnv(
		"DB_MAX_CONNECTION_LIFETIME",
		defaultDBMaxConnectionLifetime,
	)
	maxConnectionLifetime, err := time.ParseDuration(maxConnectionLifetimeValue)
	if err != nil || maxConnectionLifetime <= 0 {
		return DatabaseConfig{}, fmt.Errorf("DB_MAX_CONNECTION_LIFETIME must be a positive duration")
	}

	return DatabaseConfig{
		URL:                   databaseURL,
		MaxConnections:        maxConnections,
		MinConnections:        minConnections,
		MaxConnectionLifetime: maxConnectionLifetime,
	}, nil
}

func loadOIDCConfig() (OIDCConfig, error) {
	issuerURL, err := parseHTTPURL(
		"OIDC_ISSUER_URL",
		getEnv("OIDC_ISSUER_URL", ""),
		false,
	)
	if err != nil {
		return OIDCConfig{}, err
	}

	jwksURL, err := parseHTTPURL(
		"OIDC_JWKS_URL",
		getEnv("OIDC_JWKS_URL", ""),
		true,
	)
	if err != nil {
		return OIDCConfig{}, err
	}

	audience := getEnv("OIDC_AUDIENCE", defaultOIDCAudience)
	if audience == "" {
		return OIDCConfig{}, fmt.Errorf("OIDC_AUDIENCE must not be empty")
	}

	httpTimeout, err := parsePositiveDuration(
		"OIDC_HTTP_TIMEOUT",
		getEnv("OIDC_HTTP_TIMEOUT", defaultOIDCHTTPTimeout),
	)
	if err != nil {
		return OIDCConfig{}, err
	}

	clockSkew, err := time.ParseDuration(
		getEnv("OIDC_CLOCK_SKEW", defaultOIDCClockSkew),
	)
	if err != nil || clockSkew < 0 {
		return OIDCConfig{}, fmt.Errorf("OIDC_CLOCK_SKEW must be a nonnegative duration")
	}

	return OIDCConfig{
		IssuerURL:   issuerURL,
		JWKSURL:     jwksURL,
		Audience:    audience,
		HTTPTimeout: httpTimeout,
		ClockSkew:   clockSkew,
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
