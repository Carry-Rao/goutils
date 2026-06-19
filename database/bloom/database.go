package bloom

import (
	"errors"

	"github.com/Carry-Rao/goutils/database/api"
)

var ErrNotFound = errors.New("not found")

func (b *Database) Create(tableName string, config map[string]api.Config) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.data[tableName]; !ok {
		b.data[tableName] = make(map[string]any)
	}
	if _, ok := b.bits[tableName]; !ok {
		b.bits[tableName] = make([]bool, 1024)
	}

	field := ""
	for k, v := range config {
		if v.PrimaryKey {
			field = k
			break
		}
	}
	b.cacheKey[tableName] = field
	return nil
}

func (b *Database) GetTable(tableName string, _ any) (api.Table, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if _, ok := b.data[tableName]; !ok {
		return nil, nil
	}

	return &Table{
		db:        b,
		tableName: tableName,
		cacheKey:  b.cacheKey[tableName],
	}, nil
}

func (b *Database) DeleteTable(tableName string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.data, tableName)
	delete(b.bits, tableName)
	delete(b.cacheKey, tableName)
	return nil
}
