package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	StorageDriverMemory = "memory"
	StorageDriverJSON   = "json"
)

// Config contains public-safe local runtime configuration for the API.
type Config struct {
	APIAddr        string
	StorageDriver  string
	StoragePath    string
	AllowedOrigins []string
}

// Load reads configuration from defaults, an optional .env file, and process env.
//
// Precedence is process environment > .env file > built-in defaults.
func Load(envFilePath string) (Config, error) {
	envFileValues, err := readEnvFile(envFilePath)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		APIAddr:        valueFor("FINANCIALS_API_ADDR", envFileValues, ":8080"),
		StorageDriver:  valueFor("FINANCIALS_STORAGE_DRIVER", envFileValues, StorageDriverMemory),
		StoragePath:    valueFor("FINANCIALS_STORAGE_PATH", envFileValues, ""),
		AllowedOrigins: splitCSV(valueFor("FINANCIALS_ALLOWED_ORIGINS", envFileValues, "")),
	}

	if cfg.StorageDriver == StorageDriverMemory {
		cfg.StoragePath = ""
	}
	if cfg.StorageDriver != StorageDriverMemory && cfg.StorageDriver != StorageDriverJSON {
		return Config{}, fmt.Errorf("unsupported FINANCIALS_STORAGE_DRIVER %q", cfg.StorageDriver)
	}
	if cfg.StorageDriver == StorageDriverJSON && strings.TrimSpace(cfg.StoragePath) == "" {
		return Config{}, fmt.Errorf("FINANCIALS_STORAGE_PATH is required when FINANCIALS_STORAGE_DRIVER=json")
	}

	return cfg, nil
}

func valueFor(key string, envFileValues map[string]string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	if value, ok := envFileValues[key]; ok {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func readEnvFile(path string) (map[string]string, error) {
	values := make(map[string]string)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return values, nil
		}
		return nil, fmt.Errorf("open env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("parse env file line %d: expected KEY=VALUE", lineNumber)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"'")
		if key == "" {
			return nil, fmt.Errorf("parse env file line %d: key is required", lineNumber)
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}

	return values, nil
}
