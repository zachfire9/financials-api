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
}

func TestLoadReadsEnvFileWhenProcessEnvIsAbsent(t *testing.T) {
	t.Setenv("FINANCIALS_API_ADDR", "")
	t.Setenv("FINANCIALS_STORAGE_DRIVER", "")
	t.Setenv("FINANCIALS_STORAGE_PATH", "")
	t.Setenv("FINANCIALS_ALLOWED_ORIGINS", "")

	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte("FINANCIALS_API_ADDR=:9090\nFINANCIALS_STORAGE_DRIVER=json\nFINANCIALS_STORAGE_PATH=./local-data/financial-items.json\nFINANCIALS_ALLOWED_ORIGINS=http://localhost:5173, https://example-amplify-app.example.com\n"), 0o600); err != nil {
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
}

func TestLoadLetsProcessEnvOverrideEnvFile(t *testing.T) {
	t.Setenv("FINANCIALS_API_ADDR", ":7070")
	t.Setenv("FINANCIALS_STORAGE_DRIVER", "memory")
	t.Setenv("FINANCIALS_STORAGE_PATH", "")
	t.Setenv("FINANCIALS_ALLOWED_ORIGINS", "https://override.example.com")

	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte("FINANCIALS_API_ADDR=:9090\nFINANCIALS_STORAGE_DRIVER=json\nFINANCIALS_STORAGE_PATH=./local-data/financial-items.json\nFINANCIALS_ALLOWED_ORIGINS=http://localhost:5173\n"), 0o600); err != nil {
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
