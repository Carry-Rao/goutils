package db

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func Open(driver, dsn string) (*sql.DB, error) {
	return sql.Open(driver, dsn)
}
