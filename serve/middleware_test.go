package serve

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"spiritchat/auth"
	"testing"

	"github.com/julienschmidt/httprouter"
)

/*
Test that the middleware will abort the request with 429 if the store returns the request is rate limited.
Otherwise it should successfully call the next handler.
*/
func TestMiddlewareRateLimit(t *testing.T) {
	mockStore := &MockStore{}
	mockAuth := &MockAuth{}
	server := CreateTestServer(mockStore, mockAuth)

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
	mockStore := &MockStore{}
	mockAuth := &MockAuth{}
	server := CreateTestServer(mockStore, mockAuth)

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
func TestMiddleware(t *testing.T) {
	mockStore := &MockStore{}
	mockAuth := &MockAuth{}
	server := CreateTestServer(mockStore, mockAuth)

	nextStatus := http.StatusTeapot
	okText := "ok"
	okHandler := func(ctx context.Context, req *request, res *response) {
		res.Respond(nextStatus, nil, okText)
	}

	handler := makeHandler(server.middlewareRequireLogin(okHandler))

	router := httprouter.New()
	router.GET("/random/", handler)

	tests := map[string]map[int]func(req *http.Request, mock *MockAuth){
		"No header": {
			http.StatusForbidden: func(req *http.Request, mock *MockAuth) {
				req.Header.Set("Authorization", "")
				mock.err = nil
			},
		},
		"Good header, ok, no user": {
			http.StatusNotFound: func(req *http.Request, mock *MockAuth) {
				req.Header.Set("Authorization", "data")
				mock.err = nil
				mock.user = nil
			},
		},
		"Good header, not ok": {
			http.StatusForbidden: func(req *http.Request, mock *MockAuth) {
				req.Header.Set("Authorization", "")
				mock.err = errors.New("no")
			},
		},
		"Good header, ok, has user": {
			nextStatus: func(req *http.Request, mock *MockAuth) {
				req.Header.Set("Authorization", "data")
				mock.err = nil
				mock.user = &auth.UserData{
					Username: "beep",
					Email:    "boop",
				}
			},
		},
	}

	for testName, test := range tests {
		for expectCode, setup := range test {
			t.Run(testName, func(t *testing.T) {
				req, err := http.NewRequest("GET", "/random/", nil)
				if err != nil {
					t.Fatal(err)
				}
				setup(req, mockAuth)

				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)
				if rr.Code != expectCode {
					t.Errorf("expected status code %d, got: %d", expectCode, rr.Code)
				}
			})
		}
	}
}
