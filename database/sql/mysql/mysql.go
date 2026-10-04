// Package mysql provides a MySQL-backed database.Database.
package mysql

import (
	"database/sql"
	"fmt"
	"net"

	"github.com/Carry-Rao/goutils/database"

	"github.com/go-sql-driver/mysql"
)

type Database[T any] struct {
	database.Backend[T]
}

func init() { database.Register("mysql") }

// NewDatabase opens a connection. Recognised keys are user, password, host,
// port, dbname, tls and charset; any other key is added to the driver's
// parameters, so options such as timeout or readTimeout pass straight through.
//
// tls is never hardcoded. Left unset it is omitted and the driver default
// applies; set it to true or skip-verify for a server with no TLS configured,
// and to a registered TLS config name in production.
func NewDatabase[T any](cfg map[string]string) (*Database[T], error) {
	db, err := sql.Open("mysql", dsn(cfg))
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Database[T]{database.Backend[T]{DB: db, Dialect: database.MySQLDialect}}, nil
}

// dsn builds the connection string through the driver's own config type, which
// escapes the user, password and database name for us. Hand-writing the string
// breaks on any password containing @ or /.
func dsn(cfg map[string]string) string {
	c := mysql.NewConfig()
	c.User = cfg["user"]
	c.Passwd = cfg["password"]
	c.Net = "tcp"
	c.Addr = net.JoinHostPort(cfg["host"], cfg["port"])
	c.DBName = cfg["dbname"]
	c.ParseTime = true

	if charset := cfg["charset"]; charset != "" {
		c.Params = map[string]string{"charset": charset}
	} else {
		c.Params = map[string]string{"charset": "utf8mb4"}
	}

	// Omitted rather than defaulted, so the driver's own policy applies.
	if name := cfg["tls"]; name != "" {
		c.TLSConfig = name
	}

	for k, v := range cfg {
		switch k {
		case "user", "password", "host", "port", "dbname", "tls", "charset":
			continue
		}
		if c.Params == nil {
			c.Params = make(map[string]string)
		}
		c.Params[k] = v
	}
	return c.FormatDSN()
}
