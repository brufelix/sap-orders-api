package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	Env            string
	DatabaseURL    string
	AzureTenantID  string
	AzureClientID  string
	AzureAudience  string
	SAPRFCFunction string
	SAPRFCEnabled  bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	sapEnabled, err := strconv.ParseBool(getEnv("SAP_RFC_ENABLED", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid SAP_RFC_ENABLED: %w", err)
	}

	cfg := &Config{
		Port:           getEnv("PORT", "8081"),
		Env:            getEnv("ENV", "development"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://saporders:saporders@localhost:5434/saporders?sslmode=disable"),
		AzureTenantID:  os.Getenv("AZURE_TENANT_ID"),
		AzureClientID:  os.Getenv("AZURE_CLIENT_ID"),
		AzureAudience:  getEnv("AZURE_AUDIENCE", os.Getenv("AZURE_CLIENT_ID")),
		SAPRFCFunction: getEnv("SAP_RFC_FUNCTION", "Z_UPDATE_DEMAND"),
		SAPRFCEnabled:  sapEnabled,
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
