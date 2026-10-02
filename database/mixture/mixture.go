// Package mixture chains several stores into one lookup path, falling through
// to the next layer when a layer misses or fails.
//
// The layers are cache.Table, so a chain is Filter → Cache → Database over
// whichever stores each layer was built from.
package mixture

import (
	"errors"

	"github.com/Carry-Rao/goutils/database/cache"
)

// ErrAction decides what happens when a layer fails.
type ErrAction int

const (
	// Continue falls through to the next layer.
	Continue ErrAction = iota
	// Return aborts the chain and surfaces the error.
	Return
)

// Database is a chain of cache tables.
type Database[T any] struct {
	layers []*Layer[T]
}

// Layer pairs one table with the action to take when it errors.
type Layer[T any] struct {
	Table cache.Table[T]
	Act   ErrAction
}

// New creates an empty chain.
func New[T any]() *Database[T] { return &Database[T]{} }

// NewDatabase satisfies the shape the other backends use; the chain is empty
// until layers are added.
func NewDatabase[T any](_ map[string]string) (*Database[T], error) {
	return New[T](), nil
}

// Close closes every layer and reports every failure.
func (m *Database[T]) Close() error {
	var errs []error
	for _, l := range m.layers {
		if c, ok := l.Table.(interface{ Close() error }); ok {
			if err := c.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// Add appends a layer to the chain.
func (m *Database[T]) Add(tbl cache.Table[T], act ErrAction) {
	m.layers = append(m.layers, &Layer[T]{Table: tbl, Act: act})
}

// Table returns the chain as a single cache.Table.
func (m *Database[T]) Table() cache.Table[T] {
	return &Table[T]{layers: m.layers}
}

// Table is the chain seen as one cache table.
type Table[T any] struct {
	layers []*Layer[T]
}

var _ cache.Table[int] = (*Table[int])(nil)

// Ins writes to every layer. A cache-aside chain needs the row in the filter,
// the cache and the store alike, so a layer accepting the write does not stop
// the others.
func (t *Table[T]) Ins(val T) error {
	return t.forEach(func(l *Layer[T]) error { return l.Table.Ins(val) })
}

func (t *Table[T]) Set(old, updated T) error {
	return t.forEach(func(l *Layer[T]) error { return l.Table.Set(old, updated) })
}

func (t *Table[T]) Del(val T) error {
	return t.forEach(func(l *Layer[T]) error { return l.Table.Del(val) })
}

// Get returns the first non-empty result. A layer that misses without an error
// is not a hit, so the chain keeps going.
func (t *Table[T]) Get(val T) ([]T, error) {
	var lastErr error
	for _, l := range t.layers {
		res, err := l.Table.Get(val)
		if err != nil {
			lastErr = err
			if l.Act == Return {
				return nil, err
			}
			continue
		}
		if len(res) > 0 {
			return res, nil
		}
	}
	return nil, lastErr
}

// forEach applies a write to every layer, honouring Return and reporting all
// failures.
func (t *Table[T]) forEach(op func(*Layer[T]) error) error {
	var errs []error
	for _, l := range t.layers {
		if err := op(l); err != nil {
			errs = append(errs, err)
			if l.Act == Return {
				break
			}
		}
	}
	return errors.Join(errs...)
}
