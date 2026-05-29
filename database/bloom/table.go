package bloom

import (
	"fmt"
	"hash/fnv"
)

type Table struct {
	db        *Database
	tableName string
	cacheKey  string
}

func (t *Table) Create(data map[string]any) error {
	if t.cacheKey == "" {
		return ErrNotFound
	}

	idVal, ok := data[t.cacheKey]
	if !ok {
		return ErrNotFound
	}

	key := fmt.Sprintf("%s_%v", t.tableName, idVal)
	t.add(key)

	t.db.mu.Lock()
	defer t.db.mu.Unlock()
	t.db.data[t.tableName][key] = data
	return nil
}

func (t *Table) Get(where map[string]any) ([]any, error) {
	if t.cacheKey == "" {
		return nil, ErrNotFound
	}

	idVal, ok := where[t.cacheKey]
	if !ok {
		return nil, ErrNotFound
	}

	key := fmt.Sprintf("%s_%v", t.tableName, idVal)
	if !t.contains(key) {
		return nil, ErrNotFound
	}

	t.db.mu.RLock()
	defer t.db.mu.RUnlock()

	val, ok := t.db.data[t.tableName][key]
	if !ok {
		return nil, ErrNotFound
	}

	m, ok := val.(map[string]any)
	if !ok {
		return nil, ErrNotFound
	}

	res := make([]any, 0, len(m))
	for _, v := range m {
		res = append(res, v)
	}
	return res, nil
}

func (t *Table) Set(data map[string]any) error {
	if t.cacheKey == "" {
		return ErrNotFound
	}

	idVal, ok := data[t.cacheKey]
	if !ok {
		return ErrNotFound
	}

	key := fmt.Sprintf("%s_%v", t.tableName, idVal)
	if !t.contains(key) {
		return ErrNotFound
	}

	t.db.mu.Lock()
	defer t.db.mu.Unlock()
	t.db.data[t.tableName][key] = data
	return nil
}

func (t *Table) Delete(where map[string]any) error {
	if t.cacheKey == "" {
		return ErrNotFound
	}

	idVal, ok := where[t.cacheKey]
	if !ok {
		return ErrNotFound
	}

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
