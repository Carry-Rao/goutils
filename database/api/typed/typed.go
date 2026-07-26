package typed

import (
	"time"

	"github.com/Carry-Rao/goutils/database/api"
)

type TypedTable[T any] struct {
	inner api.Table
}

func NewTable[T any](inner api.Table) *TypedTable[T] {
	return &TypedTable[T]{inner: inner}
}

func (t *TypedTable[T]) Ins(val T, ttl time.Duration) error {
	return t.inner.Ins(val, ttl)
}

func (t *TypedTable[T]) Get(example T, where []string, ttl time.Duration) ([]T, error) {
	raw, err := t.inner.Get(example, where, ttl)
	if err != nil {
		return nil, err
	}
	out := make([]T, len(raw))
	for i, r := range raw {
		out[i] = r.(T)
	}
	return out, nil
}

func (t *TypedTable[T]) Set(val T, where []string, ttl time.Duration) error {
	return t.inner.Set(val, where, ttl)
}

func (t *TypedTable[T]) Del(val T, where []string, ttl time.Duration) error {
	return t.inner.Del(val, where, ttl)
}
