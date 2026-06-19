package mysql

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Carry-Rao/goutils/database/api"
)

func (m *Database) Create(tableName string, config map[string]api.Config) error {
	var columns []string
	var pks []string

	for field, cfg := range config {
		col := fmt.Sprintf("%s %s", field, cfg.Type)
		if !cfg.NullAble {
			col += " NOT NULL"
		}
		if cfg.Identity {
			col += " AUTO_INCREMENT"
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

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", tableName, strings.Join(columns, ","))
	_, err := m.db.Exec(sql)
	return err
}

func (m *Database) GetTable(tableName string, example any) (api.Table, error) {
	var exists bool
	err := m.db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?", tableName).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("table not exists")
	}
	return &Table{db: m.db, tableName: tableName, typ: reflect.TypeOf(example)}, nil
}

func (m *Database) DeleteTable(tableName string) error {
	_, err := m.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	return err
}
