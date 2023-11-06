package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type request struct {
	params     httprouter.Params
	rawRequest *http.Request
	header     http.Header
	ip         string // Priority: X-Forwarded-For > X-Real-IP -> Remote Addr
}

type response struct {
	rw http.ResponseWriter
}

func (r *response) Respond(status int, jsonObj interface{}, message string) {
	if jsonObj == nil {
		r.rw.Header().Set("content-type", "text/plain")
		r.rw.WriteHeader(status)
		_, err := fmt.Fprintln(r.rw, message)
		if err != nil {
			log.Printf("failed to write text response: %v", err)
		}
		return
	}

	r.rw.Header().Set("content-type", "application/json")
	r.rw.WriteHeader(status)
	err := json.NewEncoder(r.rw).Encode(jsonObj)
	if err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

// Simplified HTTP handler function
type handlerFunc func(ctx context.Context, req *request, respond *response)

// Takes a custom handler function and returns an httprouter handler
func makeHandler(handler handlerFunc) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		// Find the request IP
		ip := req.Header.Get("X-FORWARDED-FOR")
		if len(ip) == 0 {
			ip = req.Header.Get("X-REAL-IP")
			if len(ip) == 0 {
				host, _, _ := net.SplitHostPort(req.RemoteAddr)
				ip = host
			}
		}

		log.Printf("Request %s: %s from %s agent :%s", req.Method, req.URL.Path, ip, req.UserAgent())

		handler(
			req.Context(),
			&request{
				header:     req.Header,
				params:     params,
				rawRequest: req,
				ip:         ip,
			},
			&response{
				rw: rw,
			},
		)
	}
}
