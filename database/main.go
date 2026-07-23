package database

import (
	"errors"
	"fmt"

	"github.com/Carry-Rao/goutils/database/api"

	"github.com/Carry-Rao/goutils/database/bloom"
	"github.com/Carry-Rao/goutils/database/memory"
	"github.com/Carry-Rao/goutils/database/mixture"
	"github.com/Carry-Rao/goutils/database/mysql"
	"github.com/Carry-Rao/goutils/database/postgresql"
	"github.com/Carry-Rao/goutils/database/redis"
	"github.com/Carry-Rao/goutils/database/sqlite"
)

func NewDatabase(dbType api.DbType, config map[string]string) (api.Database, error) {
	switch dbType {
	case api.MySQL:
		return mysql.NewDatabase(config)
	case api.PostgreSQL:
		return postgresql.NewDatabase(config)
	case api.SQLite:
		return sqlite.NewDatabase(config)
	case api.Redis:
		return redis.NewDatabase(config)
	case api.Memory:
		return memory.NewDatabase(config)
	case api.Bloom:
		return bloom.NewDatabase(config)
	case api.Mixture:
		return mixture.NewDatabase(config)
	}
	return nil, errors.New(fmt.Sprintf("unknown database type: %d", dbType))
}
