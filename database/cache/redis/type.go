package redis

import (
	"context"
	"sync"

	"github.com/go-redis/redis/v8"

	"github.com/Carry-Rao/goutils/database/cache"
)

type Database[T any] struct {
	client     *redis.Client
	tables     map[string]struct{}
	mu         sync.RWMutex
	cacheField map[string]string
}

func NewDatabase[T any](cfg map[string]string) (*Database[T], error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg["addr"],
		Password: cfg["password"],
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &Database[T]{
		client:     client,
		tables:     make(map[string]struct{}),
		cacheField: make(map[string]string),
	}, nil
}

func (r *Database[T]) Close() error { return r.client.Close() }

func (r *Database[T]) Create(tableName string, config map[string]cache.Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tables[tableName] = struct{}{}
	for k, v := range config {
		if v.PrimaryKey {
			r.cacheField[tableName] = k
			break
		}
	}
	return nil
}

func (r *Database[T]) GetTable(tableName string, _ T) (cache.Table[T], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.tables[tableName]; !ok {
		return nil, nil
	}
	schema := cache.SchemaOf[T]()
	pk, err := schema.PrimaryKey()
	if err != nil {
		return nil, err
	}
	return &Table[T]{db: r, tableName: tableName, schema: schema, pk: pk}, nil
}

func (r *Database[T]) DeleteTable(tableName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tables, tableName)
	delete(r.cacheField, tableName)
	return nil
}
