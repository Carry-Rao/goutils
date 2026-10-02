// Package cache provides a key-value cache abstraction.
//
// It is deliberately separate from the database package: caches address
// records by primary key only, so they have no WHERE clause, no column-level
// assignment, and no ordering or paging to honour.
package cache

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Config describes one column when declaring a table.
type Config struct {
	Type       string
	PrimaryKey bool
}

// Table is the CRUD surface of one cached table. Every operation addresses a
// single record by its primary key.
//
// Set takes the old and new values separately so that a changed primary key
// moves the record rather than duplicating it.
type Table[T any] interface {
	Ins(val T) error
	Get(val T) ([]T, error)
	Set(old, updated T) error
	Del(val T) error
}

// FieldInfo describes one mapped struct field.
type FieldInfo struct {
	Index       int
	ColumnName  string
	IsPrimary   bool
	GoFieldName string
	GoKind      reflect.Kind
}

// Schema is the parsed form of a model type, cached per type.
type Schema struct {
	Type     reflect.Type
	Fields   []FieldInfo
	FieldMap map[string]FieldInfo
	PKIndex  int
}

var schemaCache sync.Map

// SchemaOf parses and caches the mapping for T.
func SchemaOf[T any]() *Schema {
	typ := TypeOf[T]()
	if cached, ok := schemaCache.Load(typ); ok {
		return cached.(*Schema)
	}
	s := buildSchema(typ)
	schemaCache.Store(typ, s)
	return s
}

// TypeOf returns T's underlying struct type, dereferencing pointers.
func TypeOf[T any]() reflect.Type {
	return TypeOfValue((*T)(nil))
}

// TypeOfValue is TypeOf for a runtime value.
func TypeOfValue(v any) reflect.Type {
	t := reflect.TypeOf(v)
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func buildSchema(typ reflect.Type) *Schema {
	s := &Schema{Type: typ, PKIndex: -1, FieldMap: make(map[string]FieldInfo)}

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		field := parseField(f, i)
		s.Fields = append(s.Fields, field)
		s.FieldMap[field.ColumnName] = field
		if field.IsPrimary {
			s.PKIndex = len(s.Fields) - 1
		}
	}
	return s
}

func parseField(f reflect.StructField, index int) FieldInfo {
	field := FieldInfo{
		Index:       index,
		GoFieldName: f.Name,
		GoKind:      f.Type.Kind(),
	}

	parts := strings.Split(f.Tag.Get("db"), ",")
	if len(parts) > 0 && parts[0] != "" {
		field.ColumnName = parts[0]
	} else {
		field.ColumnName = f.Name
	}
	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "primary" {
			field.IsPrimary = true
		}
	}
	return field
}

// Lookup resolves a column name or a Go field name.
func (s *Schema) Lookup(name string) (FieldInfo, bool) {
	if f, ok := s.FieldMap[name]; ok {
		return f, true
	}
	for _, f := range s.Fields {
		if f.GoFieldName == name {
			return f, true
		}
	}
	return FieldInfo{}, false
}

// PrimaryKey returns the primary key field, or an error when the model has none.
func (s *Schema) PrimaryKey() (FieldInfo, error) {
	if s.PKIndex < 0 {
		return FieldInfo{}, fmt.Errorf("cache: model has no primary key, tag one with `db:\"name,primary\"`")
	}
	return s.Fields[s.PKIndex], nil
}

// KeyOf renders the cache key for a value from the given prefix.
func (s *Schema) KeyOf(prefix string, v reflect.Value, pk FieldInfo) string {
	fv := v.Field(pk.Index)
	switch pk.GoKind {
	case reflect.String:
		return prefix + fv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return prefix + strconv.FormatInt(fv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return prefix + strconv.FormatUint(fv.Uint(), 10)
	default:
		return prefix + fmt.Sprintf("%v", fv.Interface())
	}
}

// Unwrap dereferences a value to its struct form.
func Unwrap(v any) (reflect.Value, error) {
	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("cache: value must be a struct or pointer to struct")
	}
	return val, nil
}

// Register records a driver name. Backends call it from init so the set of
// available caches can be listed and resolved by name.
func Register(name string) {
	driverRegistry.mu.Lock()
	defer driverRegistry.mu.Unlock()
	driverRegistry.names[name] = struct{}{}
}

// Drivers lists the registered cache driver names.
func Drivers() []string {
	driverRegistry.mu.RLock()
	defer driverRegistry.mu.RUnlock()

	names := make([]string, 0, len(driverRegistry.names))
	for n := range driverRegistry.names {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

var driverRegistry = struct {
	mu    sync.RWMutex
	names map[string]struct{}
}{names: make(map[string]struct{})}

// ErrNotFound reports a cache miss.
var ErrNotFound = errors.New("cache: not found")
