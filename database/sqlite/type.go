package sqlite

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(cfg map[string]string) (*Database, error) {
	dbPath := cfg["path"]
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Database{db: db}, nil
}

func (s *Database) Close() error {
	return s.db.Close()
}

// Exec executes a raw SQL query (useful for inserts and other operations not covered by the Table interface).
func (s *Database) Exec(query string, args ...any) error {
	_, err := s.db.Exec(query, args...)
	return err
}
