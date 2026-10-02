package router

import (
	"net/http"
)

func notFound(w http.ResponseWriter, _ *http.Request, _ []string) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found (404)"))
}

func methodNotAllowed(w http.ResponseWriter, _ *http.Request, _ []string) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Write([]byte("method not allowed (405)"))
}

var NotFound func(http.ResponseWriter, *http.Request, []string) = notFound

var MethodNotAllowed func(http.ResponseWriter, *http.Request, []string) = methodNotAllowed
