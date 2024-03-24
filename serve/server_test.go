package serve

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"spiritchat/auth"
	"spiritchat/data"
	"testing"
)

type MockStore struct {
	isRateLimited   bool
	err             error
	getThreadView   *data.ThreadView
	getCategories   []*data.Category
	getCategory     *data.Category
	getCategoryView *data.CatView
}

func (ms *MockStore) Cleanup(ctx context.Context) error {
	panic("not implemented") // TODO: Implement
}

func (ms *MockStore) IsRateLimited(identifier string, resource string) (bool, error) {
	return ms.isRateLimited, nil
}

func (ms *MockStore) RateLimit(identifier string, resource string, _ int) error {
	return nil
}

func (ms *MockStore) WriteCategory(ctx context.Context, tag string, name string) error {
	panic("not implemented") // TODO: Implement
}

func (ms *MockStore) RemoveCategory(ctx context.Context, catName string) (int64, error) {
	panic("not implemented") // TODO: Implement
}

func (ms *MockStore) GetThreadCount(ctx context.Context, catName string) (int, error) {
	panic("not implemented") // TODO: Implement
}

func (ms *MockStore) GetCategories(ctx context.Context) ([]*data.Category, error) {
	return ms.getCategories, ms.err
}

func (ms *MockStore) GetPostByNumber(ctx context.Context, catName string, num int) (*data.Post, error) {
	panic("not implemented") // TODO: Implement
}

func (ms *MockStore) GetThreadView(ctx context.Context, catName string, threadNum int) (*data.ThreadView, error) {
	return ms.getThreadView, ms.err
}

func (ms *MockStore) GetCategory(ctx context.Context, catName string) (*data.Category, error) {
	return ms.getCategory, ms.err
}

func (ms *MockStore) GetCategoryView(ctx context.Context, catName string) (*data.CatView, error) {
	return ms.getCategoryView, ms.err
}

func (ms *MockStore) WritePost(ctx context.Context, catName string, parentThreadNumber int, subject string, content string, username string, email string, ip string) error {
	return ms.err
}

func (ms *MockStore) RemovePost(ctx context.Context, categoryTag string, number int) (int, error) {
	return 0, ms.err
}

func (ms *MockStore) EmailMatches(ctx context.Context, categoryTag string, postNumber int, email string) (bool, error) {
	return true, ms.err
}

func (ms *MockStore) GetPostsByEmail(ctx context.Context, email string) ([]*data.Post, error) {
	var d []*data.Post
	return d, ms.err
}

type MockAuth struct {
	err  error
	user *auth.UserData
}

func (ma *MockAuth) RequestSignUp(
	ctx context.Context,
	username string, email string, password string,
) (*auth.UserData, error) {
	return ma.user, ma.err
}

func (ma *MockAuth) GetUserFromToken(
	ctx context.Context,
	token string,
) (*auth.UserData, error) {
	return ma.user, ma.err
}

func CreateTestServer(mockStore *MockStore, mockAuth *MockAuth) *Server {
	return NewServer(mockStore, mockAuth, ServerOptions{
		Address:             "0.0.0.0",
		PostCooldownSeconds: 0,
		CorsOriginAllow:     "",
	})
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
		if resAllowedHeaders != "Content-Type,Authorization" {
			t.Errorf("expected Content-Type header allowed in CORS response, got: %s", resAllowedHeaders)
		}
	}
}

type RouteMockTest struct {
	route        string
	setup        func(*MockStore, *MockAuth, *http.Request)
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
				route:        "/v1/categories",
				expectedCode: http.StatusOK,
			},
			"Category view (Not Found)": {
				route:        "/v1/categories/none",
				expectedCode: http.StatusNotFound,
				setup: func(ms *MockStore, ma *MockAuth, r *http.Request) {
					ms.err = data.ErrNotFound
				},
			},
			"Category view (Valid)": {
				expectedCode: http.StatusOK,
				route:        "/v1/categories/valid",
				setup: func(ms *MockStore, ma *MockAuth, r *http.Request) {
					ms.getCategoryView = &data.CatView{
						Category: &data.Category{
							Tag: "beep",
						},
						Threads: []*data.Post{},
					}
				},
			},
			"Thread View (not found)": {
				expectedCode: http.StatusNotFound,
				route:        "/v1/categories/nothing/5",
				setup: func(ms *MockStore, ma *MockAuth, r *http.Request) {
					ms.err = data.ErrNotFound
				},
			},
			"Thread View (bad formatting)": {
				expectedCode: http.StatusBadRequest,
				route:        "/v1/categories/something/here?",
			},
			"Thread View (valid)": {
				expectedCode: http.StatusOK,
				route:        "/v1/categories/something/1",
			},
		},
		"POST": {
			"Write Thread (bad formatting)": {
				expectedCode: http.StatusBadRequest,
				route:        "/v1/categories/cat/beepboop",
				setup: func(ms *MockStore, ma *MockAuth, r *http.Request) {
					r.Header.Add("Authorization", "ok")
					ma.err = nil
					ma.user = &auth.UserData{
						Username:   "test user",
						Email:      "test@gmail.com",
						IsVerified: true,
					}
				},
			},
			"Write Thread (bad empty thread)": {
				expectedCode: http.StatusBadRequest,
				route:        "/v1/categories/cat/1",
				body:         []byte(`{"Content": ""}`),
				setup: func(ms *MockStore, ma *MockAuth, r *http.Request) {
					r.Header.Add("Authorization", "ok")
					ma.err = nil
					ma.user = &auth.UserData{
						Username:   "test user",
						Email:      "test@gmail.com",
						IsVerified: true,
					}
				},
			},
			"Write Thread (not found)": {
				expectedCode: http.StatusNotFound,
				route:        "/v1/categories/cat/5",
				body:         []byte(`{"Content": "hello!"}`),
				setup: func(ms *MockStore, ma *MockAuth, r *http.Request) {
					r.Header.Add("Authorization", "ok")
					ma.err = nil
					ma.user = &auth.UserData{
						Username:   "test user",
						Email:      "test@gmail.com",
						IsVerified: true,
					}
					ms.err = data.ErrNotFound
				},
			},
			"Write Thread (valid)": {
				expectedCode: http.StatusOK,
				body:         []byte(`{"Content": "hello!"}`),
				route:        "/v1/categories/cat/1",
				setup: func(ms *MockStore, ma *MockAuth, r *http.Request) {
					r.Header.Add("Authorization", "ok")
					ma.err = nil
					ma.user = &auth.UserData{
						Username:   "test user",
						Email:      "test@gmail.com",
						IsVerified: true,
					}
				},
			},
			"Sign Up (no username)": {
				expectedCode: http.StatusBadRequest,
				route:        "/v1/signup",
				body:         []byte(`{"username": "", password: "beep", email:"nah@gmail.com"}`),
			},
			"Sign Up (no password)": {
				expectedCode: http.StatusBadRequest,
				route:        "/v1/signup",
				body:         []byte(`{"username": "awdawdwad", password: "", email:"nah@gmail.com"}`),
			},
			"Sign Up (bad email)": {
				expectedCode: http.StatusBadRequest,
				route:        "/v1/signup",
				body:         []byte(`{"username": "sdflkmmlksdf", password: "beep", email:"naha.com"}`),
			},
		},
	}

	for method, routeTest := range tests {
		for testName, test := range routeTest {
			test := test
			t.Run(fmt.Sprintf("%s %s", method, testName), func(t *testing.T) {
				mockAuth := &MockAuth{}
				mockStore := &MockStore{}
				req, err := http.NewRequest(method, test.route, bytes.NewReader(test.body))
				if err != nil {
					t.Fatal(err)
				}

				if test.setup != nil {
					test.setup(mockStore, mockAuth, req)
				}

				server := CreateTestServer(mockStore, mockAuth)

				rr := httptest.NewRecorder()

				server.ServeHTTP(rr, req)
				if rr.Code != test.expectedCode {
					t.Errorf("%s: %s, expected status %d, got: %d", method, test.route, test.expectedCode, rr.Code)
				}
			})
		}
	}
}
