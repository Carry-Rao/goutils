package bloom

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"time"

	"github.com/Carry-Rao/goutils/database/helpers"
)

type Table struct {
	db        *Database
	tableName string
	cacheKey  string
}

func (t *Table) Ins(example any, _ time.Duration) error {
	if t.cacheKey == "" {
		return ErrNotFound
	}

	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	f, found := helpers.FindFieldByDBTag(typ, t.cacheKey)
	if !found {
		return ErrNotFound
	}
	idVal := val.FieldByIndex(f.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	t.add(key)

	t.db.mu.Lock()
	defer t.db.mu.Unlock()
	t.db.data[t.tableName][key] = example
	return nil
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) ([]any, error) {
	if t.cacheKey == "" {
		return nil, ErrNotFound
	}

	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	f, found := helpers.FindFieldByDBTag(typ, t.cacheKey)
	if !found {
		return nil, ErrNotFound
	}
	idVal := val.FieldByIndex(f.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	if !t.contains(key) {
		return nil, ErrNotFound
	}

	t.db.mu.RLock()
	defer t.db.mu.RUnlock()

	d, ok := t.db.data[t.tableName][key]
	if !ok {
		return nil, ErrNotFound
	}

	return []any{d}, nil
}

func (t *Table) Set(example any, whereFields []string, _ time.Duration) error {
	if t.cacheKey == "" {
		return ErrNotFound
	}

	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	f, found := helpers.FindFieldByDBTag(typ, t.cacheKey)
	if !found {
		return ErrNotFound
	}
	idVal := val.FieldByIndex(f.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	if !t.contains(key) {
		return ErrNotFound
	}

	t.db.mu.Lock()
	defer t.db.mu.Unlock()
	t.db.data[t.tableName][key] = example
	return nil
}

func (t *Table) Del(example any, whereFields []string, _ time.Duration) error {
	if t.cacheKey == "" {
		return ErrNotFound
	}

	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	f, found := helpers.FindFieldByDBTag(typ, t.cacheKey)
	if !found {
		return ErrNotFound
	}
	idVal := val.FieldByIndex(f.Index).Interface()
	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	if !t.contains(key) {
		return ErrNotFound
	}

	t.db.mu.Lock()
	defer t.db.mu.Unlock()
	delete(t.db.data[t.tableName], key)
	return nil
}

func (t *Table) add(s string) {
	h1 := t.hash(s, 1)
	h2 := t.hash(s, 2)
	bits := t.db.bits[t.tableName]
	for i := 0; i < 3; i++ {
		idx := (h1 + uint64(i)*h2) % 1024
		bits[idx] = true
	}
}

func (t *Table) contains(s string) bool {
	h1 := t.hash(s, 1)
	h2 := t.hash(s, 2)
	bits := t.db.bits[t.tableName]
	for i := 0; i < 3; i++ {
		idx := (h1 + uint64(i)*h2) % 1024
		if !bits[idx] {
			return false
		}
	}
	return true
}

func (t *Table) hash(s string, seed uint64) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64() + seed
}