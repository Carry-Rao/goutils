package bloom

import (
	"strconv"
	"testing"

	"github.com/Carry-Rao/goutils/database/api"
)

type benchItem struct {
	ID    int    `db:"id,primary"`
	Value string `db:"value"`
}

func benchBloom(b *testing.B) *Table {
	b.Helper()
	db, _ := NewDatabase(nil)
	_ = db.Create("items", map[string]api.Config{
		"id": {PrimaryKey: true},
	})
	tbl, err := db.GetTable("items", &benchItem{})
	if err != nil {
		b.Fatal(err)
	}
	table := tbl.(*Table)
	for i := 0; i < b.N; i++ {
		_ = table.Ins(&benchItem{ID: i, Value: "v"}, 0)
	}
	return table
}

func BenchmarkIns(b *testing.B) {
	db, _ := NewDatabase(nil)
	_ = db.Create("items", map[string]api.Config{"id": {PrimaryKey: true}})
	tbl, _ := db.GetTable("items", &benchItem{})
	table := tbl.(*Table)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.Ins(&benchItem{ID: i, Value: "v"}, 0)
	}
}

func BenchmarkGetHit(b *testing.B) {
	table := benchBloom(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = table.Get(&benchItem{ID: i}, nil, 0)
	}
}

func BenchmarkGetMiss(b *testing.B) {
	table := benchBloom(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = table.Get(&benchItem{ID: b.N + i + 1}, nil, 0)
	}
}

func BenchmarkContains(b *testing.B) {
	table := benchBloom(b)
	key := "items_1"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		table.contains(key)
	}
}

func BenchmarkAdd(b *testing.B) {
	table := benchBloom(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		table.add("items_" + string(rune(i)))
	}
}

// ============ Standard library map baseline ============

func BenchmarkStdlibMapContains(b *testing.B) {
	m := make(map[string]struct{}, b.N)
	for i := 0; i < b.N; i++ {
		m["items_"+strconv.Itoa(i)] = struct{}{}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m["items_1"]
	}
}

