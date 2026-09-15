package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zachfire9/financials-api/internal/financialitems"
)

func TestHealthEndpoint(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}

	var response healthResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Fatalf("expected status ok, got %q", response.Status)
	}
}

func TestUnknownRouteReturnsNotFound(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)

	NewHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestCORSMiddlewareAllowsConfiguredOrigins(t *testing.T) {
	handler := NewHandlerWithRepositoryAndCORS(
		financialitems.NewInMemoryRepository(),
		CORSConfig{AllowedOrigins: []string{"https://example-amplify-app.example.com"}},
	)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://example-amplify-app.example.com")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "https://example-amplify-app.example.com" {
		t.Fatalf("expected allowed origin header, got %q", origin)
	}
	if vary := recorder.Header().Get("Vary"); vary != "Origin" {
		t.Fatalf("expected Vary: Origin, got %q", vary)
	}
}

func TestCORSMiddlewareRejectsUnconfiguredOrigins(t *testing.T) {
	handler := NewHandlerWithRepositoryAndCORS(
		financialitems.NewInMemoryRepository(),
		CORSConfig{AllowedOrigins: []string{"https://example-amplify-app.example.com"}},
	)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://unexpected.example.com")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Fatalf("expected no allowed origin header for rejected origin, got %q", origin)
	}
}

func TestCORSMiddlewareHandlesPreflightForConfiguredOrigins(t *testing.T) {
	handler := NewHandlerWithRepositoryAndCORS(
		financialitems.NewInMemoryRepository(),
		CORSConfig{AllowedOrigins: []string{"https://example-amplify-app.example.com"}},
	)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/financial-items", nil)
	request.Header.Set("Origin", "https://example-amplify-app.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "https://example-amplify-app.example.com" {
		t.Fatalf("expected allowed origin header, got %q", origin)
	}
	if methods := recorder.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(methods, http.MethodPost) || !strings.Contains(methods, http.MethodOptions) {
		t.Fatalf("expected CORS methods to include POST and OPTIONS, got %q", methods)
	}
	if headers := recorder.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(headers, "Content-Type") {
		t.Fatalf("expected CORS headers to include Content-Type, got %q", headers)
	}
}

func TestFinancialItemsEndpointCreatesAndListsItems(t *testing.T) {
	handler := NewHandlerWithRepository(financialitems.NewInMemoryRepository())

	created := createFinancialItem(t, handler, `{
		"name":"Example brokerage",
		"amountCents":1250000,
		"currency":"USD",
		"annualReturnRateBasisPoints":700,
		"annualContributionCents":300000,
		"sortOrder":2
	}`)

	if created.ID == "" {
		t.Fatal("expected generated id")
	}
	if created.Name != "Example brokerage" || created.AmountCents != 1250000 || created.Currency != "USD" {
		t.Fatalf("unexpected created item: %+v", created)
	}
	if created.AnnualReturnRateBasisPoints != 700 || created.AnnualContributionCents != 300000 || created.SortOrder != 2 {
		t.Fatalf("unexpected projection fields: %+v", created)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("expected timestamps on created item: %+v", created)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/financial-items", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var listed []financialitems.FinancialItem
	decodeJSON(t, recorder, &listed)
	if len(listed) != 1 {
		t.Fatalf("expected one listed item, got %d", len(listed))
	}
	if listed[0].ID != created.ID {
		t.Fatalf("expected listed item %q, got %+v", created.ID, listed[0])
	}
}

func TestFinancialItemsEndpointReadsUpdatesAndDeletesItems(t *testing.T) {
	handler := NewHandlerWithRepository(financialitems.NewInMemoryRepository())
	created := createFinancialItem(t, handler, `{
		"name":"Cash savings",
		"amountCents":400000,
		"currency":"USD",
		"annualReturnRateBasisPoints":450,
		"annualContributionCents":50000,
		"sortOrder":1
	}`)

	getRecorder := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/financial-items/"+created.ID, nil)
	handler.ServeHTTP(getRecorder, getRequest)

	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, getRecorder.Code, getRecorder.Body.String())
	}
	var fetched financialitems.FinancialItem
	decodeJSON(t, getRecorder, &fetched)
	if fetched.ID != created.ID || fetched.Name != "Cash savings" {
		t.Fatalf("unexpected fetched item: %+v", fetched)
	}

	putRecorder := httptest.NewRecorder()
	putRequest := httptest.NewRequest(http.MethodPut, "/financial-items/"+created.ID, strings.NewReader(`{
		"name":"House down payment",
		"amountCents":550000,
		"currency":"USD",
		"annualReturnRateBasisPoints":300,
		"annualContributionCents":75000,
		"sortOrder":3
	}`))
	handler.ServeHTTP(putRecorder, putRequest)

	if putRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, putRecorder.Code, putRecorder.Body.String())
	}
	var updated financialitems.FinancialItem
	decodeJSON(t, putRecorder, &updated)
	if updated.ID != created.ID || updated.Name != "House down payment" || updated.AmountCents != 550000 {
		t.Fatalf("unexpected updated item: %+v", updated)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("expected updated timestamp after create timestamp, got created=%s updated=%s", created.UpdatedAt, updated.UpdatedAt)
	}

	deleteRecorder := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/financial-items/"+created.ID, nil)
	handler.ServeHTTP(deleteRecorder, deleteRequest)

	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusNoContent, deleteRecorder.Code, deleteRecorder.Body.String())
	}

	missingRecorder := httptest.NewRecorder()
	missingRequest := httptest.NewRequest(http.MethodGet, "/financial-items/"+created.ID, nil)
	handler.ServeHTTP(missingRecorder, missingRequest)

	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d after delete, got %d", http.StatusNotFound, missingRecorder.Code)
	}
}

func TestFinancialItemsEndpointReturnsValidationFailures(t *testing.T) {
	handler := NewHandlerWithRepository(financialitems.NewInMemoryRepository())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/financial-items", strings.NewReader(`{
		"name":" ",
		"amountCents":-1,
		"currency":"usd",
		"annualReturnRateBasisPoints":100001,
		"annualContributionCents":-1,
		"sortOrder":-1
	}`))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}

	var response errorResponse
	decodeJSON(t, recorder, &response)
	for _, want := range []string{"name is required", "amountCents", "currency", "annualReturnRateBasisPoints", "annualContributionCents", "sortOrder"} {
		if !strings.Contains(response.Error, want) {
			t.Fatalf("expected validation response to contain %q, got %q", want, response.Error)
		}
	}
}

func TestFinancialItemsEndpointReturnsNotFoundForMissingItems(t *testing.T) {
	handler := NewHandlerWithRepository(financialitems.NewInMemoryRepository())

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			body := bytes.NewBufferString(`{
				"name":"Missing",
				"amountCents":1,
				"currency":"USD",
				"annualReturnRateBasisPoints":1,
				"annualContributionCents":1,
				"sortOrder":1
			}`)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(method, "/financial-items/item_missing", body)
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusNotFound {
				t.Fatalf("expected status %d, got %d with body %s", http.StatusNotFound, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func createFinancialItem(t *testing.T, handler http.Handler, body string) financialitems.FinancialItem {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/financial-items", strings.NewReader(body))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var created financialitems.FinancialItem
	decodeJSON(t, recorder, &created)
	return created
}

func decodeJSON(t *testing.T, recorder *httptest.ResponseRecorder, destination any) {
	t.Helper()

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}
	if err := json.NewDecoder(recorder.Body).Decode(destination); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
