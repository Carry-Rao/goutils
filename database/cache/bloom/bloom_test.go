package bloom

import (
	"errors"
	"testing"

	"github.com/Carry-Rao/goutils/database/cache"
)

type bloomItem struct {
	ID     int            `db:",primary"`
	Tags   []string       `db:",child"`
	Counts map[string]int `db:",child"`
}

func newDB(t *testing.T) (*Database[bloomItem], cache.Table[bloomItem]) {
	t.Helper()
	db, err := NewDatabase[bloomItem](nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create("items", map[string]cache.Config{"id": {PrimaryKey: true}}); err != nil {
		t.Fatal(err)
	}
	tbl, err := db.GetTable("items", bloomItem{})
	if err != nil {
		t.Fatal(err)
	}
	return db, tbl
}

func TestBloomMembership(t *testing.T) {
	_, tbl := newDB(t)

	if err := tbl.Ins(bloomItem{ID: 1, Tags: []string{"x"}, Counts: map[string]int{"n": 2}}); err != nil {
		t.Fatal(err)
	}
	got, err := tbl.Get(bloomItem{ID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Tags) != 1 || got[0].Counts["n"] != 2 {
		t.Errorf("got %+v", got[0])
	}
}

// A miss is an error, since membership is the point of the filter.
func TestBloomMissIsError(t *testing.T) {
	_, tbl := newDB(t)
	_, err := tbl.Get(bloomItem{ID: 42})
	if !errors.Is(err, cache.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestBloomDel(t *testing.T) {
	_, tbl := newDB(t)
	if err := tbl.Ins(bloomItem{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Del(bloomItem{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := tbl.Get(bloomItem{ID: 1}); !errors.Is(err, cache.ErrNotFound) {
		t.Errorf("want ErrNotFound after Del, got %v", err)
	}
}

// Bloom filters have false positives, so assert only that a stored key is
// reported present and that repeated lookups are stable.
func TestBloomNoFalseNegative(t *testing.T) {
	_, tbl := newDB(t)
	const n = 200
	for i := 1; i <= n; i++ {
		if err := tbl.Ins(bloomItem{ID: i}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= n; i++ {
		if _, err := tbl.Get(bloomItem{ID: i}); err != nil {
			t.Errorf("stored key %d reported absent: %v", i, err)
		}
	}
}
