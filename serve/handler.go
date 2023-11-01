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
	// Favours: X-Forwarded-For > X-Real-IP -> Remote Addr
	ip string
}

type respondFunc func(status int, jsonObj interface{}, message string)

type handlerFunc func(ctx context.Context, req *request, respond respondFunc)

func genHandler(handler handlerFunc) httprouter.Handle {
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
			func(status int, jsonObj interface{}, message string) {
				rw.WriteHeader(status)
				if jsonObj == nil {
					_, err := fmt.Fprintln(rw, message)
					if err != nil {
						rw.Header().Set("content-type", "text/plain")
						log.Printf("failed to write text response: %v", err)
					}
					return
				}

				err := json.NewEncoder(rw).Encode(jsonObj)
				if err != nil {
					rw.Header().Set("content-type", "application/json")
					log.Printf("failed to write JSON response: %v", err)
				}
			},
		)
	}
}
