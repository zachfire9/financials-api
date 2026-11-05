package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadUsesDefaultsWhenEnvFileAndProcessEnvAreAbsent(t *testing.T) {
	t.Setenv("FINANCIALS_API_ADDR", "")
	t.Setenv("FINANCIALS_STORAGE_DRIVER", "")
	t.Setenv("FINANCIALS_STORAGE_PATH", "")
	t.Setenv("FINANCIALS_ALLOWED_ORIGINS", "")
	t.Setenv("FINANCIALS_ACCESS_TOKEN", "")

	cfg, err := Load(filepath.Join(t.TempDir(), ".env"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.APIAddr != ":8080" {
		t.Fatalf("expected default API address, got %q", cfg.APIAddr)
	}
	if cfg.StorageDriver != StorageDriverMemory {
		t.Fatalf("expected memory storage driver, got %q", cfg.StorageDriver)
	}
	if cfg.StoragePath != "" {
		t.Fatalf("expected empty storage path for memory driver, got %q", cfg.StoragePath)
	}
	if len(cfg.AllowedOrigins) != 0 {
		t.Fatalf("expected no default CORS origins, got %v", cfg.AllowedOrigins)
	}
	if cfg.AccessToken != "" {
		t.Fatalf("expected no default access token, got %q", cfg.AccessToken)
	}
}

// unsetEnv removes a variable for the duration of the test and restores it afterwards.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
}

func TestLoadReadsEnvFileWhenProcessEnvIsAbsent(t *testing.T) {
	unsetEnv(t, "FINANCIALS_API_ADDR")
	unsetEnv(t, "FINANCIALS_STORAGE_DRIVER")
	unsetEnv(t, "FINANCIALS_STORAGE_PATH")
	unsetEnv(t, "FINANCIALS_ALLOWED_ORIGINS")
	unsetEnv(t, "FINANCIALS_ACCESS_TOKEN")

	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte("FINANCIALS_API_ADDR=:9090\nFINANCIALS_STORAGE_DRIVER=json\nFINANCIALS_STORAGE_PATH=./local-data/financial-items.json\nFINANCIALS_ALLOWED_ORIGINS=http://localhost:5173, https://example-amplify-app.example.com\nFINANCIALS_ACCESS_TOKEN=example-local-token\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.APIAddr != ":9090" {
		t.Fatalf("expected env file API address, got %q", cfg.APIAddr)
	}
	if cfg.StorageDriver != StorageDriverJSON {
		t.Fatalf("expected json storage driver, got %q", cfg.StorageDriver)
	}
	if cfg.StoragePath != "./local-data/financial-items.json" {
		t.Fatalf("expected env file storage path, got %q", cfg.StoragePath)
	}
	wantOrigins := []string{"http://localhost:5173", "https://example-amplify-app.example.com"}
	if !reflect.DeepEqual(cfg.AllowedOrigins, wantOrigins) {
		t.Fatalf("expected env file allowed origins %v, got %v", wantOrigins, cfg.AllowedOrigins)
	}
	if cfg.AccessToken != "example-local-token" {
		t.Fatalf("expected env file access token, got %q", cfg.AccessToken)
	}
}

func TestLoadLetsProcessEnvOverrideEnvFile(t *testing.T) {
	t.Setenv("FINANCIALS_API_ADDR", ":7070")
	t.Setenv("FINANCIALS_STORAGE_DRIVER", "memory")
	t.Setenv("FINANCIALS_STORAGE_PATH", "")
	t.Setenv("FINANCIALS_ALLOWED_ORIGINS", "https://override.example.com")
	t.Setenv("FINANCIALS_ACCESS_TOKEN", "override-token")

	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte("FINANCIALS_API_ADDR=:9090\nFINANCIALS_STORAGE_DRIVER=json\nFINANCIALS_STORAGE_PATH=./local-data/financial-items.json\nFINANCIALS_ALLOWED_ORIGINS=http://localhost:5173\nFINANCIALS_ACCESS_TOKEN=example-local-token\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.APIAddr != ":7070" {
		t.Fatalf("expected process env API address, got %q", cfg.APIAddr)
	}
	if cfg.StorageDriver != StorageDriverMemory {
		t.Fatalf("expected process env storage driver, got %q", cfg.StorageDriver)
	}
	if cfg.StoragePath != "" {
		t.Fatalf("expected process env empty storage path, got %q", cfg.StoragePath)
	}
	wantOrigins := []string{"https://override.example.com"}
	if !reflect.DeepEqual(cfg.AllowedOrigins, wantOrigins) {
		t.Fatalf("expected process env allowed origins %v, got %v", wantOrigins, cfg.AllowedOrigins)
	}
	if cfg.AccessToken != "override-token" {
		t.Fatalf("expected process env access token, got %q", cfg.AccessToken)
	}
}

func TestLoadAcceptsEphemeralStorageDriverWithoutAStoragePath(t *testing.T) {
	t.Setenv("FINANCIALS_API_ADDR", "")
	t.Setenv("FINANCIALS_STORAGE_DRIVER", "ephemeral")
	t.Setenv("FINANCIALS_STORAGE_PATH", "./ignored-for-ephemeral.json")
	t.Setenv("FINANCIALS_ALLOWED_ORIGINS", "")

	cfg, err := Load(filepath.Join(t.TempDir(), ".env"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.StorageDriver != StorageDriverEphemeral {
		t.Fatalf("expected ephemeral storage driver, got %q", cfg.StorageDriver)
	}
	if cfg.StoragePath != "" {
		t.Fatalf("expected empty storage path for ephemeral driver, got %q", cfg.StoragePath)
	}
}

func TestLoadRequiresStoragePathForJSONDriver(t *testing.T) {
	t.Setenv("FINANCIALS_API_ADDR", "")
	t.Setenv("FINANCIALS_STORAGE_DRIVER", "json")
	t.Setenv("FINANCIALS_STORAGE_PATH", "")

	_, err := Load(filepath.Join(t.TempDir(), ".env"))
	if err == nil {
		t.Fatal("expected JSON storage path validation error")
	}
}

func TestLoadLetsEmptyProcessEnvOverrideEnvFileWithDefault(t *testing.T) {
	unsetEnv(t, "FINANCIALS_API_ADDR")
	unsetEnv(t, "FINANCIALS_STORAGE_DRIVER")
	unsetEnv(t, "FINANCIALS_STORAGE_PATH")
	unsetEnv(t, "FINANCIALS_ALLOWED_ORIGINS")
	t.Setenv("FINANCIALS_ACCESS_TOKEN", "")

	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte("FINANCIALS_ACCESS_TOKEN=example-local-token\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AccessToken != "" {
		t.Fatalf("expected empty process env to clear the .env access token, got %q", cfg.AccessToken)
	}
}
