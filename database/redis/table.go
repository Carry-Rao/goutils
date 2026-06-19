package redis

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type Table struct {
	db        *Database
	tableName string
	cacheKey  string
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

func (t *Table) Ins(example any, ttl time.Duration) error {
	if t.cacheKey == "" {
		return nil
	}

	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	f, found := findFieldByDBTag(typ, t.cacheKey)
	if !found {
		return nil
	}
	idVal := val.FieldByIndex(f.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	// Build a map to marshal
	data := make(map[string]any)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		colName := getDBColumnName(field)
		data[colName] = val.Field(i).Interface()
	}

	return t.db.client.HSet(context.Background(), key, data).Err()
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) ([]any, error) {
	if t.cacheKey == "" {
		return nil, fmt.Errorf("not found")
	}

	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	f, found := findFieldByDBTag(typ, t.cacheKey)
	if !found {
		return nil, fmt.Errorf("not found")
	}
	idVal := val.FieldByIndex(f.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	result := reflect.New(typ).Elem()

	data, err := t.db.client.HGetAll(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("not found")
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		colName := getDBColumnName(field)
		if strVal, ok := data[colName]; ok {
			setFieldFromString(result.Field(i), strVal)
		}
	}

	return []any{result.Addr().Interface()}, nil
}

func (t *Table) Set(example any, whereFields []string, ttl time.Duration) error {
	if t.cacheKey == "" {
		return nil
	}

	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	f, found := findFieldByDBTag(typ, t.cacheKey)
	if !found {
		return fmt.Errorf("missing cache key")
	}
	idVal := val.FieldByIndex(f.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	// Build a map and use HSet
	data := make(map[string]any)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		colName := getDBColumnName(field)
		data[colName] = val.Field(i).Interface()
	}

	return t.db.client.HSet(context.Background(), key, data).Err()
}

func (t *Table) Del(example any, whereFields []string, _ time.Duration) error {
	if t.cacheKey == "" {
		return nil
	}

	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	f, found := findFieldByDBTag(typ, t.cacheKey)
	if !found {
		return nil
	}
	idVal := val.FieldByIndex(f.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	return t.db.client.Del(context.Background(), key).Err()
}

// setFieldFromString sets a reflect.Value from a string (Redis stores hashes as strings).
func setFieldFromString(fv reflect.Value, str string) {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(str)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var v int64
		fmt.Sscanf(str, "%d", &v)
		fv.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var v uint64
		fmt.Sscanf(str, "%d", &v)
		fv.SetUint(v)
	case reflect.Float32, reflect.Float64:
		var v float64
		fmt.Sscanf(str, "%f", &v)
		fv.SetFloat(v)
	case reflect.Bool:
		fv.SetBool(str == "1" || str == "true")
	}
}
