package serve

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"spiritchat/data"
	"testing"

	"github.com/julienschmidt/httprouter"
)

type MockStore struct {
	isRateLimited   bool
	err             error
	getThreadView   *data.ThreadView
	getCategories   []*data.Category
	getCategory     *data.Category
	getCategoryView *data.CatView
}

// Cleanup cleans the underlying connection to the data store.
func (ms *MockStore) Cleanup(ctx context.Context) error {
	panic("not implemented") // TODO: Implement
}

// IsRateLimited returns true if the given IP is being rate limited.
func (ms *MockStore) IsRateLimited(identifier string, resource string) (bool, error) {
	return ms.isRateLimited, nil
}

// RateLimit marks IP & Resource as rate limited for n ms.
func (ms *MockStore) RateLimit(identifier string, resource string, _ int) error {
	return nil
}

// WriteCategory adds a new category to the database.
func (ms *MockStore) WriteCategory(ctx context.Context, catName string) error {
	panic("not implemented") // TODO: Implement
}

/*
RemoveCategory removes all posts under category catName and removes the category.
Returns affected rows.
*/
func (ms *MockStore) RemoveCategory(ctx context.Context, catName string) (int64, error) {
	panic("not implemented") // TODO: Implement
}

// GetThreadCount returns the number of threads in a category.
func (ms *MockStore) GetThreadCount(ctx context.Context, catName string) (int, error) {
	panic("not implemented") // TODO: Implement
}

// GetCategories returns all categories.
func (ms *MockStore) GetCategories(ctx context.Context) ([]*data.Category, error) {
	return ms.getCategories, ms.err
}

/*
GetPostByNumber returns a post in a category by its number.
Should return ErrNotFound if no such post.
*/
func (ms *MockStore) GetPostByNumber(ctx context.Context, catName string, num int) (*data.Post, error) {
	panic("not implemented") // TODO: Implement
}

/*
GetThreadView returns all the posts in a thread, and the category they're on.
Should return ErrNotFound if the requested thread is not an OP thread, or the category
is invalid
*/
func (ms *MockStore) GetThreadView(ctx context.Context, catName string, threadNum int) (*data.ThreadView, error) {
	return ms.getThreadView, ms.err
}

/*
GetCategory returns a single category. May return ErrNotFound if the given category
name is invalid.
*/
func (ms *MockStore) GetCategory(ctx context.Context, catName string) (*data.Category, error) {
	return ms.getCategory, ms.err
}

/*
GetCatView returns information about a category, and all the threads on it.
May return an ErrNotFound if the given category name is invalid.
*/
func (ms *MockStore) GetCatView(ctx context.Context, catName string) (*data.CatView, error) {
	return ms.getCategoryView, ms.err
}

/*
Creates a post.
Optional parent thread can be provided if it's a reply.
Should return ErrNotFound if invalid post or category.
*/
func (ms *MockStore) WritePost(ctx context.Context, catName string, parentThreadNumber int, p *data.UserPost) error {
	return ms.err
}

func CreateMockStore() *MockStore {
	return &MockStore{}
}

func CreateTestServer(mockStore *MockStore) *Server {
	return NewServer(mockStore, ServerOptions{
		Address:             "0.0.0.0",
		PostCooldownSeconds: 0,
		CorsOriginAllow:     "",
	})
}

/*
Test that the middleware will abort the request with 429 if the store returns the request is rate limited.
Otherwise it should successfully call the next handler.
*/
func TestMiddlewareRateLimit(t *testing.T) {
	mockStore := CreateMockStore()
	server := CreateTestServer(mockStore)

	okStatus := http.StatusTeapot
	okText := "all g"
	okHandler := func(ctx context.Context, req *request, respond respondFunc) {
		respond(okStatus, nil, okText)
	}

	handler := genHandler(server.middlewareRateLimit(okHandler, 0, "dogs"))

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

func TestHandleCORSPreflight(t *testing.T) {
	tests := []string{
		"www.google.com",
		"http://localhost:7070",
		"0.0.0.0:3000",
		"myapi.cooldogs.com",
	}

	for _, allowedOrigin := range tests {
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("OPTIONS", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		handler := handleCORSPreflight(allowedOrigin)
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Errorf("expected preflight status %d, got: %d", http.StatusNoContent, rr.Code)
		}

		resAllowedOrigin := rr.Header().Get("Access-Control-Allow-Origin")
		if resAllowedOrigin != allowedOrigin {
			t.Errorf("expected allowed origin header to match %s, got: %s", allowedOrigin, resAllowedOrigin)
		}

		resAllowedMethods := rr.Header().Get("Access-Control-Allow-Methods")
		if resAllowedMethods != "GET,POST" {
			t.Errorf("expected allowed methods header for GET,POST, got: %s", resAllowedMethods)
		}

		resAllowedHeaders := rr.Header().Get("Access-Control-Allow-Headers")
		if resAllowedHeaders != "Content-Type" {
			t.Errorf("expected Content-Type header allowed in CORS response, got: %s", resAllowedHeaders)
		}
	}
}

type RouteMockTest struct {
	route        string
	setup        func(*MockStore)
	expectedCode int
	body         []byte
}

func TestRoutes(t *testing.T) {
	tests := map[string]map[string]RouteMockTest{
		"GET": {
			"Invalid URL": {
				route:        "/nothing-here",
				expectedCode: http.StatusNotFound,
			},
			"Gategories": {
				route:        "/v1",
				expectedCode: http.StatusOK,
			},
			"Category view (Not Found)": {
				route:        "/v1/none",
				expectedCode: http.StatusNotFound,
				setup: func(ms *MockStore) {
					ms.err = data.ErrNotFound
				},
			},
			"Category view (Valid)": {
				expectedCode: http.StatusOK,
				route:        "/v1/valid",
				setup: func(ms *MockStore) {
					ms.getCategoryView = &data.CatView{
						Category: &data.Category{
							Name: "beep",
						},
						Threads: []*data.Post{},
					}
				},
			},
			"Thread View (not found)": {
				expectedCode: http.StatusNotFound,
				route:        "/v1/nothing/5",
				setup: func(ms *MockStore) {
					ms.err = data.ErrNotFound
				},
			},
			"Thread View (bad formatting)": {
				expectedCode: http.StatusBadRequest,
				route:        "/v1/something/here?",
			},
			"Thread View (valid)": {
				expectedCode: http.StatusOK,
				route:        "/v1/something/1",
			},
		},
		"POST": {
			"Write Thread (bad formatting)": {
				expectedCode: http.StatusBadRequest,
				route:        "/v1/cat/beepboop",
			},
			"Write Thread (bad empty thread)": {
				expectedCode: http.StatusBadRequest,
				route:        "/v1/cat/1",
				body:         []byte(`{"Content": ""}`),
			},
			"Write Thread (not found)": {
				expectedCode: http.StatusNotFound,
				route:        "/v1/cat/5",
				body:         []byte(`{"Content": "hello!"}`),
				setup: func(ms *MockStore) {
					ms.err = data.ErrNotFound
				},
			},
			"Write Thread (valid)": {
				expectedCode: http.StatusOK,
				body:         []byte(`{"Content": "hello!"}`),
				route:        "/v1/cat/1",
			},
		},
	}

	for method, routeTest := range tests {
		for testName, test := range routeTest {
			test := test
			t.Run(fmt.Sprintf("%s %s", method, testName), func(t *testing.T) {
				mockStore := CreateMockStore()
				if test.setup != nil {
					test.setup(mockStore)
				}
				server := CreateTestServer(mockStore)

				rr := httptest.NewRecorder()
				req, err := http.NewRequest(method, test.route, bytes.NewReader(test.body))
				if err != nil {
					t.Fatal(err)
				}
				server.ServeHTTP(rr, req)
				if rr.Code != test.expectedCode {
					t.Errorf("%s: %s, expected status %d, got: %d", method, test.route, test.expectedCode, rr.Code)
				}
			})
		}
	}
}
