package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

var benchHeader = make(http.Header)

type noopWriter struct{}

func (*noopWriter) Header() http.Header         { return benchHeader }
func (*noopWriter) Write(p []byte) (int, error)   { return len(p), nil }
func (*noopWriter) WriteHeader(int)                {}

var benchHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

// ============ Static exact path ============

func BenchmarkServeMux_Static(b *testing.B) {
	mux := http.NewServeMux()
	mux.Handle("/hello", benchHandler)

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_Static(b *testing.B) {
	r := New()
	r.GET("/hello", func(w http.ResponseWriter, r *http.Request, _ []string) {})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// ============ Variable path ============

func BenchmarkServeMux_Var(b *testing.B) {
	mux := http.NewServeMux()
	mux.Handle("GET /hello/{id}", benchHandler)

	req := httptest.NewRequest(http.MethodGet, "/hello/world", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_StringVar(b *testing.B) {
	r := New()
	r.GET("/hello/:string", func(w http.ResponseWriter, r *http.Request, _ []string) {})

	req := httptest.NewRequest(http.MethodGet, "/hello/world", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_IntVar(b *testing.B) {
	r := New()
	r.GET("/hello/:int", func(w http.ResponseWriter, r *http.Request, _ []string) {})

	req := httptest.NewRequest(http.MethodGet, "/hello/12345", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// ============ Deep nested path ============

func BenchmarkServeMux_Deep(b *testing.B) {
	mux := http.NewServeMux()
	mux.Handle("GET /a/b/c/d/e/f/g/h", benchHandler)

	req := httptest.NewRequest(http.MethodGet, "/a/b/c/d/e/f/g/h", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_Deep(b *testing.B) {
	r := New()
	r.GET("/a/b/c/d/e/f/g/h", func(w http.ResponseWriter, r *http.Request, _ []string) {})

	req := httptest.NewRequest(http.MethodGet, "/a/b/c/d/e/f/g/h", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// ============ 404 (no match) ============

func BenchmarkServeMux_NotFound(b *testing.B) {
	mux := http.NewServeMux()
	mux.Handle("GET /hello", benchHandler)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_NotFound(b *testing.B) {
	r := New()
	r.GET("/hello", func(w http.ResponseWriter, r *http.Request, _ []string) {})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// ============ Many routes (99 static + 1 match) ============

func BenchmarkServeMux_ManyRoutes(b *testing.B) {
	mux := http.NewServeMux()
	for i := 0; i < 99; i++ {
		mux.Handle(http.MethodGet+"/r/"+string(rune('a'+i%26))+string(rune('0'+i%10)), benchHandler)
	}
	mux.Handle("GET /target", benchHandler)

	req := httptest.NewRequest(http.MethodGet, "/target", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_ManyRoutes(b *testing.B) {
	r := New()
	for i := 0; i < 99; i++ {
		r.GET("/r/"+string(rune('a'+i%26))+string(rune('0'+i%10)), func(w http.ResponseWriter, r *http.Request, _ []string) {})
	}
	r.GET("/target", func(w http.ResponseWriter, r *http.Request, _ []string) {})

	req := httptest.NewRequest(http.MethodGet, "/target", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// ============ Middleware (0 vs 2) ============

func BenchmarkRouter_NoMiddleware(b *testing.B) {
	r := New()
	r.GET("/hello", func(w http.ResponseWriter, r *http.Request, _ []string) {})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_TwoMiddleware(b *testing.B) {
	r := New()
	r.Medium(func(_ http.ResponseWriter, _ *http.Request, _ []string) bool { return true })
	r.Medium(func(_ http.ResponseWriter, _ *http.Request, _ []string) bool { return true })
	r.GET("/hello", func(w http.ResponseWriter, r *http.Request, _ []string) {})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// ============ CORS ============

func BenchmarkRouter_CORS(b *testing.B) {
	r := New()
	r.GET("/api/data", func(w http.ResponseWriter, r *http.Request, _ []string) {})
	r.Option("/api/:string").Enable().Origin("*")

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_CORSPreflight(b *testing.B) {
	r := New()
	r.GET("/api/data", func(w http.ResponseWriter, r *http.Request, _ []string) {})
	r.Option("/api/:string").Enable().Origin("*")

	req := httptest.NewRequest(http.MethodOptions, "/api/data", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// ============ Mixed variable + literal routes ============

func BenchmarkRouter_MixedVar(b *testing.B) {
	r := New()
	r.GET("/users/:int/posts/:string/comments/:int", func(w http.ResponseWriter, r *http.Request, _ []string) {})

	req := httptest.NewRequest(http.MethodGet, "/users/42/posts/hello/comments/7", nil)
	w := &noopWriter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// prevent compiler from optimising away the benchmark result
var _ = io.Discard
