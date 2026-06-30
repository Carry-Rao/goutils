package sqlite

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/Carry-Rao/goutils/database/api"
	"github.com/Carry-Rao/goutils/database/helpers"
)

func (s *Database) Create(tableName string, config map[string]api.Config) error {
	var columns []string
	var pks []string

	typeMap := map[string]string{
		"string":  "TEXT",
		"text":    "TEXT",
		"int":     "INTEGER",
		"integer": "INTEGER",
		"float":   "REAL",
		"real":    "REAL",
		"bool":    "INTEGER",
	}

	for field, cfg := range config {
		sqlType := typeMap[cfg.Type]
		if sqlType == "" {
			sqlType = "TEXT"
		}

		var buf strings.Builder
		fmt.Fprintf(&buf, "`%s` %s", field, sqlType)

		if cfg.Identity {
			buf.WriteString(" PRIMARY KEY AUTOINCREMENT")
		} else {
			if !cfg.NullAble {
				buf.WriteString(" NOT NULL")
			}
			if cfg.PrimaryKey {
				pks = append(pks, fmt.Sprintf("`%s`", field))
			}
		}
		if cfg.Unique {
			buf.WriteString(" UNIQUE")
		}

		columns = append(columns, buf.String())
	}

	if len(columns) == 0 {
		return fmt.Errorf("empty config")
	}

	if len(pks) > 0 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pks, ",")))
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s)", tableName, strings.Join(columns, ","))
	_, err := s.db.Exec(query)
	return err
}

func (s *Database) GetTable(tableName string, example any) (api.Table, error) {
	var name string
	err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if name == "" {
		if err := s.CreateFromStruct(tableName, example); err != nil {
			return nil, err
		}
	}
	return &Table{typ: reflect.TypeOf(example), db: s.db, tableName: tableName}, nil
}

func (s *Database) CreateFromStruct(tableName string, example any) error {
	t := helpers.UnwrapType(reflect.TypeOf(example))
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	var columns []string
	var pks []string

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		colName, tags := helpers.ParseDBTag(f)
		sqlType := helpers.MapGoTypeToSQL(f.Type.Kind())

		var buf strings.Builder
		fmt.Fprintf(&buf, "`%s` %s", colName, sqlType)

		if tags["autoinc"] {
			buf.WriteString(" PRIMARY KEY AUTOINCREMENT")
		} else {
			if !tags["null"] {
				buf.WriteString(" NOT NULL")
			}
			if tags["primary"] {
				pks = append(pks, fmt.Sprintf("`%s`", colName))
			}
		}
		if tags["unique"] {
			buf.WriteString(" UNIQUE")
		}

		columns = append(columns, buf.String())
	}

	if len(columns) == 0 {
		return fmt.Errorf("struct has no exported fields")
	}

	if len(pks) > 0 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pks, ",")))
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s)", tableName, strings.Join(columns, ","))
	_, err := s.db.Exec(query)
	return err
}

func (s *Database) DeleteTable(tableName string) error {
	_, err := s.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	return err
}