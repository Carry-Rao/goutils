package bloom

import (
	"errors"

	"github.com/Carry-Rao/goutils/database/api"
)

var ErrNotFound = errors.New("not found")

func (b *Database) Create(tableName string, config map[string]api.Config) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.tables[tableName]; !ok {
		b.tables[tableName] = &tableData{
			data: make(map[string]any),
		}
	}

	for k, v := range config {
		if v.PrimaryKey {
			b.tables[tableName].cacheKey = k
			break
		}
	}
	return nil
}

func (b *Database) GetTable(tableName string, example any) (api.Table, error) {
	b.mu.RLock()
	td, ok := b.tables[tableName]
	b.mu.RUnlock()

	if !ok {
		return nil, nil
	}

	schema := api.GetOrBuildSchema(api.TypeOf(example))
	return &Table{
		td:        td,
		tableName: tableName,
		schema:    schema,
	}, nil
}

func (b *Database) DeleteTable(tableName string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.tables, tableName)
	return nil
}
