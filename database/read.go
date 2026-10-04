package database

import (
	"reflect"
	"strings"
)

// buildWhere renders the WHERE clause from opts.Where, starting the bind
// counter at paramStart.
func (t *SQLTable[T]) buildWhere(opts Options, paramStart int) (string, []any) {
	if len(opts.Where) == 0 {
		return "", nil
	}
	pk, _ := t.schema.PrimaryKey()

	var parts []string
	var args []any
	idx := paramStart

	for _, c := range opts.Where {
		f, _ := t.schema.Lookup(c.Column)
		if f.IsChild {
			clause, cargs := t.childCondition(f, pk, c, idx)
			parts = append(parts, clause)
			args = append(args, cargs...)
			idx += len(cargs)
			continue
		}

		col := t.dialect.Quote(f.ColumnName)
		switch c.Op {
		case OpIsNull:
			parts = append(parts, col+" IS NULL")
		case OpIsNotNull:
			parts = append(parts, col+" IS NOT NULL")
		case OpIn, OpNotIn:
			if len(c.Values) == 0 {
				continue
			}
			holes := make([]string, 0, len(c.Values))
			for _, v := range c.Values {
				holes = append(holes, t.dialect.placeholder(idx))
				args = append(args, v)
				idx++
			}
			word := " IN ("
			if c.Op == OpNotIn {
				word = " NOT IN ("
			}
			parts = append(parts, col+word+strings.Join(holes, ",")+")")
		case OpBetween:
			lo := t.dialect.placeholder(idx)
			hi := t.dialect.placeholder(idx + 1)
			parts = append(parts, col+" BETWEEN "+lo+" AND "+hi)
			args = append(args, c.Value, c.Values[0])
			idx += 2
		default:
			parts = append(parts, col+sqlOp(c.Op)+t.dialect.placeholder(idx))
			args = append(args, c.Value)
			idx++
		}
	}
	return strings.Join(parts, " AND "), args
}

func sqlOp(op ConditionOp) string {
	switch op {
	case OpEq:
		return "="
	case OpNeq:
		return "!="
	case OpGt:
		return ">"
	case OpGte:
		return ">="
	case OpLt:
		return "<"
	case OpLte:
		return "<="
	case OpLike:
		return " LIKE "
	case OpNotLike:
		return " NOT LIKE "
	}
	return "="
}

// childCondition renders a collection predicate as a correlated EXISTS against
// the parent row.
func (t *SQLTable[T]) childCondition(f FieldInfo, pk FieldInfo, c Condition, paramStart int) (string, []any) {
	tbl := t.dialect.QuoteTable(t.childTable(f))
	parent := t.dialect.Quote(t.parentColumn(pk))
	link := "c." + parent + " = " + t.dialect.QuoteTable(t.table) + "." + t.dialect.Quote(pk.ColumnName)

	wrap := func(clause string, args ...any) (string, []any) {
		out := "EXISTS (SELECT 1 FROM " + tbl + " c WHERE " + link + " AND " + clause + ")"
		return out, args
	}

	valCol := t.dialect.Quote(f.ColumnName)
	keyCol := t.dialect.Quote("k")
	vCol := t.dialect.Quote("v")

	switch c.Op {
	case OpContainsKV:
		return wrap(keyCol+" = "+t.dialect.placeholder(paramStart)+
			" AND "+vCol+" = "+t.dialect.placeholder(paramStart+1), c.Key, c.Value)

	case OpContains:
		if f.GoKind == reflect.Map {
			return wrap(keyCol+" = "+t.dialect.placeholder(paramStart), c.Value)
		}
		return wrap(valCol+" = "+t.dialect.placeholder(paramStart), c.Value)

	case OpContainsAll:
		// Every element must be present, and no single row can hold them all,
		// so this expands to one EXISTS per element ANDed together.
		var parts []string
		var args []any
		for i, v := range c.Values {
			clause, cargs := wrap(valCol+" = "+t.dialect.placeholder(paramStart+i), v)
			parts = append(parts, clause)
			args = append(args, cargs...)
		}
		return strings.Join(parts, " AND "), args

	case OpContainsAny:
		// One row holding any of them is enough.
		parts := make([]string, 0, len(c.Values))
		args := make([]any, 0, len(c.Values))
		for i, v := range c.Values {
			parts = append(parts, valCol+" = "+t.dialect.placeholder(paramStart+i))
			args = append(args, v)
		}
		if len(parts) == 0 {
			return "", nil
		}
		return wrap("("+strings.Join(parts, " OR ")+")", args...)
	}
	return "", nil
}

// Get reads the matching rows and fills in their collections.
func (t *SQLTable[T]) Get(val T, opts Options) ([]T, error) {
	if err := opts.Validate(t.schema); err != nil {
		return nil, err
	}
	if err := t.assertStruct(val); err != nil {
		return nil, err
	}

	where, args := t.buildWhere(opts, 1)
	query := "SELECT * FROM " + t.dialect.QuoteTable(t.table)
	if where != "" {
		query += " WHERE " + where
	}
	if clause, ok := opts.orderBy(t.schema, t.dialect); ok {
		query += clause
	}
	query += opts.limitClause(t.dialect)

	rows, err := t.exec.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out, err := ScanRows[T](rows, t.schema)
	if err != nil {
		return nil, err
	}
	return t.hydrate(out)
}

// assertStruct checks a value is usable as a model.
func (t *SQLTable[T]) assertStruct(val T) error {
	if _, err := Unwrap(val); err != nil {
		return err
	}
	return nil
}

// hydrate fills in every collection field with one query per field.
//
// Scanned rows are copies when T is a value type, so each row is rebuilt into
// an addressable value before its collections are assigned.
func (t *SQLTable[T]) hydrate(rows []T) ([]T, error) {
	if len(rows) == 0 || len(t.schema.Collections) == 0 {
		return rows, nil
	}
	pk, err := t.schema.PrimaryKey()
	if err != nil {
		return nil, err
	}

	// Deduplicate parent keys so the IN list stays small.
	var keys []any
	seen := make(map[string]bool, len(rows))
	for _, r := range rows {
		v, err := Unwrap(r)
		if err != nil {
			return nil, err
		}
		k := KeyString(v.Field(pk.Index))
		if !seen[k] {
			seen[k] = true
			keys = append(keys, v.Field(pk.Index).Interface())
		}
	}

	loaded := make(map[string]map[string][][2]string, len(rows))
	for _, f := range t.schema.Collections {
		pairs, err := t.loadChild(f, t.parentColumn(pk), keys)
		if err != nil {
			return nil, err
		}
		for key, p := range pairs {
			if loaded[key] == nil {
				loaded[key] = make(map[string][][2]string, len(t.schema.Collections))
			}
			loaded[key][f.ColumnName] = p
		}
	}

	out := make([]T, 0, len(rows))
	for _, r := range rows {
		v, err := Unwrap(r)
		if err != nil {
			return nil, err
		}

		nv := reflect.New(t.schema.Type).Elem()
		nv.Set(v)

		byField := loaded[KeyString(v.Field(pk.Index))]
		for _, f := range t.schema.Collections {
			assignChild(nv.Field(f.Index), byField[f.ColumnName], f)
		}

		row, ok := cellToT[T](nv.Addr())
		if !ok {
			return nil, &ScanTypeError{Got: t.schema.Type}
		}
		out = append(out, row)
	}
	return out, nil
}

// loadChild reads one collection field for the given parent keys.
func (t *SQLTable[T]) loadChild(f FieldInfo, parentCol string, keys []any) (map[string][][2]string, error) {
	tbl := t.dialect.QuoteTable(t.childTable(f))
	parent := t.dialect.Quote(parentCol)

	holes := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys))
	for i, k := range keys {
		holes = append(holes, t.dialect.placeholder(i+1))
		args = append(args, k)
	}

	sel := parent
	if f.GoKind == reflect.Map {
		sel += ", " + t.dialect.Quote("k") + ", " + t.dialect.Quote("v")
	} else {
		sel += ", " + t.dialect.Quote(f.ColumnName)
	}

	q := "SELECT " + sel + " FROM " + tbl + " WHERE " + parent + " IN (" + strings.Join(holes, ",") + ")"
	rows, err := t.exec.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][][2]string, len(keys))
	for rows.Next() {
		var parent any
		var k, v sqlNullString
		if f.GoKind == reflect.Map {
			if err := rows.Scan(&parent, &k, &v); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&parent, &v); err != nil {
				return nil, err
			}
		}
		out[KeyString(reflect.ValueOf(parent))] = append(out[KeyString(reflect.ValueOf(parent))], [2]string{k.String, v.String})
	}
	return out, rows.Err()
}

// assignChild fills a collection field from its loaded pairs.
func assignChild(fv reflect.Value, pairs [][2]string, f FieldInfo) {
	if f.GoKind == reflect.Map {
		m := reflect.MakeMap(fv.Type())
		for _, p := range pairs {
			m.SetMapIndex(reflect.ValueOf(p[0]), reflect.ValueOf(p[1]))
		}
		fv.Set(m)
		return
	}
	s := reflect.MakeSlice(fv.Type(), 0, len(pairs))
	for _, p := range pairs {
		s = reflect.Append(s, reflect.ValueOf(p[1]).Convert(f.GoElem))
	}
	fv.Set(s)
}
