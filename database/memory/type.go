package memory

import (
	"sync"
	"time"
)

type entry struct {
	data    any
	expires time.Time
}

type Database struct {
	data  map[string]map[string]any
	cache map[string]string
	mu    sync.RWMutex
}

func NewDatabase(cfg map[string]string) (*Database, error) {
	return &Database{
		data:  make(map[string]map[string]any),
		cache: make(map[string]string),
	}, nil
}
