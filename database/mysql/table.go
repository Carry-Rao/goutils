package mysql

import (
	"database/sql"
	"fmt"
	"strings"
)

type Table struct {
	db        *sql.DB
	tableName string
}

func (t *Table) Create(data map[string]any) error {
	var fields []string
	var placeholders []string
	var args []any
	for k, v := range data {
		fields = append(fields, k)
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", t.tableName, strings.Join(fields, ","), strings.Join(placeholders, ","))
	_, err := t.db.Exec(query, args...)
	return err
}

func (t *Table) Get(where map[string]any) ([]any, error) {
	var conds []string
	var args []any
	for k, v := range where {
		conds = append(conds, fmt.Sprintf("%s=?", k))
		args = append(args, v)
	}
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 1", t.tableName, strings.Join(conds, " AND "))
	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	return vals, nil
}

func (t *Table) Set(data map[string]any) error {
	var sets []string
	var args []any
	var id any
	hasID := false
	for k, v := range data {
		if k == "id" {
			id = v
			hasID = true
			continue
		}
		sets = append(sets, fmt.Sprintf("%s=?", k))
		args = append(args, v)
	}
	if !hasID {
		return fmt.Errorf("missing id")
	}
	args = append(args, id)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id=?", t.tableName, strings.Join(sets, ","))
	_, err := t.db.Exec(query, args...)
	return err
}

func (t *Table) Delete(where map[string]any) error {
	var conds []string
	var args []any
	for k, v := range where {
		conds = append(conds, fmt.Sprintf("%s=?", k))
		args = append(args, v)
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", t.tableName, strings.Join(conds, " AND "))
	_, err := t.db.Exec(query, args...)
	return err
}
