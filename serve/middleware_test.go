package serve

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
)

/*
Test that the middleware will abort the request with 429 if the store returns the request is rate limited.
Otherwise it should successfully call the next handler.
*/
func TestMiddlewareRateLimit(t *testing.T) {
	mockStore := CreateMockStore()
	server := CreateTestServer(mockStore)

	okStatus := http.StatusTeapot
	okText := "all g"
	okHandler := func(ctx context.Context, req *request, res *response) {
		res.Respond(okStatus, nil, okText)
	}

	handler := makeHandler(server.middlewareRateLimit(okHandler, 0, "dogs"))

	router := httprouter.New()
	router.GET("/random/", handler)
	req, err := http.NewRequest("GET", "/random/", nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]bool{
		"Ok":     true,
		"Not ok": false,
	}

	for testName, isRateLimited := range tests {
		t.Run(testName, func(t *testing.T) {
			mockStore.isRateLimited = isRateLimited
			expectedStatus := okStatus
			if isRateLimited {
				expectedStatus = http.StatusTooManyRequests
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != expectedStatus {
				t.Errorf("expected status code %d, got: %d", expectedStatus, rr.Code)
			}
		})
	}
}

func TestMiddlewareCors(t *testing.T) {
	mockStore := CreateMockStore()
	server := CreateTestServer(mockStore)

	allowedOrigin := "example.net"
	okHandler := func(ctx context.Context, req *request, res *response) {
		res.Respond(200, nil, "")
	}

	handler := makeHandler(server.middlewareCORS(okHandler, allowedOrigin))

	router := httprouter.New()
	router.GET("/random/", handler)
	req, err := http.NewRequest("GET", "/random/", nil)
	if err != nil {
		t.Errorf("request creation failure: %v", err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	originResponse := rr.Header().Get("Access-Control-Allow-Origin")
	if originResponse != allowedOrigin {
		t.Errorf("expected allowed origin %s, got %s", allowedOrigin, originResponse)
	}

}
