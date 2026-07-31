package memory

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/Carry-Rao/goutils/database/api"
)

type Table struct {
	db        *Database
	tableName string
	cacheKey  string
	keyPrefix string
	pkField   api.FieldInfo
	hasPK     bool
	schema    *api.CachedSchema
	mu        sync.RWMutex
}

func (t *Table) buildKey(val reflect.Value) string {
	fv := val.Field(t.pkField.Index)
	switch t.pkField.GoKind {
	case reflect.String:
		return t.keyPrefix + fv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return t.keyPrefix + strconv.FormatInt(fv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return t.keyPrefix + strconv.FormatUint(fv.Uint(), 10)
	default:
		return t.keyPrefix + fmt.Sprintf("%v", fv.Interface())
	}
}

func newEntry(data any, ttl time.Duration) *entry {
	if ttl <= 0 {
		return &entry{data: data}
	}
	return &entry{data: data, expires: time.Now().Add(ttl)}
}

func (t *Table) Ins(example any, ttl time.Duration) error {
	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	if !t.hasPK {
		return fmt.Errorf("primary key field %q not found", t.cacheKey)
	}

	key := t.buildKey(val)

	t.mu.Lock()
	defer t.mu.Unlock()
	t.db.data[t.tableName][key] = newEntry(example, ttl)
	return nil
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) ([]any, error) {
	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}

	if !t.hasPK {
		return nil, fmt.Errorf("primary key field %q not found", t.cacheKey)
	}

	key := t.buildKey(val)

	t.mu.RLock()
	defer t.mu.RUnlock()

	raw, ok := t.db.data[t.tableName][key]
	if !ok {
		return nil, nil
	}

	ent, ok := raw.(*entry)
	if !ok {
		return nil, fmt.Errorf("invalid entry type")
	}

	if !ent.expires.IsZero() && time.Now().After(ent.expires) {
		delete(t.db.data[t.tableName], key)
		return nil, nil
	}

	return []any{ent.data}, nil
}

func (t *Table) Set(example any, whereFields []string, ttl time.Duration) error {
	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	if !t.hasPK {
		return fmt.Errorf("primary key field %q not found", t.cacheKey)
	}

	key := t.buildKey(val)

	t.mu.Lock()
	defer t.mu.Unlock()
	t.db.data[t.tableName][key] = newEntry(example, ttl)
	return nil
}

func (t *Table) Del(example any, whereFields []string, _ time.Duration) error {
	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	if !t.hasPK {
		return fmt.Errorf("primary key field %q not found", t.cacheKey)
	}

	key := t.buildKey(val)

	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.db.data[t.tableName], key)
	return nil
}
