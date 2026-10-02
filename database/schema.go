package database

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// FieldInfo describes one mapped struct field.
type FieldInfo struct {
	Index       int
	ColumnName  string
	GoFieldName string
	GoKind      reflect.Kind

	IsAutoInc  bool
	IsNullable bool
	IsPrimary  bool
	IsUnique   bool

	// IsChild marks a slice, array or map whose values live in a companion
	// table instead of a column.
	IsChild    bool
	ChildTable string // overrides the derived table name
	GoElem     reflect.Type
}

// Schema is the parsed, cached form of a model type.
type Schema struct {
	Type     reflect.Type
	Fields   []FieldInfo
	FieldMap map[string]FieldInfo
	PKIndex  int

	// Scalar holds the fields stored as columns, PKIndex into Fields.
	Scalars []FieldInfo
	// Collections holds the fields stored in companion tables.
	Collections []FieldInfo

	insertCols string
	insertArgs string
}

var schemaCache sync.Map

// TypeOf returns T's struct type, dereferencing pointers.
func TypeOf[T any]() reflect.Type {
	t := reflect.TypeOf((*T)(nil))
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

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

// Unwrap dereferences a value down to its struct form.
func Unwrap(v any) (reflect.Value, error) {
	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("value must be a struct or pointer to struct")
	}
	return val, nil
}

// NewStructValue allocates an addressable T.
func NewStructValue[T any]() reflect.Value {
	return reflect.New(SchemaOf[T]().Type).Elem()
}

func buildSchema(typ reflect.Type) *Schema {
	s := &Schema{Type: typ, PKIndex: -1, FieldMap: make(map[string]FieldInfo)}

	var cols, args []string

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		field, err := parseField(f, i)
		if err != nil {
			panic(fmt.Sprintf("database: %s.%s: %v", typ.Name(), f.Name, err))
		}
		s.Fields = append(s.Fields, field)
		s.FieldMap[field.ColumnName] = field

		switch {
		case field.IsPrimary:
			s.PKIndex = len(s.Fields) - 1
			if field.IsAutoInc {
				continue
			}
			s.Scalars = append(s.Scalars, field)
			cols = append(cols, field.ColumnName)
			args = append(args, "?")
		case field.IsChild:
			s.Collections = append(s.Collections, field)
		default:
			s.Scalars = append(s.Scalars, field)
			cols = append(cols, field.ColumnName)
			args = append(args, "?")
		}
	}

	s.insertCols = strings.Join(cols, ",")
	s.insertArgs = strings.Join(args, ",")
	return s
}

// parseField reads a field's db tag. The tag is comma separated: a non-empty
// first element is the column name, the rest are constraints. An absent or empty
// tag leaves the column named after the field.
//
//	`db:"col"`                  column "col"
//	`db:",primary"`             column from the field name, marked primary
//	`db:",child"`               stored in a companion table
//	`db:",child=tag_links"`     companion table named explicitly
func parseField(f reflect.StructField, index int) (FieldInfo, error) {
	field := FieldInfo{
		Index:       index,
		GoFieldName: f.Name,
		GoKind:      f.Type.Kind(),
	}

	parts := strings.Split(f.Tag.Get("db"), ",")
	if parts[0] != "" {
		field.ColumnName = parts[0]
	} else {
		field.ColumnName = f.Name
	}

	for _, part := range parts[1:] {
		name, value, _ := strings.Cut(strings.TrimSpace(part), "=")
		switch name {
		case "autoinc":
			field.IsAutoInc = true
		case "null":
			field.IsNullable = true
		case "primary":
			field.IsPrimary = true
		case "unique":
			field.IsUnique = true
		case "child":
			field.IsChild = true
			field.ChildTable = value
		default:
			return field, fmt.Errorf("unknown db constraint %q", name)
		}
	}

	if err := applyKind(&field, f.Type); err != nil {
		return field, err
	}
	return field, nil
}

// applyKind validates that collections are marked as children and records their
// element type. Silently dropping an unmarked collection would lose data.
func applyKind(field *FieldInfo, typ reflect.Type) error {
	var elem reflect.Type

	switch typ.Kind() {
	case reflect.Slice, reflect.Array:
		if !field.IsChild {
			return fmt.Errorf("slice field must be tagged `db:\",child\"` to be stored")
		}
		elem = typ.Elem()
	case reflect.Map:
		if !field.IsChild {
			return fmt.Errorf("map field must be tagged `db:\",child\"` to be stored")
		}
		// A map's key must survive a round trip through a text column too, since
		// the companion table stores it as one.
		if !storableKind(typ.Key().Kind()) {
			return fmt.Errorf("map key %s cannot be stored as a child value", typ.Key())
		}
		elem = typ.Elem()
	default:
		return nil
	}

	if !storableKind(elem.Kind()) {
		return fmt.Errorf("element %s cannot be stored as a child value; "+
			"use a basic type such as string, int, bool or float", elem)
	}
	field.GoElem = elem
	return nil
}

// storableKind reports whether a kind round-trips through a text column, which
// is what the companion tables hold. Checking up front turns a panic deep in
// hydration into an error at schema time.
func storableKind(k reflect.Kind) bool {
	switch k {
	case reflect.String,
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// PrimaryKey returns the primary key field. Only single-column keys are supported.
func (s *Schema) PrimaryKey() (FieldInfo, error) {
	if s.PKIndex < 0 {
		return FieldInfo{}, fmt.Errorf("no primary key; tag one field `db:\",primary\"`")
	}
	return s.Fields[s.PKIndex], nil
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

// ChildTableName resolves the companion table for a collection field. Without an
// explicit name it is derived from the main table and the field's column.
func (s *Schema) ChildTableName(mainTable string, f FieldInfo) string {
	if f.ChildTable != "" {
		return f.ChildTable
	}
	return mainTable + "_" + strings.ToLower(f.ColumnName)
}

// IsCollection reports whether op makes sense for the field's type.
func (s *Schema) IsCollection(name string) bool {
	f, ok := s.Lookup(name)
	return ok && f.IsChild
}

func (s *Schema) InsertQuery(d Dialect, table string) string {
	cols := make([]string, 0, len(s.Scalars))
	args := make([]string, 0, len(s.Scalars))
	idx := 1
	for _, f := range s.Scalars {
		cols = append(cols, d.Quote(f.ColumnName))
		args = append(args, d.placeholder(idx))
		idx++
	}
	return "INSERT INTO " + d.QuoteTable(table) +
		" (" + strings.Join(cols, ",") + ") VALUES (" + strings.Join(args, ",") + ")"
}

// InsertValues gathers bind arguments for an insert.
func (s *Schema) InsertValues(v reflect.Value) []any {
	args := make([]any, 0, len(s.Scalars))
	for _, f := range s.Scalars {
		args = append(args, v.Field(f.Index).Interface())
	}
	return args
}

// KeyString renders a value as the string used in collection and map keys.
func KeyString(v reflect.Value) string {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return itoa(int(v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return itoa(int(v.Uint()))
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", v.Float())
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// stringify renders a collection element as the string stored in a child table.
func stringify(v reflect.Value) string {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.String {
		return v.String()
	}
	return KeyString(v)
}
