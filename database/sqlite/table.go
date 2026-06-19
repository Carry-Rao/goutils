package sqlite

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type Table struct {
	typ       reflect.Type
	db        *sql.DB
	tableName string
}

func getDBColumnName(f reflect.StructField) string {
	tag := f.Tag.Get("db")
	if tag == "" {
		return f.Name
	}
	parts := strings.Split(tag, ",")
	if parts[0] != "" {
		return parts[0]
	}
	return f.Name
}

func findFieldByDBTag(typ reflect.Type, name string) (reflect.StructField, bool) {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		colName := getDBColumnName(f)
		if colName == name || f.Name == name {
			return f, true
		}
	}
	return reflect.StructField{}, false
}

func buildWhereClause(typ reflect.Type, val reflect.Value, whereFields []string) (string, []any, error) {
	var conds []string
	var args []any
	for _, fieldName := range whereFields {
		f, found := findFieldByDBTag(typ, fieldName)
		if !found {
			return "", nil, fmt.Errorf("field %q not found in struct", fieldName)
		}
		fv := val.FieldByIndex(f.Index)
		colName := getDBColumnName(f)
		conds = append(conds, fmt.Sprintf("`%s`=?", colName))
		args = append(args, fv.Interface())
	}
	return strings.Join(conds, " AND "), args, nil
}

func (t *Table) Ins(example any, _ time.Duration) error {
	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	var cols []string
	var placeholders []string
	var args []any
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		colName := getDBColumnName(f)
		tag := f.Tag.Get("db")
		if strings.Contains(tag, "autoinc") {
			continue
		}
		cols = append(cols, fmt.Sprintf("`%s`", colName))
		placeholders = append(placeholders, "?")
		args = append(args, val.Field(i).Interface())
	}
	if len(cols) == 0 {
		return fmt.Errorf("struct has no exported fields")
	}
	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		t.tableName,
		strings.Join(cols, ","),
		strings.Join(placeholders, ","))
	_, err := t.db.Exec(query, args...)
	return err
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) ([]any, error) {
	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	whereClause, args, err := buildWhereClause(typ, val, whereFields)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM `%s` WHERE %s", t.tableName, whereClause)
	if whereClause == "" {
		query = fmt.Sprintf("SELECT * FROM `%s`", t.tableName)
	}
	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	colIndex := make(map[string]int, len(cols))
	for i, c := range cols {
		colIndex[c] = i
	}

	var results []any
	for rows.Next() {
		result := reflect.New(typ).Elem()
		scanPtrs := make([]any, len(cols))
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if !f.IsExported() {
				continue
			}
			colName := getDBColumnName(f)
			if idx, ok := colIndex[colName]; ok {
				scanPtrs[idx] = result.Field(i).Addr().Interface()
			}
		}
		if err := rows.Scan(scanPtrs...); err != nil {
			return nil, err
		}
		results = append(results, result.Addr().Interface())
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("not found")
	}

	return results, nil
}

func (t *Table) Set(example any, whereFields []string, _ time.Duration) error {
	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	whereClause, args, err := buildWhereClause(typ, val, whereFields)
	if err != nil {
		return err
	}

	whereSet := make(map[string]bool)
	for _, fn := range whereFields {
		whereSet[fn] = true
	}

	var sets []string
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		colName := getDBColumnName(f)
		if whereSet[colName] || whereSet[f.Name] {
			continue
		}
		fv := val.Field(i)
		sets = append(sets, fmt.Sprintf("`%s`=?", colName))
		args = append(args, fv.Interface())
	}

	if len(sets) == 0 {
		return fmt.Errorf("no fields to set")
	}

	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE %s", t.tableName, strings.Join(sets, ","), whereClause)
	_, err = t.db.Exec(query, args...)
	return err
}

func (t *Table) Del(example any, whereFields []string, _ time.Duration) error {
	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	whereClause, args, err := buildWhereClause(typ, val, whereFields)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM `%s`", t.tableName)
	if whereClause != "" {
		query = fmt.Sprintf("DELETE FROM `%s` WHERE %s", t.tableName, whereClause)
	}
	_, err = t.db.Exec(query, args...)
	return err
}
