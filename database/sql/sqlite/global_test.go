package sqlite_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Carry-Rao/goutils/database"
	"github.com/Carry-Rao/goutils/database/sql/sqlite"
)

// owner stands in for the auth model: a slice of ownership claims where each
// value must belong to exactly one row across the whole table.
type owner struct {
	ID    int64    `db:",primary"`
	Name  string   `db:"name"`
	OAuth []string `db:",child,global"`
}

// OnlyUnique is the contrast case: the default per-parent key, which lets the
// same value sit under two rows.
type onlyUnique struct {
	ID     int64    `db:",primary"`
	Name   string   `db:"name"`
	Scopes []string `db:",child"`
}

func openOwner(t *testing.T) (database.Database[owner], func()) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "owner.db")
	db, err := sqlite.NewDatabase[owner](map[string]string{"path": path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return db, func() { db.Close() }
}

// A second row claiming a value another row already holds must be reported as
// ErrDuplicate, and it must be distinguishable from any other failure.
func TestGlobalUniqueRejectsSecondOwner(t *testing.T) {
	db, done := openOwner(t)
	defer done()

	tbl, err := db.GetTable("user")
	if err != nil {
		t.Fatalf("table: %v", err)
	}

	if err := tbl.Ins(owner{ID: 1, Name: "alice", OAuth: []string{"github:1"}}, database.Options{}); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	err = tbl.Ins(owner{ID: 2, Name: "bob", OAuth: []string{"github:1"}}, database.Options{})
	if err == nil {
		t.Fatal("second owner of github:1 was accepted")
	}
	if !errors.Is(err, database.ErrDuplicate) {
		t.Fatalf("want ErrDuplicate, got %v", err)
	}
}

// The same value is fine inside one row: the constraint is about ownership
// across rows, not about repeating an element.
func TestGlobalUniqueAllowsDistinctValuesPerRow(t *testing.T) {
	db, done := openOwner(t)
	defer done()

	tbl, _ := db.GetTable("user")
	if err := tbl.Ins(owner{ID: 1, Name: "alice", OAuth: []string{"github:1", "google:2"}}, database.Options{}); err != nil {
		t.Fatalf("alice: %v", err)
	}
	if err := tbl.Ins(owner{ID: 2, Name: "bob", OAuth: []string{"gitlab:3"}}, database.Options{}); err != nil {
		t.Fatalf("bob: %v", err)
	}

	got, err := tbl.Get(owner{}, database.Options{
		Where: []database.Condition{database.Eq("ID", 2)},
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got) != 1 || got[0].Name != "bob" || len(got[0].OAuth) != 1 || got[0].OAuth[0] != "gitlab:3" {
		t.Fatalf("unexpected row: %+v", got)
	}

	// The ownership question the auth model actually asks: who holds this
	// third-party identity? It is a collection lookup, so it must go through
	// Contains — Eq on a collection is rejected at validation.
	owners, err := tbl.Get(owner{}, database.Options{
		Where: []database.Condition{database.Contains("OAuth", "github:1")},
	})
	if err != nil {
		t.Fatalf("lookup by claim: %v", err)
	}
	if len(owners) != 1 || owners[0].Name != "alice" {
		t.Fatalf("github:1 should belong to alice only, got %+v", owners)
	}
}

// Rewriting a row must not collide with itself: Set addresses the row by its
// primary key, and that row's existing claims are its own.
func TestGlobalUniqueIgnoresSelfOnUpdate(t *testing.T) {
	db, done := openOwner(t)
	defer done()

	tbl, _ := db.GetTable("user")
	if err := tbl.Ins(owner{ID: 1, Name: "alice", OAuth: []string{"github:1"}}, database.Options{}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := tbl.Set(
		owner{ID: 1}, owner{ID: 1, Name: "alice2", OAuth: []string{"github:1"}},
		database.Options{},
	); err != nil {
		t.Fatalf("set to the same claim: %v", err)
	}
}

// The default per-parent uniqueness is unchanged: two rows may hold the same
// value, because the key is (column, parent).
func TestDefaultChildUniqueStillPerParent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scopes.db")
	db, err := sqlite.NewDatabase[onlyUnique](map[string]string{"path": path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	tbl, _ := db.GetTable("user")
	if err := tbl.Ins(onlyUnique{ID: 1, Name: "alice", Scopes: []string{"read"}}, database.Options{}); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := tbl.Ins(onlyUnique{ID: 2, Name: "bob", Scopes: []string{"read"}}, database.Options{}); err != nil {
		t.Fatalf("second owner of the same scope was rejected: %v", err)
	}
}

// Without the global constraint the same insert must succeed, which proves the
// rejection above comes from the constraint and not from something incidental.
func TestWithoutGlobalNoDuplicateError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain.db")
	db, err := sqlite.NewDatabase[owner2](map[string]string{"path": path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	tbl, _ := db.GetTable("user")
	if err := tbl.Ins(owner2{ID: 1, OAuth: []string{"github:1"}}, database.Options{}); err != nil {
		t.Fatalf("first: %v", err)
	}
	err = tbl.Ins(owner2{ID: 2, Name: "bob", OAuth: []string{"github:1"}}, database.Options{})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if errors.Is(err, database.ErrDuplicate) {
		t.Fatal("ErrDuplicate reported where no unique constraint exists")
	}
}

type owner2 struct {
	ID    int64    `db:",primary"`
	Name  string   `db:"name"`
	OAuth []string `db:",child"`
}
