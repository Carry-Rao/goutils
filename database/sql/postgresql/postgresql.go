// Package postgresql provides a PostgreSQL-backed database.Database.
package postgresql

import (
	"database/sql"
	"fmt"
	"net"
	"net/url"

	"github.com/Carry-Rao/goutils/database"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database[T any] struct {
	database.Backend[T]
}

func init() { database.Register("postgresql") }

// NewDatabase opens a connection. Recognised keys are user, password, host,
// port, dbname and sslmode; any other key is appended to the query string, so
// options such as connect_timeout or application_name pass straight through.
//
// sslmode is never hardcoded. Left unset it is omitted entirely and pgx applies
// its own default; set it to verify-full in production, or disable for a local
// server with no TLS configured.
func NewDatabase[T any](cfg map[string]string) (*Database[T], error) {
	db, err := sql.Open("pgx", dsn(cfg))
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Database[T]{database.Backend[T]{DB: db, Dialect: database.PostgreSQLDialect}}, nil
}

// dsn builds the connection URL. Credentials go through net/url rather than
// fmt.Sprintf, so a password containing @ / ? # or : still produces a valid URL
// instead of a silently mis-parsed one.
func dsn(cfg map[string]string) string {
	u := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(cfg["host"], cfg["port"]),
		Path:   "/" + cfg["dbname"],
	}

	if pass, ok := cfg["password"]; ok && pass != "" {
		u.User = url.UserPassword(cfg["user"], pass)
	} else if user := cfg["user"]; user != "" {
		u.User = url.User(user)
	}

	// url.URL.Query returns a copy, so the parameters have to be written back
	// or they are silently dropped.
	q := u.Query()

	// Recognised keys are consumed above; everything else is a libpq parameter.
	for k, v := range cfg {
		switch k {
		case "user", "password", "host", "port", "dbname", "sslmode":
			continue
		}
		q.Set(k, v)
	}

	// Omitted rather than defaulted, so the driver's own policy applies.
	if mode := cfg["sslmode"]; mode != "" {
		q.Set("sslmode", mode)
	}

	u.RawQuery = q.Encode()
	return u.String()
}
