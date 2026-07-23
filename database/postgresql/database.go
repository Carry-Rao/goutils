package postgresql

import (
	"context"
	"fmt"
	"strings"

	"github.com/Carry-Rao/goutils/database/api"
)

func (p *Database) Create(tableName string, config map[string]api.Config) error {
	if len(config) == 0 {
		return fmt.Errorf("empty config")
	}

	var columns []string
	var pks []string

	for field, cfg := range config {
		col := fmt.Sprintf("%s %s", field, cfg.Type)
		if !cfg.NullAble {
			col += " NOT NULL"
		}
		if cfg.Identity {
			col += " GENERATED ALWAYS AS IDENTITY"
		}
		if cfg.Unique {
			col += " UNIQUE"
		}
		if cfg.PrimaryKey {
			pks = append(pks, field)
		}
		columns = append(columns, col)
	}

	if len(pks) > 0 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pks, ", ")))
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(columns, ", "))

	_, err := p.pool.Exec(context.Background(), sql)
	if err != nil {
		return fmt.Errorf("exec create table: %w", err)
	}
	return nil
}

func (p *Database) GetTable(tableName string, example any) (api.Table, error) {
	var exists bool
	err := p.pool.QueryRow(
		context.Background(),
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name=$1)`,
		tableName,
	).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("table %s not exists", tableName)
	}
	schema := api.GetOrBuildSchema(api.TypeOf(example))
	return &Table{pool: p.pool, table: tableName, schema: schema, insertQry: schema.BuildInsertQueryPG(tableName)}, nil
}

func (p *Database) DeleteTable(tableName string) error {
	_, err := p.pool.Exec(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	return err
}
