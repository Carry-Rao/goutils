// Package sqlite provides a SQLite-backed database.Database. The driver is
// embedded, so no server is needed.
package sqlite

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/Carry-Rao/goutils/database"

	"github.com/mattn/go-sqlite3"
)

type Database[T any] struct {
	database.Backend[T]
}

func init() {
	database.Register("sqlite")
	database.Classify(isDuplicate)
}

// isDuplicate recognises SQLite's extended constraint result. The driver
// reports every constraint failure — not-null, check, foreign key — under one
// code and separates them only by message, so the primary-key and unique
// messages are matched as well to avoid reporting a check failure as a
// duplicate.
func isDuplicate(err error) bool {
	var se sqlite3.Error
	if !errors.As(err, &se) || se.Code != sqlite3.ErrConstraint {
		return false
	}
	msg := se.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "PRIMARY KEY must be unique")
}

func NewDatabase[T any](cfg map[string]string) (*Database[T], error) {
	db, err := sql.Open("sqlite3", cfg["path"])
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &Database[T]{database.Backend[T]{DB: db, Dialect: database.SQLiteDialect}}, nil
}
