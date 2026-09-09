package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	TautulliURL       string
	TautulliAPIKey    string
	TautulliBasePath  string
	TautulliTimeout   time.Duration
	TautulliVerifySSL bool
	ListenHost        string
	ListenPort        int
	LogLevel          string
	Version           string
}

// LoadConfig loads configuration from environment variables
func LoadConfig(version string) Config {
	return Config{
		TautulliURL:       getEnv("TAUTULLI_URL", "http://127.0.0.1:8181"),
		TautulliAPIKey:    getEnv("TAUTULLI_APIKEY", ""),
		TautulliBasePath:  getEnv("TAUTULLI_BASE_PATH", ""),
		TautulliTimeout:   time.Duration(getEnvAsInt("TAUTULLI_TIMEOUT", 5)) * time.Second,
		TautulliVerifySSL: getEnvAsBool("TAUTULLI_VERIFY_SSL", true),
		ListenHost:        getEnv("LISTEN_HOST", "0.0.0.0"),
		ListenPort:        getEnvAsInt("LISTEN_PORT", 9105),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		Version:           version,
	}
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if valueStr := getEnv(key, ""); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if valueStr := strings.ToLower(getEnv(key, "")); valueStr != "" {
		return valueStr == "true" || valueStr == "1"
	}
	return defaultValue
}
