package lambdahttp

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// Adapter translates API Gateway HTTP API v2 requests into the existing net/http handler.
type Adapter struct {
	handler http.Handler
}

// NewAdapter wraps a net/http handler for Lambda/API Gateway HTTP API v2.
func NewAdapter(handler http.Handler) Adapter {
	return Adapter{handler: handler}
}

// Proxy handles one API Gateway HTTP API v2 request.
func (adapter Adapter) Proxy(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if adapter.handler == nil {
		return events.APIGatewayV2HTTPResponse{}, fmt.Errorf("lambda HTTP adapter requires a handler")
	}

	httpRequest, err := requestToHTTP(ctx, request)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	recorder := httptest.NewRecorder()
	adapter.handler.ServeHTTP(recorder, httpRequest)

	response := recorder.Result()
	defer response.Body.Close()

	headers := make(map[string]string)
	for key, values := range response.Header {
		if len(values) > 0 {
			headers[key] = strings.Join(values, ", ")
		}
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode:      response.StatusCode,
		Headers:         headers,
		Body:            recorder.Body.String(),
		IsBase64Encoded: false,
	}, nil
}

func requestToHTTP(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	body := []byte(request.Body)
	if request.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			return nil, fmt.Errorf("decode base64 request body: %w", err)
		}
		body = decoded
	}

	method := request.RequestContext.HTTP.Method
	if method == "" {
		method = http.MethodGet
	}

	path := request.RawPath
	if path == "" {
		path = request.RequestContext.HTTP.Path
	}
	if path == "" {
		path = "/"
	}

	target := path
	if request.RawQueryString != "" {
		target += "?" + request.RawQueryString
	}

	httpRequest, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for key, value := range request.Headers {
		httpRequest.Header.Set(key, value)
	}

	return httpRequest, nil
}
