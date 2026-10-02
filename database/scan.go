package database

import (
	"database/sql"
	"reflect"
)

// ScanRows reads every row into a T. T may be the struct type or a pointer to
// it. Columns absent from the schema are discarded.
func ScanRows[T any](rows *sql.Rows, schema *Schema) ([]T, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	cell := reflect.New(schema.Type)
	targets := scanTargets(cols, schema, cell)
	results := make([]T, 0, 8)

	for rows.Next() {
		cell.Elem().Set(reflect.Zero(schema.Type))
		if err := rows.Scan(targets...); err != nil {
			return nil, err
		}
		out, ok := cellToT[T](cell)
		if !ok {
			return nil, &ScanTypeError{Got: schema.Type}
		}
		results = append(results, out)
	}

	// Iteration stops early on error, so a short result set is not success.
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// scanTargets builds the rows.Scan destination. cell must be the value
// ScanRows reads back, since the targets alias its fields.
func scanTargets(cols []string, schema *Schema, cell reflect.Value) []any {
	colIndex := make(map[string]int, len(cols))
	for i, c := range cols {
		colIndex[c] = i
	}

	targets := make([]any, len(cols))
	for i := range targets {
		targets[i] = new(any) // unknown columns are read and dropped
	}

	for _, f := range schema.Fields {
		if idx, ok := colIndex[f.ColumnName]; ok {
			targets[idx] = cell.Elem().Field(f.Index).Addr().Interface()
		}
	}
	return targets
}

// cellToT converts a *Struct pointer into T, struct or pointer T alike.
func cellToT[T any](cell reflect.Value) (T, bool) {
	var zero T
	tType := reflect.TypeOf(&zero).Elem()

	if cell.Type().AssignableTo(tType) {
		v, ok := cell.Interface().(T)
		return v, ok
	}
	if cell.Kind() == reflect.Ptr && cell.Type().Elem().AssignableTo(tType) {
		v, ok := cell.Elem().Interface().(T)
		return v, ok
	}
	return zero, false
}

// ScanTypeError reports rows that could not be materialised into T.
type ScanTypeError struct {
	Got reflect.Type
}

func (e *ScanTypeError) Error() string {
	return "database: cannot scan " + e.Got.String() + " into the requested model type"
}
