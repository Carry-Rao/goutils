package sqlite

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/Carry-Rao/goutils/database/api"
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
	schema := api.GetOrBuildSchema(api.TypeOf(example))
	return &Table{schema: schema, db: s.db, tableName: tableName, insertQry: schema.BuildInsertQueryMysql(tableName)}, nil
}

func (s *Database) CreateFromStruct(tableName string, example any) error {
	schema := api.GetOrBuildSchema(api.TypeOf(example))

	var columns []string
	var pks []string

	for _, field := range schema.Fields {
		sqlType := mapGoTypeToSQL(field.GoKind)

		var buf strings.Builder
		fmt.Fprintf(&buf, "`%s` %s", field.ColumnName, sqlType)

		if field.IsAutoInc {
			buf.WriteString(" PRIMARY KEY AUTOINCREMENT")
		} else {
			if !field.IsNullable {
				buf.WriteString(" NOT NULL")
			}
			if field.IsPrimary {
				pks = append(pks, fmt.Sprintf("`%s`", field.ColumnName))
			}
		}
		if field.IsUnique {
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

func mapGoTypeToSQL(kind reflect.Kind) string {
	switch kind {
	case reflect.String:
		return "TEXT"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "INTEGER"
	case reflect.Bool:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	default:
		return "TEXT"
	}
}
