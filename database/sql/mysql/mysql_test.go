package mysql

import (
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

// A hand-written DSN breaks on any password containing @ or /, so the driver's
// own escaping is what these tests pin down. The username is deliberately plain:
// the DSN format splits on the first colon, so a colon in a username cannot be
// represented at all, by us or by anyone.
func TestDSNEscapesCredentials(t *testing.T) {
	const pw = "p@ss:w/rd?#&%"
	cfg := map[string]string{
		"user": "user", "password": pw, "host": "127.0.0.1", "port": "3306",
		"dbname": "my db",
	}

	parsed, err := mysql.ParseDSN(dsn(cfg))
	if err != nil {
		t.Fatalf("driver could not parse our own DSN: %v", err)
	}
	if parsed.Passwd != pw {
		t.Errorf("password round-trip: got %q, want %q", parsed.Passwd, pw)
	}
	if parsed.User != "user" {
		t.Errorf("user round-trip: got %q", parsed.User)
	}
	if parsed.DBName != "my db" {
		t.Errorf("database name: got %q", parsed.DBName)
	}
	if parsed.Addr != "127.0.0.1:3306" {
		t.Errorf("address: got %q", parsed.Addr)
	}
}

// tls must never be baked in, or the library silently downgrades production.
func TestDSNOmitsTLSByDefault(t *testing.T) {
	parsed, err := mysql.ParseDSN(dsn(map[string]string{
		"user": "app", "host": "db", "port": "3306", "dbname": "app",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.TLSConfig != "" {
		t.Errorf("tls must be omitted when unset, got %q", parsed.TLSConfig)
	}
}

func TestDSNReadsTLSFromConfig(t *testing.T) {
	parsed, err := mysql.ParseDSN(dsn(map[string]string{
		"user": "app", "host": "db", "port": "3306", "dbname": "app",
		"tls": "true",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.TLSConfig != "true" {
		t.Errorf("tls = %q, want true", parsed.TLSConfig)
	}
}

func TestDSNDefaults(t *testing.T) {
	parsed, err := mysql.ParseDSN(dsn(map[string]string{
		"user": "app", "host": "db", "port": "3306", "dbname": "app",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.ParseTime {
		t.Error("ParseTime should be on so time columns decode into time.Time")
	}
	// The driver consumes charset into Collation rather than Params.
	if !strings.Contains(dsn(map[string]string{
		"user": "a", "host": "h", "port": "1", "dbname": "d",
	}), "charset=utf8mb4") {
		t.Error("charset=utf8mb4 should be present in the DSN")
	}
}

// The driver parses the known options into typed fields; an unknown one must
// still arrive as a parameter rather than being dropped.
func TestDSNForwardsExtraParams(t *testing.T) {
	got := dsn(map[string]string{
		"user": "app", "host": "db", "port": "3306", "dbname": "app",
		"timeout": "5s", "readTimeout": "10s", "rejectReadOnly": "true",
	})
	parsed, err := mysql.ParseDSN(got)
	if err != nil {
		t.Fatalf("parse %q: %v", got, err)
	}
	if parsed.Timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", parsed.Timeout)
	}
	if parsed.ReadTimeout != 10*time.Second {
		t.Errorf("readTimeout = %v, want 10s", parsed.ReadTimeout)
	}
	if !parsed.RejectReadOnly {
		t.Error("rejectReadOnly was dropped")
	}
	// Consumed keys must not leak into the parameter list.
	for _, k := range []string{"user", "password", "host", "port", "dbname"} {
		if strings.Contains(got, k+"=") {
			t.Errorf("%q should not appear as a parameter", k)
		}
	}
}
