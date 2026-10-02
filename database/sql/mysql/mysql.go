// Package mysql provides a MySQL-backed database.Database.
package mysql

import (
	"database/sql"
	"fmt"

	"github.com/Carry-Rao/goutils/database"

	_ "github.com/go-sql-driver/mysql"
)

type Database[T any] struct {
	database.Backend[T]
}

func init() { database.Register("mysql") }

func NewDatabase[T any](cfg map[string]string) (*Database[T], error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
		cfg["user"], cfg["password"], cfg["host"], cfg["port"], cfg["dbname"])

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &Database[T]{database.Backend[T]{DB: db, Dialect: database.MySQLDialect}}, nil
}
