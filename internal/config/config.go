package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	Port               string
	LLMBaseURL         string
	LLMAPIKey          string
	LLMModel           string
	JWTSecret          string
	DBPath             string
	MaxHistoryMessages int
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		LLMBaseURL:         getEnv("LLM_BASE_URL", ""),
		LLMAPIKey:          getEnv("LLM_API_KEY", ""),
		LLMModel:           getEnv("LLM_MODEL", "mimo-v2.5-pro"),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		DBPath:             getEnv("DB_PATH", "./data/ops-agent.db"),
		MaxHistoryMessages: 20,
	}

	// Startup validation
	var errs []string
	if cfg.LLMAPIKey == "" {
		errs = append(errs, "LLM_API_KEY is required")
	}
	if cfg.LLMBaseURL == "" {
		errs = append(errs, "LLM_BASE_URL is required")
	}
	if cfg.JWTSecret == "" || cfg.JWTSecret == "change-me-in-production" {
		// For dev, we allow a weak secret with a warning
		if cfg.JWTSecret == "" {
			cfg.JWTSecret = "dev-only-insecure-key"
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}

	return cfg, nil
}

// LoadDotEnv loads a .env file into environment (simple implementation).
func LoadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && os.Getenv(parts[0]) == "" {
			val := strings.TrimSpace(parts[1])
			if (strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) ||
				(strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`)) {
				val = val[1 : len(val)-1]
			}
			os.Setenv(parts[0], val)
		}
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
