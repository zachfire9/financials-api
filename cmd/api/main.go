package main

import (
	"log"
	"net/http"
	"time"

	"github.com/zachfire9/financials-api/internal/app"
	"github.com/zachfire9/financials-api/internal/config"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	handler, err := app.NewHandler(cfg)
	if err != nil {
		log.Fatalf("configure application: %v", err)
	}

	server := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("financials-api listening on %s with %s storage", cfg.APIAddr, cfg.StorageDriver)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
