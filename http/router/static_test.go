package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// pubDir builds a small static tree and returns its path.
func pubDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.html", "root-index")
	write("app.css", "body{}")
	write("css/site.css", "css{}")
	write("assets/deep/nested.txt", "deep")
	return dir
}

func get(r *Router, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// The prefix must be stripped before the file lookup, otherwise FileServer
// looks for root/static/... and every request 404s.
func TestStaticServesFiles(t *testing.T) {
	dir := pubDir(t)
	r := New()
	r.Static("/static", dir)

	cases := []struct{ path, want string }{
		{"/static/app.css", "body{}"},
		{"/static/css/site.css", "css{}"},
		{"/static/assets/deep/nested.txt", "deep"},
	}
	for _, c := range cases {
		w := get(r, c.path)
		if w.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", c.path, w.Code)
			continue
		}
		if w.Body.String() != c.want {
			t.Errorf("%s: got %q, want %q", c.path, w.Body.String(), c.want)
		}
	}
}

// A file named after the mount point must not resolve to root/static/...
func TestStaticStripsPrefixNotAppends(t *testing.T) {
	dir := pubDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "static", "decoy.txt"), []byte("decoy"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := New()
	r.Static("/static", dir)

	// root/static/decoy.txt must NOT be reachable: after stripping, the lookup
	// is /decoy.txt, which does not exist. Were the prefix re-appended rather
	// than stripped, this would return 200.
	if w := get(r, "/static/decoy.txt"); w.Code != http.StatusNotFound {
		t.Errorf("root/static/ must not be reachable, got %d", w.Code)
	}
}

// http.FileServer canonicalises /index.html to ./ and serves index.html for a
// directory. Both are net/http behaviour, asserted here so a router change
// cannot silently alter them.
func TestStaticIndexCanonicalisation(t *testing.T) {
	dir := pubDir(t)
	r := New()
	r.Static("/static", dir)

	if w := get(r, "/static/index.html"); w.Code != http.StatusMovedPermanently {
		t.Errorf("expected 301 for index.html, got %d", w.Code)
	}
	w := get(r, "/static/")
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for directory, got %d", w.Code)
	}
	if w.Body.String() != "root-index" {
		t.Errorf("expected index.html content, got %q", w.Body.String())
	}
}

// A trailing slash on the pattern must behave like one without.
func TestStaticPatternTrailingSlash(t *testing.T) {
	dir := pubDir(t)
	for _, pattern := range []string{"/static", "/static/"} {
		r := New()
		r.Static(pattern, dir)
		if w := get(r, "/static/app.css"); w.Code != http.StatusOK {
			t.Errorf("pattern %q: expected 200, got %d", pattern, w.Code)
		}
	}
}

// Sub routers must compose their prefix, otherwise the strip amount is wrong.
func TestStaticInsideSub(t *testing.T) {
	dir := pubDir(t)
	r := New()
	sub := r.Sub("/api")
	sub.Static("/assets", dir)

	if w := get(r, "/api/assets/app.css"); w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if w := get(r, "/api/assets/assets/deep/nested.txt"); w.Code != http.StatusOK {
		t.Errorf("nested: expected 200, got %d", w.Code)
	}
}

// Nested Sub routers accumulate prefixes.
func TestStaticInsideNestedSub(t *testing.T) {
	dir := pubDir(t)
	r := New()
	inner := r.Sub("/v1").Sub("/static")
	inner.Static("/files", dir)

	if w := get(r, "/v1/static/files/app.css"); w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

// Mounting at the root must not break StripPrefix with an empty prefix.
func TestStaticAtRoot(t *testing.T) {
	dir := pubDir(t)
	r := New()
	r.Static("/", dir)

	if w := get(r, "/app.css"); w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w := get(r, "/css/site.css"); w.Code != http.StatusOK {
		t.Errorf("nested: expected 200, got %d", w.Code)
	}
}

// Path traversal must not escape the root; http.FileServer cleans the path,
// and stripping must not reintroduce the traversal.
func TestStaticRejectsTraversal(t *testing.T) {
	dir := pubDir(t)
	r := New()
	r.Static("/static", dir)

	for _, p := range []string{
		"/static/../../etc/passwd",
		"/static/../secret",
		"/static/css/../../escape.txt",
	} {
		w := get(r, p)
		if w.Code == http.StatusOK {
			t.Errorf("%s: traversal should not succeed, got 200 %q", p, w.Body.String())
		}
	}
}

// Middleware on the subtree path must still run, so the match node chain has
// to include the subtree node.
func TestStaticRunsMiddleware(t *testing.T) {
	dir := pubDir(t)
	r := New()

	var order []string
	sub := r.Sub("/static")
	sub.Medium(func(w http.ResponseWriter, _ *http.Request, _ []string) bool {
		order = append(order, "sub")
		return true
	})
	sub.Static("/files", dir)

	r.Medium(func(w http.ResponseWriter, _ *http.Request, _ []string) bool {
		order = append(order, "global")
		return true
	})

	w := get(r, "/static/files/app.css")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(order) != 2 || order[0] != "global" || order[1] != "sub" {
		t.Errorf("middleware order = %v, want [global sub]", order)
	}
}

// Middleware must be able to block static requests too.
func TestStaticMiddlewareCanBlock(t *testing.T) {
	dir := pubDir(t)
	r := New()
	r.Medium(func(w http.ResponseWriter, _ *http.Request, _ []string) bool { return false })
	r.Static("/static", dir)

	w := get(r, "/static/app.css")
	if w.Body.Len() != 0 || w.Header().Get("Content-Type") != "" {
		t.Errorf("blocking middleware should serve nothing, got %d %q", w.Code, w.Body.String())
	}
}

// Static paths exist for 405 detection, so a wrong method reports 405.
func TestStaticWrongMethod(t *testing.T) {
	dir := pubDir(t)
	r := New()
	r.Static("/static", dir)

	req := httptest.NewRequest(http.MethodPost, "/static/index.html", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// A subtree must not swallow sibling routes registered on the same prefix.
func TestStaticCoexistsWithAPI(t *testing.T) {
	dir := pubDir(t)
	r := New()
	r.Static("/static", dir)
	r.GET("/static/config", func(w http.ResponseWriter, _ *http.Request, _ []string) {
		w.Write([]byte("config"))
	})

	if w := get(r, "/static/config"); w.Body.String() != "config" {
		t.Errorf("literal sibling route should win, got %d %q", w.Code, w.Body.String())
	}
	if w := get(r, "/static/app.css"); w.Code != http.StatusOK {
		t.Errorf("static file still reachable, got %d", w.Code)
	}
}

// joinPath is what Sub and Static rely on to compose prefixes.
func TestJoinPath(t *testing.T) {
	cases := []struct{ prefix, pattern, want string }{
		{"", "/static", "/static"},
		{"", "static", "/static"},
		{"/", "/static", "/static"},
		{"/api", "/assets", "/api/assets"},
		{"/api/", "/assets/", "/api/assets"},
		{"/v1", "/static/files", "/v1/static/files"},
		{"", "", ""},
		{"", "/", ""},
		{"/", "/", ""},
	}
	for _, c := range cases {
		if got := joinPath(c.prefix, c.pattern); got != c.want {
			t.Errorf("joinPath(%q, %q) = %q, want %q", c.prefix, c.pattern, got, c.want)
		}
	}
}
