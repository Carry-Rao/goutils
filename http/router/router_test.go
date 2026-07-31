package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGET(t *testing.T) {
	r := New()
	var called bool
	r.GET("/test", func(w http.ResponseWriter, r *http.Request, params []string) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	r := New()
	r.GET("/test", func(w http.ResponseWriter, r *http.Request, params []string) {})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if w.Body.String() != "bad request (400)" {
		t.Errorf("expected 'bad request (400)', got '%s'", w.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	r := New()
	r.GET("/test", func(w http.ResponseWriter, r *http.Request, params []string) {})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if w.Body.String() != "not found (404)" {
		t.Errorf("expected 'not found (404)', got '%s'", w.Body.String())
	}
}

func TestIntVariable(t *testing.T) {
	r := New()
	r.GET("/users/:int", func(w http.ResponseWriter, r *http.Request, params []string) {
		if len(params) != 1 || params[0] != "123" {
			t.Errorf("expected [123], got %v", params)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestStringVariable(t *testing.T) {
	r := New()
	r.GET("/users/:string", func(w http.ResponseWriter, r *http.Request, params []string) {
		if len(params) != 1 || params[0] != "abc" {
			t.Errorf("expected [abc], got %v", params)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestIntPrecedence(t *testing.T) {
	r := New()
	var called string
	r.GET("/items/:int", func(w http.ResponseWriter, r *http.Request, params []string) {
		called = "int:" + params[0]
	})
	r.GET("/items/:string", func(w http.ResponseWriter, r *http.Request, params []string) {
		called = "string:" + params[0]
	})

	req := httptest.NewRequest(http.MethodGet, "/items/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if called != "int:42" {
		t.Errorf("expected int:42, got %s", called)
	}

	req = httptest.NewRequest(http.MethodGet, "/items/hello", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if called != "string:hello" {
		t.Errorf("expected string:hello, got %s", called)
	}
}

func TestMultipleVariables(t *testing.T) {
	r := New()
	r.GET("/users/:int/posts/:string", func(w http.ResponseWriter, r *http.Request, params []string) {
		if len(params) != 2 || params[0] != "10" || params[1] != "hello" {
			t.Errorf("expected [10 hello], got %v", params)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/users/10/posts/hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRootPath(t *testing.T) {
	r := New()
	var called bool
	r.GET("/", func(w http.ResponseWriter, r *http.Request, params []string) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Error("root handler was not called")
	}
}

func TestAll(t *testing.T) {
	r := New()
	var count int
	r.All("/test", func(w http.ResponseWriter, r *http.Request, params []string) {
		count++
	})

	for _, m := range defaultMethods {
		req := httptest.NewRequest(m, "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s expected 200, got %d", m, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS expected 204, got %d", w.Code)
	}
	if w.Header().Get("Allow") == "" {
		t.Error("OPTIONS should have Allow header")
	}
}

func TestMiddleware(t *testing.T) {
	r := New()
	var order []string

	r.Medium(func(w http.ResponseWriter, r *http.Request, ctx []string) bool {
		order = append(order, "first")
		return true
	})
	r.Medium(func(w http.ResponseWriter, r *http.Request, ctx []string) bool {
		order = append(order, "second")
		return true
	})

	r.GET("/test", func(w http.ResponseWriter, r *http.Request, params []string) {
		order = append(order, "handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expected := []string{"first", "second", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i := range order {
		if order[i] != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], order[i])
		}
	}
}

func TestMiddlewareStop(t *testing.T) {
	r := New()
	var called bool

	r.Medium(func(w http.ResponseWriter, r *http.Request, ctx []string) bool {
		return false
	})
	r.GET("/test", func(w http.ResponseWriter, r *http.Request, params []string) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if called {
		t.Error("handler should not be called when middleware returns false")
	}
}

func TestCORSPreflight(t *testing.T) {
	r := New()
	r.GET("/api/data", func(w http.ResponseWriter, r *http.Request, params []string) {})
	r.Option("/api/:string").Enable().Origin("https://example.com").Methods("GET,POST").Headers("Content-Type")

	req := httptest.NewRequest(http.MethodOptions, "/api/data", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("expected CORS origin, got '%s'", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") != "GET,POST" {
		t.Errorf("expected 'GET,POST', got '%s'", w.Header().Get("Access-Control-Allow-Methods"))
	}
}

func TestCORSActualRequest(t *testing.T) {
	r := New()
	r.GET("/api/data", func(w http.ResponseWriter, r *http.Request, params []string) {})
	r.Option("/api/:string").Enable().Origin("https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("expected CORS origin, got '%s'", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestUnsupportedMethod(t *testing.T) {
	r := New()
	r.GET("/test", func(w http.ResponseWriter, r *http.Request, params []string) {})

	req := httptest.NewRequest("INVALID", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLiteralPath(t *testing.T) {
	r := New()
	r.GET("/users/profile/settings", func(w http.ResponseWriter, r *http.Request, params []string) {
		if len(params) != 0 {
			t.Errorf("expected no params, got %v", params)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/users/profile/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDoubleSlash(t *testing.T) {
	r := New()
	var called bool
	r.GET("/test", func(w http.ResponseWriter, r *http.Request, params []string) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "//test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called for //test")
	}
}

func TestPathVariablesInMiddle(t *testing.T) {
	r := New()
	r.GET("/a/:int/b/:string/c", func(w http.ResponseWriter, r *http.Request, params []string) {
		if len(params) != 2 || params[0] != "1" || params[1] != "x" {
			t.Errorf("expected [1 x], got %v", params)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/a/1/b/x/c", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNegativeInt(t *testing.T) {
	r := New()
	r.GET("/value/:int", func(w http.ResponseWriter, r *http.Request, params []string) {
		if len(params) != 1 || params[0] != "-42" {
			t.Errorf("expected [-42], got %v", params)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/value/-42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestPOST(t *testing.T) {
	r := New()
	var called bool
	r.POST("/submit", func(w http.ResponseWriter, r *http.Request, params []string) {
		called = true
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Error("POST handler was not called")
	}
}

func TestHEAD(t *testing.T) {
	r := New()
	var called bool
	r.HEAD("/status", func(w http.ResponseWriter, r *http.Request, params []string) {
		called = true
	})

	req := httptest.NewRequest(http.MethodHead, "/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Error("HEAD handler was not called")
	}
}
