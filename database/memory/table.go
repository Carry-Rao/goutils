package memory

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/Carry-Rao/goutils/database/helpers"
)

type Table struct {
	db        *Database
	tableName string
	cacheKey  string
	mu        sync.RWMutex
}

func (t *Table) Ins(example any, ttl time.Duration) error {
	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	pkField, found := helpers.FindFieldByDBTag(typ, t.cacheKey)
	if !found {
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

func (t *Table) Get(example any, whereFields []string, _ time.Duration) ([]any, error) {
	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	pkField, found := helpers.FindFieldByDBTag(typ, t.cacheKey)
	if !found {
		return nil, fmt.Errorf("primary key field %q not found", t.cacheKey)
	}
	idVal := val.FieldByIndex(pkField.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	t.mu.RLock()
	defer t.mu.RUnlock()

	raw, ok := t.db.data[t.tableName][key]
	if !ok {
		return nil, nil
	}

	ent, ok := raw.(*entry)
	if !ok {
		return nil, fmt.Errorf("invalid entry type")
	}

	if !ent.expires.IsZero() && time.Now().After(ent.expires) {
		delete(t.db.data[t.tableName], key)
		return nil, nil
	}

	return []any{ent.data}, nil
}

func (t *Table) Set(example any, whereFields []string, ttl time.Duration) error {
	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	pkField, found := helpers.FindFieldByDBTag(typ, t.cacheKey)
	if !found {
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
	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	pkField, found := helpers.FindFieldByDBTag(typ, t.cacheKey)
	if !found {
		return fmt.Errorf("primary key field %q not found", t.cacheKey)
	}
	idVal := val.FieldByIndex(pkField.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.db.data[t.tableName], key)
	return nil
}