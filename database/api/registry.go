package api

var typeRegistry = map[string]DbType{
	"mysql":      MySQL,
	"postgresql": PostgreSQL,
	"sqlite":     SQLite,
	"redis":      Redis,
	"memory":     Memory,
	"bloom":      Bloom,
	"mixture":    Mixture,
}

func Register(name string, dbType DbType) {
	typeRegistry[name] = dbType
}

func Lookup(name string) (DbType, bool) {
	t, ok := typeRegistry[name]
	return t, ok
}
