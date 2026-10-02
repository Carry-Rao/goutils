package memory

import (
	"sync"

	"github.com/Carry-Rao/goutils/database/cache"
)

type Table[T any] struct {
	db        *Database[T]
	tableName string
	keyPrefix string
	schema    *cache.Schema
	pk        cache.FieldInfo
	mu        sync.RWMutex
}

func (t *Table[T]) key(val T) (string, error) {
	v, err := cache.Unwrap(val)
	if err != nil {
		return "", err
	}
	return t.schema.KeyOf(t.keyPrefix, v, t.pk), nil
}

func (t *Table[T]) Ins(val T) error {
	key, err := t.key(val)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.db.data[t.tableName][key] = entry[T]{data: val}
	return nil
}

func (t *Table[T]) Get(val T) ([]T, error) {
	key, err := t.key(val)
	if err != nil {
		return nil, err
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	ent, ok := t.db.data[t.tableName][key]
	if !ok {
		return nil, nil
	}
	return []T{ent.data}, nil
}

// A changed primary key moves the record: the old key is dropped.
func (t *Table[T]) Set(old, updated T) error {
	oldKey, err := t.key(old)
	if err != nil {
		return err
	}
	newKey, err := t.key(updated)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if newKey != oldKey {
		delete(t.db.data[t.tableName], oldKey)
	}
	t.db.data[t.tableName][newKey] = entry[T]{data: updated}
	return nil
}

func (t *Table[T]) Del(val T) error {
	key, err := t.key(val)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.db.data[t.tableName], key)
	return nil
}
