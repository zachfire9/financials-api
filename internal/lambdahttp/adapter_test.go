package lambdahttp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/zachfire9/financials-api/internal/financialitems"
	"github.com/zachfire9/financials-api/internal/httpapi"
)

func TestAdapterProxiesHealthRequests(t *testing.T) {
	adapter := NewAdapter(httpapi.NewHandler())

	response, err := adapter.Proxy(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/health",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: http.MethodGet},
		},
	})
	if err != nil {
		t.Fatalf("proxy: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, response.StatusCode, response.Body)
	}
	if contentType := response.Headers["Content-Type"]; contentType != "application/json" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}
	if response.Body != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected body %q", response.Body)
	}
}

func TestAdapterProxiesJSONRequestsAndHeaders(t *testing.T) {
	handler := httpapi.NewHandlerWithRepositoryAndCORS(
		financialitems.NewInMemoryRepository(),
		httpapi.CORSConfig{AllowedOrigins: []string{"https://example-static-ui.example.com"}},
	)
	adapter := NewAdapter(handler)

	body := `{"name":"Example brokerage","amountCents":1250000,"currency":"USD","annualReturnRateBasisPoints":700,"annualContributionCents":300000,"sortOrder":1}`
	response, err := adapter.Proxy(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/financial-items",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Origin":       "https://example-static-ui.example.com",
		},
		Body:            base64.StdEncoding.EncodeToString([]byte(body)),
		IsBase64Encoded: true,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: http.MethodPost},
		},
	})
	if err != nil {
		t.Fatalf("proxy: %v", err)
	}

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusCreated, response.StatusCode, response.Body)
	}
	if origin := response.Headers["Access-Control-Allow-Origin"]; origin != "https://example-static-ui.example.com" {
		t.Fatalf("expected CORS origin header, got %q", origin)
	}

	var created financialitems.FinancialItem
	if err := json.Unmarshal([]byte(response.Body), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.ID == "" || created.Name != "Example brokerage" {
		t.Fatalf("unexpected created item: %+v", created)
	}
}

func TestAdapterRejectsInvalidBase64Body(t *testing.T) {
	adapter := NewAdapter(httpapi.NewHandler())

	_, err := adapter.Proxy(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath:         "/financial-items",
		Body:            "not-base64%%%",
		IsBase64Encoded: true,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: http.MethodPost},
		},
	})
	if err == nil {
		t.Fatal("expected invalid base64 error")
	}
}
