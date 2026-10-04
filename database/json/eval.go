package json

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Carry-Rao/goutils/database"
)

// evalCondition applies one predicate to an already-decoded row.
//
// The SQL backends compile Conditions into SQL; here they run in Go over each
// row in turn, which is the scan the JSON backend trades away indexing for.
//
// Two semantics differ from SQL and are deliberate:
//
//   - NULL exists only where a Go value can be nil — pointers, interfaces,
//     slices and maps. A non-pointer field decoded from JSON null holds its
//     zero value, so IsNull reports false for it.
//   - Like is case-sensitive and understands the same % and _ wildcards.
func evalCondition(fv reflect.Value, c database.Condition) (bool, error) {
	switch c.Op {
	case database.OpIsNull:
		return isNil(fv), nil
	case database.OpIsNotNull:
		return !isNil(fv), nil
	}

	if fv.Kind() == reflect.Slice || fv.Kind() == reflect.Array || fv.Kind() == reflect.Map {
		return evalCollection(fv, c)
	}

	if isNil(fv) {
		// A NULL operand satisfies only the negated comparisons.
		switch c.Op {
		case database.OpEq:
			return c.Value == nil, nil
		case database.OpNeq:
			return c.Value != nil, nil
		}
		return false, nil
	}

	switch c.Op {
	case database.OpIn:
		return matchAny(fv, c.Values), nil
	case database.OpNotIn:
		return !matchAny(fv, c.Values), nil
	case database.OpBetween:
		if len(c.Values) != 1 {
			return false, fmt.Errorf("json: Between needs two bounds")
		}
		lo, ok1 := compare(fv, c.Value)
		hi, ok2 := compare(fv, c.Values[0])
		return ok1 && ok2 && lo >= 0 && hi <= 0, nil
	case database.OpLike:
		return likeMatch(stringOf(fv), stringOf(reflect.ValueOf(c.Value))), nil
	case database.OpNotLike:
		return !likeMatch(stringOf(fv), stringOf(reflect.ValueOf(c.Value))), nil
	}

	cmp, ok := compare(fv, c.Value)
	if !ok {
		return false, fmt.Errorf("json: cannot compare column %q with %T", c.Column, c.Value)
	}
	switch c.Op {
	case database.OpEq:
		return cmp == 0, nil
	case database.OpNeq:
		return cmp != 0, nil
	case database.OpGt:
		return cmp > 0, nil
	case database.OpGte:
		return cmp >= 0, nil
	case database.OpLt:
		return cmp < 0, nil
	case database.OpLte:
		return cmp <= 0, nil
	}
	return false, fmt.Errorf("json: unsupported operator %d on column %q", c.Op, c.Column)
}

// evalCollection applies a membership predicate to a slice or map. The meaning
// matches the SQL backends: a slice holds the element, a map holds the key.
func evalCollection(fv reflect.Value, c database.Condition) (bool, error) {
	isMap := fv.Kind() == reflect.Map

	switch c.Op {
	case database.OpContainsKV:
		if !isMap {
			return false, fmt.Errorf("json: ContainsKV needs a map column, got %s", c.Column)
		}
		entry := fv.MapIndex(reflect.ValueOf(c.Key))
		if !entry.IsValid() {
			return false, nil
		}
		return stringOf(entry) == stringOf(reflect.ValueOf(c.Value)), nil

	case database.OpContains:
		if isMap {
			return fv.MapIndex(reflect.ValueOf(c.Value)).IsValid(), nil
		}
		for i := 0; i < fv.Len(); i++ {
			if cmp, ok := compare(fv.Index(i), c.Value); ok && cmp == 0 {
				return true, nil
			}
		}
		return false, nil

	case database.OpContainsAll, database.OpContainsAny:
		if isMap {
			return false, fmt.Errorf("json: %s needs a slice column, got %s", opName(c.Op), c.Column)
		}
		if len(c.Values) == 0 {
			return false, nil
		}
		// Every element must be present for ContainsAll; one is enough for
		// ContainsAny.
		for _, want := range c.Values {
			found := false
			for i := 0; i < fv.Len(); i++ {
				if cmp, ok := compare(fv.Index(i), want); ok && cmp == 0 {
					found = true
					break
				}
			}
			if c.Op == database.OpContainsAll && !found {
				return false, nil
			}
			if c.Op == database.OpContainsAny && found {
				return true, nil
			}
		}
		return c.Op == database.OpContainsAll, nil
	}
	return false, fmt.Errorf("json: %s needs a scalar column, got %s", opName(c.Op), c.Column)
}

func opName(op database.ConditionOp) string {
	if s, ok := opNames[op]; ok {
		return s
	}
	return "condition"
}

var opNames = map[database.ConditionOp]string{
	database.OpContains:    "Contains",
	database.OpContainsAll: "ContainsAll",
	database.OpContainsAny: "ContainsAny",
	database.OpContainsKV:  "ContainsKV",
}

// matchAny reports whether the field equals any of the candidates.
func matchAny(fv reflect.Value, candidates []any) bool {
	for _, c := range candidates {
		if cmp, ok := compare(fv, c); ok && cmp == 0 {
			return true
		}
	}
	return false
}

// compare orders two values, returning -1, 0 or 1. The bool reports whether the
// comparison is meaningful; mixed kinds that cannot be ordered are not.
func compare(a reflect.Value, b any) (int, bool) {
	bv := reflect.ValueOf(b)
	if !bv.IsValid() {
		return 0, false
	}
	// An addressable value is fine to read, but a nil interface hides the type.
	if bv.Kind() == reflect.Interface {
		bv = bv.Elem()
		if !bv.IsValid() {
			return 0, false
		}
	}

	if af, aok := numeric(a); aok {
		if bf, bok := numeric(bv); bok {
			switch {
			case af < bf:
				return -1, true
			case af > bf:
				return 1, true
			default:
				return 0, true
			}
		}
		return 0, false
	}

	switch a.Kind() {
	case reflect.String:
		if bv.Kind() != reflect.String {
			return 0, false
		}
		return strings.Compare(a.String(), bv.String()), true

	case reflect.Bool:
		if bv.Kind() != reflect.Bool {
			return 0, false
		}
		switch {
		case a.Bool() == bv.Bool():
			return 0, true
		case !a.Bool():
			return -1, true
		default:
			return 1, true
		}

	// A time.Time field or a string column both order lexically once rendered.
	default:
		as, bs := stringOf(a), stringOf(bv)
		if as == "" && bs == "" {
			return 0, false
		}
		return strings.Compare(as, bs), true
	}
}

// numeric renders a value as a float when it is any numeric kind, so an int
// column can be compared against a float literal.
func numeric(v reflect.Value) (float64, bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	}
	// A value decoded from JSON may arrive boxed as an interface.
	if v.Kind() == reflect.Interface {
		return numeric(v.Elem())
	}
	return 0, false
}

// stringOf renders a value the way a text column would hold it.
func stringOf(v reflect.Value) string {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return ""
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	}
	if f, ok := numeric(v); ok {
		return fmt.Sprintf("%v", f)
	}
	return fmt.Sprintf("%v", v.Interface())
}

func isNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}

// likeMatch implements SQL LIKE with % and _ wildcards, case-sensitively.
//
// The naive two-pointer version is quadratic on pathological patterns such as
// many stars followed by a mismatch, so this tracks the last star instead.
func likeMatch(s, pattern string) bool {
	sr, pr := []rune(s), []rune(pattern)

	var si, pi int
	star, mark := -1, 0

	for si < len(sr) {
		switch {
		case pi < len(pr) && (pr[pi] == '_' || pr[pi] == sr[si]):
			si++
			pi++
		case pi < len(pr) && pr[pi] == '%':
			star = pi
			mark = si
			pi++
		case star >= 0:
			pi = star + 1
			mark++
			si = mark
		default:
			return false
		}
	}
	for pi < len(pr) && pr[pi] == '%' {
		pi++
	}
	return pi == len(pr)
}
