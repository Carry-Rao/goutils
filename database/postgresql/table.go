package postgresql

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Table struct {
	pool  *pgxpool.Pool
	table string
}

func (t *Table) Create(data map[string]any) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	var fields []string
	var placeholders []string
	var args []any
	i := 1
	for k, v := range data {
		fields = append(fields, k)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		args = append(args, v)
		i++
	}
	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		t.table,
		strings.Join(fields, ","),
		strings.Join(placeholders, ","),
	)
	_, err := t.pool.Exec(context.Background(), sql, args...)
	return err
}

func (t *Table) Get(where map[string]any) ([]any, error) {
	if len(where) == 0 {
		return nil, fmt.Errorf("empty where")
	}
	var conds []string
	var args []any
	i := 1
	for k, v := range where {
		conds = append(conds, fmt.Sprintf("%s=$%d", k, i))
		args = append(args, v)
		i++
	}
	sql := fmt.Sprintf(
		"SELECT * FROM %s WHERE %s LIMIT 1",
		t.table,
		strings.Join(conds, " AND "),
	)
	rows, err := t.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rows.Values()
}

func (t *Table) Set(data map[string]any) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	var sets []string
	var args []any
	i := 1
	var whereVal any
	hasWhere := false
	for k, v := range data {
		if k == "id" {
			whereVal = v
			hasWhere = true
			continue
		}
		sets = append(sets, fmt.Sprintf("%s=$%d", k, i))
		args = append(args, v)
		i++
	}
	if !hasWhere {
		return fmt.Errorf("missing id for where")
	}
	args = append(args, whereVal)
	sql := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id=$%d",
		t.table,
		strings.Join(sets, ","),
		i,
	)
	_, err := t.pool.Exec(context.Background(), sql, args...)
	return err
}

func (t *Table) Delete(where map[string]any) error {
	if len(where) == 0 {
		return fmt.Errorf("empty where")
	}
	var conds []string
	var args []any
	i := 1
	for k, v := range where {
		conds = append(conds, fmt.Sprintf("%s=$%d", k, i))
		args = append(args, v)
		i++
	}
	sql := fmt.Sprintf(
		"DELETE FROM %s WHERE %s",
		t.table,
		strings.Join(conds, " AND "),
	)
	_, err := t.pool.Exec(context.Background(), sql, args...)
	return err
}
