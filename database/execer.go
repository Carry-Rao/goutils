package database

import (
	"database/sql"
	"fmt"
)

// DbTxRunner runs work in a real transaction over a *sql.DB.
type DbTxRunner struct{ DB *sql.DB }

func (r DbTxRunner) Run(fn func(Execer) error) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return Normalize(err)
	}
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("%w (rollback failed: %v)", Normalize(err), rbErr)
		}
		return Normalize(err)
	}
	return Normalize(tx.Commit())
}

// TxTxRunner reuses an already-open transaction: the work joins it rather than
// opening a nested one.
type TxTxRunner struct{ Exec Execer }

// Run normalises the result so a duplicate detected inside an enclosing
// transaction still reaches the caller as ErrDuplicate once the outermost Run
// rolls it back.
func (r TxTxRunner) Run(fn func(Execer) error) error { return Normalize(fn(r.Exec)) }

// sqlNullString scans a nullable text column into a plain string.
type sqlNullString struct {
	String string
}

func (s *sqlNullString) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		s.String = ""
	case string:
		s.String = v
	case []byte:
		s.String = string(v)
	default:
		s.String = fmt.Sprintf("%v", v)
	}
	return nil
}
