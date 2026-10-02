package bloom

import (
	"sync"

	"github.com/Carry-Rao/goutils/database/cache"
)

var ErrNotFound = cache.ErrNotFound

type tableData[T any] struct {
	data map[string]T
	mu   sync.RWMutex
}

type Database[T any] struct {
	tables map[string]*tableData[T]
	mu     sync.RWMutex
}

func NewDatabase[T any](_ map[string]string) (*Database[T], error) {
	return &Database[T]{tables: make(map[string]*tableData[T])}, nil
}

// Close satisfies the cache contract. Nothing external is held.
func (b *Database[T]) Close() error { return nil }

func (b *Database[T]) Create(tableName string, config map[string]cache.Config) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.tables[tableName]; !ok {
		b.tables[tableName] = &tableData[T]{data: make(map[string]T)}
	}
	return nil
}

func (b *Database[T]) GetTable(tableName string, _ T) (cache.Table[T], error) {
	b.mu.RLock()
	td, ok := b.tables[tableName]
	b.mu.RUnlock()

	if !ok {
		return nil, nil
	}
	schema := cache.SchemaOf[T]()
	pk, err := schema.PrimaryKey()
	if err != nil {
		return nil, err
	}
	return &Table[T]{td: td, tableName: tableName, schema: schema, pk: pk}, nil
}

func (b *Database[T]) DeleteTable(tableName string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.tables, tableName)
	return nil
}
