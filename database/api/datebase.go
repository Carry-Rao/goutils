package api

type Database interface {
	Create(tableName string, config map[string]Config) error
	GetTable(tableName string) (Table, error)
	DeleteTable(tableName string) error
}

type Config struct {
	Type       string
	NullAble   bool
	Identity   bool
	PrimaryKey bool
	Unique     bool
}
