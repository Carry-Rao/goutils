package router

import (
	"net/http"
)

func notFound(w http.ResponseWriter, _ *http.Request, _ []string) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found (404)"))
}

func badRequest(w http.ResponseWriter, _ *http.Request, _ []string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("bad request (400)"))
}

var NotFound func(http.ResponseWriter, *http.Request, []string) = notFound

var BadRequest func(http.ResponseWriter, *http.Request, []string) = badRequest
