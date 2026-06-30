package postgresql

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Carry-Rao/goutils/database/helpers"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Table struct {
	pool  *pgxpool.Pool
	table string
	typ   reflect.Type
}

func (t *Table) Ins(example any, _ time.Duration) error {
	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	var cols []string
	var placeholders []string
	var args []any
	paramIdx := 1
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		_, tags := helpers.ParseDBTag(f)
		if tags["autoinc"] {
			continue
		}
		colName := helpers.GetDBColumnName(f)
		cols = append(cols, colName)
		placeholders = append(placeholders, fmt.Sprintf("$%d", paramIdx))
		args = append(args, val.Field(i).Interface())
		paramIdx++
	}
	if len(cols) == 0 {
		return fmt.Errorf("struct has no exported fields")
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		t.table,
		strings.Join(cols, ","),
		strings.Join(placeholders, ","))
	_, err := t.pool.Exec(context.Background(), query, args...)
	return err
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) ([]any, error) {
	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	whereClause, args, err := helpers.BuildWhereClause(typ, val, whereFields, "pg")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM %s", t.table)
	if whereClause != "" {
		query = fmt.Sprintf("SELECT * FROM %s WHERE %s", t.table, whereClause)
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
		result := reflect.New(typ).Elem()
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if !f.IsExported() {
				continue
			}
			colName := helpers.GetDBColumnName(f)
			if idx, ok := colIndex[colName]; ok && idx < len(values) {
				v := values[idx]
				if v != nil {
					result.Field(i).Set(reflect.ValueOf(v))
				}
			}
		}
		results = append(results, result.Addr().Interface())
	}

	return results, nil
}

func (t *Table) Set(example any, whereFields []string, _ time.Duration) error {
	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	whereClause, args, err := helpers.BuildWhereClause(typ, val, whereFields, "pg")
	if err != nil {
		return err
	}

	whereSet := make(map[string]bool)
	for _, fn := range whereFields {
		whereSet[fn] = true
	}

	var sets []string
	paramIdx := len(args) + 1
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		colName := helpers.GetDBColumnName(f)
		if whereSet[colName] || whereSet[f.Name] {
			continue
		}
		fv := val.Field(i)
		sets = append(sets, fmt.Sprintf("%s=$%d", colName, paramIdx))
		args = append(args, fv.Interface())
		paramIdx++
	}

	if len(sets) == 0 {
		return fmt.Errorf("no fields to set")
	}

	query := fmt.Sprintf("UPDATE %s SET %s", t.table, strings.Join(sets, ","))
	if whereClause != "" {
		query = fmt.Sprintf("UPDATE %s SET %s WHERE %s", t.table, strings.Join(sets, ","), whereClause)
	}
	_, err = t.pool.Exec(context.Background(), query, args...)
	return err
}

func (t *Table) Del(example any, whereFields []string, _ time.Duration) error {
	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	whereClause, args, err := helpers.BuildWhereClause(typ, val, whereFields, "pg")
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM %s", t.table)
	if whereClause != "" {
		query = fmt.Sprintf("DELETE FROM %s WHERE %s", t.table, whereClause)
	}
	_, err = t.pool.Exec(context.Background(), query, args...)
	return err
}