package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/zachfire9/financials-api/internal/config"
	"github.com/zachfire9/financials-api/internal/financialitems"
	"github.com/zachfire9/financials-api/internal/httpapi"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	repository, err := newFinancialItemsRepository(cfg)
	if err != nil {
		log.Fatalf("configure financial items repository: %v", err)
	}

	server := &http.Server{
		Addr: cfg.APIAddr,
		Handler: httpapi.NewHandlerWithRepositoryAndCORS(repository, httpapi.CORSConfig{
			AllowedOrigins: cfg.AllowedOrigins,
		}),
	}

	log.Printf("financials-api listening on %s with %s storage", cfg.APIAddr, cfg.StorageDriver)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func newFinancialItemsRepository(cfg config.Config) (financialitems.Repository, error) {
	switch cfg.StorageDriver {
	case config.StorageDriverMemory:
		return financialitems.NewInMemoryRepository(), nil
	case config.StorageDriverJSON:
		return financialitems.NewJSONFileRepository(cfg.StoragePath)
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", cfg.StorageDriver)
	}
}
