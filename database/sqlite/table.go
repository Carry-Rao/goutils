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

// findFieldByDBTag finds a struct field matching the given name (either db tag column name or Go field name).
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

// buildWhereClause builds a WHERE clause from the given fields of a struct value.
// whereFields specifies which fields to use as conditions (by db column name or Go field name).
// Returns the clause string, args slice, and any error.
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

// Ins inserts a new row into the table from the example struct.
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
		// 跳过 autoinc 字段（由数据库自动生成）
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

// Get queries the table using whereFields as conditions from the example struct.
// Returns a pointer to a new struct of the same type populated with the result.
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

	query := fmt.Sprintf("SELECT * FROM `%s` WHERE %s LIMIT 1", t.tableName, whereClause)
	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, fmt.Errorf("not found")
	}

	// Build column name -> index map
	colIndex := make(map[string]int, len(cols))
	for i, c := range cols {
		colIndex[c] = i
	}

	// Create result struct and scan destinations
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

	return result.Addr().Interface(), nil
}

// Set updates the table using whereFields as WHERE conditions, and the remaining
// struct fields as SET values.
func (t *Table) Set(example any, whereFields []string, _ time.Duration) error {
	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	// Build WHERE from whereFields
	whereClause, args, err := buildWhereClause(typ, val, whereFields)
	if err != nil {
		return err
	}
	if whereClause == "" {
		return fmt.Errorf("no where conditions specified")
	}

	// Build SET from all fields NOT in whereFields
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

// Delete removes rows matching the whereFields conditions from the example struct.
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
	if whereClause == "" {
		return fmt.Errorf("no where conditions specified")
	}

	query := fmt.Sprintf("DELETE FROM `%s` WHERE %s", t.tableName, whereClause)
	_, err = t.db.Exec(query, args...)
	return err
}
