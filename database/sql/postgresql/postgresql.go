// Package postgresql provides a PostgreSQL-backed database.Database.
package postgresql

import (
	"database/sql"
	"fmt"

	"github.com/Carry-Rao/goutils/database"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database[T any] struct {
	database.Backend[T]
}

func init() { database.Register("postgresql") }

func NewDatabase[T any](cfg map[string]string) (*Database[T], error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg["user"], cfg["password"], cfg["host"], cfg["port"], cfg["dbname"])

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Database[T]{database.Backend[T]{DB: db, Dialect: database.PostgreSQLDialect}}, nil
}
