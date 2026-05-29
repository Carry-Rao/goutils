package mixture

import "github.com/Carry-Rao/goutils/database/api"

type ErrAction int

const (
	Continue ErrAction = 0
	Return   ErrAction = 1
)

type Database struct {
	dbs     []api.Database
	errActs []ErrAction
}

func NewDatabase(_ map[string]string) (*Database, error) {
	return &Database{}, nil
}

func (m *Database) Add(db api.Database, act ErrAction) {
	m.dbs = append(m.dbs, db)
	m.errActs = append(m.errActs, act)
}

func (m *Database) Create(tableName string, config map[string]api.Config) error {
	for i, db := range m.dbs {
		err := db.Create(tableName, config)
		if err == nil {
			continue
		}
		if m.errActs[i] == Return {
			return err
		}
	}
	return nil
}

func (m *Database) GetTable(tableName string) (api.Table, error) {
	var tables []api.Table
	var acts []ErrAction

	for i, db := range m.dbs {
		tbl, err := db.GetTable(tableName)
		if err != nil {
			if m.errActs[i] == Return {
				return nil, err
			}
			continue
		}
		tables = append(tables, tbl)
		acts = append(acts, m.errActs[i])
	}

	return &Table{
		tables: tables,
		acts:   acts,
	}, nil
}

func (m *Database) DeleteTable(tableName string) error {
	for i, db := range m.dbs {
		err := db.DeleteTable(tableName)
		if err == nil {
			continue
		}
		if m.errActs[i] == Return {
			return err
		}
	}
	return nil
}
