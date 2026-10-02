package memory

import (
	"testing"

	"github.com/Carry-Rao/goutils/database/cache"
)

type memUser struct {
	ID   int `db:",primary"`
	Name string
	Tags []string          `db:",child"`
	Meta map[string]string `db:",child"`
}

func newDB(t *testing.T) (*Database[memUser], cache.Table[memUser]) {
	t.Helper()
	db, err := NewDatabase[memUser](nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create("users", map[string]cache.Config{"id": {PrimaryKey: true}}); err != nil {
		t.Fatal(err)
	}
	tbl, err := db.GetTable("users", memUser{})
	if err != nil {
		t.Fatal(err)
	}
	if tbl == nil {
		t.Fatal("nil table")
	}
	return db, tbl
}

func TestGetTableNotCreated(t *testing.T) {
	db, _ := NewDatabase[memUser](nil)
	tbl, err := db.GetTable("nope", memUser{})
	if err != nil || tbl != nil {
		t.Fatalf("want (nil, nil), got (%v, %v)", tbl, err)
	}
}

func TestCRUD(t *testing.T) {
	_, tbl := newDB(t)

	in := memUser{ID: 1, Name: "alice", Tags: []string{"a", "b"},
		Meta: map[string]string{"k": "v"}}
	if err := tbl.Ins(in); err != nil {
		t.Fatal(err)
	}

	got, err := tbl.Get(memUser{ID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if got[0].Name != "alice" || len(got[0].Tags) != 2 || got[0].Meta["k"] != "v" {
		t.Errorf("got %+v", got[0])
	}

	if err := tbl.Set(memUser{ID: 1}, memUser{ID: 1, Name: "bob"}); err != nil {
		t.Fatal(err)
	}
	got, _ = tbl.Get(memUser{ID: 1})
	if got[0].Name != "bob" {
		t.Errorf("after Set: %+v", got[0])
	}

	if err := tbl.Del(memUser{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if got, _ := tbl.Get(memUser{ID: 1}); len(got) != 0 {
		t.Errorf("after Del: %+v", got)
	}
}

// A changed primary key moves the record instead of duplicating it.
func TestSetMovesOnKeyChange(t *testing.T) {
	_, tbl := newDB(t)
	if err := tbl.Ins(memUser{ID: 1, Name: "alice"}); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Set(memUser{ID: 1}, memUser{ID: 99, Name: "alice"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := tbl.Get(memUser{ID: 1}); len(got) != 0 {
		t.Errorf("old key should be gone, got %+v", got)
	}
	if got, _ := tbl.Get(memUser{ID: 99}); len(got) != 1 {
		t.Errorf("new key missing, got %+v", got)
	}
}

// Without a primary key there is no way to address a record.
func TestMissingPrimaryKey(t *testing.T) {
	type noKey struct{ Name string }
	db, err := NewDatabase[noKey](nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create("x", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetTable("x", noKey{}); err == nil {
		t.Error("expected an error for a model with no primary key")
	}
}

func TestDrivers(t *testing.T) {
	names := cache.Drivers()
	found := false
	for _, n := range names {
		if n == "memory" {
			found = true
		}
	}
	if !found {
		t.Errorf("memory not registered: %v", names)
	}
}
