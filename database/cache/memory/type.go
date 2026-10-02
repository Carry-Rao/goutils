package memory

import (
	"sync"

	"github.com/Carry-Rao/goutils/database/cache"
)

type entry[T any] struct {
	data T
}

type Database[T any] struct {
	data  map[string]map[string]entry[T]
	cache map[string]string
	mu    sync.RWMutex
}

func NewDatabase[T any](cfg map[string]string) (*Database[T], error) {
	return &Database[T]{
		data:  make(map[string]map[string]entry[T]),
		cache: make(map[string]string),
	}, nil
}

// Close satisfies the cache contract. Nothing external is held.
func (m *Database[T]) Close() error { return nil }

func (m *Database[T]) Create(tableName string, config map[string]cache.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.data[tableName]; !ok {
		m.data[tableName] = make(map[string]entry[T])
	}
	for k, v := range config {
		if v.PrimaryKey {
			m.cache[tableName] = k
			break
		}
	}
	return nil
}

func (m *Database[T]) GetTable(tableName string, _ T) (cache.Table[T], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.data[tableName]; !ok {
		return nil, nil
	}
	schema := cache.SchemaOf[T]()
	pk, err := schema.PrimaryKey()
	if err != nil {
		return nil, err
	}
	return &Table[T]{db: m, tableName: tableName, keyPrefix: tableName + "_",
		schema: schema, pk: pk}, nil
}

func (m *Database[T]) DeleteTable(tableName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, tableName)
	delete(m.cache, tableName)
	return nil
}
