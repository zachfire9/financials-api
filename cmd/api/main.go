package main

import (
	"log"
	"net/http"
	"os"

	"github.com/zachfire9/financials-api/internal/httpapi"
)

func main() {
	addr := envOrDefault("FINANCIALS_API_ENDPOINT", ":8080")

	server := &http.Server{
		Addr:    addr,
		Handler: httpapi.NewHandler(),
	}

	log.Printf("financials-api listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
