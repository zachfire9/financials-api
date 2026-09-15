package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/zachfire9/financials-api/internal/financialitems"
)

// NewHandler builds the HTTP handler tree for the API.
//
// Keep this package independent from process startup so the same handler can be
// used by local net/http serving now and an AWS Lambda/API Gateway adapter later.
func NewHandler() http.Handler {
	return NewHandlerWithRepository(financialitems.NewInMemoryRepository())
}

// NewHandlerWithRepository builds the HTTP handler tree with an injected financial item repository.
func NewHandlerWithRepository(repository financialitems.Repository) http.Handler {
	return NewHandlerWithRepositoryAndCORS(repository, CORSConfig{})
}

// CORSConfig contains allowed browser origins for static-hosted UI deployments.
type CORSConfig struct {
	AllowedOrigins []string
}

// NewHandlerWithRepositoryAndCORS builds the HTTP handler tree with an injected financial item repository and CORS config.
func NewHandlerWithRepositoryAndCORS(repository financialitems.Repository, corsConfig CORSConfig) http.Handler {
	api := &apiHandler{financialItems: repository}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", api.handleHealth)
	mux.HandleFunc("GET /financial-items", api.handleFinancialItemsList)
	mux.HandleFunc("POST /financial-items", api.handleFinancialItemsCreate)
	mux.HandleFunc("GET /financial-items/{id}", api.handleFinancialItemsGet)
	mux.HandleFunc("PUT /financial-items/{id}", api.handleFinancialItemsUpdate)
	mux.HandleFunc("DELETE /financial-items/{id}", api.handleFinancialItemsDelete)
	return withCORS(mux, corsConfig)
}

func withCORS(next http.Handler, config CORSConfig) http.Handler {
	allowedOrigins := make(map[string]struct{}, len(config.AllowedOrigins))
	for _, origin := range config.AllowedOrigins {
		allowedOrigins[origin] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		_, isAllowedOrigin := allowedOrigins[origin]
		if origin != "" && isAllowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type apiHandler struct {
	financialItems financialitems.Repository
}

type healthResponse struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (api *apiHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func (api *apiHandler) handleFinancialItemsList(w http.ResponseWriter, r *http.Request) {
	items, err := api.financialItems.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (api *apiHandler) handleFinancialItemsCreate(w http.ResponseWriter, r *http.Request) {
	var request financialitems.CreateFinancialItemRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	item, err := api.financialItems.Create(r.Context(), request)
	if err != nil {
		writeRepositoryError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (api *apiHandler) handleFinancialItemsGet(w http.ResponseWriter, r *http.Request) {
	item, err := api.financialItems.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeRepositoryError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (api *apiHandler) handleFinancialItemsUpdate(w http.ResponseWriter, r *http.Request) {
	var request financialitems.UpdateFinancialItemRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	item, err := api.financialItems.Update(r.Context(), r.PathValue("id"), request)
	if err != nil {
		writeRepositoryError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (api *apiHandler) handleFinancialItemsDelete(w http.ResponseWriter, r *http.Request) {
	if err := api.financialItems.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeRepositoryError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeJSONBody(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func writeRepositoryError(w http.ResponseWriter, err error) {
	var validationError financialitems.ValidationError
	switch {
	case errors.Is(err, financialitems.ErrFinancialItemNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.As(err, &validationError):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeError(w http.ResponseWriter, statusCode int, err error) {
	writeJSON(w, statusCode, errorResponse{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}
