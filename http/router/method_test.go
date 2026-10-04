package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// stubsAnswerDirectly: once a path exists for one method, another method must
// get 405 without the router having to probe the other trees.
func TestStubAnswersDirectly(t *testing.T) {
	r := New()
	r.GET("/users", func(w http.ResponseWriter, _ *http.Request, _ []string) {
		w.WriteHeader(http.StatusOK)
	})

	if r.roots[http.MethodPost] == nil {
		t.Fatal("expected a POST tree to plant the stub in")
	}
	if node := lookupNode(r.roots[http.MethodPost], "users"); !node.MethodNotAllowed {
		t.Error("POST /users should hold a 405 stub")
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/users", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q, want GET", got)
	}
}

// A stub must never shadow a real route. Int outranks String in varPriority, so
// a stub on :int would otherwise hide a real handler on :string.
func TestStubDoesNotShadowRealRoute(t *testing.T) {
	r := New()
	r.GET("/user/:int", func(w http.ResponseWriter, _ *http.Request, v []string) {
		w.Write([]byte("get:" + v[0]))
	})
	r.POST("/user/:string", func(w http.ResponseWriter, _ *http.Request, v []string) {
		w.Write([]byte("post:" + v[0]))
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/user/123", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: the real :string handler must win", rec.Code)
	}
	if got := rec.Body.String(); got != "post:123" {
		t.Errorf("body = %q, want post:123", got)
	}
}

// The same in the other direction: a real :int handler still outranks a stub.
func TestRealRouteBeatsStub(t *testing.T) {
	r := New()
	r.POST("/user/:string", func(w http.ResponseWriter, _ *http.Request, v []string) {
		w.Write([]byte("post:" + v[0]))
	})
	r.GET("/user/:int", func(w http.ResponseWriter, _ *http.Request, v []string) {
		w.Write([]byte("get:" + v[0]))
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/user/123", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "post:123" {
		t.Errorf("got %d %q, want 200 post:123", rec.Code, rec.Body.String())
	}
}

// Registering a method later must replace the stub at the same path.
func TestLaterRegistrationReplacesStub(t *testing.T) {
	r := New()
	r.GET("/users", func(w http.ResponseWriter, _ *http.Request, _ []string) {
		w.Write([]byte("get"))
	})
	r.POST("/users", func(w http.ResponseWriter, _ *http.Request, _ []string) {
		w.Write([]byte("post"))
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/users", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "post" {
		t.Fatalf("got %d %q, want 200 post", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/users", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("delete = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); !strings.Contains(got, "GET") || !strings.Contains(got, "POST") {
		t.Errorf("Allow = %q, want GET and POST", got)
	}
}

// A stub must not turn an unregistered path into a 405: /users/1 was never
// registered for POST, so it stays a 404.
func TestStubDoesNotInventPaths(t *testing.T) {
	r := New()
	r.GET("/users", func(w http.ResponseWriter, _ *http.Request, _ []string) {})

	for _, path := range []string{"/users/1", "/users/1/edit", "/other"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("POST %s = %d, want 404", path, rec.Code)
		}
	}
}

// Every method listed in Allow must actually serve the path.
func TestAllowListsOnlyRealRoutes(t *testing.T) {
	r := New()
	r.GET("/users", func(w http.ResponseWriter, _ *http.Request, _ []string) {})
	r.PUT("/users", func(w http.ResponseWriter, _ *http.Request, _ []string) {})
	r.DELETE("/users", func(w http.ResponseWriter, _ *http.Request, _ []string) {})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, "/users", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}

	allow := rec.Header().Get("Allow")
	for _, want := range []string{"GET", "PUT", "DELETE"} {
		if !strings.Contains(allow, want) {
			t.Errorf("Allow = %q, missing %s", allow, want)
		}
	}
	// PATCH itself and the untouched methods must not be advertised.
	for _, unwanted := range []string{"PATCH", "POST"} {
		if strings.Contains(allow, unwanted) {
			t.Errorf("Allow = %q, should not list %s", allow, unwanted)
		}
	}

	for _, m := range []string{"GET", "PUT", "DELETE"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(m, "/users", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s = %d, want 200 (Allow promised it)", m, rec.Code)
		}
	}
}

// A method never registered still gets 405 via the fallback, since there is no
// tree to plant a stub in.
func TestUnregisteredMethodStillGets405(t *testing.T) {
	r := New()
	r.GET("/users", func(w http.ResponseWriter, _ *http.Request, _ []string) {})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/users", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "GET" {
		t.Errorf("Allow = %q, want GET", got)
	}
}

// Variable paths must 405 on the right node.
func TestStubWithVariablePaths(t *testing.T) {
	r := New()
	r.GET("/user/:int/post", func(w http.ResponseWriter, _ *http.Request, _ []string) {})

	// The stub belongs on the final segment; :int and its parent stay handlerless
	// so that POST /user/7 (without /post) still 404s.
	user := lookupNode(r.roots[http.MethodPost], "user")
	if user == nil {
		t.Fatal("expected a POST /user node")
	}
	leaf := user.SubVariablesPaths[Int]
	if leaf == nil {
		t.Fatal("expected a POST /user/:int node")
	}
	// :int is not the route, so it must stay handlerless.
	if leaf.MethodNotAllowed || leaf.Function != nil {
		t.Error("the :int node must stay handlerless, since /post is the route")
	}
	if leaf.SubPaths["post"] == nil || !leaf.SubPaths["post"].MethodNotAllowed {
		t.Error("expected the 405 stub on the /post leaf")
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/user/7/post", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}

	// A non-numeric segment does not match :int, so it is a 404.
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/user/abc/post", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("non-matching segment = %d, want 404", rec.Code)
	}
}

// Static keeps serving GET, and POST is 405 rather than 404 because the path
// does exist under another method. That matches the behaviour from before the
// stub change.
func TestStubDoesNotDisturbStatic(t *testing.T) {
	dir := t.TempDir()
	if err := writeFile(dir+"/a.txt", "hello"); err != nil {
		t.Fatal(err)
	}
	r := New()
	r.Static("/files", dir)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/a.txt", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "hello" {
		t.Errorf("GET = %d %q, want 200 hello", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/files/a.txt", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q, want GET", got)
	}
}

// Sub routers share the roots, so a stub planted on the parent must be visible
// through the prefix.
func TestStubOnSubRouter(t *testing.T) {
	r := New()
	sub := r.Sub("/api")
	sub.GET("/users", func(w http.ResponseWriter, _ *http.Request, _ []string) {})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/users", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q, want GET", got)
	}
}

// All registers every method, so nothing may 405.
func TestAllBlocks405(t *testing.T) {
	r := New()
	r.All("/x", func(w http.ResponseWriter, _ *http.Request, _ []string) {})

	for _, m := range defaultMethods {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(m, "/x", nil))
		if rec.Code == http.StatusMethodNotAllowed {
			t.Errorf("%s = 405, All must register it", m)
		}
	}
}

// The root path needs a stub too.
func TestStubAtRoot(t *testing.T) {
	r := New()
	r.GET("/", func(w http.ResponseWriter, _ *http.Request, _ []string) {})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q, want GET", got)
	}
}

// lookupNode walks literal segments only, for asserting on tree shape. The
// second result of resolve reports whether the step was a variable, which is not
// what this helper cares about.
func lookupNode(root *pathTree, segments ...string) *pathTree {
	node := root
	if node == nil {
		return nil
	}
	for _, seg := range segments {
		next, _ := node.resolve(seg)
		if next == nil {
			return nil
		}
		node = next
	}
	return node
}
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
