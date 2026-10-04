package database

import (
	"reflect"
	"strings"
	"testing"
)

type full struct {
	ID      int               `db:",primary"`
	Name    string            // no tag: column is the field name
	Email   string            `db:"email_address"`
	Age     int               `db:",null"`
	Tags    []string          `db:",child"`
	Meta    map[string]string `db:",child"`
	Renamed []string          `db:",child=tag_links"`
	skip    string            // unexported, never mapped
}

func TestParseFieldDefaults(t *testing.T) {
	s := SchemaOf[full]()

	if got := s.FieldMap["Name"]; got.GoFieldName != "Name" {
		t.Errorf("untagged field should use the field name, got %q", got.ColumnName)
	}
	if _, ok := s.FieldMap["Email"]; ok {
		t.Error("db tag must rename the column")
	}
	if _, ok := s.Lookup("email_address"); !ok {
		t.Error("renamed column not reachable by its column name")
	}
}

// Unexported fields exist only to be skipped. Referencing one keeps the field
// from reading as dead code.
func TestUnexportedFieldNotMapped(t *testing.T) {
	if got := (full{}).skip; got != "" {
		t.Errorf("unexpected value %q", got)
	}
	if _, ok := SchemaOf[full]().FieldMap["skip"]; ok {
		t.Error("unexported fields must not be mapped")
	}
}

func TestParseFieldConstraints(t *testing.T) {
	s := SchemaOf[full]()

	id := s.FieldMap["ID"]
	if !id.IsPrimary {
		t.Error("ID should be primary")
	}
	if !s.FieldMap["Age"].IsNullable {
		t.Error("Age should be nullable")
	}
	for _, name := range []string{"Tags", "Meta", "Renamed"} {
		if !s.FieldMap[name].IsChild {
			t.Errorf("%s should be a collection field", name)
		}
	}
	if s.FieldMap["Renamed"].ChildTable != "tag_links" {
		t.Errorf("child table override lost: %q", s.FieldMap["Renamed"].ChildTable)
	}
}

func TestPrimaryKey(t *testing.T) {
	s := SchemaOf[full]()
	pk, err := s.PrimaryKey()
	if err != nil {
		t.Fatal(err)
	}
	if pk.ColumnName != "ID" {
		t.Errorf("pk = %q", pk.ColumnName)
	}

	type noKey struct{ Name string }
	if _, err := SchemaOf[noKey]().PrimaryKey(); err == nil {
		t.Error("expected an error for a model with no primary key")
	}
}

func TestLookupByColumnOrFieldName(t *testing.T) {
	s := SchemaOf[full]()
	if _, ok := s.Lookup("email_address"); !ok {
		t.Error("lookup by column name failed")
	}
	if _, ok := s.Lookup("Email"); !ok {
		t.Error("lookup by Go field name failed")
	}
	if _, ok := s.Lookup("nope"); ok {
		t.Error("unknown name should not resolve")
	}
}

func TestChildTableName(t *testing.T) {
	s := SchemaOf[full]()

	if got := s.ChildTableName("users", s.FieldMap["Tags"]); got != "users_tags" {
		t.Errorf("derived child table = %q, want users_tags", got)
	}
	if got := s.ChildTableName("users", s.FieldMap["Renamed"]); got != "tag_links" {
		t.Errorf("explicit child table = %q, want tag_links", got)
	}
}

// An unmarked collection must be refused rather than silently dropped.
func TestUnmarkedCollectionRejected(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected a panic for an unmarked slice field")
		}
	}()
	type bad struct {
		ID   int
		Tags []string
	}
	_ = SchemaOf[bad]()
}

func TestUnknownConstraintRejected(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected a panic for an unknown db constraint")
		}
	}()
	type bad struct {
		ID int `db:",bogus"`
	}
	_ = SchemaOf[bad]()
}

func TestKeyString(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{"abc", "abc"},
		{42, "42"},
		{int64(-7), "-7"},
		{uint(9), "9"},
		{3.5, "3.5"},
		{true, "true"},
		{(*string)(nil), ""},
	}
	for _, c := range cases {
		if got := KeyString(reflect.ValueOf(c.in)); got != c.want {
			t.Errorf("KeyString(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A pointer model must map to the same struct schema.
func TestPointerModelSameSchema(t *testing.T) {
	if SchemaOf[*full]().Type != SchemaOf[full]().Type {
		t.Error("pointer and value models should share a schema")
	}
}

func TestSQLType(t *testing.T) {
	q := SQLiteDialect
	pg := PostgreSQLDialect

	if got := q.SQLType(reflect.String); got != "TEXT" {
		t.Errorf("string: %q", got)
	}
	if got := q.SQLType(reflect.Int); got != "INTEGER" {
		t.Errorf("int: %q", got)
	}
	if got := q.SQLType(reflect.Bool); got != "INTEGER" {
		t.Errorf("sqlite bool: %q", got)
	}
	if got := pg.SQLType(reflect.Bool); got != "BOOLEAN" {
		t.Errorf("pg bool: %q", got)
	}
}

func TestQuote(t *testing.T) {
	if got := SQLiteDialect.Quote("col"); got != "`col`" {
		t.Errorf("sqlite quote: %q", got)
	}
	if got := PostgreSQLDialect.Quote("col"); got != `"col"` {
		t.Errorf("pg quote: %q", got)
	}
	if got := SQLiteDialect.QuoteTable("main.users"); got != "`main`.`users`" {
		t.Errorf("qualified quote: %q", got)
	}
}

func TestPlaceholder(t *testing.T) {
	if got := SQLiteDialect.placeholder(3); got != "?" {
		t.Errorf("sqlite: %q", got)
	}
	if got := PostgreSQLDialect.placeholder(3); got != "$3" {
		t.Errorf("pg: %q", got)
	}
}

func TestLimitClause(t *testing.T) {
	d := SQLiteDialect
	cases := []struct {
		o    Options
		want string
	}{
		{Options{}, ""},
		{Options{Limit: 5}, " LIMIT 5"},
		{Options{Offset: 10}, " LIMIT -1 OFFSET 10"},
		{Options{Limit: 5, Offset: 10}, " LIMIT 5 OFFSET 10"},
	}
	for _, c := range cases {
		if got := c.o.limitClause(d); got != c.want {
			t.Errorf("%+v: got %q, want %q", c.o, got, c.want)
		}
	}
}

func TestChildPairs(t *testing.T) {
	got := childPairs([]string{"b", "a"})
	if len(got) != 2 || got[0][1] != "b" {
		t.Errorf("slice pairs: %v", got)
	}

	got = childPairs(map[string]string{"z": "1", "a": "2"})
	if len(got) != 2 || got[0][0] != "a" || got[1][0] != "z" {
		t.Errorf("map pairs should be sorted for stable inserts: %v", got)
	}
}

func TestMergeAndSubtract(t *testing.T) {
	a := [][2]string{{"", "x"}, {"", "y"}}
	b := [][2]string{{"", "y"}, {"", "z"}}

	merged := mergePairs(a, b)
	if len(merged) != 3 {
		t.Errorf("merge: %v", merged)
	}
	only := subtractPairs(merged, b)
	if len(only) != 1 || only[0][1] != "x" {
		t.Errorf("subtract: %v", only)
	}
}

func TestScanTypeError(t *testing.T) {
	var e error = &ScanTypeError{Got: reflect.TypeOf(full{})}
	if e.Error() == "" {
		t.Error("ScanTypeError should describe itself")
	}
}

func TestDriversRegistered(t *testing.T) {
	// Only the root package registers nothing itself; importing this package in
	// isolation should yield an empty list rather than nil.
	_ = Drivers()
}

// An element type that cannot round-trip through a text column must be refused
// at schema time rather than panicking during hydration.
func TestUnstorableChildElementRejected(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected a panic for a struct-valued child element")
			return
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, "cannot be stored") {
			t.Errorf("unhelpful message: %v", r)
		}
	}()
	type elem struct{ N int }
	type bad struct {
		ID   int    `db:",primary"`
		Tags []elem `db:",child"`
	}
	_ = SchemaOf[bad]()
}

func TestUnstorableMapKeyRejected(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected a panic for a struct map key")
			return
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, "map key") {
			t.Errorf("unhelpful message: %v", r)
		}
	}()
	type elem struct{ N int }
	type bad struct {
		ID   int             `db:",primary"`
		Meta map[elem]string `db:",child"`
	}
	_ = SchemaOf[bad]()
}

// Every basic element kind must be accepted, since they all round-trip.
func TestStorableChildElementsAccepted(t *testing.T) {
	type ok struct {
		ID int                `db:",primary"`
		S  []string           `db:",child"`
		I  []int              `db:",child"`
		B  []bool             `db:",child"`
		F  []float64          `db:",child"`
		SI []int              `db:",child"`
		M  map[string]int     `db:",child"`
		MI map[int]string     `db:",child"`
		MF map[string]float64 `db:",child"`
	}
	s := SchemaOf[ok]()
	if len(s.Collections) != 8 {
		t.Errorf("got %d collection fields, want 8", len(s.Collections))
	}
}
