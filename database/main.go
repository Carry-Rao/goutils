package database

import (
	"errors"

	"github.com/Carry-Rao/goutils/database/api"

	"github.com/Carry-Rao/goutils/database/bloom"
	"github.com/Carry-Rao/goutils/database/memory"
	"github.com/Carry-Rao/goutils/database/mixture"
	"github.com/Carry-Rao/goutils/database/mysql"
	"github.com/Carry-Rao/goutils/database/postgresql"
	"github.com/Carry-Rao/goutils/database/redis"
	"github.com/Carry-Rao/goutils/database/sqlite"
)

func NewDatabase(config map[string]string) (api.Database, error) {
	typ := config["type"]
	switch typ {
	case "redis":
		return redis.NewDatabase(config)
	case "memory":
		return memory.NewDatabase(config)
	case "mysql":
		return mysql.NewDatabase(config)
	case "sqlite":
		return sqlite.NewDatabase(config)
	case "postgresql":
		return postgresql.NewDatabase(config)
	case "bloom":
		return bloom.NewDatabase(config)
	case "mixture":
		return mixture.NewDatabase(config)
	}
	return nil, errors.New("unknown database type")
}
