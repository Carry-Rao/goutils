package bloom

import "sync"

type tableData struct {
	data     map[string]any
	cacheKey string
	mu       sync.RWMutex
}

type Database struct {
	tables map[string]*tableData
	mu     sync.RWMutex
}

func NewDatabase(_ map[string]string) (*Database, error) {
	return &Database{
		tables: make(map[string]*tableData),
	}, nil
}
