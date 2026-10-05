package database

import (
	"reflect"
	"sort"
	"strings"
)

// Ins writes one row and its collection rows atomically.
func (t *SQLTable[T]) Ins(val T, opts Options) error {
	v, err := Unwrap(val)
	if err != nil {
		return err
	}
	if err := opts.Validate(t.schema); err != nil {
		return err
	}
	pk, err := t.schema.PrimaryKey()
	if err != nil {
		return err
	}

	return t.runner.Run(func(ex Execer) error {
		if _, err := ex.Exec(t.insert, t.schema.InsertValues(v)...); err != nil {
			return err
		}
		for _, f := range t.schema.Collections {
			if err := t.replaceChild(ex, f, pk, v.Field(pk.Index).Interface(), childPairs(v.Field(f.Index).Interface())); err != nil {
				return err
			}
		}
		return nil
	})
}

// Set updates the rows matching opts.Where, or old's primary key when no
// condition is given.
//
// An empty opts.Values assigns every scalar and every collection from updated.
// Once opts.Values is given it becomes the complete specification: only the
// columns it names are written, and a collection named there uses that operation
// rather than updated's value. opts.Skip always wins.
func (t *SQLTable[T]) Set(old, updated T, opts Options) error {
	oldVal, err := Unwrap(old)
	if err != nil {
		return err
	}
	newVal, err := Unwrap(updated)
	if err != nil {
		return err
	}
	if err := opts.Validate(t.schema); err != nil {
		return err
	}
	pk, err := t.schema.PrimaryKey()
	if err != nil {
		return err
	}

	conditions := opts.Where
	if len(conditions) == 0 {
		conditions = []Condition{Eq(pk.ColumnName, oldVal.Field(pk.Index).Interface())}
	}

	// An empty Values means "assign everything from updated". Once Values is
	// given it becomes the complete specification, so a column left out of it is
	// untouched rather than blanked from updated.
	targeted := len(opts.Values) > 0

	sets := make([]string, 0, len(t.schema.Scalars))
	setArgs := make([]any, 0, len(t.schema.Scalars))
	idx := 1

	for _, f := range t.schema.Scalars {
		if opts.SkipSet(t.schema, f.ColumnName) {
			continue
		}
		if targeted {
			v, ok := opts.ValueFor(t.schema, f.ColumnName)
			if !ok || v.Op != OpAssign {
				continue
			}
		}
		sets = append(sets, t.dialect.Quote(f.ColumnName)+"="+t.dialect.placeholder(idx))
		setArgs = append(setArgs, newVal.Field(f.Index).Interface())
		idx++
	}
	// A Values naming only collection operations leaves no scalar to update,
	// which is legitimate; the UPDATE is simply skipped.
	runUpdate := len(sets) > 0

	where, whereArgs := t.buildWhere(Options{Where: conditions}, idx)
	query := ""
	if runUpdate {
		query = "UPDATE " + t.dialect.QuoteTable(t.table) + " SET " + strings.Join(sets, ",")
		if where != "" {
			query += " WHERE " + where
		}
	}

	pkVal := newVal.Field(pk.Index).Interface()

	return t.runner.Run(func(ex Execer) error {
		if runUpdate {
			if _, err := ex.Exec(query, append(setArgs, whereArgs...)...); err != nil {
				return err
			}
		}
		for _, f := range t.schema.Collections {
			if opts.SkipSet(t.schema, f.ColumnName) {
				continue
			}
			if targeted {
				if _, ok := opts.ValueFor(t.schema, f.ColumnName); !ok {
					continue // not named, so left alone
				}
			}
			if err := t.applyChild(ex, f, pk, pkVal, newVal, opts); err != nil {
				return err
			}
		}
		return nil
	})
}

// applyChild performs the requested operation on one collection field.
func (t *SQLTable[T]) applyChild(ex Execer, f FieldInfo, pk FieldInfo, pkVal any, newVal reflect.Value, opts Options) error {
	op, hasOp := opts.ValueFor(t.schema, f.ColumnName)

	if !hasOp {
		return t.replaceChild(ex, f, pk, pkVal, childPairs(newVal.Field(f.Index).Interface()))
	}

	switch op.Op {
	case OpClear:
		return t.clearChild(ex, f, pk, pkVal)
	case OpUnion:
		current, err := t.readChild(ex, f, pk, pkVal)
		if err != nil {
			return err
		}
		return t.replaceChild(ex, f, pk, pkVal, mergePairs(current, childPairs(op.Value)))
	case OpDifference:
		current, err := t.readChild(ex, f, pk, pkVal)
		if err != nil {
			return err
		}
		return t.replaceChild(ex, f, pk, pkVal, subtractPairs(current, childPairs(op.Value)))
	default:
		return t.replaceChild(ex, f, pk, pkVal, childPairs(op.Value))
	}
}

// Del removes the matching rows along with their collection rows.
func (t *SQLTable[T]) Del(val T, opts Options) error {
	v, err := Unwrap(val)
	if err != nil {
		return err
	}
	if err := opts.Validate(t.schema); err != nil {
		return err
	}
	pk, err := t.schema.PrimaryKey()
	if err != nil {
		return err
	}

	conditions := opts.Where
	if len(conditions) == 0 {
		conditions = []Condition{Eq(pk.ColumnName, v.Field(pk.Index).Interface())}
	}
	where, args := t.buildWhere(Options{Where: conditions}, 1)

	query := "DELETE FROM " + t.dialect.QuoteTable(t.table)
	if where != "" {
		query += " WHERE " + where
	}

	return t.runner.Run(func(ex Execer) error {
		for _, f := range t.schema.Collections {
			if err := t.deleteChildFor(ex, f, pk, conditions); err != nil {
				return err
			}
		}
		_, err := ex.Exec(query, args...)
		return err
	})
}

// deleteChildFor removes the collection rows of whichever parents the conditions
// select. An IN sub-select is used because SQLite rejects DELETE with a table
// alias.
func (t *SQLTable[T]) deleteChildFor(ex Execer, f FieldInfo, pk FieldInfo, conditions []Condition) error {
	where, args := t.buildWhere(Options{Where: conditions}, 1)
	query := "DELETE FROM " + t.dialect.QuoteTable(t.childTable(f))
	if where != "" {
		query += " WHERE " + t.dialect.Quote(t.parentColumn(pk)) +
			" IN (SELECT " + t.dialect.Quote(pk.ColumnName) + " FROM " +
			t.dialect.QuoteTable(t.table) + " WHERE " + where + ")"
	}
	_, err := ex.Exec(query, args...)
	return err
}

// ---------- collection row plumbing ----------

func (t *SQLTable[T]) readChild(ex Execer, f FieldInfo, pk FieldInfo, pkVal any) ([][2]string, error) {
	parent := t.dialect.Quote(t.parentColumn(pk))
	sel := t.dialect.Quote(f.ColumnName)
	if f.GoKind == reflect.Map {
		sel = t.dialect.Quote("k") + ", " + t.dialect.Quote("v")
	}

	rows, err := ex.Query("SELECT "+sel+" FROM "+t.dialect.QuoteTable(t.childTable(f))+
		" WHERE "+parent+"="+t.dialect.placeholder(1), pkVal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out [][2]string
	for rows.Next() {
		var k, v sqlNullString
		if f.GoKind == reflect.Map {
			if err := rows.Scan(&k, &v); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&v); err != nil {
				return nil, err
			}
		}
		out = append(out, [2]string{k.String, v.String})
	}
	return out, rows.Err()
}

func (t *SQLTable[T]) replaceChild(ex Execer, f FieldInfo, pk FieldInfo, pkVal any, pairs [][2]string) error {
	parent := t.dialect.Quote(t.parentColumn(pk))
	tbl := t.dialect.QuoteTable(t.childTable(f))

	if _, err := ex.Exec("DELETE FROM "+tbl+" WHERE "+parent+"="+t.dialect.placeholder(1), pkVal); err != nil {
		return err
	}
	if len(pairs) == 0 {
		return nil
	}

	cols, holes, args := t.childInsert(f, pk, parent, pkVal, pairs)
	// Duplicates are already merged for Union; IGNORE keeps a concurrent
	// double-insert from failing the whole statement.
	//
	// A globally unique collection is the exception. There, a repeat is not a
	// duplicate of this row but a claim another row already owns, and
	// suppressing it would drop the insert without a word — the binding would
	// simply not exist. Let the constraint fire so the caller learns about it.
	ignore, onConflict := t.dialect.Ignore, t.dialect.OnConflict
	if f.IsGlobalUnique {
		ignore, onConflict = t.dialect.PlainInsert(), ""
	}
	query := ignore + "INTO " + tbl + " (" + strings.Join(cols, ",") +
		") VALUES " + strings.Join(holes, ",") + onConflict
	_, err := ex.Exec(query, args...)
	return Normalize(err)
}

func (t *SQLTable[T]) clearChild(ex Execer, f FieldInfo, pk FieldInfo, pkVal any) error {
	_, err := ex.Exec("DELETE FROM "+t.dialect.QuoteTable(t.childTable(f))+
		" WHERE "+t.dialect.Quote(t.parentColumn(pk))+"="+t.dialect.placeholder(1), pkVal)
	return err
}

func (t *SQLTable[T]) childInsert(f FieldInfo, pk FieldInfo, parent string, pkVal any, pairs [][2]string) ([]string, []string, []any) {
	cols := []string{parent}
	if f.GoKind == reflect.Map {
		cols = append([]string{t.dialect.Quote("k"), t.dialect.Quote("v")}, cols...)
	} else {
		cols = append([]string{t.dialect.Quote(f.ColumnName)}, cols...)
	}

	var holes []string
	var args []any
	idx := 1

	for _, p := range pairs {
		if f.GoKind == reflect.Map {
			holes = append(holes, "("+t.dialect.placeholder(idx)+","+t.dialect.placeholder(idx+1)+","+t.dialect.placeholder(idx+2)+")")
			args = append(args, p[0], p[1], pkVal)
			idx += 3
		} else {
			holes = append(holes, "("+t.dialect.placeholder(idx)+","+t.dialect.placeholder(idx+1)+")")
			args = append(args, p[1], pkVal)
			idx += 2
		}
	}
	return cols, holes, args
}

// childPairs flattens a Go collection into stored string pairs.
func childPairs(v any) [][2]string {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([][2]string, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out = append(out, [2]string{"", stringify(rv.Index(i))})
		}
		return out
	case reflect.Map:
		out := make([][2]string, 0, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			out = append(out, [2]string{stringify(iter.Key()), stringify(iter.Value())})
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i][0] != out[j][0] {
				return out[i][0] < out[j][0]
			}
			return out[i][1] < out[j][1]
		})
		return out
	}
	return nil
}

func mergePairs(a, b [][2]string) [][2]string {
	seen := make(map[[2]string]bool, len(a)+len(b))
	out := make([][2]string, 0, len(a)+len(b))
	for _, p := range b {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	for _, p := range a {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

func subtractPairs(a, b [][2]string) [][2]string {
	drop := make(map[[2]string]bool, len(b))
	for _, p := range b {
		drop[p] = true
	}
	out := make([][2]string, 0, len(a))
	for _, p := range a {
		if !drop[p] {
			out = append(out, p)
		}
	}
	return out
}
