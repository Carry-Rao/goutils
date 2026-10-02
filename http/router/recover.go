package router

import "net/http"

func Recovery() func(http.ResponseWriter, *http.Request, []string) bool {
	return func(w http.ResponseWriter, r *http.Request, _ []string) bool {
		defer func() {
			if rec := recover(); rec != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error (500)"))
			}
		}()
		return true
	}
}

func NewRecovery() *Router {
	r := New()
	r.Recover = true
	return r
}
