package bloom

import (
	"hash/fnv"

	"github.com/Carry-Rao/goutils/database/cache"
)

type Table[T any] struct {
	td        *tableData[T]
	tableName string
	schema    *cache.Schema
	pk        cache.FieldInfo
	bf        BloomFilter
}

func (t *Table[T]) key(val T) (string, error) {
	v, err := cache.Unwrap(val)
	if err != nil {
		return "", err
	}
	return t.schema.KeyOf(t.tableName+"_", v, t.pk), nil
}

func (t *Table[T]) Ins(val T) error {
	key, err := t.key(val)
	if err != nil {
		return err
	}
	h1, h2 := t.hashPair(key)
	t.bf.Add(h1, h2)

	t.td.mu.Lock()
	defer t.td.mu.Unlock()
	t.td.data[key] = val
	return nil
}

// Get reports ErrNotFound on a miss, since membership is the whole point of a
// bloom filter. Remember that it has false positives.
func (t *Table[T]) Get(val T) ([]T, error) {
	key, err := t.key(val)
	if err != nil {
		return nil, err
	}
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
	return []T{d}, nil
}

// A changed primary key moves the record and re-registers it in the filter.
func (t *Table[T]) Set(old, updated T) error {
	oldKey, err := t.key(old)
	if err != nil {
		return err
	}
	newKey, err := t.key(updated)
	if err != nil {
		return err
	}

	h1, h2 := t.hashPair(newKey)
	t.bf.Add(h1, h2)

	t.td.mu.Lock()
	defer t.td.mu.Unlock()
	if newKey != oldKey {
		delete(t.td.data, oldKey)
	}
	t.td.data[newKey] = updated
	return nil
}

func (t *Table[T]) Del(val T) error {
	key, err := t.key(val)
	if err != nil {
		return err
	}

	h1, h2 := t.hashPair(key)
	if !t.bf.Contains(h1, h2) {
		return ErrNotFound
	}

	t.td.mu.Lock()
	defer t.td.mu.Unlock()
	delete(t.td.data, key)
	return nil
}

func (t *Table[T]) hashPair(s string) (uint64, uint64) {
	h := fnv.New64a()
	h.Write([]byte(s))
	v := h.Sum64()
	return v + 1, v + 2
}
