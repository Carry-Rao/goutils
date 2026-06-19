package memory

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

type Table struct {
	db        *Database
	tableName string
	cacheKey  string
	mu        sync.RWMutex
}

func (t *Table) Ins(example any, ttl time.Duration) error {
	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	// Find primary key field
	pkField := findFieldByDBTag(typ, t.cacheKey)
	if pkField.Index == nil {
		return fmt.Errorf("primary key field %q not found", t.cacheKey)
	}
	idVal := val.FieldByIndex(pkField.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	entry := &entry{
		data:    example,
		expires: time.Now().Add(ttl),
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.db.data[t.tableName][key] = entry
	return nil
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) (any, error) {
	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}

	// Build key from whereFields cacheKey
	pkField := findFieldByDBTag(val.Type(), t.cacheKey)
	if pkField.Index == nil {
		return nil, fmt.Errorf("primary key field %q not found", t.cacheKey)
	}
	idVal := val.FieldByIndex(pkField.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	t.mu.RLock()
	defer t.mu.RUnlock()

	raw, ok := t.db.data[t.tableName][key]
	if !ok {
		return nil, fmt.Errorf("not found")
	}

	ent, ok := raw.(*entry)
	if !ok {
		return nil, fmt.Errorf("invalid entry type")
	}

	// Check expiry
	if !ent.expires.IsZero() && time.Now().After(ent.expires) {
		// Remove expired entry
		delete(t.db.data[t.tableName], key)
		return nil, fmt.Errorf("not found")
	}

	return ent.data, nil
}

func (t *Table) Set(example any, whereFields []string, ttl time.Duration) error {
	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	// Build key from whereFields cacheKey
	pkField := findFieldByDBTag(typ, t.cacheKey)
	if pkField.Index == nil {
		return fmt.Errorf("primary key field %q not found", t.cacheKey)
	}
	idVal := val.FieldByIndex(pkField.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	t.mu.Lock()
	defer t.mu.Unlock()

	entry := &entry{
		data:    example,
		expires: time.Now().Add(ttl),
	}
	t.db.data[t.tableName][key] = entry
	return nil
}

func (t *Table) Del(example any, whereFields []string, _ time.Duration) error {
	val := reflect.ValueOf(example)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	// Build key from whereFields cacheKey
	pkField := findFieldByDBTag(val.Type(), t.cacheKey)
	if pkField.Index == nil {
		return fmt.Errorf("primary key field %q not found", t.cacheKey)
	}
	idVal := val.FieldByIndex(pkField.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.db.data[t.tableName], key)
	return nil
}

// findFieldByDBTag finds a struct field matching the given name.
func findFieldByDBTag(typ reflect.Type, name string) reflect.StructField {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		colName := getDBColumnName(f)
		if colName == name || f.Name == name {
			return f
		}
	}
	return reflect.StructField{}
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
