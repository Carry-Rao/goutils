package bloom

import (
	"sync"
)

type Database struct {
	data     map[string]map[string]any
	bits     map[string][]bool
	cacheKey map[string]string
	mu       sync.RWMutex
}

func NewDatabase(_ map[string]string) (*Database, error) {
	return &Database{
		data:     make(map[string]map[string]any),
		bits:     make(map[string][]bool),
		cacheKey: make(map[string]string),
	}, nil
}
