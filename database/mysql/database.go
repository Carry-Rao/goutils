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
		col := fmt.Sprintf("`%s` %s", field, cfg.Type)
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
			pks = append(pks, "`"+field+"`")
		}
		columns = append(columns, col)
	}

	if len(pks) > 0 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pks, ",")))
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", tableName, strings.Join(columns, ","))
	_, err := m.db.Exec(query)
	return err
}

func (m *Database) GetTable(tableName string, example any) (api.Table, error) {
	var exists bool
	err := m.db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?", tableName).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := m.CreateFromStruct(tableName, example); err != nil {
			return nil, err
		}
	}
	schema := api.GetOrBuildSchema(api.TypeOf(example))
	return &Table{db: m.db, tableName: tableName, schema: schema, insertQry: schema.BuildInsertQueryMysql(tableName)}, nil
}

func (m *Database) CreateFromStruct(tableName string, example any) error {
	schema := api.GetOrBuildSchema(api.TypeOf(example))

	var columns []string
	var pks []string

	for _, field := range schema.Fields {
		sqlType := mapGoTypeToSQL(field.GoKind)

		var buf strings.Builder
		fmt.Fprintf(&buf, "`%s` %s", field.ColumnName, sqlType)

		if field.IsAutoInc {
			buf.WriteString(" PRIMARY KEY AUTO_INCREMENT")
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

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", tableName, strings.Join(columns, ","))
	_, err := m.db.Exec(query)
	return err
}

func (m *Database) DeleteTable(tableName string) error {
	_, err := m.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName))
	return err
}

func mapGoTypeToSQL(kind reflect.Kind) string {
	switch kind {
	case reflect.String:
		return "TEXT"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "BIGINT"
	case reflect.Bool:
		return "TINYINT(1)"
	case reflect.Float32, reflect.Float64:
		return "DOUBLE"
	default:
		return "TEXT"
	}
}
