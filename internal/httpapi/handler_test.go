package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zachfire9/financials-api/internal/financialitems"
	"github.com/zachfire9/financials-api/internal/projections"
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
		"drawdownAnnualReturnRateBasisPoints":350,
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
	if created.DrawdownAnnualReturnRateBasisPoints == nil || *created.DrawdownAnnualReturnRateBasisPoints != 350 {
		t.Fatalf("unexpected drawdown return rate: %+v", created.DrawdownAnnualReturnRateBasisPoints)
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
		"drawdownAnnualReturnRateBasisPoints":150,
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
		"drawdownAnnualReturnRateBasisPoints":100,
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
	if updated.DrawdownAnnualReturnRateBasisPoints == nil || *updated.DrawdownAnnualReturnRateBasisPoints != 100 {
		t.Fatalf("unexpected updated drawdown return rate: %+v", updated.DrawdownAnnualReturnRateBasisPoints)
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
		"drawdownAnnualReturnRateBasisPoints":100001,
		"annualContributionCents":-1,
		"sortOrder":-1
	}`))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}

	var response errorResponse
	decodeJSON(t, recorder, &response)
	for _, want := range []string{"name is required", "amountCents", "currency", "annualReturnRateBasisPoints", "drawdownAnnualReturnRateBasisPoints", "annualContributionCents", "sortOrder"} {
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

func TestProjectionEndpointCalculatesHypotheticalItemsWithoutSaving(t *testing.T) {
	handler := NewHandlerWithRepository(financialitems.NewInMemoryRepository())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/projections", strings.NewReader(`{
		"years":2,
		"items":[
			{
				"name":"Example brokerage",
				"amountCents":20000,
				"currency":"USD",
				"annualReturnRateBasisPoints":1000,
				"annualContributionCents":2000,
				"sortOrder":2
			},
			{
				"name":"Example savings",
				"amountCents":10000,
				"currency":"USD",
				"annualReturnRateBasisPoints":500,
				"annualContributionCents":1000,
				"sortOrder":1
			}
		]
	}`))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var projection projections.Projection
	decodeJSON(t, recorder, &projection)
	if projection.Years != 2 || projection.Currency != "USD" {
		t.Fatalf("unexpected projection metadata: %+v", projection)
	}
	if len(projection.Items) != 2 {
		t.Fatalf("expected two projected items, got %d", len(projection.Items))
	}
	if projection.Items[0].Name != "Example savings" || projection.Items[1].Name != "Example brokerage" {
		t.Fatalf("expected projection items sorted by sortOrder, got %+v", projection.Items)
	}
	assertProjectionBalance(t, projection.Items[0].YearlyBalances[2], 2, 13075, 1000, 575)
	assertProjectionBalance(t, projection.Items[1].YearlyBalances[2], 2, 28400, 2000, 2400)
	assertProjectionBalance(t, projection.Totals[2], 2, 41475, 3000, 2975)

	listRecorder := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/financial-items", nil)
	handler.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected financial item list status %d, got %d", http.StatusOK, listRecorder.Code)
	}
	var listed []financialitems.FinancialItem
	decodeJSON(t, listRecorder, &listed)
	if len(listed) != 0 {
		t.Fatalf("expected hypothetical projection items not to be saved, got %+v", listed)
	}
}

func TestProjectionEndpointCalculatesDrawdownHypotheticalItemsWithoutSaving(t *testing.T) {
	handler := NewHandlerWithRepository(financialitems.NewInMemoryRepository())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/projections", strings.NewReader(`{
		"savingYears":1,
		"drawdownYears":2,
		"annualWithdrawalCents":6000000,
		"annualWithdrawalInflationRateBasisPoints":300,
		"items":[
			{
				"name":"Example retirement account",
				"amountCents":20000000,
				"currency":"USD",
				"annualReturnRateBasisPoints":0,
				"drawdownAnnualReturnRateBasisPoints":0,
				"annualContributionCents":100000,
				"sortOrder":1
			}
		]
	}`))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var projection projections.Projection
	decodeJSON(t, recorder, &projection)
	if projection.Years != 3 || projection.SavingYears != 1 || projection.DrawdownYears != 2 || projection.Currency != "USD" {
		t.Fatalf("unexpected drawdown projection metadata: %+v", projection)
	}
	if len(projection.Items) != 1 {
		t.Fatalf("expected one projected item, got %d", len(projection.Items))
	}
	if projection.Items[0].DrawdownAnnualReturnRateBasisPoints == nil || *projection.Items[0].DrawdownAnnualReturnRateBasisPoints != 0 {
		t.Fatalf("expected drawdown return rate in response, got %+v", projection.Items[0].DrawdownAnnualReturnRateBasisPoints)
	}
	assertProjectionPhaseBalance(t, projection.Items[0].YearlyBalances[0], 0, projections.PhaseStarting, 20000000, 0, 0, 0, 0)
	assertProjectionPhaseBalance(t, projection.Items[0].YearlyBalances[1], 1, projections.PhaseSaving, 20100000, 100000, 0, 0, 0)
	assertProjectionPhaseBalance(t, projection.Items[0].YearlyBalances[2], 2, projections.PhaseDrawdown, 14100000, 0, 6000000, 0, 0)
	assertProjectionPhaseBalance(t, projection.Items[0].YearlyBalances[3], 3, projections.PhaseDrawdown, 7920000, 0, 6180000, 0, 0)
	assertProjectionPhaseBalance(t, projection.Totals[3], 3, projections.PhaseDrawdown, 7920000, 0, 6180000, 0, 0)

	listRecorder := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/financial-items", nil)
	handler.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected financial item list status %d, got %d", http.StatusOK, listRecorder.Code)
	}
	var listed []financialitems.FinancialItem
	decodeJSON(t, listRecorder, &listed)
	if len(listed) != 0 {
		t.Fatalf("expected hypothetical drawdown projection items not to be saved, got %+v", listed)
	}
}

func TestProjectionEndpointUsesRepositoryItemsForDrawdownWhenItemsOmittedOrEmpty(t *testing.T) {
	repository := financialitems.NewInMemoryRepository()
	handler := NewHandlerWithRepository(repository)
	created := createFinancialItem(t, handler, `{
		"name":"Stored retirement account",
		"amountCents":10000000,
		"currency":"USD",
		"annualReturnRateBasisPoints":0,
		"drawdownAnnualReturnRateBasisPoints":500,
		"annualContributionCents":0,
		"sortOrder":1
	}`)

	for _, body := range []string{`{"savingYears":0,"drawdownYears":1,"annualWithdrawalCents":1000000}`, `{"savingYears":0,"drawdownYears":1,"annualWithdrawalCents":1000000,"items":[]}`} {
		t.Run(body, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/projections", strings.NewReader(body))
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
			}
			var projection projections.Projection
			decodeJSON(t, recorder, &projection)
			if projection.SavingYears != 0 || projection.DrawdownYears != 1 {
				t.Fatalf("unexpected phase metadata: %+v", projection)
			}
			if len(projection.Items) != 1 || projection.Items[0].ID != created.ID {
				t.Fatalf("expected repository item identity in projection, got %+v", projection.Items)
			}
			assertProjectionPhaseBalance(t, projection.Items[0].YearlyBalances[1], 1, projections.PhaseDrawdown, 9500000, 0, 1000000, 500000, 0)
		})
	}
}

func TestProjectionEndpointUsesRepositoryItemsWhenItemsOmittedOrEmpty(t *testing.T) {
	repository := financialitems.NewInMemoryRepository()
	handler := NewHandlerWithRepository(repository)
	created := createFinancialItem(t, handler, `{
		"name":"Stored brokerage",
		"amountCents":20000,
		"currency":"USD",
		"annualReturnRateBasisPoints":1000,
		"annualContributionCents":2000,
		"sortOrder":1
	}`)

	for _, body := range []string{`{"years":1}`, `{"years":1,"items":[]}`} {
		t.Run(body, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/projections", strings.NewReader(body))
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
			}
			var projection projections.Projection
			decodeJSON(t, recorder, &projection)
			if len(projection.Items) != 1 {
				t.Fatalf("expected one repository-backed projection item, got %d", len(projection.Items))
			}
			if projection.Items[0].ID != created.ID || projection.Items[0].Name != "Stored brokerage" {
				t.Fatalf("expected repository item identity in projection, got %+v", projection.Items[0])
			}
			assertProjectionBalance(t, projection.Items[0].YearlyBalances[1], 1, 24000, 2000, 2000)
		})
	}
}

func TestProjectionEndpointReturnsValidationFailures(t *testing.T) {
	handler := NewHandlerWithRepository(financialitems.NewInMemoryRepository())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/projections", strings.NewReader(`{
		"years":0,
		"items":[
			{
				"name":" ",
				"amountCents":-1,
				"currency":"usd",
				"annualReturnRateBasisPoints":100001,
				"annualContributionCents":-1,
				"sortOrder":-1
			}
		]
	}`))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	var response errorResponse
	decodeJSON(t, recorder, &response)
	for _, want := range []string{"years", "name", "amountCents", "currency", "annualReturnRateBasisPoints", "annualContributionCents", "sortOrder"} {
		if !strings.Contains(response.Error, want) {
			t.Fatalf("expected projection validation response to contain %q, got %q", want, response.Error)
		}
	}
}

func TestProjectionEndpointReturnsDrawdownValidationFailures(t *testing.T) {
	handler := NewHandlerWithRepository(financialitems.NewInMemoryRepository())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/projections", strings.NewReader(`{
		"years":10,
		"savingYears":1,
		"drawdownYears":1,
		"annualWithdrawalCents":0,
		"annualWithdrawalInflationRateBasisPoints":-1,
		"items":[
			{
				"name":"Example retirement account",
				"amountCents":10000000,
				"currency":"USD",
				"annualReturnRateBasisPoints":0,
				"drawdownAnnualReturnRateBasisPoints":100001,
				"annualContributionCents":0,
				"sortOrder":1
			}
		]
	}`))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	var response errorResponse
	decodeJSON(t, recorder, &response)
	for _, want := range []string{"mutually exclusive", "annualWithdrawalCents", "annualWithdrawalInflationRateBasisPoints", "drawdownAnnualReturnRateBasisPoints"} {
		if !strings.Contains(response.Error, want) {
			t.Fatalf("expected drawdown validation response to contain %q, got %q", want, response.Error)
		}
	}
}

func TestProjectionEndpointRejectsUnknownFields(t *testing.T) {
	handler := NewHandlerWithRepository(financialitems.NewInMemoryRepository())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/projections", strings.NewReader(`{
		"years":10,
		"unexpected":"field"
	}`))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	var response errorResponse
	decodeJSON(t, recorder, &response)
	if !strings.Contains(response.Error, "unknown field") {
		t.Fatalf("expected unknown field error, got %q", response.Error)
	}
}

func assertProjectionBalance(t *testing.T, got projections.YearlyBalance, year int, balanceCents int64, contributionCents int64, growthCents int64) {
	t.Helper()
	if got.Year != year || got.BalanceCents != balanceCents || got.ContributionCents != contributionCents || got.GrowthCents != growthCents {
		t.Fatalf("unexpected projection balance: got %+v, want year=%d balance=%d contribution=%d growth=%d", got, year, balanceCents, contributionCents, growthCents)
	}
}

func assertProjectionPhaseBalance(t *testing.T, got projections.YearlyBalance, year int, phase projections.Phase, balanceCents int64, contributionCents int64, withdrawalCents int64, growthCents int64, unfundedWithdrawalCents int64) {
	t.Helper()
	if got.Year != year || got.Phase != phase || got.BalanceCents != balanceCents || got.ContributionCents != contributionCents || got.WithdrawalCents != withdrawalCents || got.GrowthCents != growthCents || got.UnfundedWithdrawalCents != unfundedWithdrawalCents {
		t.Fatalf("unexpected projection phase balance: got %+v, want year=%d phase=%s balance=%d contribution=%d withdrawal=%d growth=%d unfundedWithdrawal=%d", got, year, phase, balanceCents, contributionCents, withdrawalCents, growthCents, unfundedWithdrawalCents)
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
