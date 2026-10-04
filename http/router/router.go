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
	Recover     bool

	// prefix is this router's own path prefix, set by Sub.
	prefix string
}

func New() *Router {
	r := &Router{
		roots:       make(map[string]*pathTree),
		corsConfigs: nil,
	}
	// Every method gets a tree up front, so registration always has somewhere to
	// plant its 405 stub. Without this a method that was never registered would
	// have no tree to answer from and would fall back to probing the others.
	for _, method := range defaultMethods {
		r.roots[method] = r.newRoot()
	}
	return r
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
		root = r.newRoot()
		r.roots[method] = root
	}
	segments := splitPath(pattern)
	root.addRoute(segments, handler)

	// Plant a 405 stub for every other method at the same path, so a request
	// finds its answer in its own tree instead of probing the others.
	r.plantStubs(method, segments)
}

// plantStubs gives each other method a 405 responder at this path, but only
// where it has no real handler. Registering a route later overwrites the stub,
// because a real handler always outranks one.
func (r *Router) plantStubs(registered string, segments []string) {
	for method := range r.roots {
		if method == registered {
			continue
		}
		r.roots[method].plant(segments)
	}
}

// plant installs a stub at the given segments unless a real route already
// occupies that node.
//
// Intermediate nodes are created along the way. That is safe because they carry
// no handler: match keeps descending through a node whose Function is nil and
// only stops where the stub is, so POST /users still 404s when only GET
// /users exists.
func (p *pathTree) plant(segments []string) {
	if len(segments) == 0 {
		return
	}
	seg := segments[0]

	var child *pathTree
	if varType, ok := parseVarType(seg); ok {
		if child = p.SubVariablesPaths[varType]; child == nil {
			child = p.newNode(seg)
			p.SubVariablesPaths[varType] = child
		}
	} else {
		if child = p.SubPaths[seg]; child == nil {
			child = p.newNode(seg)
			p.SubPaths[seg] = child
		}
	}

	if len(segments) == 1 {
		if child.Function == nil {
			child.Function = notAllowedHandler
			child.MethodNotAllowed = true
		}
		return
	}
	child.plant(segments[1:])
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
			root = r.newRoot()
			r.roots[method] = root
		}
		root.Middleware = append(root.Middleware, mw)
	}
}

func (r *Router) Sub(prefix string) *Router {
	sub := &Router{
		roots:  make(map[string]*pathTree),
		prefix: joinPath(r.prefix, prefix),
	}
	for _, method := range defaultMethods {
		root := r.roots[method]
		if root == nil {
			root = r.newRoot()
			r.roots[method] = root
		}
		sub.roots[method] = root.getOrCreatePrefix(splitPath(prefix))
	}
	return sub
}

// Static serves the files under root for pattern and every path below it. The
// pattern is stripped before the lookup, so "/static/css/app.css" resolves to
// "./public/css/app.css".
func (r *Router) Static(pattern, root string) {
	full := joinPath(r.prefix, pattern)
	handler := http.StripPrefix(full, http.FileServer(http.Dir(root)))

	tree := r.roots[http.MethodGet]
	if tree == nil {
		tree = r.newRoot()
		r.roots[http.MethodGet] = tree
	}
	// Relative to this router: on a Sub the tree already sits at the prefix.
	tree.addSubtree(splitPath(pattern), func(w http.ResponseWriter, req *http.Request, _ []string) {
		handler.ServeHTTP(w, req)
	})
}

func (r *Router) newRoot() *pathTree {
	return &pathTree{
		SubPaths:          make(map[string]*pathTree),
		SubVariablesPaths: make(map[Type]*pathTree),
	}
}

// joinPath combines a prefix and a pattern; an empty result means the root.
func joinPath(prefix, pattern string) string {
	parts := make([]string, 0, 2)
	for _, s := range []string{prefix, pattern} {
		if s = strings.Trim(s, "/"); s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "/" + strings.Join(parts, "/")
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

		if varType, ok := parseVarType(pseg); ok {
			if !matchType(qseg, varType) {
				return false
			}
		} else if pseg != qseg {
			return false
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

func (r *Router) allowedMethods(path string) []string {
	methods := make([]string, 0, len(r.roots))
	for method, root := range r.roots {
		if root.pathExists(path) {
			methods = append(methods, method)
		}
	}
	return methods
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.Recover {
		defer func() {
			if rec := recover(); rec != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error (500)"))
			}
		}()
	}

	if cfg := r.matchCORS(req.URL.Path); cfg != nil {
		cfg.applyHeaders(w)
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	// A nil root means the method was never registered. Otherwise the tree
	// answers directly: either a real route, or a 405 stub planted at
	// registration time. Only the latter needs the other trees, to build Allow.
	if root := r.roots[req.Method]; root != nil {
		if m, ok := root.match(req.URL.Path); ok {
			if m.notAllowed {
				r.writeNotAllowed(w, req, req.URL.Path)
				return
			}
			m.exec(w, req)
			return
		}
	}

	// The method tree exists but the path is not in it, which happens when a
	// method was registered after this path and so no stub was planted.
	if allowed := r.allowedMethods(req.URL.Path); len(allowed) > 0 {
		r.writeNotAllowed(w, req, req.URL.Path)
		return
	}
	NotFound(w, req, nil)
}

// writeNotAllowed replies 405 with the methods that do serve this path.
func (r *Router) writeNotAllowed(w http.ResponseWriter, req *http.Request, path string) {
	w.Header().Set("Allow", strings.Join(r.allowedMethods(path), ", "))
	MethodNotAllowed(w, req, nil)
}

func (r *Router) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, r)
}

func (r *Router) ListenAndServeTLS(addr, certFile, keyFile string) error {
	return http.ListenAndServeTLS(addr, certFile, keyFile, r)
}
