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

	// Mapping from Config.Type to SQLite types
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

		buf := strings.Builder{}
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

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s)", tableName, strings.Join(columns, ","))
	_, err := s.db.Exec(sql)
	return err
}

func (s *Database) GetTable(tableName string, example any) (api.Table, error) {
	var name string
	err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	exists := name != ""
	if !exists {
		err = s.CreateFromStruct(tableName, example)
		if err != nil {
			return nil, err
		}
	}
	return &Table{typ: reflect.TypeOf(example), db: s.db, tableName: tableName}, nil
}

// CreateFromStruct creates a table from a struct example, keeping backward compatibility.
func (s *Database) CreateFromStruct(tableName string, example any) error {
	t := reflect.TypeOf(example)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
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

		tagParts := strings.Split(f.Tag.Get("db"), ",")
		var colName string
		tagsMap := make(map[string]bool)
		for _, part := range tagParts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if colName == "" {
				colName = part
			} else {
				tagsMap[part] = true
			}
		}
		if colName == "" {
			colName = f.Name
		}

		var sqlType string
		ft := f.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		switch ft.Kind() {
		case reflect.String:
			sqlType = "TEXT"
		case reflect.Int, reflect.Int64, reflect.Int32:
			sqlType = "INTEGER"
		case reflect.Bool:
			sqlType = "INTEGER"
		case reflect.Float32, reflect.Float64:
			sqlType = "REAL"
		default:
			sqlType = "TEXT"
		}

		buf := strings.Builder{}
		fmt.Fprintf(&buf, "`%s` %s", colName, sqlType)

		isPrimary := tagsMap["primary"]
		isAutoInc := tagsMap["autoinc"]

		if isAutoInc {
			buf.WriteString(" PRIMARY KEY AUTOINCREMENT")
		} else {
			if tagsMap["null"] {
				buf.WriteString(" NOT NULL")
			}
			if isPrimary {
				pks = append(pks, fmt.Sprintf("`%s`", colName))
			}
		}
		if tagsMap["unique"] {
			buf.WriteString(" UNIQUE")
		}

		columns = append(columns, buf.String())
	}

	if len(columns) == 0 {
		return fmt.Errorf("struct has no export fields")
	}

	if len(pks) > 0 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pks, ",")))
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s)", tableName, strings.Join(columns, ","))
	_, err := s.db.Exec(sql)
	return err
}

func (s *Database) DeleteTable(tableName string) error {
	_, err := s.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	return err
}
