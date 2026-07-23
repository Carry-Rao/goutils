package postgresql

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Carry-Rao/goutils/database/api"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Table struct {
	pool      *pgxpool.Pool
	table     string
	schema    *api.CachedSchema
	insertQry string
}

func (t *Table) Ins(example any, _ time.Duration) error {
	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	args := t.schema.ExtractValues(val)
	if len(args) == 0 {
		return fmt.Errorf("struct has no exported fields")
	}
	_, err := t.pool.Exec(context.Background(), t.insertQry, args...)
	return err
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) ([]any, error) {
	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}

	whereClause, args, err := t.schema.BuildWhereClause(val, whereFields, "pg")
	if err != nil {
		return nil, err
	}

	query := "SELECT * FROM " + t.table
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	rows, err := t.pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := rows.FieldDescriptions()
	if len(cols) == 0 {
		return nil, nil
	}

	colIndex := make(map[string]int, len(cols))
	for i, c := range cols {
		colIndex[string(c.Name)] = i
	}

	var results []any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		result := api.NewStruct(t.schema.Type)
		for _, field := range t.schema.Fields {
			if idx, ok := colIndex[field.ColumnName]; ok && idx < len(values) {
				v := values[idx]
				if v != nil {
					result.Field(field.Index).Set(reflect.ValueOf(v))
				}
			}
		}
		results = append(results, result.Addr().Interface())
	}

	return results, nil
}

func (t *Table) Set(example any, whereFields []string, _ time.Duration) error {
	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	whereClause, args, err := t.schema.BuildWhereClause(val, whereFields, "pg")
	if err != nil {
		return err
	}

	whereSet := make(map[string]bool, len(whereFields))
	for _, fn := range whereFields {
		whereSet[fn] = true
	}

	var sets []string
	paramIdx := len(args) + 1
	for _, field := range t.schema.Insertable {
		if whereSet[field.ColumnName] || whereSet[field.GoFieldName] {
			continue
		}
		sets = append(sets, field.ColumnName+"=$"+itoa(paramIdx))
		args = append(args, val.Field(field.Index).Interface())
		paramIdx++
	}

	if len(sets) == 0 {
		return fmt.Errorf("no fields to set")
	}

	query := "UPDATE " + t.table + " SET " + strings.Join(sets, ",")
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	_, err = t.pool.Exec(context.Background(), query, args...)
	return err
}

func (t *Table) Del(example any, whereFields []string, _ time.Duration) error {
	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	whereClause, args, err := t.schema.BuildWhereClause(val, whereFields, "pg")
	if err != nil {
		return err
	}

	query := "DELETE FROM " + t.table
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	_, err = t.pool.Exec(context.Background(), query, args...)
	return err
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
