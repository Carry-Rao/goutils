package bloom

import (
	"fmt"
	"hash/fnv"
	"time"

	"github.com/Carry-Rao/goutils/database/api"
)

type Table struct {
	td        *tableData
	tableName string
	schema    *api.CachedSchema
	bf        BloomFilter
}

func (t *Table) Ins(example any, _ time.Duration) error {
	if t.td.cacheKey == "" {
		return ErrNotFound
	}

	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	f, found := t.schema.FieldMap[t.td.cacheKey]
	if !found {
		return ErrNotFound
	}
	idVal := val.Field(f.Index).Interface()
	key := t.tableName + "_" + fmt.Sprintf("%v", idVal)

	h1, h2 := t.hashPair(key)
	t.bf.Add(h1, h2)

	t.td.mu.Lock()
	t.td.data[key] = example
	t.td.mu.Unlock()
	return nil
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) ([]any, error) {
	if t.td.cacheKey == "" {
		return nil, ErrNotFound
	}

	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}

	f, found := t.schema.FieldMap[t.td.cacheKey]
	if !found {
		return nil, ErrNotFound
	}
	idVal := val.Field(f.Index).Interface()
	key := t.tableName + "_" + fmt.Sprintf("%v", idVal)

	h1, h2 := t.hashPair(key)
	if !t.bf.Contains(h1, h2) {
		return nil, ErrNotFound
	}

	t.td.mu.RLock()
	d, ok := t.td.data[key]
	t.td.mu.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}
	return []any{d}, nil
}

func (t *Table) Set(example any, whereFields []string, _ time.Duration) error {
	if t.td.cacheKey == "" {
		return ErrNotFound
	}

	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	f, found := t.schema.FieldMap[t.td.cacheKey]
	if !found {
		return ErrNotFound
	}
	idVal := val.Field(f.Index).Interface()
	key := t.tableName + "_" + fmt.Sprintf("%v", idVal)

	h1, h2 := t.hashPair(key)
	if !t.bf.Contains(h1, h2) {
		return ErrNotFound
	}

	t.td.mu.Lock()
	t.td.data[key] = example
	t.td.mu.Unlock()
	return nil
}

func (t *Table) Del(example any, whereFields []string, _ time.Duration) error {
	if t.td.cacheKey == "" {
		return ErrNotFound
	}

	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	f, found := t.schema.FieldMap[t.td.cacheKey]
	if !found {
		return ErrNotFound
	}
	idVal := val.Field(f.Index).Interface()
	key := t.tableName + "_" + fmt.Sprintf("%v", idVal)

	h1, h2 := t.hashPair(key)
	if !t.bf.Contains(h1, h2) {
		return ErrNotFound
	}

	t.td.mu.Lock()
	delete(t.td.data, key)
	t.td.mu.Unlock()
	return nil
}

func (t *Table) hashPair(s string) (uint64, uint64) {
	h := fnv.New64a()
	h.Write([]byte(s))
	v := h.Sum64()
	return v + 1, v + 2
}
