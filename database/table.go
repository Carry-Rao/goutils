package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// Execer runs SQL. Both *sql.DB and *sql.Tx satisfy it, which is how one Table
// implementation serves both a database and a transaction.
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
}

// Table is the CRUD surface of one table.
type Table[T any] interface {
	// Ins writes one row, including any collections.
	Ins(val T, opts Options) error

	// Get reads the matching rows with their collections filled in.
	Get(val T, opts Options) ([]T, error)

	// Set updates the rows matching opts.Where, taking values from updated.
	Set(old, updated T, opts Options) error

	// Del removes the matching rows along with their collection rows.
	Del(val T, opts Options) error
}

// SQLTable is the shared implementation for the SQL backends.
type SQLTable[T any] struct {
	exec    Execer
	runner  TxRunner
	schema  *Schema
	table   string
	dialect Dialect
	insert  string
}

// TxRunner runs a unit of work atomically. A table built over a *sql.DB starts
// a real transaction; one built inside a transaction reuses it.
type TxRunner interface {
	Run(fn func(Execer) error) error
}

// NewSQLTable builds a Table over exec. It creates the main table and the
// companion tables for any collection fields when they do not exist.
func NewSQLTable[T any](exec Execer, runner TxRunner, table string, d Dialect) (*SQLTable[T], error) {
	schema := SchemaOf[T]()
	t := &SQLTable[T]{exec: exec, runner: runner, schema: schema, table: table, dialect: d, insert: schema.InsertQuery(d, table)}

	if _, err := schema.PrimaryKey(); err != nil {
		return nil, err
	}
	if err := t.createMain(); err != nil {
		return nil, err
	}
	if err := t.createChildren(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *SQLTable[T]) createMain() error {
	var cols, pkCols []string

	for _, f := range t.schema.Scalars {
		def := fmt.Sprintf("%s %s", t.dialect.Quote(f.ColumnName), t.dialect.SQLType(f.GoKind))
		switch {
		case f.IsAutoInc && f.IsPrimary:
			def += " PRIMARY KEY " + t.dialect.AutoInc
		default:
			if !f.IsNullable {
				def += " NOT NULL"
			}
			if f.IsUnique {
				def += " UNIQUE"
			}
			if f.IsPrimary {
				pkCols = append(pkCols, t.dialect.Quote(f.ColumnName))
			}
		}
		cols = append(cols, def)
	}

	if len(pkCols) == 0 {
		pk, _ := t.schema.PrimaryKey()
		cols = append(cols, fmt.Sprintf("%s %s", t.dialect.Quote(pk.ColumnName), t.dialect.SQLType(pk.GoKind)))
		pkCols = append(pkCols, t.dialect.Quote(pk.ColumnName))
	}
	cols = append(cols, "PRIMARY KEY ("+strings.Join(pkCols, ",")+")")

	stmt := "CREATE TABLE IF NOT EXISTS " + t.dialect.QuoteTable(t.table) + " (" + strings.Join(cols, ",") + ")"
	_, err := t.exec.Exec(stmt)
	return err
}

// childTable returns the companion table name and its metadata for a field.
func (t *SQLTable[T]) childTable(f FieldInfo) string {
	return t.schema.ChildTableName(t.table, f)
}

// parentColumn is the column a child row references.
func (t *SQLTable[T]) parentColumn(pk FieldInfo) string {
	return "parent_" + strings.ToLower(pk.ColumnName)
}

func (t *SQLTable[T]) createChildren() error {
	pk, err := t.schema.PrimaryKey()
	if err != nil {
		return err
	}
	pkType := t.dialect.SQLType(pk.GoKind)

	for _, f := range t.schema.Collections {
		table := t.childTable(f)
		elemType := "TEXT"

		var cols []string
		if f.GoKind == reflect.Map {
			cols = append(cols, fmt.Sprintf("%s %s", t.dialect.Quote("k"), elemType))
			cols = append(cols, fmt.Sprintf("%s %s", t.dialect.Quote("v"), elemType))
		} else {
			cols = append(cols, fmt.Sprintf("%s %s", t.dialect.Quote(f.ColumnName), elemType))
		}
		cols = append(cols, fmt.Sprintf("%s %s", t.dialect.Quote(t.parentColumn(pk)), pkType))

		unique := []string{t.dialect.Quote(f.ColumnName), t.dialect.Quote(t.parentColumn(pk))}
		if f.GoKind == reflect.Map {
			unique = []string{t.dialect.Quote("k"), t.dialect.Quote("v"), t.dialect.Quote(t.parentColumn(pk))}
		}
		if f.IsGlobalUnique {
			// Drop the parent column from the key: the value must not appear
			// under any other parent either. For a map the key alone is the
			// identity, so v is dropped too — one key, one owner.
			if f.GoKind == reflect.Map {
				unique = []string{t.dialect.Quote("k")}
			} else {
				unique = []string{t.dialect.Quote(f.ColumnName)}
			}
		}

		ddl := "CREATE TABLE IF NOT EXISTS " + t.dialect.QuoteTable(table) +
			" (" + strings.Join(cols, ",") + ", UNIQUE(" + strings.Join(unique, ",") + "))"
		if _, err := t.exec.Exec(ddl); err != nil {
			return err
		}
	}
	return nil
}
