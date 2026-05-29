package memory

import (
	"fmt"
	"sync"
)

type Table struct {
	db        *Database
	tableName string
	cacheKey  string
	mu        sync.RWMutex
}

func (t *Table) Create(data map[string]any) error {
	if t.cacheKey == "" {
		return nil
	}

	idVal, ok := data[t.cacheKey]
	if !ok {
		return nil
	}

	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	t.mu.Lock()
	defer t.mu.Unlock()
	t.db.data[t.tableName][key] = data
	return nil
}

func (t *Table) Get(where map[string]any) ([]any, error) {
	if t.cacheKey == "" {
		return nil, nil
	}

	idVal, ok := where[t.cacheKey]
	if !ok {
		return nil, nil
	}

	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	t.mu.RLock()
	defer t.mu.RUnlock()

	val, ok := t.db.data[t.tableName][key]
	if !ok {
		return nil, nil
	}

	m, ok := val.(map[string]any)
	if !ok {
		return nil, nil
	}

	res := make([]any, 0, len(m))
	for _, v := range m {
		res = append(res, v)
	}
	return res, nil
}

func (t *Table) Set(data map[string]any) error {
	if t.cacheKey == "" {
		return nil
	}

	idVal, ok := data[t.cacheKey]
	if !ok {
		return fmt.Errorf("missing cache key")
	}

	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	t.mu.Lock()
	defer t.mu.Unlock()
	t.db.data[t.tableName][key] = data
	return nil
}

func (t *Table) Delete(where map[string]any) error {
	if t.cacheKey == "" {
		return nil
	}

	idVal, ok := where[t.cacheKey]
	if !ok {
		return nil
	}

	key := fmt.Sprintf("%s_%v", t.tableName, idVal)

	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.db.data[t.tableName], key)
	return nil
}
