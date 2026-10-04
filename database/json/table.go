package json

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/Carry-Rao/goutils/database"
)

func init() { database.Register("json") }

// Table is one table inside the JSON file. Rows are addressed by primary key,
// so a key lookup costs no scan; anything else walks the table in Go.
type Table[T any] struct {
	store  *store
	name   string
	schema *database.Schema
	pk     database.FieldInfo
}

// toT converts a decoded struct value into T, whether T is the struct or a
// pointer to it. The database package keeps its own copy private, so the
// conversion is repeated here rather than widening that package's API.
func toT[T any](v reflect.Value) (T, error) {
	var zero T
	want := reflect.TypeOf((*T)(nil)).Elem()

	if v.Type() == want {
		out, ok := v.Interface().(T)
		if !ok {
			return zero, &database.ScanTypeError{Got: v.Type()}
		}
		return out, nil
	}
	if v.CanAddr() && v.Addr().Type() == want {
		out, ok := v.Addr().Interface().(T)
		if !ok {
			return zero, &database.ScanTypeError{Got: v.Type()}
		}
		return out, nil
	}
	return zero, &database.ScanTypeError{Got: want}
}

// decode turns stored bytes into T plus the addressable struct behind it.
func (t *Table[T]) decode(raw json.RawMessage) (T, reflect.Value, error) {
	var zero T
	ptr := reflect.New(t.schema.Type)
	if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
		return zero, reflect.Value{}, fmt.Errorf("json: decode %s: %w", t.name, err)
	}
	out, err := toT[T](ptr.Elem())
	return out, ptr.Elem(), err
}

// keyOf renders a row's primary key as the object key it is stored under.
func (t *Table[T]) keyOf(row reflect.Value) string {
	return database.KeyString(row.Field(t.pk.Index))
}

func (t *Table[T]) encode(row reflect.Value) (json.RawMessage, error) {
	raw, err := json.Marshal(row.Interface())
	if err != nil {
		return nil, fmt.Errorf("json: encode: %w", err)
	}
	return raw, nil
}

// rows returns a live table's rows, creating the table when absent. It is only
// called inside store.update, where the document is a private draft.
func rows(f file, name string) map[string]json.RawMessage {
	if f[name] == nil {
		f[name] = make(map[string]json.RawMessage)
	}
	return f[name]
}

// rowsOf returns a table's rows, or an empty set when it does not exist. Used on
// the read path, where nothing should be created.
func rowsOf(f file, name string) map[string]json.RawMessage {
	if r, ok := f[name]; ok {
		return r
	}
	return map[string]json.RawMessage{}
}

// Ins writes one row, replacing any row already stored under the same key.
func (t *Table[T]) Ins(val T, opts database.Options) error {
	if err := opts.Validate(t.schema); err != nil {
		return err
	}
	row, err := database.Unwrap(val)
	if err != nil {
		return err
	}
	raw, err := t.encode(row)
	if err != nil {
		return err
	}
	key := t.keyOf(row)

	return t.store.update(func(f file) error {
		rows(f, t.name)[key] = raw
		return nil
	})
}

// Get reads the matching rows. An empty opts.Where returns the whole table,
// matching the SQL backends.
func (t *Table[T]) Get(val T, opts database.Options) ([]T, error) {
	if err := opts.Validate(t.schema); err != nil {
		return nil, err
	}
	if _, err := database.Unwrap(val); err != nil {
		return nil, err
	}

	var out []T
	var failure error

	t.store.read(func(f file) {
		existing := rowsOf(f, t.name)

		// A lone equality test on the primary key is answered from the index.
		if keys, ok := t.keyLookup(opts); ok {
			for _, k := range keys {
				raw, hit := existing[k]
				if !hit {
					continue
				}
				decoded, _, err := t.decode(raw)
				if err != nil {
					failure = err
					return
				}
				out = append(out, decoded)
			}
			return
		}

		for _, k := range sortedKeys(existing) {
			decoded, row, err := t.decode(existing[k])
			if err != nil {
				failure = err
				return
			}
			match, err := t.matches(row, opts)
			if err != nil {
				failure = err
				return
			}
			if match {
				out = append(out, decoded)
			}
		}
	})

	if failure != nil {
		return nil, failure
	}
	if err := t.sort(out, opts); err != nil {
		return nil, err
	}
	return page(out, opts), nil
}

// keyLookup recognises a read that the primary-key index can answer directly.
// Anything else needs the row in hand to evaluate the condition.
func (t *Table[T]) keyLookup(opts database.Options) ([]string, bool) {
	if len(opts.Where) != 1 || len(opts.Order) > 0 || opts.Limit > 0 || opts.Offset > 0 {
		return nil, false
	}
	c := opts.Where[0]
	if c.Op != database.OpEq {
		return nil, false
	}
	f, ok := t.schema.Lookup(c.Column)
	if !ok || f.Index != t.pk.Index {
		return nil, false
	}
	return []string{database.KeyString(reflect.ValueOf(c.Value))}, true
}

// matches reports whether a decoded row satisfies every condition.
func (t *Table[T]) matches(row reflect.Value, opts database.Options) (bool, error) {
	for _, c := range opts.Where {
		f, ok := t.schema.Lookup(c.Column)
		if !ok {
			return false, fmt.Errorf("json: unknown field %q", c.Column)
		}
		hit, err := evalCondition(row.Field(f.Index), c)
		if err != nil {
			return false, err
		}
		if !hit {
			return false, nil
		}
	}
	return true, nil
}

// Set updates the rows matching opts.Where, or old's primary key when no
// condition is given.
//
// An empty opts.Values assigns the whole record from updated. Once opts.Values
// is given it becomes the complete specification, and a column left out of it is
// untouched. opts.Skip always wins. The rule matches the SQL backends exactly.
func (t *Table[T]) Set(old, updated T, opts database.Options) error {
	if err := opts.Validate(t.schema); err != nil {
		return err
	}
	oldRow, err := database.Unwrap(old)
	if err != nil {
		return err
	}
	newRow, err := database.Unwrap(updated)
	if err != nil {
		return err
	}

	conditions := opts.Where
	if len(conditions) == 0 {
		conditions = []database.Condition{{
			Column: t.pk.ColumnName,
			Op:     database.OpEq,
			Value:  oldRow.Field(t.pk.Index).Interface(),
		}}
	}

	return t.store.update(func(f file) error {
		existing := rows(f, t.name)
		targeted := len(opts.Values) > 0

		for _, key := range sortedKeys(existing) {
			_, row, err := t.decode(existing[key])
			if err != nil {
				return err
			}
			match, err := t.matches(row, database.Options{Where: conditions})
			if err != nil {
				return err
			}
			if !match {
				continue
			}

			if err := t.applyRow(row, newRow, opts, targeted); err != nil {
				return err
			}
			raw, err := t.encode(row)
			if err != nil {
				return err
			}
			existing[key] = raw
		}
		return nil
	})
}

// applyRow folds updated into an existing row according to opts.
func (t *Table[T]) applyRow(row, newRow reflect.Value, opts database.Options, targeted bool) error {
	for _, f := range t.schema.Fields {
		if opts.SkipSet(t.schema, f.ColumnName) {
			continue
		}

		if targeted {
			v, named := opts.ValueFor(t.schema, f.ColumnName)
			if !named {
				continue
			}
			if err := assign(row, newRow, f, v); err != nil {
				return err
			}
			continue
		}

		// Untargeted: every field takes updated's value verbatim.
		row.Field(f.Index).Set(newRow.Field(f.Index))
	}
	return nil
}

// assign applies one named Value to one field of the stored row.
func assign(row, newRow reflect.Value, f database.FieldInfo, v database.Value) error {
	target := row.Field(f.Index)

	if !f.IsChild {
		// Only Assign and Clear apply to a scalar; validation rejects the rest.
		if v.Op == database.OpClear {
			target.Set(reflect.Zero(target.Type()))
			return nil
		}
		target.Set(newRow.Field(f.Index))
		return nil
	}

	switch v.Op {
	case database.OpClear:
		target.Set(reflect.Zero(target.Type()))

	case database.OpUnion:
		if err := mergeValues(target, reflect.ValueOf(v.Value)); err != nil {
			return err
		}

	case database.OpDifference:
		current := target
		current = copyValue(current)
		if err := removeValues(current, reflect.ValueOf(v.Value)); err != nil {
			return err
		}
		target.Set(current)

	default: // Assign
		target.Set(reflect.Zero(target.Type()))
		if err := mergeValues(target, reflect.ValueOf(v.Value)); err != nil {
			return err
		}
	}
	return nil
}

// Del removes the matching rows, or the row at val's primary key when no
// condition is given.
func (t *Table[T]) Del(val T, opts database.Options) error {
	if err := opts.Validate(t.schema); err != nil {
		return err
	}
	probe, err := database.Unwrap(val)
	if err != nil {
		return err
	}

	conditions := opts.Where
	if len(conditions) == 0 {
		key := t.keyOf(probe)
		return t.store.update(func(f file) error {
			delete(rows(f, t.name), key)
			return nil
		})
	}

	return t.store.update(func(f file) error {
		existing := rows(f, t.name)
		for _, key := range sortedKeys(existing) {
			_, row, err := t.decode(existing[key])
			if err != nil {
				return err
			}
			match, err := t.matches(row, opts)
			if err != nil {
				return err
			}
			if match {
				delete(existing, key)
			}
		}
		return nil
	})
}

// sort applies opts.Order in place, falling back to the primary key so paging
// over an unordered table stays stable.
func (t *Table[T]) sort(rows []T, opts database.Options) error {
	if len(rows) < 2 {
		return nil
	}
	keys := opts.Order
	if len(keys) == 0 {
		keys = []database.Order{{Column: t.pk.ColumnName}}
	}

	values := make([]reflect.Value, len(rows))
	for i, r := range rows {
		v, err := database.Unwrap(r)
		if err != nil {
			return err
		}
		values[i] = v
	}

	order := make([]int, len(rows))
	for i := range order {
		order[i] = i
	}

	sort.SliceStable(order, func(a, b int) bool {
		for _, k := range keys {
			f, ok := t.schema.Lookup(k.Column)
			if !ok {
				continue
			}
			cmp, ok := compare(values[order[a]].Field(f.Index), values[order[b]].Field(f.Index).Interface())
			if !ok || cmp == 0 {
				continue
			}
			if k.Desc {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})

	out := make([]T, len(rows))
	for i, idx := range order {
		out[i] = rows[idx]
	}
	copy(rows, out)
	return nil
}

// page trims rows to Limit and Offset.
func page[T any](rows []T, opts database.Options) []T {
	if opts.Offset > 0 {
		if opts.Offset >= len(rows) {
			return []T{}
		}
		rows = rows[opts.Offset:]
	}
	if opts.Limit > 0 && opts.Limit < len(rows) {
		rows = rows[:opts.Limit]
	}
	return rows
}

func sortedKeys(rows map[string]json.RawMessage) []string {
	out := make([]string, 0, len(rows))
	for k := range rows {
		out = append(out, k)
	}
	// Numeric keys sort numerically, so 2 precedes 10 rather than following it.
	sort.Slice(out, func(i, j int) bool {
		a, aok := strconv.ParseFloat(out[i], 64)
		b, bok := strconv.ParseFloat(out[j], 64)
		if aok == nil && bok == nil {
			return a < b
		}
		return out[i] < out[j]
	})
	return out
}
