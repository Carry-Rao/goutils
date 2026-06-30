package mysql

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Carry-Rao/goutils/database/helpers"
)

type Table struct {
	db        *sql.DB
	tableName string
	typ       reflect.Type
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
		cols = append(cols, fmt.Sprintf("`%s`", colName))
		placeholders = append(placeholders, "?")
		args = append(args, val.Field(i).Interface())
	}
	if len(cols) == 0 {
		return fmt.Errorf("struct has no exported fields")
	}
	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		t.tableName,
		strings.Join(cols, ","),
		strings.Join(placeholders, ","))
	_, err := t.db.Exec(query, args...)
	return err
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) ([]any, error) {
	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	whereClause, args, err := helpers.BuildWhereClause(typ, val, whereFields, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM `%s`", t.tableName)
	if whereClause != "" {
		query = fmt.Sprintf("SELECT * FROM `%s` WHERE %s", t.tableName, whereClause)
	}

	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	colIndex := make(map[string]int, len(cols))
	for i, c := range cols {
		colIndex[c] = i
	}

	var results []any
	for rows.Next() {
		result := reflect.New(typ).Elem()
		scanPtrs := make([]any, len(cols))
		tmpBuf := make([]*interface{}, len(cols))
		for i := range tmpBuf {
			tmpBuf[i] = new(interface{})
			scanPtrs[i] = tmpBuf[i]
		}
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if !f.IsExported() {
				continue
			}
			colName := helpers.GetDBColumnName(f)
			if idx, ok := colIndex[colName]; ok {
				scanPtrs[idx] = result.Field(i).Addr().Interface()
			}
		}
		if err := rows.Scan(scanPtrs...); err != nil {
			return nil, err
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

	whereClause, args, err := helpers.BuildWhereClause(typ, val, whereFields, "")
	if err != nil {
		return err
	}

	whereSet := make(map[string]bool)
	for _, fn := range whereFields {
		whereSet[fn] = true
	}

	var sets []string
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
		sets = append(sets, fmt.Sprintf("`%s`=?", colName))
		args = append(args, fv.Interface())
	}

	if len(sets) == 0 {
		return fmt.Errorf("no fields to set")
	}

	query := fmt.Sprintf("UPDATE `%s` SET %s", t.tableName, strings.Join(sets, ","))
	if whereClause != "" {
		query = fmt.Sprintf("UPDATE `%s` SET %s WHERE %s", t.tableName, strings.Join(sets, ","), whereClause)
	}
	_, err = t.db.Exec(query, args...)
	return err
}

func (t *Table) Del(example any, whereFields []string, _ time.Duration) error {
	val := helpers.UnwrapValue(reflect.ValueOf(example))
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}
	typ := val.Type()

	whereClause, args, err := helpers.BuildWhereClause(typ, val, whereFields, "")
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM `%s`", t.tableName)
	if whereClause != "" {
		query = fmt.Sprintf("DELETE FROM `%s` WHERE %s", t.tableName, whereClause)
	}
	_, err = t.db.Exec(query, args...)
	return err
}