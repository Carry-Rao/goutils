package redis

import (
	"github.com/Carry-Rao/goutils/database/api"
)

func (r *Database) Create(tableName string, config map[string]api.Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tables[tableName]; !ok {
		r.tables[tableName] = make(map[string]any)
	}

	field := ""
	for k, v := range config {
		if v.PrimaryKey {
			field = k
			break
		}
	}
	r.cacheField[tableName] = field
	return nil
}

func (r *Database) GetTable(tableName string) (api.Table, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.tables[tableName]; !ok {
		return nil, nil
	}

	return &Table{
		db:        r,
		tableName: tableName,
		cacheKey:  r.cacheField[tableName],
	}, nil
}

func (r *Database) DeleteTable(tableName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tables, tableName)
	delete(r.cacheField, tableName)
	return nil
}
