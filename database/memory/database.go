package memory

import (
	"github.com/Carry-Rao/goutils/database/api"
)

func (m *Database) Create(tableName string, config map[string]api.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.data[tableName]; !ok {
		m.data[tableName] = make(map[string]any)
	}

	field := ""
	for k, v := range config {
		if v.PrimaryKey {
			field = k
			break
		}
	}
	m.cache[tableName] = field
	return nil
}

func (m *Database) GetTable(tableName string, example any) (api.Table, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.data[tableName]; !ok {
		return nil, nil
	}

	schema := api.GetOrBuildSchema(api.TypeOf(example))
	cacheKey := m.cache[tableName]
	pkField, hasPK := schema.FieldMap[cacheKey]
	return &Table{
		db:        m,
		tableName: tableName,
		cacheKey:  cacheKey,
		keyPrefix: tableName + "_",
		pkField:   pkField,
		hasPK:     hasPK,
		schema:    schema,
	}, nil
}

func (m *Database) DeleteTable(tableName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, tableName)
	delete(m.cache, tableName)
	return nil
}
