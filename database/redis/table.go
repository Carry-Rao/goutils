package redis

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Carry-Rao/goutils/database/api"
)

type Table struct {
	db        *Database
	tableName string
	cacheKey  string
	schema    *api.CachedSchema
}

func (t *Table) Ins(example any, ttl time.Duration) error {
	if t.cacheKey == "" {
		return nil
	}

	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	f, found := t.schema.FieldMap[t.cacheKey]
	if !found {
		return nil
	}
	idVal := val.Field(f.Index).Interface()
	key := t.tableName + "_" + fmt.Sprintf("%v", idVal)

	data := make(map[string]any, len(t.schema.Fields))
	for _, field := range t.schema.Fields {
		data[field.ColumnName] = val.Field(field.Index).Interface()
	}

	return t.db.client.HSet(context.Background(), key, data).Err()
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) ([]any, error) {
	if t.cacheKey == "" {
		return nil, fmt.Errorf("not found")
	}

	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}

	f, found := t.schema.FieldMap[t.cacheKey]
	if !found {
		return nil, fmt.Errorf("not found")
	}
	idVal := val.Field(f.Index).Interface()
	key := t.tableName + "_" + fmt.Sprintf("%v", idVal)

	result := api.NewStruct(t.schema.Type)

	data, err := t.db.client.HGetAll(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	for _, field := range t.schema.Fields {
		if strVal, ok := data[field.ColumnName]; ok {
			setFieldFromString(result.Field(field.Index), strVal)
		}
	}

	return []any{result.Addr().Interface()}, nil
}

func (t *Table) Set(example any, whereFields []string, ttl time.Duration) error {
	if t.cacheKey == "" {
		return nil
	}

	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	f, found := t.schema.FieldMap[t.cacheKey]
	if !found {
		return fmt.Errorf("missing cache key")
	}
	idVal := val.Field(f.Index).Interface()
	key := t.tableName + "_" + fmt.Sprintf("%v", idVal)

	data := make(map[string]any, len(t.schema.Fields))
	for _, field := range t.schema.Fields {
		data[field.ColumnName] = val.Field(field.Index).Interface()
	}

	return t.db.client.HSet(context.Background(), key, data).Err()
}

func (t *Table) Del(example any, whereFields []string, _ time.Duration) error {
	if t.cacheKey == "" {
		return nil
	}

	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	f, found := t.schema.FieldMap[t.cacheKey]
	if !found {
		return nil
	}
	idVal := val.Field(f.Index).Interface()
	key := t.tableName + "_" + fmt.Sprintf("%v", idVal)

	return t.db.client.Del(context.Background(), key).Err()
}

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
