package sqlite

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Carry-Rao/goutils/database"
)

type User struct {
	ID    int `db:",primary"`
	Name  string
	Email string            `db:"email_address"`
	Age   int               `db:",null"`
	Tags  []string          `db:",child"`
	Meta  map[string]string `db:",child"`
}

// plain has no collections, for exercising the scalar-only paths.
type plain struct {
	ID   int `db:",primary"`
	Name string
}

func openDB[T any](t *testing.T) *Database[T] {
	t.Helper()
	db, err := NewDatabase[T](map[string]string{"path": filepath.Join(t.TempDir(), "t.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func open(t *testing.T) *Database[User] { return openDB[User](t) }

func table(t *testing.T, db *Database[User]) database.Table[User] {
	t.Helper()
	tbl, err := db.GetTable("users")
	if err != nil {
		t.Fatal(err)
	}
	return tbl
}

func names(rows []User) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Name
	}
	return out
}

func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func seed(t *testing.T, tbl database.Table[User]) {
	t.Helper()
	for _, u := range []User{
		{ID: 3, Name: "carol", Age: 30, Tags: []string{"beta"}},
		{ID: 1, Name: "alice", Age: 20, Tags: []string{"vip", "beta"},
			Meta: map[string]string{"theme": "dark"}},
		{ID: 4, Name: "dave", Age: 40, Tags: []string{"beta", "gamma"}},
		{ID: 2, Name: "bob", Age: 25},
	} {
		if err := tbl.Ins(u, database.Options{}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInsAndGet(t *testing.T) {
	tbl := table(t, open(t))

	in := User{ID: 1, Name: "alice", Email: "a@x.com", Age: 30,
		Tags: []string{"vip", "beta"},
		Meta: map[string]string{"theme": "dark"}}
	if err := tbl.Ins(in, database.Options{}); err != nil {
		t.Fatal(err)
	}

	got, err := tbl.Get(User{ID: 1}, database.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 row, got %d", len(got))
	}
	u := got[0]
	if u.Name != "alice" || u.Email != "a@x.com" || u.Age != 30 {
		t.Errorf("scalars wrong: %+v", u)
	}
	if len(u.Tags) != 2 || u.Meta["theme"] != "dark" {
		t.Errorf("collections wrong: %+v", u)
	}
}

// A record with no collections gets empty ones, not nil-map surprises.
func TestGetEmptyCollections(t *testing.T) {
	tbl := table(t, open(t))
	if err := tbl.Ins(User{ID: 1, Name: "solo"}, database.Options{}); err != nil {
		t.Fatal(err)
	}
	got, err := tbl.Get(User{ID: 1}, database.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 row, got %d", len(got))
	}
	if got[0].Tags == nil || len(got[0].Tags) != 0 {
		t.Errorf("Tags = %v, want empty non-nil", got[0].Tags)
	}
	if got[0].Meta == nil || len(got[0].Meta) != 0 {
		t.Errorf("Meta = %v, want empty non-nil", got[0].Meta)
	}
}

func TestScalarConditions(t *testing.T) {
	tbl := table(t, open(t))
	seed(t, tbl)

	got, err := tbl.Get(User{}, database.Options{
		Where: []database.Condition{database.Gte("Age", 25)},
		Order: database.Ascending("Age"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !same(names(got), []string{"bob", "carol", "dave"}) {
		t.Errorf("Gte: %v", names(got))
	}

	got, _ = tbl.Get(User{}, database.Options{
		Where: []database.Condition{database.In("ID", 1, 3)},
		Order: database.Ascending("ID"),
	})
	if !same(names(got), []string{"alice", "carol"}) {
		t.Errorf("In: %v", names(got))
	}

	got, _ = tbl.Get(User{}, database.Options{
		Where: []database.Condition{
			database.Gt("Age", 20),
			database.Lt("Age", 40),
		},
		Order: database.Ascending("Age"),
	})
	if !same(names(got), []string{"bob", "carol"}) {
		t.Errorf("range: %v", names(got))
	}

	got, _ = tbl.Get(User{}, database.Options{
		Where: []database.Condition{database.Like("Name", "a%")},
		Order: database.Ascending("Name"),
	})
	if !same(names(got), []string{"alice"}) {
		t.Errorf("Like: %v", names(got))
	}

	got, _ = tbl.Get(User{}, database.Options{
		Where: []database.Condition{database.Between("Age", 25, 35)},
		Order: database.Ascending("Age"),
	})
	if !same(names(got), []string{"bob", "carol"}) {
		t.Errorf("Between: %v", names(got))
	}
}

func TestCollectionConditions(t *testing.T) {
	tbl := table(t, open(t))
	seed(t, tbl)

	cases := []struct {
		name string
		cond database.Condition
		want []string
	}{
		{"Contains vip", database.Contains("Tags", "vip"), []string{"alice"}},
		{"Contains beta", database.Contains("Tags", "beta"),
			[]string{"alice", "carol", "dave"}},
		{"ContainsAll", database.ContainsAll("Tags", "beta", "gamma"), []string{"dave"}},
		{"ContainsAny", database.ContainsAny("Tags", "vip", "gamma"),
			[]string{"alice", "dave"}},
		{"ContainsKV", database.ContainsKV("Meta", "theme", "dark"), []string{"alice"}},
		{"Contains key", database.Contains("Meta", "theme"), []string{"alice"}},
	}

	for _, c := range cases {
		got, err := tbl.Get(User{}, database.Options{
			Where: []database.Condition{c.cond},
			Order: database.Ascending("Name"),
		})
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !same(names(got), c.want) {
			t.Errorf("%s: got %v, want %v", c.name, names(got), c.want)
		}
	}
}

func TestSetAssignsScalars(t *testing.T) {
	tbl := table(t, open(t))
	if err := tbl.Ins(User{ID: 1, Name: "alice", Age: 20}, database.Options{}); err != nil {
		t.Fatal(err)
	}

	err := tbl.Set(User{ID: 1}, User{ID: 1, Name: "alice2", Age: 21}, database.Options{})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{ID: 1}, database.Options{})
	if got[0].Name != "alice2" || got[0].Age != 21 {
		t.Errorf("got %+v", got[0])
	}
}

func TestSetSkip(t *testing.T) {
	tbl := table(t, open(t))
	if err := tbl.Ins(User{ID: 1, Name: "alice", Age: 20}, database.Options{}); err != nil {
		t.Fatal(err)
	}
	err := tbl.Set(User{ID: 1}, User{ID: 1, Name: "new", Age: 99},
		database.Options{Skip: []string{"Age"}})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{ID: 1}, database.Options{})
	if got[0].Name != "new" {
		t.Errorf("Name should change: %+v", got[0])
	}
	if got[0].Age != 20 {
		t.Errorf("Age should be skipped: %+v", got[0])
	}
}

// A collection named in Values uses its operation; the model's own value is
// ignored for that field.
func TestSetCollectionOps(t *testing.T) {
	tbl := table(t, open(t))
	if err := tbl.Ins(User{ID: 1, Tags: []string{"a", "b"},
		Meta: map[string]string{"x": "1"}}, database.Options{}); err != nil {
		t.Fatal(err)
	}

	err := tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
		Values: []database.Value{
			database.Union("Tags", []string{"c"}),
			database.Union("Meta", map[string]string{"y": "2"}),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{ID: 1}, database.Options{})
	if len(got[0].Tags) != 3 {
		t.Errorf("Union Tags: %v", got[0].Tags)
	}
	if len(got[0].Meta) != 2 {
		t.Errorf("Union Meta: %v", got[0].Meta)
	}

	err = tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
		Values: []database.Value{database.Difference("Tags", []string{"b"})},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ = tbl.Get(User{ID: 1}, database.Options{})
	for _, tag := range got[0].Tags {
		if tag == "b" {
			t.Errorf("b should be gone: %v", got[0].Tags)
		}
	}

	err = tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
		Values: []database.Value{database.Clear("Tags")},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ = tbl.Get(User{ID: 1}, database.Options{})
	if len(got[0].Tags) != 0 {
		t.Errorf("Clear: %v", got[0].Tags)
	}
	// Clearing one collection must not disturb the other.
	if len(got[0].Meta) != 2 {
		t.Errorf("Meta should be untouched: %v", got[0].Meta)
	}
}

// Set replaces a collection wholesale when Values does not mention it.
func TestSetReplacesCollection(t *testing.T) {
	tbl := table(t, open(t))
	if err := tbl.Ins(User{ID: 1, Tags: []string{"a", "b"}}, database.Options{}); err != nil {
		t.Fatal(err)
	}
	err := tbl.Set(User{ID: 1}, User{ID: 1, Tags: []string{"z"}}, database.Options{})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{ID: 1}, database.Options{})
	if len(got[0].Tags) != 1 || got[0].Tags[0] != "z" {
		t.Errorf("want [z], got %v", got[0].Tags)
	}
}

func TestDel(t *testing.T) {
	tbl := table(t, open(t))
	seed(t, tbl)

	err := tbl.Del(User{}, database.Options{
		Where: []database.Condition{database.Eq("Name", "bob")},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{}, database.Options{Order: database.Ascending("ID")})
	if len(got) != 3 {
		t.Errorf("want 3 remaining, got %d", len(got))
	}
}

// Collection rows must be gone once the parent is deleted.
func TestDelRemovesCollections(t *testing.T) {
	db := open(t)
	tbl := table(t, db)
	if err := tbl.Ins(User{ID: 1, Tags: []string{"ghost"}}, database.Options{}); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Del(User{ID: 1}, database.Options{}); err != nil {
		t.Fatal(err)
	}

	// The collection row for the deleted parent must no longer be reachable.
	got, err := tbl.Get(User{}, database.Options{
		Where: []database.Condition{database.Contains("Tags", "ghost")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("orphan collection row still matches: %v", names(got))
	}
}

func TestTxRollback(t *testing.T) {
	db := open(t)
	tbl := table(t, db)
	if err := tbl.Ins(User{ID: 1, Name: "keep"}, database.Options{}); err != nil {
		t.Fatal(err)
	}

	err := db.Tx(func(tx database.Tx[User]) error {
		tt, err := tx.GetTable("users")
		if err != nil {
			return err
		}
		if err := tt.Ins(User{ID: 2, Name: "doomed"}, database.Options{}); err != nil {
			return err
		}
		return errRollback
	})
	if err == nil {
		t.Fatal("expected the sentinel error")
	}

	got, _ := tbl.Get(User{}, database.Options{})
	if len(got) != 1 || got[0].ID != 1 {
		t.Errorf("rollback failed, rows: %v", names(got))
	}
}

func TestTxCommit(t *testing.T) {
	db := open(t)
	tbl := table(t, db)

	err := db.Tx(func(tx database.Tx[User]) error {
		tt, err := tx.GetTable("users")
		if err != nil {
			return err
		}
		return tt.Ins(User{ID: 5, Name: "committed"}, database.Options{})
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{ID: 5}, database.Options{})
	if len(got) != 1 {
		t.Errorf("commit failed: %v", names(got))
	}
}

// A model without collections must still work; no child tables are created.
func TestModelWithoutCollections(t *testing.T) {
	db := openDB[plain](t)
	tbl, err := db.GetTable("things")
	if err != nil {
		t.Fatal(err)
	}
	if err := tbl.Ins(plain{ID: 1, Name: "x"}, database.Options{}); err != nil {
		t.Fatal(err)
	}
	got, err := tbl.Get(plain{ID: 1}, database.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "x" {
		t.Errorf("got %+v", got)
	}
}

// A model with no primary key cannot be keyed, so it must be rejected up front.
func TestModelWithoutPrimaryKey(t *testing.T) {
	type noKey struct{ Name string }
	db := openDB[noKey](t)
	if _, err := db.GetTable("x"); err == nil {
		t.Error("expected an error for a model with no primary key")
	}
}

// Operator/field mismatches must be caught before any SQL runs.
func TestOptionValidation(t *testing.T) {
	tbl := table(t, open(t))
	cases := []struct {
		name string
		opts database.Options
	}{
		{"Eq on a collection", database.Options{Where: []database.Condition{database.Eq("Tags", "a")}}},
		{"Contains on a scalar", database.Options{Where: []database.Condition{database.Contains("Name", "a")}}},
		{"ContainsAll on a map", database.Options{Where: []database.Condition{database.ContainsAll("Meta", "a")}}},
		{"ContainsKV on a slice", database.Options{Where: []database.Condition{database.ContainsKV("Tags", "k", "v")}}},
		{"Union on a scalar", database.Options{Values: []database.Value{database.Union("Name", "x")}}},
		{"unknown field", database.Options{Where: []database.Condition{database.Eq("nope", 1)}}},
		{"unknown order", database.Options{Order: database.Ascending("nope")}},
		{"negative limit", database.Options{Limit: -1}},
	}
	for _, c := range cases {
		if _, err := tbl.Get(User{}, c.opts); err == nil {
			t.Errorf("%s: expected an error", c.name)
		}
	}
}

func TestQuery(t *testing.T) {
	db := open(t)
	tbl := table(t, db)
	seed(t, tbl)

	got, err := db.Query("SELECT * FROM users WHERE `Age` >= ?", 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("want 2, got %d", len(got))
	}

	if err := db.Exec("UPDATE users SET Name = ? WHERE ID = ?", "renamed", 1); err != nil {
		t.Fatal(err)
	}
	got, _ = tbl.Get(User{ID: 1}, database.Options{})
	if got[0].Name != "renamed" {
		t.Errorf("Exec did not apply: %+v", got[0])
	}
}

func TestDeleteTable(t *testing.T) {
	db := open(t)
	tbl := table(t, db)
	if err := tbl.Ins(User{ID: 1}, database.Options{}); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteTable("users"); err != nil {
		t.Fatal(err)
	}
	// Recreating must work from scratch.
	if _, err := db.GetTable("users"); err != nil {
		t.Fatal(err)
	}
}

func TestLimitOffset(t *testing.T) {
	tbl := table(t, open(t))
	seed(t, tbl)

	got, _ := tbl.Get(User{}, database.Options{Order: database.Ascending("ID"), Limit: 2})
	if !same(names(got), []string{"alice", "bob"}) {
		t.Errorf("limit: %v", names(got))
	}

	got, _ = tbl.Get(User{}, database.Options{
		Order: database.Ascending("ID"), Limit: 2, Offset: 2,
	})
	if !same(names(got), []string{"carol", "dave"}) {
		t.Errorf("limit+offset: %v", names(got))
	}

	got, _ = tbl.Get(User{}, database.Options{Order: database.Ascending("ID"), Offset: 3})
	if !same(names(got), []string{"dave"}) {
		t.Errorf("offset only: %v", names(got))
	}
}

var errRollback = errors.New("rollback on purpose")

// A scalar named in Values is written from updated, and other scalars are not.
func TestSetAssignSingleColumn(t *testing.T) {
	tbl := table(t, open(t))
	if err := tbl.Ins(User{ID: 1, Name: "alice", Age: 20}, database.Options{}); err != nil {
		t.Fatal(err)
	}
	err := tbl.Set(User{ID: 1}, User{ID: 1, Name: "bob", Age: 99},
		database.Options{Values: []database.Value{database.Assign("Name", "bob")}})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{ID: 1}, database.Options{})
	if got[0].Name != "bob" {
		t.Errorf("Name should change: %+v", got[0])
	}
	if got[0].Age != 20 {
		t.Errorf("Age is not in Values, so it must be untouched: %+v", got[0])
	}
}

// Skip wins over Values, even for a collection.
func TestSetSkipBeatsValues(t *testing.T) {
	tbl := table(t, open(t))
	if err := tbl.Ins(User{ID: 1, Name: "alice", Tags: []string{"a"}},
		database.Options{}); err != nil {
		t.Fatal(err)
	}
	err := tbl.Set(User{ID: 1}, User{ID: 1, Name: "bob"},
		database.Options{
			Skip:   []string{"Name", "Tags"},
			Values: []database.Value{database.Assign("Name", "ignored"), database.Clear("Tags")},
		})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{ID: 1}, database.Options{})
	if got[0].Name != "alice" {
		t.Errorf("Name was skipped: %+v", got[0])
	}
	if len(got[0].Tags) != 1 {
		t.Errorf("Tags was skipped: %+v", got[0])
	}
}
