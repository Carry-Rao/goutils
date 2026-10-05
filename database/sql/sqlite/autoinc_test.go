package sqlite_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/Carry-Rao/goutils/database"
	"github.com/Carry-Rao/goutils/database/sql/sqlite"
)

type autoIncModel struct {
	ID   int    `db:",primary,autoinc"`
	Name string `db:"name"`
}

// An auto-increment key is kept out of Scalars so it stays out of the INSERT
// column list. That also means the table builder cannot see it, and the key used
// to be declared as a plain nullable column: the CREATE succeeded, the insert
// succeeded, and every read saw NULL. The DDL is asserted as well as the
// round trip, because the failure mode was silent at every step but the read.
func TestAutoIncKeyIsDeclaredAsPrimaryKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai.db")
	db, err := sqlite.NewDatabase[autoIncModel](map[string]string{"path": path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.GetTable("t"); err != nil {
		t.Fatalf("table: %v", err)
	}

	raw, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	defer raw.Close()

	var ddl string
	if err := raw.QueryRow("SELECT sql FROM sqlite_master WHERE name = 't'").Scan(&ddl); err != nil {
		t.Fatalf("read ddl: %v", err)
	}
	if !contains(ddl, "PRIMARY KEY AUTOINCREMENT") {
		t.Errorf("auto-increment clause missing from %q", ddl)
	}
	// Two primary keys in one CREATE TABLE is rejected outright, so this also
	// covers the table-level clause not being emitted alongside the inline one.
	if n := countOccurrences(ddl, "PRIMARY KEY"); n != 1 {
		t.Errorf("%d PRIMARY KEY clauses in %q, want 1", n, ddl)
	}
}

// The value the database assigns has to come back on a primary-key lookup.
func TestAutoIncAssignsAndReadsBack(t *testing.T) {
	db, err := sqlite.NewDatabase[autoIncModel](map[string]string{
		"path": filepath.Join(t.TempDir(), "ai2.db"),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	tbl, err := db.GetTable("t")
	if err != nil {
		t.Fatalf("table: %v", err)
	}
	// No ID supplied: the database assigns it.
	if err := tbl.Ins(autoIncModel{Name: "first"}, database.Options{}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := tbl.Ins(autoIncModel{Name: "second"}, database.Options{}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rows, err := tbl.Get(autoIncModel{}, database.Options{
		Where: []database.Condition{database.Eq("ID", 1)},
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].ID != 1 || rows[0].Name != "first" {
		t.Errorf("row = %+v", rows[0])
	}
}

// A non-auto-increment primary key still gets the table-level clause, and still
// refuses a duplicate.
func TestPlainPrimaryKeyKeepsTableLevelClause(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pk.db")
	db, err := sqlite.NewDatabase[owner](map[string]string{"path": path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.GetTable("user"); err != nil {
		t.Fatalf("table: %v", err)
	}

	raw, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var ddl string
	if err := raw.QueryRow("SELECT sql FROM sqlite_master WHERE name = 'user'").Scan(&ddl); err != nil {
		t.Fatal(err)
	}
	// owner has a plain int64 primary key, so the clause is table-level and the
	// key is a single declared column.
	if !contains(ddl, "PRIMARY KEY (") {
		t.Errorf("table-level primary key missing from %q", ddl)
	}

	tbl, _ := db.GetTable("user")
	if err := tbl.Ins(owner{ID: 1, Name: "alice"}, database.Options{}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	err = tbl.Ins(owner{ID: 1, Name: "impostor"}, database.Options{})
	if err == nil {
		t.Fatal("duplicate primary key accepted")
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func countOccurrences(haystack, needle string) int {
	n := 0
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			n++
		}
	}
	return n
}

var _ = database.Options{}
