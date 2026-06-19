package mixture

import (
	"time"

	"github.com/Carry-Rao/goutils/database/api"
)

type Table struct {
	tables []api.Table
	acts   []ErrAction
}

func (t *Table) Ins(example any, ttl time.Duration) error {
	var lastErr error
	for i, tbl := range t.tables {
		err := tbl.Ins(example, ttl)
		if err == nil {
			return nil
		}
		lastErr = err
		if t.acts[i] == Return {
			return err
		}
	}
	return lastErr
}

func (t *Table) Get(example any, whereFields []string, ttl time.Duration) (any, error) {
	var lastErr error
	for i, tbl := range t.tables {
		res, err := tbl.Get(example, whereFields, ttl)
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

func (t *Table) Set(example any, whereFields []string, ttl time.Duration) error {
	var lastErr error
	for i, tbl := range t.tables {
		err := tbl.Set(example, whereFields, ttl)
		if err == nil {
			return nil
		}
		lastErr = err
		if t.acts[i] == Return {
			return err
		}
	}
	return lastErr
}

func (t *Table) Del(example any, whereFields []string, ttl time.Duration) error {
	var lastErr error
	for i, tbl := range t.tables {
		err := tbl.Del(example, whereFields, ttl)
		if err == nil {
			return nil
		}
		lastErr = err
		if t.acts[i] == Return {
			return err
		}
	}
	return lastErr
}
