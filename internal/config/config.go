package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	Env                string
	DatabaseURL        string
	AzureTenantID      string
	AzureClientID      string
	AzureAudience      string
	SAPRFCFunction     string
	SAPRFCEnabled      bool
	TLSCertFile        string
	TLSKeyFile         string
	TLSRedirect        bool
	RateLimitRequests  int
	RateLimitWindowSec int
	MaxBodyBytes       int64
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	sapEnabled, err := strconv.ParseBool(getEnv("SAP_RFC_ENABLED", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid SAP_RFC_ENABLED: %w", err)
	}

	rateLimitRequests, err := strconv.Atoi(getEnv("RATE_LIMIT_REQUESTS", "100"))
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_REQUESTS: %w", err)
	}

	rateLimitWindowSec, err := strconv.Atoi(getEnv("RATE_LIMIT_WINDOW_SEC", "60"))
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_WINDOW_SEC: %w", err)
	}

	maxBodyBytes, err := strconv.ParseInt(getEnv("MAX_BODY_BYTES", "1048576"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_BODY_BYTES: %w", err)
	}

	env := getEnv("ENV", "development")
	databaseURL := getEnv("DATABASE_URL", "postgres://saporders:saporders@localhost:5434/saporders?sslmode=disable")
	onRailway := os.Getenv("RAILWAY_ENVIRONMENT") != ""

	if env == "production" && !onRailway && strings.Contains(databaseURL, "sslmode=disable") {
		return nil, fmt.Errorf("DATABASE_URL must not use sslmode=disable in production")
	}

	tlsCertFile := os.Getenv("TLS_CERT_FILE")
	tlsKeyFile := os.Getenv("TLS_KEY_FILE")
	tlsTerminatedAtEdge := getEnv("TLS_TERMINATED_AT_EDGE", "false") == "true"
	tlsRedirect, err := strconv.ParseBool(getEnv("TLS_REDIRECT", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid TLS_REDIRECT: %w", err)
	}

	requireAppTLS := env == "production" && !onRailway && !tlsTerminatedAtEdge
	if requireAppTLS && (tlsCertFile == "" || tlsKeyFile == "") {
		return nil, fmt.Errorf("TLS_CERT_FILE and TLS_KEY_FILE are required in production unless TLS is terminated at the edge")
	}

	cfg := &Config{
		Port:               getEnv("PORT", "8081"),
		Env:                env,
		DatabaseURL:        databaseURL,
		AzureTenantID:      os.Getenv("AZURE_TENANT_ID"),
		AzureClientID:      os.Getenv("AZURE_CLIENT_ID"),
		AzureAudience:      getEnv("AZURE_AUDIENCE", os.Getenv("AZURE_CLIENT_ID")),
		SAPRFCFunction:     getEnv("SAP_RFC_FUNCTION", "Z_UPDATE_DEMAND"),
		SAPRFCEnabled:      sapEnabled,
		TLSCertFile:        tlsCertFile,
		TLSKeyFile:         tlsKeyFile,
		TLSRedirect:        tlsRedirect,
		RateLimitRequests:  rateLimitRequests,
		RateLimitWindowSec: rateLimitWindowSec,
		MaxBodyBytes:       maxBodyBytes,
	}

	if cfg.AzureTenantID == "" || cfg.AzureClientID == "" {
		return nil, fmt.Errorf("AZURE_TENANT_ID and AZURE_CLIENT_ID are required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
