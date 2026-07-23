package mysql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Carry-Rao/goutils/database/api"
)

type Table struct {
	db        *sql.DB
	tableName string
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
	_, err := t.db.Exec(t.insertQry, args...)
	return err
}

func (t *Table) Get(example any, whereFields []string, _ time.Duration) ([]any, error) {
	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return nil, fmt.Errorf("example must be struct or pointer to struct")
	}

	whereClause, args, err := t.schema.BuildWhereClause(val, whereFields, "")
	if err != nil {
		return nil, err
	}

	query := "SELECT * FROM `" + t.tableName + "`"
	if whereClause != "" {
		query += " WHERE " + whereClause
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
		result := api.NewStruct(t.schema.Type)
		scanPtrs := make([]any, len(cols))
		tmpBuf := make([]*interface{}, len(cols))
		for i := range tmpBuf {
			tmpBuf[i] = new(interface{})
			scanPtrs[i] = tmpBuf[i]
		}
		for _, field := range t.schema.Fields {
			if idx, ok := colIndex[field.ColumnName]; ok {
				scanPtrs[idx] = result.Field(field.Index).Addr().Interface()
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
	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	whereSet := make(map[string]bool, len(whereFields))
	for _, fn := range whereFields {
		whereSet[fn] = true
	}

	var sets []string
	var setArgs []any
	for _, field := range t.schema.Insertable {
		if whereSet[field.ColumnName] || whereSet[field.GoFieldName] {
			continue
		}
		sets = append(sets, "`"+field.ColumnName+"`=?")
		setArgs = append(setArgs, val.Field(field.Index).Interface())
	}

	if len(sets) == 0 {
		return fmt.Errorf("no fields to set")
	}

	whereClause, whereArgs, err := t.schema.BuildWhereClause(val, whereFields, "")
	if err != nil {
		return err
	}

	args := append(setArgs, whereArgs...)
	query := "UPDATE `" + t.tableName + "` SET " + strings.Join(sets, ",")
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	_, err = t.db.Exec(query, args...)
	return err
}

func (t *Table) Del(example any, whereFields []string, _ time.Duration) error {
	val := api.UnwrapValueOf(example)
	if val.Kind() != api.KindStruct {
		return fmt.Errorf("example must be struct or pointer to struct")
	}

	whereClause, args, err := t.schema.BuildWhereClause(val, whereFields, "")
	if err != nil {
		return err
	}

	query := "DELETE FROM `" + t.tableName + "`"
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	_, err = t.db.Exec(query, args...)
	return err
}
