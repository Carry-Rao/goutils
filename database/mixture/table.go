package mixture

import "github.com/Carry-Rao/goutils/database/api"

type Table struct {
	tables []api.Table
	acts   []ErrAction
}

func (t *Table) Create(data map[string]any) error {
	for i, tbl := range t.tables {
		err := tbl.Create(data)
		if err == nil {
			continue
		}
		if t.acts[i] == Return {
			return err
		}
	}
	return nil
}

func (t *Table) Get(where map[string]any) ([]any, error) {
	var lastErr error
	for i, tbl := range t.tables {
		res, err := tbl.Get(where)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if t.acts[i] == Return {
			return nil, err
		}
	}
	return nil, lastErr
}

func (t *Table) Set(data map[string]any) error {
	for i, tbl := range t.tables {
		err := tbl.Set(data)
		if err == nil {
			continue
		}
		if t.acts[i] == Return {
			return err
		}
	}
	return nil
}

func (t *Table) Delete(where map[string]any) error {
	for i, tbl := range t.tables {
		err := tbl.Delete(where)
		if err == nil {
			continue
		}
		if t.acts[i] == Return {
			return err
		}
	}
	return nil
}
