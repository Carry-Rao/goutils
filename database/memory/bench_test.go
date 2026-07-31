package memory

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Carry-Rao/goutils/database/api"
)

type benchUser struct {
	ID   int    `db:"id,primary"`
	Name string `db:"name"`
	Age  int    `db:"age"`
}

func benchMemory(b *testing.B) *Table {
	b.Helper()
	db, _ := NewDatabase(nil)
	_ = db.Create("users", map[string]api.Config{
		"id": {PrimaryKey: true},
	})
	tbl, err := db.GetTable("users", &benchUser{})
	if err != nil {
		b.Fatal(err)
	}
	table := tbl.(*Table)
	for i := 0; i < b.N; i++ {
		_ = table.Ins(&benchUser{ID: i, Name: "user", Age: 20}, 0)
	}
	return table
}

func BenchmarkIns(b *testing.B) {
	db, _ := NewDatabase(nil)
	_ = db.Create("users", map[string]api.Config{"id": {PrimaryKey: true}})
	tbl, _ := db.GetTable("users", &benchUser{})
	table := tbl.(*Table)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.Ins(&benchUser{ID: i, Name: "user", Age: 20}, 0)
	}
}

func BenchmarkGet(b *testing.B) {
	table := benchMemory(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = table.Get(&benchUser{ID: i}, nil, 0)
	}
}

func BenchmarkSet(b *testing.B) {
	table := benchMemory(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.Set(&benchUser{ID: i, Name: "updated", Age: 21}, nil, 0)
	}
}

func BenchmarkDel(b *testing.B) {
	table := benchMemory(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.Del(&benchUser{ID: i}, nil, 0)
	}
}

func BenchmarkGetMiss(b *testing.B) {
	table := benchMemory(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = table.Get(&benchUser{ID: b.N + i + 1}, nil, 0)
	}
}

// ============ Standard library map baseline ============

func BenchmarkStdlibMapIns(b *testing.B) {
	m := make(map[string]any)
	var mu sync.RWMutex
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m[fmt.Sprintf("users_%d", i)] = &benchUser{ID: i, Name: "user", Age: 20}
		mu.Unlock()
	}
}

func BenchmarkStdlibMapGet(b *testing.B) {
	m := make(map[string]any, b.N)
	var mu sync.RWMutex
	for i := 0; i < b.N; i++ {
		m[fmt.Sprintf("users_%d", i)] = &benchUser{ID: i, Name: "user", Age: 20}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		_ = m[fmt.Sprintf("users_%d", i)]
		mu.RUnlock()
	}
}

func BenchmarkStdlibMapSet(b *testing.B) {
	m := make(map[string]any, b.N)
	var mu sync.RWMutex
	for i := 0; i < b.N; i++ {
		m[fmt.Sprintf("users_%d", i)] = &benchUser{ID: i, Name: "user", Age: 20}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m[fmt.Sprintf("users_%d", i)] = &benchUser{ID: i, Name: "updated", Age: 21}
		mu.Unlock()
	}
}

func BenchmarkStdlibMapDel(b *testing.B) {
	m := make(map[string]any, b.N)
	var mu sync.RWMutex
	for i := 0; i < b.N; i++ {
		m[fmt.Sprintf("users_%d", i)] = &benchUser{ID: i, Name: "user", Age: 20}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		delete(m, fmt.Sprintf("users_%d", i))
		mu.Unlock()
	}
}

var _ = time.Second
