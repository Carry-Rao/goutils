package api

type Factory func(config map[string]string) (Database, error)

var typeRegistry = map[string]DbType{
	"mysql":      MySQL,
	"postgresql": PostgreSQL,
	"sqlite":     SQLite,
	"redis":      Redis,
	"memory":     Memory,
	"bloom":      Bloom,
	"mixture":    Mixture,
}

var factoryRegistry = map[string]Factory{}

func Register(name string, dbType DbType) {
	typeRegistry[name] = dbType
}

func RegisterFactory(name string, factory Factory) {
	factoryRegistry[name] = factory
}

func Lookup(name string) (DbType, bool) {
	t, ok := typeRegistry[name]
	return t, ok
}

func LookupFactory(name string) (Factory, bool) {
	f, ok := factoryRegistry[name]
	return f, ok
}
