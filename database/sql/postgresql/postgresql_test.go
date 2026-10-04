package postgresql

import (
	"net/url"
	"strings"
	"testing"
)

// sslmode must never be baked in: a library that forces one either breaks a TLS
// deployment or silently downgrades a local one.
func TestDSNOmitsSSLModeByDefault(t *testing.T) {
	got := dsn(map[string]string{
		"user": "app", "password": "pw", "host": "db", "port": "5432", "dbname": "app",
	})
	if strings.Contains(got, "sslmode") {
		t.Errorf("sslmode must be omitted when unset, got %q", got)
	}
}

func TestDSNReadsSSLModeFromConfig(t *testing.T) {
	got := dsn(map[string]string{
		"user": "app", "password": "pw", "host": "db", "port": "5432",
		"dbname": "app", "sslmode": "verify-full",
	})
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse %q: %v", got, err)
	}
	if u.Query().Get("sslmode") != "verify-full" {
		t.Errorf("sslmode = %q, want verify-full", u.Query().Get("sslmode"))
	}
}

// A password with URL metacharacters must survive intact.
func TestDSNEscapesCredentials(t *testing.T) {
	const pw = "p@ss:w/rd?#&%"
	got := dsn(map[string]string{
		"user": "us:er", "password": pw, "host": "127.0.0.1", "port": "5432",
		"dbname": "my db",
	})

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse %q: %v", got, err)
	}
	if pass, _ := u.User.Password(); pass != pw {
		t.Errorf("password round-trip: got %q, want %q", pass, pw)
	}
	if gotPass := u.User.Username(); gotPass != "us:er" {
		t.Errorf("user round-trip: got %q", gotPass)
	}
	if u.Path != "/my db" {
		t.Errorf("database name: got %q", u.Path)
	}
	if u.Host != "127.0.0.1:5432" {
		t.Errorf("host: got %q", u.Host)
	}
}

// An empty password must not emit a stray colon.
func TestDSNOmitsEmptyPassword(t *testing.T) {
	got := dsn(map[string]string{
		"user": "app", "host": "db", "port": "5432", "dbname": "app",
	})
	if strings.Contains(got, "app:@") {
		t.Errorf("empty password should be omitted, got %q", got)
	}
}

// Unrecognised keys are libpq parameters and must be forwarded.
func TestDSNForwardsExtraParams(t *testing.T) {
	got := dsn(map[string]string{
		"user": "app", "host": "db", "port": "5432", "dbname": "app",
		"connect_timeout": "5", "application_name": "goutils",
	})
	u, _ := url.Parse(got)
	q := u.Query()
	if q.Get("connect_timeout") != "5" || q.Get("application_name") != "goutils" {
		t.Errorf("extra params lost: %q", u.RawQuery)
	}
	// Consumed keys must not leak into the query string.
	for _, k := range []string{"user", "password", "host", "port", "dbname"} {
		if q.Has(k) {
			t.Errorf("%q should not appear in the query string", k)
		}
	}
}
