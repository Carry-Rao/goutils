package postgresql

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Table struct {
	pool  *pgxpool.Pool
	table string
	typ   reflect.Type
}

// getDBColumnName extracts the column name from a struct field's `db` tag.
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

// findFieldByDBTag finds a struct field matching the given name.
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

// buildWhereClause builds a WHERE clause with $1, $2, ... placeholders.
func buildWhereClause(typ reflect.Type, val reflect.Value, whereFields []string) (string, []any, error) {
	var conds []string
	var args []any
	paramIdx := 1
	for _, fieldName := range whereFields {
		f, found := findFieldByDBTag(typ, fieldName)
		if !found {
			return "", nil, fmt.Errorf("field %q not found in struct", fieldName)
		}
		fv := val.FieldByIndex(f.Index)
		colName := getDBColumnName(f)
		conds = append(conds, fmt.Sprintf("%s=$%d", colName, paramIdx))
		args = append(args, fv.Interface())
		paramIdx++
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
	paramIdx := 1
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
		cols = append(cols, colName)
		placeholders = append(placeholders, fmt.Sprintf("$%d", paramIdx))
		args = append(args, val.Field(i).Interface())
		paramIdx++
	}
	if len(cols) == 0 {
		return fmt.Errorf("struct has no exported fields")
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		t.table,
		strings.Join(cols, ","),
		strings.Join(placeholders, ","))
	_, err := t.pool.Exec(context.Background(), sql, args...)
	return err
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) (any, error) {
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
	if whereClause == "" {
		return nil, fmt.Errorf("no where conditions specified")
	}

	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 1", t.table, whereClause)
	rows, err := t.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := rows.FieldDescriptions()
	if len(cols) == 0 {
		return nil, fmt.Errorf("not found")
	}

	if !rows.Next() {
		return nil, fmt.Errorf("not found")
	}

	colIndex := make(map[string]int, len(cols))
	for i, c := range cols {
		colIndex[string(c.Name)] = i
	}

	values, err := rows.Values()
	if err != nil {
		return nil, err
	}

	result := reflect.New(typ).Elem()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		colName := getDBColumnName(f)
		if idx, ok := colIndex[colName]; ok && idx < len(values) {
			v := values[idx]
			if v != nil {
				result.Field(i).Set(reflect.ValueOf(v))
			}
		}
	}

	return result.Addr().Interface(), nil
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
	if whereClause == "" {
		return fmt.Errorf("no where conditions specified")
	}

	whereSet := make(map[string]bool)
	for _, fn := range whereFields {
		whereSet[fn] = true
	}

	var sets []string
	paramIdx := len(args) + 1
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
		sets = append(sets, fmt.Sprintf("%s=$%d", colName, paramIdx))
		args = append(args, fv.Interface())
		paramIdx++
	}

	if len(sets) == 0 {
		return fmt.Errorf("no fields to set")
	}

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s", t.table, strings.Join(sets, ","), whereClause)
	_, err = t.pool.Exec(context.Background(), sql, args...)
	return err
}

func (t *Table) Delete(example any, whereFields []string, _ time.Duration) error {
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
	if whereClause == "" {
		return fmt.Errorf("no where conditions specified")
	}

	sql := fmt.Sprintf("DELETE FROM %s WHERE %s", t.table, whereClause)
	_, err = t.pool.Exec(context.Background(), sql, args...)
	return err
}
