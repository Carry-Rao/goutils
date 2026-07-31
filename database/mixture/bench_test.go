package mixture

import (
	"testing"

	"github.com/Carry-Rao/goutils/database/api"
	"github.com/Carry-Rao/goutils/database/bloom"
	"github.com/Carry-Rao/goutils/database/memory"
)

type benchRec struct {
	ID   int    `db:"id,primary"`
	Name string `db:"name"`
}

func benchMixture(b *testing.B) *Table {
	b.Helper()
	bloomDB, _ := bloom.NewDatabase(nil)
	memDB, _ := memory.NewDatabase(nil)

	mix := &Database{}
	mix.Add(bloomDB, Continue)
	mix.Add(memDB, Continue)

	_ = mix.Create("recs", map[string]api.Config{"id": {PrimaryKey: true}})
	tbl, err := mix.GetTable("recs", &benchRec{})
	if err != nil {
		b.Fatal(err)
	}
	table := tbl.(*Table)
	for i := 0; i < b.N; i++ {
		_ = table.Ins(&benchRec{ID: i, Name: "n"}, 0)
	}
	return table
}

func BenchmarkIns(b *testing.B) {
	bloomDB, _ := bloom.NewDatabase(nil)
	memDB, _ := memory.NewDatabase(nil)
	mix := &Database{}
	mix.Add(bloomDB, Continue)
	mix.Add(memDB, Continue)
	_ = mix.Create("recs", map[string]api.Config{"id": {PrimaryKey: true}})
	tbl, _ := mix.GetTable("recs", &benchRec{})
	table := tbl.(*Table)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.Ins(&benchRec{ID: i, Name: "n"}, 0)
	}
}

func BenchmarkGetHit(b *testing.B) {
	table := benchMixture(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = table.Get(&benchRec{ID: i}, nil, 0)
	}
}

func BenchmarkSet(b *testing.B) {
	table := benchMixture(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.Set(&benchRec{ID: i, Name: "u"}, nil, 0)
	}
}

func BenchmarkDel(b *testing.B) {
	table := benchMixture(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.Del(&benchRec{ID: i}, nil, 0)
	}
}
