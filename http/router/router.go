package router

import (
	"net/http"
	"strings"
)

var defaultMethods = []string{
	http.MethodGet, http.MethodPost, http.MethodPut,
	http.MethodDelete, http.MethodPatch, http.MethodHead,
}

type Router struct {
	roots       map[string]*pathTree
	corsConfigs []*CORSConfig
}

func New() *Router {
	return &Router{
		roots:       make(map[string]*pathTree),
		corsConfigs: nil,
	}
}

func splitPath(path string) []string {
	paths := strings.Split(path, "/")
	if strings.HasPrefix(path, "/") {
		paths = paths[1:]
	}
	return cleanPaths(paths)
}

func (r *Router) Handle(method, pattern string, handler func(http.ResponseWriter, *http.Request, []string)) {
	root := r.roots[method]
	if root == nil {
		root = &pathTree{
			SubPaths:          make(map[string]*pathTree),
			SubVariablesPaths: make(map[Type]*pathTree),
		}
		r.roots[method] = root
	}
	root.addRoute(splitPath(pattern), handler)
}

func (r *Router) GET(pattern string, handler func(http.ResponseWriter, *http.Request, []string)) {
	r.Handle(http.MethodGet, pattern, handler)
}

func (r *Router) POST(pattern string, handler func(http.ResponseWriter, *http.Request, []string)) {
	r.Handle(http.MethodPost, pattern, handler)
}

func (r *Router) PUT(pattern string, handler func(http.ResponseWriter, *http.Request, []string)) {
	r.Handle(http.MethodPut, pattern, handler)
}

func (r *Router) DELETE(pattern string, handler func(http.ResponseWriter, *http.Request, []string)) {
	r.Handle(http.MethodDelete, pattern, handler)
}

func (r *Router) PATCH(pattern string, handler func(http.ResponseWriter, *http.Request, []string)) {
	r.Handle(http.MethodPatch, pattern, handler)
}

func (r *Router) HEAD(pattern string, handler func(http.ResponseWriter, *http.Request, []string)) {
	r.Handle(http.MethodHead, pattern, handler)
}

func (r *Router) OPTIONS(pattern string, handler func(http.ResponseWriter, *http.Request, []string)) {
	r.Handle(http.MethodOptions, pattern, handler)
}

func (r *Router) All(pattern string, handler func(http.ResponseWriter, *http.Request, []string)) {
	for _, m := range defaultMethods {
		r.Handle(m, pattern, handler)
	}
	r.Handle(http.MethodOptions, pattern, func(w http.ResponseWriter, _ *http.Request, _ []string) {
		w.Header().Set("Allow", "GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
	})
}

func (r *Router) Medium(mw func(http.ResponseWriter, *http.Request, []string) bool) {
	for _, method := range defaultMethods {
		root := r.roots[method]
		if root == nil {
			root = &pathTree{
				SubPaths:          make(map[string]*pathTree),
				SubVariablesPaths: make(map[Type]*pathTree),
			}
			r.roots[method] = root
		}
		root.Middleware = append(root.Middleware, mw)
	}
}

func (r *Router) Sub(prefix string) *Router {
	sub := &Router{
		roots: make(map[string]*pathTree),
	}
	for _, method := range defaultMethods {
		root := r.roots[method]
		if root == nil {
			root = &pathTree{
				SubPaths:          make(map[string]*pathTree),
				SubVariablesPaths: make(map[Type]*pathTree),
			}
			r.roots[method] = root
		}
		sub.roots[method] = root.getOrCreatePrefix(splitPath(prefix))
	}
	return sub
}

func (r *Router) Static(pattern, root string) {
	fs := http.FileServer(http.Dir(root))
	r.GET(pattern, func(w http.ResponseWriter, req *http.Request, _ []string) {
		fs.ServeHTTP(w, req)
	})
}

func (r *Router) Option(pattern string) *CORSConfig {
	cfg := &CORSConfig{pattern: pattern}
	r.corsConfigs = append(r.corsConfigs, cfg)
	return cfg
}

func matchCORSPattern(pattern, path string) bool {
	pi, pj := 0, len(pattern)
	qi, qj := 0, len(path)
	for {
		for pi < pj && pattern[pi] == '/' {
			pi++
		}
		for qi < qj && path[qi] == '/' {
			qi++
		}
		if pi >= pj && qi >= qj {
			return true
		}
		if pi >= pj || qi >= qj {
			return false
		}
		psi := pi
		for pi < pj && pattern[pi] != '/' {
			pi++
		}
		pseg := pattern[psi:pi]

		qsi := qi
		for qi < qj && path[qi] != '/' {
			qi++
		}
		qseg := path[qsi:qi]

		switch pseg {
		case ":int":
			if !isInt(qseg) {
				return false
			}
		case ":string":
		default:
			if pseg != qseg {
				return false
			}
		}
	}
}

func (r *Router) matchCORS(path string) *CORSConfig {
	for _, cfg := range r.corsConfigs {
		if cfg.enabled && matchCORSPattern(cfg.pattern, path) {
			return cfg
		}
	}
	return nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if cfg := r.matchCORS(req.URL.Path); cfg != nil {
		cfg.applyHeaders(w)
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	root := r.roots[req.Method]
	if root == nil {
		BadRequest(w, req, nil)
		return
	}

	handler, vars, ok := root.visitPath(req.URL.Path, w, req)
	if handler == nil || !ok {
		return
	}
	handler(w, req, vars)
}

func (r *Router) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, r)
}

func (r *Router) ListenAndServeTLS(addr, certFile, keyFile string) error {
	return http.ListenAndServeTLS(addr, certFile, keyFile, r)
}
