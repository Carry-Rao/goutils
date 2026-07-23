package postgresql

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(config map[string]string) (*Database, error) {
	user := config["user"]
	password := config["password"]
	host := config["host"]
	port := config["port"]
	dbname := config["dbname"]

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname,
	)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &Database{db: db}, nil
}

func (p *Database) Close() {
	p.db.Close()
}
