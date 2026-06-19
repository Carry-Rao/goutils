package api

type Database interface {
	Create(string, map[string]Config) error
	GetTable(string, any) (Table, error)
	DeleteTable(string) error
}

type Config struct {
	Type       string
	NullAble   bool
	Identity   bool
	PrimaryKey bool
	Unique     bool
}
