package database

import (
	"errors"
	"sync"
)

// ErrDuplicate reports that a write violated a unique constraint. Backends
// speak different dialects for the same condition — MySQL error 1062,
// PostgreSQL SQLSTATE 23505, SQLite's constraint code, and nothing at all for
// the JSON file — so each one registers a detector through Classify and this
// package turns the match into one sentinel callers can test portably:
//
//	_, err := tbl.Ins(val, database.Options{})
//	if errors.Is(err, database.ErrDuplicate) {
//		// unique violation, not an outage
//	}
var ErrDuplicate = errors.New("database: duplicate key")

// duplicateDetectors holds the backend-supplied predicates, registered from each
// backend's init. Keeping them here rather than importing the drivers directly
// is what lets the core package stay driver-free: the sql/mysql, sql/postgresql
// and sql/sqlite packages own their own driver types.
var duplicateDetectors struct {
	mu  sync.RWMutex
	fns []func(error) bool
}

// Classify registers a predicate that recognises one backend's
// duplicate-key error. Backends call it from init.
//
// The first registered predicate that matches wins, so a later backend cannot
// reclassify an error an earlier one already claimed.
func Classify(fn func(error) bool) {
	if fn == nil {
		return
	}
	duplicateDetectors.mu.Lock()
	defer duplicateDetectors.mu.Unlock()
	duplicateDetectors.fns = append(duplicateDetectors.fns, fn)
}

// Classify registers a predicate that recognises one backend's
// duplicate-key error. Backends call it from init.
//
// The first registered predicate that matches wins, so a later backend cannot
// reclassify an error an earlier one already claimed.
// Normalize maps a backend's own duplicate-key error onto ErrDuplicate and
// returns every other error unchanged. Backends call this on the way out.
func Normalize(err error) error {
	if err == nil {
		return nil
	}
	// A caller may pass something that already carries the sentinel, either
	// because it came back through a transaction or because it was normalised
	// at an inner layer. Either way, stop here.
	if errors.Is(err, ErrDuplicate) {
		return err
	}

	duplicateDetectors.mu.RLock()
	fns := duplicateDetectors.fns
	duplicateDetectors.mu.RUnlock()

	for _, fn := range fns {
		if fn(err) {
			return errors.Join(ErrDuplicate, err)
		}
	}
	return err
}
