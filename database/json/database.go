// Package json provides a file-backed database.Database. One file holds the
// whole database, so it suits tests, local tools and small datasets rather than
// concurrent writers or anything large.
//
// Because there is no query planner, every predicate that is not a primary-key
// equality is a linear scan in Go. Prefer SQL when the data does not fit in
// memory.
package json

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/Carry-Rao/goutils/database"
)

// Database is a JSON file holding one or more tables.
type Database[T any] struct {
	store  *store
	schema *database.Schema
}

// NewDatabase opens filename, creating it if absent. The filename key is
// required.
func NewDatabase[T any](cfg map[string]string) (*Database[T], error) {
	filename := cfg["filename"]
	if filename == "" {
		return nil, fmt.Errorf("json: filename is required")
	}

	schema := database.SchemaOf[T]()
	if _, err := schema.PrimaryKey(); err != nil {
		return nil, err
	}

	s, err := newStore(filename)
	if err != nil {
		return nil, err
	}
	return &Database[T]{store: s, schema: schema}, nil
}

// GetTable returns a table, creating it in the file when it is not there yet.
// This mirrors the SQL backends, which run CREATE TABLE IF NOT EXISTS.
func (d *Database[T]) GetTable(tableName string) (database.Table[T], error) {
	pk, err := d.schema.PrimaryKey()
	if err != nil {
		return nil, err
	}
	return &Table[T]{store: d.store, name: tableName, schema: d.schema, pk: pk}, nil
}

// DeleteTable drops a table and its rows.
func (d *Database[T]) DeleteTable(tableName string) error {
	return d.store.update(func(f file) error {
		delete(f, tableName)
		return nil
	})
}

// Tables lists the table names in the file.
func (d *Database[T]) Tables() []string {
	var names []string
	d.store.read(func(f file) { names = f.tables() })
	return names
}

// Query returns every row in the file, across all tables, as T. The SQL text and
// arguments are ignored: there is no dialect here, so a caller wanting a subset
// should use Table.Get with conditions instead.
func (d *Database[T]) Query(_ string, _ ...any) ([]T, error) {
	var out []T
	var failure error

	d.store.read(func(f file) {
		for _, name := range f.tables() {
			stored := rowsOf(f, name)
			for _, key := range sortedKeys(stored) {
				decoded, _, err := d.decodeRow(stored[key])
				if err != nil {
					failure = err
					return
				}
				out = append(out, decoded)
			}
		}
	})
	if failure != nil {
		return nil, failure
	}
	return out, nil
}

// Exec replaces one table's rows with the given JSON: the first argument names
// the table, the second must be a JSON array of objects. A payload that is not
// an array is rejected rather than silently clearing the table.
func (d *Database[T]) Exec(tableName string, args ...any) error {
	return execReplace(d.store, d.schema, tableName, args...)
}

// Tx buffers every write and commits only if fn returns nil. Each statement runs
// the ordinary write path against a draft document, and the file is rewritten
// once at the end, so a failed transaction leaves it byte-for-byte unchanged.
func (d *Database[T]) Tx(fn func(database.Tx[T]) error) error {
	return d.store.update(func(draft file) error {
		scoped := &store{path: d.store.path, data: draft, transient: true}
		return fn(&txContext[T]{
			store:  scoped,
			schema: d.schema,
			parent: d.store,
		})
	})
}

// Close is a no-op: every write is already flushed before it returns.
func (d *Database[T]) Close() error { return nil }

// decodeRow materialises one stored row.
func (d *Database[T]) decodeRow(raw json.RawMessage) (T, reflect.Value, error) {
	ptr := reflect.New(d.schema.Type)
	if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
		var zero T
		return zero, reflect.Value{}, fmt.Errorf("json: decode: %w", err)
	}
	out, err := toT[T](ptr.Elem())
	if err != nil {
		var zero T
		return zero, reflect.Value{}, err
	}
	return out, ptr.Elem(), nil
}

// execReplace is the shared implementation behind Database.Exec and
// txContext.Exec. Only which store it writes to differs.
func execReplace(s *store, schema *database.Schema, tableName string, args ...any) error {
	if tableName == "" {
		return fmt.Errorf("json: Exec needs a table name")
	}
	if len(args) != 1 {
		return fmt.Errorf("json: Exec takes a table name and exactly one payload")
	}

	payload, err := asPayload(args[0])
	if err != nil {
		return err
	}

	var rows []json.RawMessage
	if err := json.Unmarshal(payload, &rows); err != nil {
		return fmt.Errorf("json: Exec payload must be a JSON array of rows: %w", err)
	}

	// Decode into the model before writing anything, so a malformed row cannot
	// leave a half-replaced table behind.
	pk, err := schema.PrimaryKey()
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(rows))
	for i, raw := range rows {
		ptr := reflect.New(schema.Type)
		if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
			return fmt.Errorf("json: Exec row %d: %w", i, err)
		}
		keys = append(keys, database.KeyString(ptr.Elem().Field(pk.Index)))
	}

	return s.update(func(f file) error {
		replacement := make(map[string]json.RawMessage, len(rows))
		for i, raw := range rows {
			replacement[keys[i]] = raw
		}
		f[tableName] = replacement
		return nil
	})
}

func asPayload(v any) ([]byte, error) {
	switch p := v.(type) {
	case []byte:
		return p, nil
	case string:
		return []byte(p), nil
	case json.RawMessage:
		return p, nil
	default:
		return nil, fmt.Errorf("json: Exec payload must be []byte or string, got %T", v)
	}
}

// txContext serves tables and raw statements inside a transaction. Its store
// writes into a draft document rather than the file.
type txContext[T any] struct {
	store  *store
	schema *database.Schema
	parent *store
}

func (c *txContext[T]) GetTable(tableName string) (database.Table[T], error) {
	pk, err := c.schema.PrimaryKey()
	if err != nil {
		return nil, err
	}
	return &Table[T]{store: c.store, name: tableName, schema: c.schema, pk: pk}, nil
}

// Query reads through to the committed file. Rows written earlier in this
// transaction are invisible here, which is the usual transactional isolation and
// keeps the two code paths from disagreeing.
func (c *txContext[T]) Query(sqlText string, args ...any) ([]T, error) {
	db := &Database[T]{store: c.parent, schema: c.schema}
	return db.Query(sqlText, args...)
}

func (c *txContext[T]) Exec(tableName string, args ...any) error {
	return execReplace(c.store, c.schema, tableName, args...)
}
