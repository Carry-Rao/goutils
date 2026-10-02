package database

import (
	"database/sql"
	"errors"
	"sort"
	"sync"
)

// Database is a SQL database handle typed to one model.
type Database[T any] interface {
	// GetTable returns the table, creating it and its collection tables when
	// they do not exist.
	GetTable(tableName string) (Table[T], error)

	// DeleteTable drops the table and its collection tables.
	DeleteTable(tableName string) error

	// Query runs arbitrary SQL and scans the rows into T.
	Query(sqlText string, args ...any) ([]T, error)

	// Exec runs a statement that returns no rows.
	Exec(sqlText string, args ...any) error

	// Tx runs fn atomically. Returning an error rolls everything back.
	Tx(fn func(Tx[T]) error) error

	// Close releases the connection pool.
	Close() error
}

// Tx is the transactional counterpart of Database.
type Tx[T any] interface {
	GetTable(tableName string) (Table[T], error)
	Query(sqlText string, args ...any) ([]T, error)
	Exec(sqlText string, args ...any) error
}

// Config describes a column when declaring a table by hand. Tables derived from
// a model do not need it.
type Config struct {
	Type       string
	NullAble   bool
	Identity   bool
	PrimaryKey bool
	Unique     bool
}

// Backend is the shared machinery every SQL driver embeds.
type Backend[T any] struct {
	DB      *sql.DB
	Dialect Dialect
}

// NewSQLTable builds a Table over the pool, starting a transaction per write.
func (b *Backend[T]) NewSQLTable(table string) (*SQLTable[T], error) {
	return NewSQLTable[T](b.DB, DbTxRunner{DB: b.DB}, table, b.Dialect)
}

// GetTable returns the table, creating it when missing.
func (b *Backend[T]) GetTable(tableName string) (Table[T], error) {
	return b.NewSQLTable(tableName)
}

// Query runs arbitrary SQL and scans the rows into T.
func (b *Backend[T]) Query(sqlText string, args ...any) ([]T, error) {
	rows, err := b.DB.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return ScanRows[T](rows, SchemaOf[T]())
}

// Exec runs a statement that returns no rows.
func (b *Backend[T]) Exec(sqlText string, args ...any) error {
	_, err := b.DB.Exec(sqlText, args...)
	return err
}

// DeleteTable drops the table and its collection tables.
func (b *Backend[T]) DeleteTable(tableName string) error {
	schema := SchemaOf[T]()
	tables := make([]string, 0, len(schema.Collections)+1)
	for _, f := range schema.Collections {
		tables = append(tables, schema.ChildTableName(tableName, f))
	}
	tables = append(tables, tableName)

	for _, name := range tables {
		if _, err := b.DB.Exec("DROP TABLE IF EXISTS " + b.Dialect.QuoteTable(name)); err != nil {
			return err
		}
	}
	return nil
}

// Tx runs fn inside a transaction.
func (b *Backend[T]) Tx(fn func(Tx[T]) error) error {
	sqlTx, err := b.DB.Begin()
	if err != nil {
		return err
	}
	if err := fn(&txContext[T]{tx: sqlTx, dialect: b.Dialect}); err != nil {
		if rbErr := sqlTx.Rollback(); rbErr != nil {
			return errors.Join(err, rbErr)
		}
		return err
	}
	return sqlTx.Commit()
}

// Close releases the connection pool.
func (b *Backend[T]) Close() error { return b.DB.Close() }

type txContext[T any] struct {
	tx      *sql.Tx
	dialect Dialect
}

func (c *txContext[T]) GetTable(tableName string) (Table[T], error) {
	// Inside a transaction the writes join the caller's transaction instead of
	// opening a nested one.
	return NewSQLTable[T](c.tx, TxTxRunner{Exec: c.tx}, tableName, c.dialect)
}

func (c *txContext[T]) Query(sqlText string, args ...any) ([]T, error) {
	rows, err := c.tx.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return ScanRows[T](rows, SchemaOf[T]())
}

func (c *txContext[T]) Exec(sqlText string, args ...any) error {
	_, err := c.tx.Exec(sqlText, args...)
	return err
}

// ---------- driver name registry ----------

var driverRegistry = struct {
	mu    sync.RWMutex
	names map[string]struct{}
}{names: make(map[string]struct{})}

// Register records a driver name; backends call it from init.
func Register(name string) {
	driverRegistry.mu.Lock()
	defer driverRegistry.mu.Unlock()
	driverRegistry.names[name] = struct{}{}
}

// Drivers lists the registered driver names.
func Drivers() []string {
	driverRegistry.mu.RLock()
	defer driverRegistry.mu.RUnlock()

	out := make([]string, 0, len(driverRegistry.names))
	for n := range driverRegistry.names {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
