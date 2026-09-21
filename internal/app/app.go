package app

import (
	"fmt"
	"net/http"

	"github.com/zachfire9/financials-api/internal/config"
	"github.com/zachfire9/financials-api/internal/financialitems"
	"github.com/zachfire9/financials-api/internal/httpapi"
)

// NewHandler builds the application HTTP handler from runtime configuration.
func NewHandler(cfg config.Config) (http.Handler, error) {
	repository, err := NewFinancialItemsRepository(cfg)
	if err != nil {
		return nil, err
	}

	return httpapi.NewHandlerWithRepositoryAndCORS(repository, httpapi.CORSConfig{
		AllowedOrigins: cfg.AllowedOrigins,
	}), nil
}

// NewFinancialItemsRepository selects the configured financial item repository adapter.
func NewFinancialItemsRepository(cfg config.Config) (financialitems.Repository, error) {
	switch cfg.StorageDriver {
	case config.StorageDriverMemory, config.StorageDriverEphemeral:
		return financialitems.NewInMemoryRepository(), nil
	case config.StorageDriverJSON:
		return financialitems.NewJSONFileRepository(cfg.StoragePath)
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", cfg.StorageDriver)
	}
}
