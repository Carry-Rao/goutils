# HTTP Router Module

This module provides a high-performance HTTP router based on a **prefix tree (radix tree)**, supporting static paths, typed path variables, middleware, and CORS configuration.

## Core Features

- **Typed Path Variables**: Supports `:int` (integer) and `:string` (string) path variables, with `int` matching first
- **Method Routing**: Built-in quick registration for GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
- **Multi-method Registration**: The `All` method registers the same path for all HTTP methods at once
- **Middleware**: Chainable middleware; returning `false` terminates the request
- **CORS Support**: Per-pattern CORS configuration with automatic preflight (OPTIONS) responses
- **Custom Error Handling**: Replaceable default 404 / 400 response handlers

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/Carry-Rao/goutils/http/router"
)

func main() {
    r := router.New()

    // Static path
    r.GET("/", func(w http.ResponseWriter, r *http.Request, params []string) {
        w.Write([]byte("hello"))
    })

    // Typed path variable; params holds matched values in declaration order
    r.GET("/users/:int", func(w http.ResponseWriter, r *http.Request, params []string) {
        // params[0] is the integer user ID
        w.Write([]byte("user " + params[0]))
    })

    r.GET("/files/:string", func(w http.ResponseWriter, r *http.Request, params []string) {
        w.Write([]byte("file " + params[0]))
    })

    r.ListenAndServe(":8080")
}
```

## Path Variables

Path variables start with `:`. Two types are currently supported:

| Variable | Matching rule | Priority |
|----------|---------------|----------|
| `:int` | Integer (including negatives), e.g. `-42`, `123` | Higher than `:string` |
| `:string` | Any non-empty string | Lower than `:int` |

```go
r.GET("/items/:int", func(w http.ResponseWriter, r *http.Request, params []string) {
    // Request /items/42 → params = ["42"]
})
```

Multiple variables fill `params` in declaration order:

```go
r.GET("/users/:int/posts/:string", func(w http.ResponseWriter, r *http.Request, params []string) {
    // Request /users/10/posts/hello → params = ["10", "hello"]
})
```

## Method Registration

```go
r.GET("/path", handler)
r.POST("/path", handler)
r.PUT("/path", handler)
r.DELETE("/path", handler)
r.PATCH("/path", handler)
r.HEAD("/path", handler)
r.OPTIONS("/path", handler)
```

Register the same path for all HTTP methods:

```go
r.All("/path", handler)
// OPTIONS automatically returns Allow: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
```

## Middleware

Register middleware with `Medium`; they run in registration order. Returning `false` terminates the request, skipping remaining middleware and the handler:

```go
r.Medium(func(w http.ResponseWriter, r *http.Request, params []string) bool {
    // Auth, logging, etc.
    return true // return false to terminate the request
})

r.Medium(func(w http.ResponseWriter, r *http.Request, params []string) bool {
    return true
})

r.GET("/secret", func(w http.ResponseWriter, r *http.Request, params []string) {
    // Only runs after middleware passes
})
```

## CORS Configuration

Use `Option` to create a CORS config, chain setters, and finally call `Enable` to activate it:

```go
r.Option("/api/:string").
    Enable().
    Origin("https://example.com", "https://app.example.com").
    Methods("GET,POST").
    Headers("Content-Type,Authorization").
    Credentials(true).
    MaxAge(3600)
```

CORS config methods:

| Method | Description |
|--------|-------------|
| `Enable()` | Enable this CORS config |
| `Origin(origins...)` | Set allowed origins |
| `Methods(methods...)` | Set allowed request methods |
| `Headers(headers...)` | Set allowed request headers |
| `Credentials(allow)` | Whether to allow credentials (Access-Control-Allow-Credentials) |
| `MaxAge(seconds)` | Preflight cache duration (Access-Control-Max-Age) |

When a CORS config matches, `OPTIONS` preflight requests return 204 immediately with the CORS response headers set.

> CORS path patterns support `:int` / `:string` type variables too.

## Custom Error Handling

The default 404 and 400 responses can be customized by replacing the package-level variables:

```go
router.NotFound = func(w http.ResponseWriter, r *http.Request, params []string) {
    http.Error(w, "resource not found", http.StatusNotFound)
}

router.BadRequest = func(w http.ResponseWriter, r *http.Request, params []string) {
    http.Error(w, "method not allowed", http.StatusBadRequest)
}
```

- **404**: the path matched no route
- **400**: the request method is not registered (e.g. only GET was registered but a POST arrived)

## Serving

```go
r.ListenAndServe(":8080")                // HTTP
r.ListenAndServeTLS(":8443", "cert.pem", "key.pem") // HTTPS
```

`Router` implements the `http.Handler` interface, so it can be passed directly to the standard library server.
