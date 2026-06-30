package helpers

import (
	"fmt"
	"reflect"
	"strings"
)

func UnwrapType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func UnwrapValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func GetDBColumnName(f reflect.StructField) string {
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

func ParseDBTag(f reflect.StructField) (colName string, tags map[string]bool) {
	tag := f.Tag.Get("db")
	parts := strings.Split(tag, ",")
	tagsMap := make(map[string]bool)
	for _, part := range parts {
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
	return colName, tagsMap
}

func FindFieldByDBTag(typ reflect.Type, name string) (reflect.StructField, bool) {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		colName := GetDBColumnName(f)
		if colName == name || f.Name == name {
			return f, true
		}
	}
	return reflect.StructField{}, false
}

func BuildWhereClause(typ reflect.Type, val reflect.Value, whereFields []string, paramStyle string) (string, []any, error) {
	var conds []string
	var args []any
	paramIdx := 1
	for _, fieldName := range whereFields {
		f, found := FindFieldByDBTag(typ, fieldName)
		if !found {
			return "", nil, fmt.Errorf("field %q not found in struct", fieldName)
		}
		fv := val.FieldByIndex(f.Index)
		colName := GetDBColumnName(f)
		var placeholder string
		switch paramStyle {
		case "pg":
			placeholder = fmt.Sprintf("%s=$%d", colName, paramIdx)
		default:
			placeholder = fmt.Sprintf("`%s`=?", colName)
		}
		conds = append(conds, placeholder)
		args = append(args, fv.Interface())
		paramIdx++
	}
	return strings.Join(conds, " AND "), args, nil
}

func MapGoTypeToSQL(kind reflect.Kind) string {
	switch kind {
	case reflect.String:
		return "TEXT"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "INTEGER"
	case reflect.Bool:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	default:
		return "TEXT"
	}
}