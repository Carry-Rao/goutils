package kv

import (
	"context"
	"path"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Data struct {
	Value     interface{}
	ExpiresAt time.Time
}

type Client struct {
	client *redis.Client
	data   map[string]Data
	mu     sync.RWMutex
}

type Options struct {
	Memory   bool
	Addr     string
	Password string
	DB       int
	Protocol int
}

func NewClient(options *Options) (*Client, error) {
	client := &Client{}

	if options.Memory {
		client.data = make(map[string]Data)
		return client, nil
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     options.Addr,
		Password: options.Password,
		DB:       options.DB,
		Protocol: options.Protocol,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	client.client = redisClient
	return client, nil
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if c.client == nil {
		c.mu.Lock()
		defer c.mu.Unlock()

		var expiresAt time.Time
		if expiration > 0 {
			expiresAt = time.Now().Add(expiration)
		}
		c.data[key] = Data{
			Value:     value,
			ExpiresAt: expiresAt,
		}
		return nil
	}

	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *Client) Get(ctx context.Context, key string) (interface{}, error) {
	if c.client == nil {
		c.mu.RLock()
		data, ok := c.data[key]
		c.mu.RUnlock()

		if !ok {
			return nil, redis.Nil
		}

		if !data.ExpiresAt.IsZero() && time.Now().After(data.ExpiresAt) {
			c.mu.Lock()
			delete(c.data, key)
			c.mu.Unlock()
			return nil, redis.Nil
		}

		return data.Value, nil
	}

	return c.client.Get(ctx, key).Result()
}

func (c *Client) Del(ctx context.Context, key string) error {
	if c.client == nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.data, key)
		return nil
	}

	return c.client.Del(ctx, key).Err()
}

func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	if c.client == nil {
		c.mu.RLock()
		defer c.mu.RUnlock()

		var keys []string
		now := time.Now()
		for key, data := range c.data {
			if !data.ExpiresAt.IsZero() && now.After(data.ExpiresAt) {
				continue
			}
			if match, _ := path.Match(pattern, key); match {
				keys = append(keys, key)
			}
		}
		return keys, nil
	}

	return c.client.Keys(ctx, pattern).Result()
}

func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
