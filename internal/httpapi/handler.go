package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/zachfire9/financials-api/internal/financialitems"
	"github.com/zachfire9/financials-api/internal/projections"
)

const accessTokenHeader = "X-Financials-Access-Token"

// maxRequestBodyBytes caps JSON request bodies. A backup with thousands of
// accounts is still well under this.
const maxRequestBodyBytes = 1 << 20

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

// HandlerConfig contains cross-cutting HTTP settings for browser-hosted deployments.
type HandlerConfig struct {
	CORS          CORSConfig
	AccessControl AccessControlConfig
	// StatelessProjectionsOnly serves only GET /health and POST /projections,
	// and requires projection requests to include their items. Used for the
	// ephemeral storage driver, where each Lambda instance would otherwise keep
	// its own unshared in-memory copy of saved items.
	StatelessProjectionsOnly bool
}

// CORSConfig contains allowed browser origins for static-hosted UI deployments.
type CORSConfig struct {
	AllowedOrigins []string
}

// AccessControlConfig contains the optional shared-token gate for deployed personal-use APIs.
type AccessControlConfig struct {
	Token string
}

// NewHandlerWithRepositoryAndCORS builds the HTTP handler tree with an injected financial item repository and CORS config.
func NewHandlerWithRepositoryAndCORS(repository financialitems.Repository, corsConfig CORSConfig) http.Handler {
	return NewHandlerWithRepositoryAndConfig(repository, HandlerConfig{CORS: corsConfig})
}

// NewHandlerWithRepositoryAndConfig builds the HTTP handler tree with injected dependencies and deployment config.
func NewHandlerWithRepositoryAndConfig(repository financialitems.Repository, handlerConfig HandlerConfig) http.Handler {
	api := &apiHandler{financialItems: repository, statelessProjectionsOnly: handlerConfig.StatelessProjectionsOnly}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", api.handleHealth)
	mux.HandleFunc("POST /projections", api.handleProjectionsCreate)
	if handlerConfig.StatelessProjectionsOnly {
		return withCORS(withAccessControl(withJSONRoutingErrors(mux), handlerConfig.AccessControl), handlerConfig.CORS)
	}

	mux.HandleFunc("GET /financial-items", api.handleFinancialItemsList)
	mux.HandleFunc("POST /financial-items", api.handleFinancialItemsCreate)
	mux.HandleFunc("GET /financial-items/{id}", api.handleFinancialItemsGet)
	mux.HandleFunc("PUT /financial-items/{id}", api.handleFinancialItemsUpdate)
	mux.HandleFunc("DELETE /financial-items/{id}", api.handleFinancialItemsDelete)
	mux.HandleFunc("GET /financial-items/backup", api.handleFinancialItemsBackupExport)
	mux.HandleFunc("POST /financial-items/backup", api.handleFinancialItemsBackupImport)
	return withCORS(withAccessControl(withJSONRoutingErrors(mux), handlerConfig.AccessControl), handlerConfig.CORS)
}

// withJSONRoutingErrors makes the mux's own 404 and 405 responses JSON, like
// every other error the API returns. Matched routes are served unchanged.
func withJSONRoutingErrors(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(&jsonErrorWriter{ResponseWriter: w}, r)
	})
}

// jsonErrorWriter replaces a plain-text error body (from http.Error) with a
// JSON error body. Non-error responses such as path-cleaning redirects pass through.
type jsonErrorWriter struct {
	http.ResponseWriter
	replaced bool
}

func (w *jsonErrorWriter) WriteHeader(statusCode int) {
	if statusCode < http.StatusBadRequest {
		w.ResponseWriter.WriteHeader(statusCode)
		return
	}
	w.replaced = true
	w.Header().Del("X-Content-Type-Options")
	writeJSON(w.ResponseWriter, statusCode, errorResponse{Error: strings.ToLower(http.StatusText(statusCode))})
}

func (w *jsonErrorWriter) Write(body []byte) (int, error) {
	if w.replaced {
		return len(body), nil
	}
	return w.ResponseWriter.Write(body)
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
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, "+accessTokenHeader)
		}

		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withAccessControl(next http.Handler, config AccessControlConfig) http.Handler {
	expectedToken := strings.TrimSpace(config.Token)
	if expectedToken == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || (r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "") {
			next.ServeHTTP(w, r)
			return
		}
		providedToken := r.Header.Get(accessTokenHeader)
		if subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) != 1 {
			writeError(w, http.StatusUnauthorized, errors.New("missing or invalid access token"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

type apiHandler struct {
	financialItems           financialitems.Repository
	statelessProjectionsOnly bool
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
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeDecodeError(w, err)
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
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeDecodeError(w, err)
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

func (api *apiHandler) handleFinancialItemsBackupExport(w http.ResponseWriter, r *http.Request) {
	backup, err := api.financialItems.ExportBackup(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, backup)
}

func (api *apiHandler) handleFinancialItemsBackupImport(w http.ResponseWriter, r *http.Request) {
	var backup financialitems.Backup
	if err := decodeJSONBody(w, r, &backup); err != nil {
		writeDecodeError(w, err)
		return
	}

	items, err := api.financialItems.ImportBackup(r.Context(), backup)
	if err != nil {
		writeRepositoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (api *apiHandler) handleProjectionsCreate(w http.ResponseWriter, r *http.Request) {
	var request projections.Request
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	if len(request.Items) == 0 && !api.statelessProjectionsOnly {
		items, err := api.financialItems.List(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		request.Items = projectionInputsFromFinancialItems(items)
	}

	projection, err := projections.Calculate(request)
	if err != nil {
		writeProjectionError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, projection)
}

func projectionInputsFromFinancialItems(items []financialitems.FinancialItem) []projections.ItemInput {
	inputs := make([]projections.ItemInput, 0, len(items))
	for _, item := range items {
		inputs = append(inputs, projections.ItemInput{
			ID:                                  item.ID,
			Name:                                item.Name,
			AmountCents:                         item.AmountCents,
			Currency:                            item.Currency,
			AnnualReturnRateBasisPoints:         item.AnnualReturnRateBasisPoints,
			DrawdownAnnualReturnRateBasisPoints: item.DrawdownAnnualReturnRateBasisPoints,
			AnnualContributionCents:             item.AnnualContributionCents,
			InflateAnnualContribution:           item.InflateAnnualContribution,
			SortOrder:                           item.SortOrder,
		})
	}
	return inputs
}

// errTrailingJSON is returned when a request body has data after its JSON value.
var errTrailingJSON = errors.New("request body must contain a single JSON value")

func decodeJSONBody(w http.ResponseWriter, r *http.Request, destination any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return err
		}
		return errTrailingJSON
	}
	return nil
}

// writeDecodeError reports a request body that couldn't be decoded.
func writeDecodeError(w http.ResponseWriter, err error) {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("request body must be %d bytes or less", maxBytesError.Limit))
		return
	}
	writeError(w, http.StatusBadRequest, err)
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

func writeProjectionError(w http.ResponseWriter, err error) {
	var validationError projections.ValidationError
	if errors.As(err, &validationError) {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeError(w, http.StatusInternalServerError, err)
}

// writeError writes a JSON error. Server errors are logged and replaced with a
// generic message so internal details (file paths, wrapped errors) don't reach clients.
func writeError(w http.ResponseWriter, statusCode int, err error) {
	message := err.Error()
	if statusCode >= http.StatusInternalServerError {
		log.Printf("internal error (%d): %v", statusCode, err)
		message = "internal error"
	}
	writeJSON(w, statusCode, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}
