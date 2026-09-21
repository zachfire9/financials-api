package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zachfire9/financials-api/internal/config"
)

func TestNewHandlerUsesConfiguredCORSAndEphemeralStorage(t *testing.T) {
	handler, err := NewHandler(config.Config{
		StorageDriver:  config.StorageDriverEphemeral,
		AllowedOrigins: []string{"https://example-static-ui.example.com"},
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://example-static-ui.example.com")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "https://example-static-ui.example.com" {
		t.Fatalf("expected configured CORS origin, got %q", origin)
	}
}

func TestNewFinancialItemsRepositoryRejectsUnknownStorageDriver(t *testing.T) {
	_, err := NewFinancialItemsRepository(config.Config{StorageDriver: "dynamodb"})
	if err == nil {
		t.Fatal("expected unknown storage driver error")
	}
}
