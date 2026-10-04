# HTTP Router Module

> 中文文档：[database_zh.md](database_zh.md) · [http_zh.md](http_zh.md) · [log_zh.md](log_zh.md)

A prefix-tree router. Compared with `http.ServeMux` it adds typed path variables, middleware, per-pattern CORS, and a correct 404/405 distinction.

## Quick start

```go
r := router.New()

r.GET("/users/:int", func(w http.ResponseWriter, req *http.Request, params []string) {
	w.Write([]byte("User " + params[0]))
})

if err := r.ListenAndServe(":8080"); err != nil {
	panic(err)
}
```

## Handler signature

```go
func(http.ResponseWriter, *http.Request, []string)
```

The third argument holds path variable values in declaration order. For `/users/:int/posts/:string` matching `/users/1/posts/hello`, `params` is `["1", "hello"]`.

The same `params` slice is passed to every middleware along the matched node chain, so middleware can read variables too.

## Registration

```go
r.Handle(method, pattern, handler) // arbitrary method
r.GET(pattern, handler)
r.POST(pattern, handler)
r.PUT(pattern, handler)
r.DELETE(pattern, handler)
r.PATCH(pattern, handler)
r.HEAD(pattern, handler)
r.OPTIONS(pattern, handler)
r.All(pattern, handler) // every method
```

`All` additionally registers an `OPTIONS` handler that returns an `Allow` header and `204`.

## Path variables

| Syntax | Matches | Examples |
|--------|---------|----------|
| `:int` | Optional minus sign plus digits | `123`, `-42` |
| `:float` | Optional minus sign plus digits, and **must contain a decimal point** | `3.14`, `-0.5` |
| `:alpha` | Letters only | `abc`, `AbC` |
| `:alphanum` | Letters and digits | `abc123`, `ABC` |
| `:uuid` | Standard 8-4-4-4-12 hex | `550e8400-e29b-41d4-a716-446655440000` |
| `:string` | Any non-empty segment (**never spans `/`**) | `abc`, `a-b_c` |

Points that easily go wrong:

- `:float` requires a decimal point, so `3` does not match `:float` (it matches `:int`)
- `:uuid` accepts either case; non-hex characters or misplaced dashes do not match
- `:string` is a **single-segment** catch-all and does not span `/`. `/path/one/two` does not match `/path/:string`

To match a trailing multi-segment path this library provides **no** wildcard
variable. Mount files with `Static`; otherwise scope a prefix with `Sub` and
read the remaining path inside that handler.

### Match priority

When several variable types coexist at one node, they are tried in a fixed order, **most specific first**:

```
literal  >  :int  >  :float  >  :uuid  >  :alpha  >  :alphanum  >  :string
```

So with both `/items/:int` and `/items/:string` registered, `/items/42` hits `:int` and `/items/hello` hits `:string`.

Literals win outright: `/users/profile` is matched before `/users/:string`.

## Middleware

A middleware is `func(http.ResponseWriter, *http.Request, []string) bool`. Returning `false` aborts the request and the handler is not invoked.

```go
r.Medium(func(w http.ResponseWriter, req *http.Request, params []string) bool {
	if req.Header.Get("Authorization") == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return false // abort
	}
	return true // continue
})
```

`Medium` applies to every route on that `Router`. To scope middleware to a subset, carve out a sub-router:

```go
r.Medium(globalMiddleware) // global

api := r.Sub("/api")
api.Medium(authMiddleware) // /api/* only

admin := r.Sub("/admin")
admin.Medium(adminMiddleware) // /admin/* only
```

Execution runs root to leaf: global middleware, then sub-router middleware, then the handler.

Because middleware runs only after a successful match, requests that match no route (404/405) never trigger middleware.

## CORS

Configured per pattern with chained calls:

```go
r.Option("/api/:string").
	Enable().
	Origin("https://example.com", "https://app.example.com").
	Methods("GET", "POST", "PUT").
	Headers("Content-Type", "Authorization").
	Credentials(true).
	MaxAge(3600)
```

- The pattern supports every path variable type
- Non-`OPTIONS` requests also receive the headers
- An `OPTIONS` preflight returns `204` and ends immediately
- A config without `Enable()` has no effect
- Without `Origin` set, no `Access-Control-Allow-Origin` is emitted

## 404 and 405

Routes are held in one tree per method. Registering a route also plants a **405
stub** at that same path in every *other* method's tree, so the common case never
inspects the other trees:

1. `GET /users` registers a handler on the GET tree, and a stub answering 405 on
   the POST, PUT, DELETE, PATCH and HEAD trees.
2. A request matches against **its own method's tree only**.
   - a real handler → run middleware then the handler
   - a 405 stub → `405 Method Not Allowed`, with `Allow` listing the methods that
     do serve the path
   - nothing → `404 Not Found`

```go
r.GET("/users", h) // GET only

// POST /users  =>  405, Allow: GET
// POST /users/1 =>  404   (no stub was planted for a path nobody registered)
// GET  /nope    =>  404
```

Only a stub needs the other trees, and only to build the `Allow` header — 405 is a
rare path, so the normal request pays nothing.

A stub is planted only where a real route is absent, and **a real handler always
beats a stub** during matching. Without that rule the stub for `/user/:int` would
shadow a real `POST /user/:string` handler, because `Int` outranks `String` in
variable priority. Registering a method later simply overwrites the stub.

This fixes cases where the standard library's `ServeMux` returns 405 without `Allow`, and cases where it wrongly reports 404.

## Customising error responses

`NotFound` and `MethodNotAllowed` are replaceable package-level variables:

```go
router.NotFound = func(w http.ResponseWriter, r *http.Request, _ []string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error":"not found"}`))
}

router.MethodNotAllowed = func(w http.ResponseWriter, r *http.Request, _ []string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Write([]byte(`{"error":"method not allowed"}`))
}
```

The defaults are the plain text `not found (404)` and `method not allowed (405)`.

These are package-level, so replacing one affects every `Router` in the process. For per-router behaviour, assign them during setup.

## Panic recovery

```go
r := router.New()
r.Recover = true
```

or via the constructor:

```go
r := router.NewRecovery()
```

With this enabled, a panic in a handler is caught and answered with `500 internal server error (500)` instead of taking down the process. The body is fixed; for something else, replace the package-level variables or handle it inside your handler.

> `Recover` only affects request handling in `ServeHTTP`; it does not cover errors from `ListenAndServe` itself.

## Static files

```go
r.Static("/static", "./public")   // /static/css/app.css → ./public/css/app.css
r.Static("/", "./public")         // mounted at the root

api := r.Sub("/api")
api.Static("/assets", "./public") // /api/assets/x → ./public/x
```

The mount point is **stripped** before the lookup, so `/static/css/app.css`
resolves to `./public/css/app.css`, not `./public/static/css/app.css`. A `Sub`
prefix is folded in automatically.

Every path below the mount point reaches the handler, but an explicit sibling
route wins:

```go
r.Static("/static", "./public")
r.GET("/static/config", handler) // matches here, not the file server
```

Only `GET` is registered, so `HEAD` will not match. Middleware still runs,
because the mount point stays on the routing tree.

Backed by `http.FileServer`, which provides `Range`, `ETag`, directory indexes
and path-traversal protection. Note it canonicalises `/index.html` to `./` and
serves `index.html` directly for a directory that has one — both standard
`net/http` behaviour.

## Serving

```go
r.ListenAndServe(":8080")
r.ListenAndServeTLS(":8443", "cert.pem", "key.pem")
```

Both mirror the `net/http` equivalents. `Router` implements `http.Handler`, so you can hand it to an `http.Server` to control timeouts and connection limits:

```go
srv := &http.Server{
	Addr:              ":8080",
	Handler:           r,
	ReadHeaderTimeout: 5 * time.Second,
}
log.Fatal(srv.ListenAndServe())
```

## Performance

Matching is O(path depth): one map lookup per level plus at most one type check.

- Literals are tried first; type checks only run when no literal matched
- Variable types are probed in a fixed order, and a check runs only when the corresponding child node exists
- Middleware runs only after a successful match, so unmatched requests pay nothing
- Matching and execution are separated: `match` is side-effect free, `exec` runs middleware and the handler

A `405` is answered from a stub in the current method's own tree, so normally routed requests never look at another tree. The other trees are consulted only to build the `Allow` header on an actual `405`.

`bench_test.go` provides comparisons against `http.ServeMux`. Run it with `go test -bench=. -benchmem ./http/router/`.

## Breaking changes

- `BadRequest` was renamed to `MethodNotAllowed`; the status code changed from `400` to `405`
- 405 is answered from a stub planted at registration time instead of by probing every method tree; a real handler always outranks a stub
- Middleware no longer runs for unmatched requests (404/405)
- Middleware receives the complete variable list rather than the prefix matched so far
- `:any` was removed; `:string` is the only catch-all and does not span `/`