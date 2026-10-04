package json

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Carry-Rao/goutils/database"
)

type User struct {
	ID    int `db:",primary"`
	Name  string
	Email string `db:"email_address"`
	Age   int
	Tags  []string          `db:",child"`
	Meta  map[string]string `db:",child"`
}

var errRollback = errors.New("rollback on purpose")

func open(t *testing.T) *Database[User] {
	t.Helper()
	db, err := NewDatabase[User](map[string]string{
		"filename": filepath.Join(t.TempDir(), "app.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

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

// The file must exist and hold one table keyed by primary key.
func TestFileLayout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.json")
	db, err := NewDatabase[User](map[string]string{"filename": path})
	if err != nil {
		t.Fatal(err)
	}
	tbl, _ := db.GetTable("users")
	if err := tbl.Ins(User{ID: 7, Name: "zoe"}, database.Options{}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Primary key is the object key, so a lookup needs no scan.
	if !contains(string(raw), `"users"`) || !contains(string(raw), `"7"`) {
		t.Errorf("unexpected file contents:\n%s", raw)
	}

	// A second handle on the same file sees the row.
	again, err := NewDatabase[User](map[string]string{"filename": path})
	if err != nil {
		t.Fatal(err)
	}
	t2, _ := again.GetTable("users")
	got, err := t2.Get(User{ID: 7}, database.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "zoe" {
		t.Errorf("second handle: %v", names(got))
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

func TestRequiresFilename(t *testing.T) {
	if _, err := NewDatabase[User](nil); err == nil {
		t.Error("expected an error when filename is missing")
	}
}

func TestRequiresPrimaryKey(t *testing.T) {
	type noKey struct{ Name string }
	if _, err := NewDatabase[noKey](map[string]string{
		"filename": filepath.Join(t.TempDir(), "x.json"),
	}); err == nil {
		t.Error("expected an error for a model with no primary key")
	}
}

// A model without a key cannot be indexed, so the file must not be created.
func TestNoStrayFileOnBadModel(t *testing.T) {
	type noKey struct{ Name string }
	path := filepath.Join(t.TempDir(), "x.json")
	if _, err := NewDatabase[noKey](map[string]string{"filename": path}); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a rejected model should not leave a file behind")
	}
}

func TestInsAndGet(t *testing.T) {
	tbl := table(t, open(t))
	in := User{ID: 1, Name: "alice", Age: 30,
		Tags: []string{"vip", "beta"}, Meta: map[string]string{"theme": "dark"}}
	if err := tbl.Ins(in, database.Options{}); err != nil {
		t.Fatal(err)
	}

	got, err := tbl.Get(User{ID: 1}, database.Options{
		Where: []database.Condition{database.Eq("ID", 1)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if got[0].Name != "alice" || got[0].Age != 30 {
		t.Errorf("scalars: %+v", got[0])
	}
	if len(got[0].Tags) != 2 || got[0].Meta["theme"] != "dark" {
		t.Errorf("collections should nest in the row: %+v", got[0])
	}
}

// No condition means the whole table, matching the SQL backends.
func TestGetAll(t *testing.T) {
	tbl := table(t, open(t))
	seed(t, tbl)
	got, err := tbl.Get(User{}, database.Options{Order: database.Ascending("ID")})
	if err != nil {
		t.Fatal(err)
	}
	if !same(names(got), []string{"alice", "bob", "carol", "dave"}) {
		t.Errorf("got %v", names(got))
	}
}

func TestScanConditions(t *testing.T) {
	tbl := table(t, open(t))
	seed(t, tbl)

	cases := []struct {
		name string
		cond database.Condition
		want []string
	}{
		{"Gte", database.Gte("Age", 30), []string{"carol", "dave"}},
		{"Gt", database.Gt("Age", 25), []string{"carol", "dave"}},
		{"Lt", database.Lt("Age", 30), []string{"alice", "bob"}},
		{"Eq", database.Eq("Name", "bob"), []string{"bob"}},
		{"Neq", database.Neq("Name", "bob"), []string{"alice", "carol", "dave"}},
		{"In", database.In("ID", 1, 3), []string{"alice", "carol"}},
		{"NotIn", database.NotIn("ID", 1, 3), []string{"bob", "dave"}},
		{"Between", database.Between("Age", 20, 30), []string{"alice", "bob", "carol"}},
		{"Like", database.Like("Name", "%o%"), []string{"bob", "carol"}},
		{"NotLike", database.NotLike("Name", "%o%"), []string{"alice", "dave"}},
		{"Contains", database.Contains("Tags", "beta"),
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

// Several conditions are combined with AND.
func TestCombinedConditions(t *testing.T) {
	tbl := table(t, open(t))
	seed(t, tbl)
	got, err := tbl.Get(User{}, database.Options{
		Where: []database.Condition{
			database.Gte("Age", 25),
			database.Lt("Age", 40),
			database.Contains("Tags", "beta"),
		},
		Order: database.Ascending("Age"),
	})
	if err != nil {
		t.Fatal(err)
	}
	// bob is in the age range but carries no Tags, so only carol matches.
	if !same(names(got), []string{"carol"}) {
		t.Errorf("got %v", names(got))
	}
}

func TestOrderAndPaging(t *testing.T) {
	tbl := table(t, open(t))
	seed(t, tbl)

	got, _ := tbl.Get(User{}, database.Options{Order: database.Descending("Age")})
	if !same(names(got), []string{"dave", "carol", "bob", "alice"}) {
		t.Errorf("desc: %v", names(got))
	}

	got, _ = tbl.Get(User{}, database.Options{
		Order: database.Ascending("Age"), Limit: 2,
	})
	if !same(names(got), []string{"alice", "bob"}) {
		t.Errorf("limit: %v", names(got))
	}

	got, _ = tbl.Get(User{}, database.Options{
		Order: database.Ascending("Age"), Limit: 2, Offset: 2,
	})
	if !same(names(got), []string{"carol", "dave"}) {
		t.Errorf("limit+offset: %v", names(got))
	}

	got, _ = tbl.Get(User{}, database.Options{
		Order: database.Ascending("Age"), Offset: 3,
	})
	if !same(names(got), []string{"dave"}) {
		t.Errorf("offset only: %v", names(got))
	}
}

// Without an explicit order paging must still be stable.
func TestDefaultOrderIsPrimaryKey(t *testing.T) {
	tbl := table(t, open(t))
	seed(t, tbl)
	got, _ := tbl.Get(User{}, database.Options{Limit: 2})
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 2 {
		t.Errorf("want IDs 1,2 got %+v", got)
	}
}

func TestSetReplacesEverything(t *testing.T) {
	tbl := table(t, open(t))
	if err := tbl.Ins(User{ID: 1, Name: "alice", Age: 20,
		Tags: []string{"a", "b"}}, database.Options{}); err != nil {
		t.Fatal(err)
	}
	err := tbl.Set(User{ID: 1}, User{ID: 1, Name: "bob", Tags: []string{"z"}},
		database.Options{})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{}, database.Options{
		Where: []database.Condition{database.Eq("ID", 1)},
	})
	if got[0].Name != "bob" || len(got[0].Tags) != 1 || got[0].Tags[0] != "z" {
		t.Errorf("got %+v", got[0])
	}
}

// Values naming one collection must leave the others alone.
func TestSetValuesIsComplete(t *testing.T) {
	tbl := table(t, open(t))
	if err := tbl.Ins(User{ID: 1, Tags: []string{"a"}, Age: 20,
		Meta: map[string]string{"x": "1"}}, database.Options{}); err != nil {
		t.Fatal(err)
	}
	err := tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
		Values: []database.Value{database.Union("Tags", []string{"c"})},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{}, database.Options{
		Where: []database.Condition{database.Eq("ID", 1)},
	})
	if len(got[0].Tags) != 2 {
		t.Errorf("union: %v", got[0].Tags)
	}
	if got[0].Age != 20 {
		t.Errorf("Age not in Values, so untouched: %+v", got[0])
	}
	if got[0].Meta["x"] != "1" {
		t.Errorf("Meta not in Values, so untouched: %v", got[0].Meta)
	}
}

func TestSetSkipBeatsValues(t *testing.T) {
	tbl := table(t, open(t))
	if err := tbl.Ins(User{ID: 1, Name: "alice", Tags: []string{"a"}},
		database.Options{}); err != nil {
		t.Fatal(err)
	}
	err := tbl.Set(User{ID: 1}, User{ID: 1, Name: "bob"}, database.Options{
		Skip:   []string{"Name"},
		Values: []database.Value{database.Assign("Name", "ignored")},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{}, database.Options{
		Where: []database.Condition{database.Eq("ID", 1)},
	})
	if got[0].Name != "alice" {
		t.Errorf("Name was skipped: %+v", got[0])
	}
}

func TestSetCollectionOps(t *testing.T) {
	tbl := table(t, open(t))
	if err := tbl.Ins(User{ID: 1, Tags: []string{"a", "b", "a"},
		Meta: map[string]string{"x": "1"}}, database.Options{}); err != nil {
		t.Fatal(err)
	}

	byID := database.Options{
		Where: []database.Condition{database.Eq("ID", 1)},
	}

	if err := tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
		Values: []database.Value{
			database.Union("Tags", []string{"c", "a"}),
			database.Union("Meta", map[string]string{"y": "2"}),
		},
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{}, database.Options{Where: byID.Where})
	if len(got[0].Tags) != 3 {
		t.Errorf("union should not duplicate: %v", got[0].Tags)
	}
	if len(got[0].Meta) != 2 {
		t.Errorf("map union: %v", got[0].Meta)
	}

	if err := tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
		Values: []database.Value{database.Difference("Tags", []string{"b"})},
	}); err != nil {
		t.Fatal(err)
	}
	got, _ = tbl.Get(User{}, database.Options{Where: byID.Where})
	for _, tag := range got[0].Tags {
		if tag == "b" {
			t.Errorf("b should be gone: %v", got[0].Tags)
		}
	}

	if err := tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
		Values: []database.Value{database.Clear("Tags")},
	}); err != nil {
		t.Fatal(err)
	}
	got, _ = tbl.Get(User{}, database.Options{Where: byID.Where})
	if len(got[0].Tags) != 0 {
		t.Errorf("clear: %v", got[0].Tags)
	}
	if len(got[0].Meta) != 2 {
		t.Errorf("Meta must be untouched: %v", got[0].Meta)
	}
}

// A difference must not mutate the value the caller still holds.
func TestDifferenceDoesNotAliasCallerValue(t *testing.T) {
	tbl := table(t, open(t))
	original := User{ID: 1, Tags: []string{"a", "b"}}
	if err := tbl.Ins(original, database.Options{}); err != nil {
		t.Fatal(err)
	}
	if err := tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
		Values: []database.Value{database.Difference("Tags", []string{"a"})},
	}); err != nil {
		t.Fatal(err)
	}
	if len(original.Tags) != 2 {
		t.Errorf("caller's slice was mutated: %v", original.Tags)
	}
}

func TestDel(t *testing.T) {
	tbl := table(t, open(t))
	seed(t, tbl)

	if err := tbl.Del(User{ID: 1}, database.Options{}); err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{}, database.Options{Order: database.Ascending("ID")})
	if len(got) != 3 || got[0].ID != 2 {
		t.Errorf("after Del: %v", names(got))
	}

	if err := tbl.Del(User{}, database.Options{
		Where: []database.Condition{database.Eq("Name", "bob")},
	}); err != nil {
		t.Fatal(err)
	}
	got, _ = tbl.Get(User{}, database.Options{Order: database.Ascending("ID")})
	if len(got) != 2 {
		t.Errorf("after conditional Del: %v", names(got))
	}
}

// Query returns the whole file regardless of the SQL it is handed.
func TestQueryReturnsEverything(t *testing.T) {
	db := open(t)
	tbl := table(t, db)
	seed(t, tbl)

	other, err := db.GetTable("more")
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Ins(User{ID: 9, Name: "eve"}, database.Options{}); err != nil {
		t.Fatal(err)
	}

	got, err := db.Query("SELECT * FROM users WHERE nonsense")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Errorf("want every row across tables, got %d", len(got))
	}
}

// Exec replaces a table wholesale from a JSON payload.
func TestExecOverwritesTable(t *testing.T) {
	db := open(t)
	tbl := table(t, db)
	seed(t, tbl)

	payload := `[{"ID":42,"Name":"new","Age":1}]`
	if err := db.Exec("users", []byte(payload)); err != nil {
		t.Fatal(err)
	}
	got, err := tbl.Get(User{}, database.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "new" {
		t.Errorf("after Exec: %v", names(got))
	}

	// A string payload is accepted too.
	if err := db.Exec("users", `[{"ID":43,"Name":"str"}]`); err != nil {
		t.Fatal(err)
	}
	got, _ = tbl.Get(User{}, database.Options{})
	if len(got) != 1 || got[0].ID != 43 {
		t.Errorf("string payload: %v", names(got))
	}
}

// A malformed payload must be refused, not silently clear the table.
func TestExecRejectsBadPayloads(t *testing.T) {
	db := open(t)
	tbl := table(t, db)
	seed(t, tbl)

	for name, args := range map[string][]any{
		"not an array": {`{"ID":1}`},
		"not JSON":     {`nope`},
		"wrong type":   {`[{"ID":"not a number"}]`},
		"no payload":   {},
		"bad type":     {42},
	} {
		if err := db.Exec("users", args...); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}

	got, _ := tbl.Get(User{}, database.Options{})
	if len(got) != 4 {
		t.Errorf("a rejected payload must not change the table: %v", names(got))
	}
}

func TestDeleteTable(t *testing.T) {
	db := open(t)
	tbl := table(t, db)
	seed(t, tbl)

	if err := db.DeleteTable("users"); err != nil {
		t.Fatal(err)
	}
	got, _ := tbl.Get(User{}, database.Options{})
	if len(got) != 0 {
		t.Errorf("after DeleteTable: %v", names(got))
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
		if err := tt.Ins(User{ID: 1, Name: "kept"}, database.Options{}); err != nil {
			return err
		}
		inner, err := tx.GetTable("inner")
		if err != nil {
			return err
		}
		return inner.Ins(User{ID: 2, Name: "also kept"}, database.Options{})
	})
	if err != nil {
		t.Fatal(err)
	}

	got, _ := tbl.Get(User{}, database.Options{})
	if len(got) != 1 || got[0].Name != "kept" {
		t.Errorf("commit: %v", names(got))
	}
	if !has(db.Tables(), "inner") {
		t.Errorf("second table should be committed: %v", db.Tables())
	}
}

// A failed transaction must leave the file untouched.
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
	if !errors.Is(err, errRollback) {
		t.Fatalf("want errRollback, got %v", err)
	}

	got, _ := tbl.Get(User{}, database.Options{})
	if len(got) != 1 || got[0].ID != 1 {
		t.Errorf("rollback failed: %v", names(got))
	}
	if has(db.Tables(), "users") != true {
		t.Error("the pre-existing table should still exist")
	}
}

func has(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func TestOptionValidation(t *testing.T) {
	tbl := table(t, open(t))
	cases := map[string]database.Options{
		"Eq on a collection":    {Where: []database.Condition{database.Eq("Tags", "a")}},
		"Contains on a scalar":  {Where: []database.Condition{database.Contains("Name", "a")}},
		"ContainsAll on a map":  {Where: []database.Condition{database.ContainsAll("Meta", "a")}},
		"ContainsKV on a slice": {Where: []database.Condition{database.ContainsKV("Tags", "k", "v")}},
		"Union on a scalar":     {Values: []database.Value{database.Union("Name", "x")}},
		"unknown where field":   {Where: []database.Condition{database.Eq("nope", 1)}},
		"unknown values field":  {Values: []database.Value{database.Assign("nope", 1)}},
		"unknown order":         {Order: database.Ascending("nope")},
		"negative limit":        {Limit: -1},
	}
	for name, opts := range cases {
		if _, err := tbl.Get(User{}, opts); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// A column named by a db tag must be reachable under both its column name and
// its Go field name.
func TestLookupByFieldOrColumn(t *testing.T) {
	db := open(t)
	tbl := table(t, db)
	if err := tbl.Ins(User{ID: 1, Name: "alice", Email: "a@x.com"},
		database.Options{}); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"Email", "email_address"} {
		got, err := tbl.Get(User{}, database.Options{
			Where: []database.Condition{database.Eq(name, "a@x.com")},
		})
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(got) != 1 || got[0].Name != "alice" {
			t.Errorf("%s: got %v", name, names(got))
		}
	}
}

// Two tables in one file stay independent.
func TestMultipleTables(t *testing.T) {
	db := open(t)
	users, _ := db.GetTable("users")
	posts, _ := db.GetTable("posts")

	if err := users.Ins(User{ID: 1, Name: "alice"}, database.Options{}); err != nil {
		t.Fatal(err)
	}
	if err := posts.Ins(User{ID: 1, Name: "post"}, database.Options{}); err != nil {
		t.Fatal(err)
	}

	got, _ := users.Get(User{}, database.Options{})
	if len(got) != 1 || got[0].Name != "alice" {
		t.Errorf("users: %v", names(got))
	}
	got, _ = posts.Get(User{}, database.Options{})
	if len(got) != 1 || got[0].Name != "post" {
		t.Errorf("posts: %v", names(got))
	}
}

// Every write rewrites the file, so concurrent writers must not corrupt it.
func TestConcurrentWrites(t *testing.T) {
	db := open(t)
	tbl := table(t, db)

	const n = 20
	errs := make(chan error, n)
	for i := 1; i <= n; i++ {
		go func(i int) {
			errs <- tbl.Ins(User{ID: i, Name: "u"}, database.Options{})
		}(i)
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}

	got, err := tbl.Get(User{}, database.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != n {
		t.Errorf("want %d rows, got %d", n, len(got))
	}
}

// A corrupt file must be reported, not silently replaced.
func TestCorruptFileReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewDatabase[User](map[string]string{"filename": path}); err == nil {
		t.Error("expected an error for a corrupt file")
	}
}

func TestLikeWildcards(t *testing.T) {
	cases := []struct {
		s, pattern string
		want       bool
	}{
		{"alice", "a%", true},
		{"alice", "%ice", true},
		{"alice", "%lic%", true},
		{"alice", "a_ice", true},
		{"alice", "a_ce", false},
		{"alice", "%", true},
		{"", "%", true},
		{"", "?", false},
		{"alice", "", false},
		// A naive matcher backtracks badly here.
		{"aaaab", "%a%a%a%b", true},
		{"aaa", "%a%b", false},
	}
	for _, c := range cases {
		if got := likeMatch(c.s, c.pattern); got != c.want {
			t.Errorf("likeMatch(%q, %q) = %v, want %v", c.s, c.pattern, got, c.want)
		}
	}
}
