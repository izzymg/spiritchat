package serve

import (
	"fmt"
	"net/http"
)

type ok struct {
	Message string `json:"message"`
}

func internalError(message string) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(rw, message)
	}
}

func notFound(message string) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(rw, message)
	}
}

func badRequest(message string) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(rw, message)
	}
}
