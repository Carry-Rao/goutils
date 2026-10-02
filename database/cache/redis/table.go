package redis

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Carry-Rao/goutils/database/cache"
)

type Table[T any] struct {
	db        *Database[T]
	tableName string
	schema    *cache.Schema
	pk        cache.FieldInfo
}

func (t *Table[T]) key(val T) (string, error) {
	v, err := cache.Unwrap(val)
	if err != nil {
		return "", err
	}
	return t.schema.KeyOf(t.tableName+"_", v, t.pk), nil
}

func (t *Table[T]) write(v reflect.Value, key string) error {
	data := make(map[string]any, len(t.schema.Fields))
	for _, f := range t.schema.Fields {
		data[f.ColumnName] = v.Field(f.Index).Interface()
	}
	return t.db.client.HSet(context.Background(), key, data).Err()
}

func (t *Table[T]) Ins(val T) error {
	key, err := t.key(val)
	if err != nil {
		return err
	}
	v, err := cache.Unwrap(val)
	if err != nil {
		return err
	}
	return t.write(v, key)
}

// Get returns (nil, nil) on a miss, matching the other cache backends.
func (t *Table[T]) Get(val T) ([]T, error) {
	key, err := t.key(val)
	if err != nil {
		return nil, err
	}

	data, err := t.db.client.HGetAll(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	result := reflect.New(t.schema.Type).Elem()
	for _, f := range t.schema.Fields {
		if s, ok := data[f.ColumnName]; ok {
			setFieldFromString(result.Field(f.Index), s)
		}
	}
	return []T{result.Addr().Interface().(T)}, nil
}

// A changed primary key moves the record.
func (t *Table[T]) Set(old, updated T) error {
	oldKey, err := t.key(old)
	if err != nil {
		return err
	}
	newKey, err := t.key(updated)
	if err != nil {
		return err
	}
	if oldKey != newKey {
		if err := t.db.client.Del(context.Background(), oldKey).Err(); err != nil {
			return err
		}
	}
	v, err := cache.Unwrap(updated)
	if err != nil {
		return err
	}
	return t.write(v, newKey)
}

func (t *Table[T]) Del(val T) error {
	key, err := t.key(val)
	if err != nil {
		return err
	}
	return t.db.client.Del(context.Background(), key).Err()
}

// setFieldFromString fills a field from its stored string form, leaving the zero
// value when the conversion does not apply.
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
