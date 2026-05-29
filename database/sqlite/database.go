package sqlite

import (
	"fmt"
	"strings"

	"github.com/Carry-Rao/goutils/database/api"
)

func (s *Database) Create(tableName string, config map[string]api.Config) error {
	var columns []string
	var pks []string

	for field, cfg := range config {
		col := fmt.Sprintf("%s %s", field, cfg.Type)
		if !cfg.NullAble {
			col += " NOT NULL"
		}
		if cfg.Identity {
			col += " AUTOINCREMENT"
		}
		if cfg.Unique {
			col += " UNIQUE"
		}
		if cfg.PrimaryKey {
			pks = append(pks, field)
		}
		columns = append(columns, col)
	}

	if len(pks) > 0 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pks, ",")))
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(columns, ","))
	_, err := s.db.Exec(sql)
	return err
}

func (s *Database) GetTable(tableName string) (api.Table, error) {
	var exists bool
	err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("table not exists")
	}
	return &Table{db: s.db, tableName: tableName}, nil
}

func (s *Database) DeleteTable(tableName string) error {
	_, err := s.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	return err
}
