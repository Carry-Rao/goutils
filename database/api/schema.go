package api

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type FieldInfo struct {
	Index       int
	ColumnName  string
	IsAutoInc   bool
	IsNullable  bool
	IsPrimary   bool
	IsUnique    bool
	GoFieldName string
	GoKind      reflect.Kind
}

type CachedSchema struct {
	Type       reflect.Type
	Fields     []FieldInfo
	Insertable []FieldInfo
	PKIndex    int
	FieldMap   map[string]FieldInfo

	InsertColsMysql  string
	InsertPlaceMysql string
	InsertColsPG     string
	InsertPlacePG    string
}

var schemaCache sync.Map

func GetOrBuildSchema(typ reflect.Type) *CachedSchema {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if cached, ok := schemaCache.Load(typ); ok {
		return cached.(*CachedSchema)
	}
	schema := buildSchema(typ)
	schemaCache.Store(typ, schema)
	return schema
}

func buildSchema(typ reflect.Type) *CachedSchema {
	schema := &CachedSchema{
		Type:     typ,
		PKIndex:  -1,
		FieldMap: make(map[string]FieldInfo),
	}

	var insertCols []string
	var insertPlaceMysql []string
	var insertPlacePG []string
	paramIdx := 1

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		field := parseField(f, i)
		schema.Fields = append(schema.Fields, field)
		schema.FieldMap[field.ColumnName] = field

		if field.IsPrimary {
			schema.PKIndex = len(schema.Fields) - 1
		}

		if !field.IsAutoInc {
			schema.Insertable = append(schema.Insertable, field)
			insertCols = append(insertCols, "`"+field.ColumnName+"`")
			insertPlaceMysql = append(insertPlaceMysql, "?")
			insertPlacePG = append(insertPlacePG, "$"+strconv.Itoa(paramIdx))
			paramIdx++
		}
	}

	schema.InsertColsMysql = strings.Join(insertCols, ",")
	schema.InsertPlaceMysql = strings.Join(insertPlaceMysql, ",")
	schema.InsertColsPG = strings.Join(insertCols, ",")
	schema.InsertPlacePG = strings.Join(insertPlacePG, ",")
	return schema
}

func parseField(f reflect.StructField, index int) FieldInfo {
	field := FieldInfo{
		Index:       index,
		GoFieldName: f.Name,
		GoKind:      f.Type.Kind(),
	}

	tag := f.Tag.Get("db")
	parts := strings.Split(tag, ",")
	if len(parts) > 0 && parts[0] != "" {
		field.ColumnName = parts[0]
	} else {
		field.ColumnName = f.Name
	}

	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		switch part {
		case "autoinc":
			field.IsAutoInc = true
		case "null":
			field.IsNullable = true
		case "primary":
			field.IsPrimary = true
		case "unique":
			field.IsUnique = true
		}
	}
	return field
}

func (s *CachedSchema) BuildInsertQueryMysql(tableName string) string {
	return "INSERT INTO `" + tableName + "` (" + s.InsertColsMysql + ") VALUES (" + s.InsertPlaceMysql + ")"
}

func (s *CachedSchema) BuildInsertQueryPG(tableName string) string {
	return "INSERT INTO " + tableName + " (" + s.InsertColsPG + ") VALUES (" + s.InsertPlacePG + ")"
}

func (s *CachedSchema) ExtractValues(val reflect.Value) []any {
	args := make([]any, 0, len(s.Insertable))
	for _, f := range s.Insertable {
		args = append(args, val.Field(f.Index).Interface())
	}
	return args
}

func (s *CachedSchema) BuildWhereClause(val reflect.Value, whereFields []string, paramStyle string) (string, []any, error) {
	var conds []string
	var args []any
	paramIdx := 1
	for _, fieldName := range whereFields {
		field, found := s.FieldMap[fieldName]
		if !found {
			f, ok := s.findFieldByGoName(fieldName)
			if !ok {
				return "", nil, &FieldNotFoundError{Name: fieldName}
			}
			field = f
		}
		fv := val.Field(field.Index)
		var placeholder string
		switch paramStyle {
		case "pg":
			placeholder = field.ColumnName + "=$" + strconv.Itoa(paramIdx)
		default:
			placeholder = "`" + field.ColumnName + "`=?"
		}
		conds = append(conds, placeholder)
		args = append(args, fv.Interface())
		paramIdx++
	}
	if len(conds) == 0 {
		return "", nil, nil
	}
	return strings.Join(conds, " AND "), args, nil
}

func (s *CachedSchema) findFieldByGoName(name string) (FieldInfo, bool) {
	for _, f := range s.Fields {
		if f.GoFieldName == name {
			return f, true
		}
	}
	return FieldInfo{}, false
}

type FieldNotFoundError struct {
	Name string
}

func (e *FieldNotFoundError) Error() string {
	return "field " + e.Name + " not found in struct"
}
