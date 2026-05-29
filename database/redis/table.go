package redis

import (
	"context"
	"encoding/json"
	"fmt"
)

type Table struct {
	db        *Database
	tableName string
	cacheKey  string
}

func (t *Table) Create(data map[string]any) error {
	if t.cacheKey == "" {
		return nil
	}

	idVal, ok := data[t.cacheKey]
	if !ok {
		return nil
	}

	key := fmt.Sprintf("%s_%v", t.tableName, idVal)
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return t.db.client.Set(context.Background(), key, val, 0).Err()
}

func (t *Table) Get(where map[string]any) ([]any, error) {
	if t.cacheKey == "" {
		return nil, nil
	}

	idVal, ok := where[t.cacheKey]
	if !ok {
		return nil, nil
	}

	key := fmt.Sprintf("%s_%v", t.tableName, idVal)
	data, err := t.db.client.Get(context.Background(), key).Bytes()
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	res := make([]any, 0, len(m))
	for _, v := range m {
		res = append(res, v)
	}
	return res, nil
}

func (t *Table) Set(data map[string]any) error {
	if t.cacheKey == "" {
		return nil
	}

	idVal, ok := data[t.cacheKey]
	if !ok {
		return fmt.Errorf("missing cache key")
	}

	key := fmt.Sprintf("%s_%v", t.tableName, idVal)
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return t.db.client.Set(context.Background(), key, val, 0).Err()
}

func (t *Table) Delete(where map[string]any) error {
	if t.cacheKey == "" {
		return nil
	}

	idVal, ok := where[t.cacheKey]
	if !ok {
		return nil
	}

	key := fmt.Sprintf("%s_%v", t.tableName, idVal)
	return t.db.client.Del(context.Background(), key).Err()
}
