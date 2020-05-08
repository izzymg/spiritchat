package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type request struct {
	params     httprouter.Params
	rawRequest *http.Request
	header     http.Header
}

type respondFunc func(status int, jsonObj interface{}, message string)

type handlerFunc func(ctx context.Context, req *request, respond respondFunc)

func genHandler(handler handlerFunc) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		handler(
			req.Context(),
			&request{
				header:     req.Header,
				params:     params,
				rawRequest: req,
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
