package database

import (
	"fmt"
	"reflect"
	"strings"
)

// Options carries everything a Table method needs beyond the model itself:
// predicates, assignments, ordering and paging. The zero value is valid
// everywhere.
type Options struct {
	// Where filters rows. Conditions are combined with AND.
	Where []Condition

	// Values assigns columns on Set. A collection column listed here overrides
	// the default whole-value replacement with the given operation.
	Values []Value

	// Skip names columns Set must not touch.
	Skip []string

	Order  []Order
	Limit  int
	Offset int
}

// Condition is one predicate.
type Condition struct {
	Column string
	Op     ConditionOp
	Value  any
	Values []any // second operand, used by Between
	Key    any   // key operand for map collections
}

// ConditionOp enumerates the predicates.
type ConditionOp int

const (
	OpEq ConditionOp = iota
	OpNeq
	OpGt
	OpGte
	OpLt
	OpLte
	OpLike
	OpNotLike
	OpIn
	OpNotIn
	OpBetween
	OpIsNull
	OpIsNotNull

	// OpContains applies to collections: a slice must hold the element, a map
	// must hold the key.
	OpContains
	// OpContainsAll requires every listed element to be present.
	OpContainsAll
	// OpContainsAny requires at least one listed element.
	OpContainsAny
	// OpContainsKV requires map[k] == v.
	OpContainsKV
)

// ValueOp enumerates the assignment operations.
type ValueOp int

const (
	// OpAssign replaces the column, or the whole collection by default.
	OpAssign ValueOp = iota
	// OpUnion merges into a collection.
	OpUnion
	// OpDifference removes from a collection.
	OpDifference
	// OpClear empties a collection, or sets a scalar column to NULL.
	OpClear
)

// Order is one sort key.
type Order struct {
	Column string
	Desc   bool
}

// Value is one assignment on Set.
type Value struct {
	Column string
	Op     ValueOp
	Value  any
}

// ---------- condition constructors ----------

func Eq(column string, v any) Condition   { return Condition{Column: column, Op: OpEq, Value: v} }
func Neq(column string, v any) Condition  { return Condition{Column: column, Op: OpNeq, Value: v} }
func Gt(column string, v any) Condition   { return Condition{Column: column, Op: OpGt, Value: v} }
func Gte(column string, v any) Condition  { return Condition{Column: column, Op: OpGte, Value: v} }
func Lt(column string, v any) Condition   { return Condition{Column: column, Op: OpLt, Value: v} }
func Lte(column string, v any) Condition  { return Condition{Column: column, Op: OpLte, Value: v} }
func Like(column string, v any) Condition { return Condition{Column: column, Op: OpLike, Value: v} }
func NotLike(column string, v any) Condition {
	return Condition{Column: column, Op: OpNotLike, Value: v}
}
func IsNull(column string) Condition    { return Condition{Column: column, Op: OpIsNull} }
func IsNotNull(column string) Condition { return Condition{Column: column, Op: OpIsNotNull} }

func In(column string, vs ...any) Condition { return Condition{Column: column, Op: OpIn, Values: vs} }
func NotIn(column string, vs ...any) Condition {
	return Condition{Column: column, Op: OpNotIn, Values: vs}
}

func Between(column string, lo, hi any) Condition {
	return Condition{Column: column, Op: OpBetween, Value: lo, Values: []any{hi}}
}

// Contains matches a collection field: a slice must hold the element, a map must
// hold the key.
func Contains(column string, v any) Condition {
	return Condition{Column: column, Op: OpContains, Value: v}
}

// ContainsAll requires every element to be present in a slice.
func ContainsAll(column string, vs ...any) Condition {
	return Condition{Column: column, Op: OpContainsAll, Values: vs}
}

// ContainsAny requires at least one element in a slice.
func ContainsAny(column string, vs ...any) Condition {
	return Condition{Column: column, Op: OpContainsAny, Values: vs}
}

// ContainsKV requires map[key] == value.
func ContainsKV(column string, key, value any) Condition {
	return Condition{Column: column, Op: OpContainsKV, Key: key, Value: value}
}

// ---------- value constructors ----------

func Assign(column string, v any) Value { return Value{Column: column, Op: OpAssign, Value: v} }
func Union(column string, v any) Value  { return Value{Column: column, Op: OpUnion, Value: v} }
func Difference(column string, v any) Value {
	return Value{Column: column, Op: OpDifference, Value: v}
}
func Clear(column string) Value { return Value{Column: column, Op: OpClear} }

// ---------- order constructors ----------

func Asc(column string) Order  { return Order{Column: column} }
func Desc(column string) Order { return Order{Column: column, Desc: true} }

func Ascending(column string) []Order  { return []Order{Asc(column)} }
func Descending(column string) []Order { return []Order{Desc(column)} }

// ---------- validation ----------

// scalarOnly lists the operators that do not apply to collections.
var scalarOnly = map[ConditionOp]bool{
	OpEq: true, OpNeq: true, OpGt: true, OpGte: true, OpLt: true, OpLte: true,
	OpLike: true, OpNotLike: true, OpIn: true, OpNotIn: true, OpBetween: true,
	OpIsNull: true, OpIsNotNull: true,
}

var collectionOnly = map[ConditionOp]bool{
	OpContains: true, OpContainsAll: true, OpContainsAny: true, OpContainsKV: true,
}

var sliceOnly = map[ConditionOp]bool{
	OpContainsAll: true, OpContainsAny: true,
}

var mapOnly = map[ConditionOp]bool{
	OpContainsKV: true,
}

func (o Options) Validate(s *Schema) error {
	for _, c := range o.Where {
		if err := c.validate(s); err != nil {
			return err
		}
	}
	for _, v := range o.Values {
		if err := v.validate(s); err != nil {
			return err
		}
	}
	for _, ord := range o.Order {
		if _, ok := s.Lookup(ord.Column); !ok {
			return fmt.Errorf("order: unknown field %q", ord.Column)
		}
	}
	if o.Limit < 0 {
		return fmt.Errorf("options: limit must not be negative")
	}
	if o.Offset < 0 {
		return fmt.Errorf("options: offset must not be negative")
	}
	return nil
}

func (c Condition) validate(s *Schema) error {
	f, ok := s.Lookup(c.Column)
	if !ok {
		return fmt.Errorf("where: unknown field %q", c.Column)
	}
	switch {
	case scalarOnly[c.Op] && f.IsChild:
		return fmt.Errorf("%s: %q holds a collection, use Contains or a collection operator", opName(c.Op), c.Column)
	case collectionOnly[c.Op] && !f.IsChild:
		return fmt.Errorf("%s: %q is not a collection", opName(c.Op), c.Column)
	case sliceOnly[c.Op] && f.GoKind == reflect.Map:
		return fmt.Errorf("%s: %q is a map, use ContainsKV", opName(c.Op), c.Column)
	case mapOnly[c.Op] && f.GoKind != reflect.Map:
		return fmt.Errorf("%s: %q is not a map", opName(c.Op), c.Column)
	}
	return nil
}

func (v Value) validate(s *Schema) error {
	f, ok := s.Lookup(v.Column)
	if !ok {
		return fmt.Errorf("values: unknown field %q", v.Column)
	}
	if !f.IsChild && v.Op != OpAssign && v.Op != OpClear {
		return fmt.Errorf("value op %d: %q is not a collection", v.Op, v.Column)
	}
	return nil
}

func opName(op ConditionOp) string {
	switch op {
	case OpEq:
		return "Eq"
	case OpNeq:
		return "Neq"
	case OpGt:
		return "Gt"
	case OpGte:
		return "Gte"
	case OpLt:
		return "Lt"
	case OpLte:
		return "Lte"
	case OpLike:
		return "Like"
	case OpNotLike:
		return "NotLike"
	case OpIn:
		return "In"
	case OpNotIn:
		return "NotIn"
	case OpBetween:
		return "Between"
	case OpIsNull:
		return "IsNull"
	case OpIsNotNull:
		return "IsNotNull"
	case OpContains:
		return "Contains"
	case OpContainsAll:
		return "ContainsAll"
	case OpContainsAny:
		return "ContainsAny"
	case OpContainsKV:
		return "ContainsKV"
	default:
		return "operator"
	}
}

// skipSet reports whether a column is excluded from Set.
func (o Options) skipSet(s *Schema, column string) bool {
	for _, name := range o.Skip {
		if f, ok := s.Lookup(name); ok && f.ColumnName == column {
			return true
		}
	}
	return false
}

// valueOp returns the assignment recorded for a column, if any.
func (o Options) valueOp(s *Schema, column string) (Value, bool) {
	for _, v := range o.Values {
		if f, ok := s.Lookup(v.Column); ok && f.ColumnName == column {
			return v, true
		}
	}
	return Value{}, false
}

// orderBy renders the ORDER BY clause and reports whether there is one.
func (o Options) orderBy(s *Schema, d Dialect) (string, bool) {
	if len(o.Order) == 0 {
		return "", false
	}
	parts := make([]string, 0, len(o.Order))
	for _, ord := range o.Order {
		f, _ := s.Lookup(ord.Column)
		part := d.Quote(f.ColumnName)
		if ord.Desc {
			part += " DESC"
		}
		parts = append(parts, part)
	}
	return " ORDER BY " + strings.Join(parts, ","), true
}

func (o Options) limitClause(d Dialect) string {
	out := ""
	if o.Limit > 0 {
		out = " LIMIT " + itoa(o.Limit)
	}
	if o.Offset > 0 {
		if out == "" {
			out = " LIMIT -1" // required by MySQL when OFFSET is used alone
		}
		out += " OFFSET " + itoa(o.Offset)
	}
	return out
}
