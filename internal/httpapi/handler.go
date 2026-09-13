package httpapi

import (
	"encoding/json"
	"net/http"
)

// NewHandler builds the HTTP handler tree for the API.
//
// Keep this package independent from process startup so the same handler can be
// used by local net/http serving now and an AWS Lambda/API Gateway adapter later.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	return mux
}

type healthResponse struct {
	Status string `json:"status"`
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}
