package redis

import (
	"context"
	"sync"

	"github.com/go-redis/redis/v8"
)

type Database struct {
	client     *redis.Client
	tables     map[string]map[string]any
	mu         sync.RWMutex
	cacheField map[string]string
}

func NewDatabase(cfg map[string]string) (*Database, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg["addr"],
		Password: cfg["password"],
		DB:       0,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &Database{
		client:     client,
		tables:     make(map[string]map[string]any),
		cacheField: make(map[string]string),
	}, nil
}

func (r *Database) Close() error {
	return r.client.Close()
}
