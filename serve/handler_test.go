package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
)

func TestGenHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("SOME_METHOD", "/", nil)
	params := []httprouter.Param{{
		Key:   "1",
		Value: "2",
	}}

	type testJSON struct {
		Name string `json:"name"`
	}

	genHandler(func(ctx context.Context, req *request, res respondFunc) {
		if req.params.ByName("1") != "2" {
			t.Fatalf("Unexpected route parameter %s", req.params.ByName("1"))
		}

		res(http.StatusTeapot, testJSON{
			Name: "Jason",
		}, "")

	})(recorder, req, params)

	if recorder.Code != http.StatusTeapot {
		t.Fatal("Expected status teapot")
	}

	resJSON := testJSON{}
	err := json.NewDecoder(recorder.Body).Decode(&resJSON)
	if err != nil {
		t.Fatal(err)
	}

	if resJSON.Name != "Jason" {
		t.Fatalf("Unexpected response JSON field: %s", resJSON.Name)
	}
}

func TestHandlerIP(t *testing.T) {
	var tests = map[string]string{
		"X-FORWARDED-FOR": "44.5.512334.5",
		"X-REAL-IP":       "xxx-xx-xxx",
	}

	for header, ip := range tests {
		forwardedReq := httptest.NewRequest("GET", "/", nil)
		forwardedReq.Header.Set(header, ip)

		recorder := httptest.NewRecorder()

		genHandler(func(ctx context.Context, req *request, respond respondFunc) {
			if req.ip != ip {
				t.Fatalf("Expected request IP %s == %s", req.ip, ip)
			}
		})(recorder, forwardedReq, nil)
	}
}
