package serve

import (
	"fmt"
	"net/http"
)

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
