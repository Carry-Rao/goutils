# Database Module

> 中文文档：[database_zh.md](database_zh.md) · [http_zh.md](http_zh.md) · [log_zh.md](log_zh.md)

Two independent abstractions:

- **`database`** — SQL only. Tables, columns, conditions, ordering, paging, transactions.
- **`database/cache`** — key-value only. Records addressed by primary key, nothing else.

They are separate on purpose. A cache has no `WHERE` clause, no column-level
assignment and no ordering, so a single interface covering both would be mostly
`if` statements. Splitting them keeps each surface honest.

## Package layout

```
database/                     SQL: interfaces, schema, dialects, Options, shared Table
database/sql/mysql/           thin backends — embed database.Backend[T]
database/sql/postgresql/
database/sql/sqlite/
database/cache/               key-value: its own Table, Schema, Config
database/cache/memory/
database/cache/bloom/
database/cache/redis/
database/mixture/             chains cache tables into one lookup path
```

There is no import cycle. `database` imports nothing internal; each backend
imports only the root package; `mixture` imports only `database/cache`.

The three SQL backends are ~30 lines each. Everything they share — DDL
generation, condition rendering, collection storage, transactions — lives once
in `database.SQLTable[T]`. A backend only supplies a `*sql.DB` and a dialect:

```go
type Database[T any] struct {
	database.Backend[T]
}

func NewDatabase[T any](cfg map[string]string) (*Database[T], error) {
	db, err := sql.Open("sqlite3", cfg["path"])
	// ...
	return &Database[T]{Backend[T]{DB: db, Dialect: database.SQLiteDialect}}, nil
}
```

## Model definition

```go
type User struct {
	ID    int               `db:",primary"`
	Name  string
	Email string            `db:"email_address"`
	Age   int               `db:",null"`
	Tags  []string          `db:",child"`
	Meta  map[string]string `db:",child"`
}
```

Tag syntax is `column[,constraint]...`. **The first element is optional**, so an
empty first element means "name the column after the field" and everything after
it is a constraint:

| Tag | Meaning |
|-----|---------|
| *(none)* | Column named after the field, NOT NULL |
| `db:"email_address"` | Column renamed |
| `db:",primary"` | Primary key |
| `db:",autoinc"` | Auto increment; excluded from `INSERT` |
| `db:",unique"` | Unique index |
| `db:",null"` | Nullable |
| `db:",child"` | Slice/map stored in a companion table |
| `db:",child=tag_links"` | Companion table named explicitly |

Exactly one field must be `primary`; a model without one is rejected when the
table is opened. Only single-column primary keys are supported.

### Collections are ordinary fields

A slice, array or map marked `child` is stored in its own table. It is read and
written like any other field — there is no separate `Children` type, no
`AddChild`, and nothing to remember to call:

```go
u := User{ID: 1, Name: "alice", Tags: []string{"vip", "beta"}}
tbl.Ins(u, database.Options{})

got, _ := tbl.Get(User{ID: 1}, database.Options{})
got[0].Tags // []string{"vip", "beta"} — filled in automatically
```

`Get` issues one extra query per collection field (`WHERE parent_id IN (...)`)
and fills the fields in. A slice field yields an empty non-nil slice when the
record has no entries, so `len(u.Tags) == 0` is safe without a nil check.

An unmarked slice or map is a **panic at first use**. Silently dropping it would
lose data, so the mistake is made loud instead.

Elements must be a type that round-trips through a text column — `string`, the
integer and float kinds, or `bool`. A `[]SomeStruct` child is rejected at schema
time rather than panicking later during hydration. Map keys are checked the same
way.

### Table names

The companion table defaults to the main table plus the lowercased column:

| Main | Field | Companion |
|------|-------|-----------|
| `user` | `Tags` | `user_tags` |
| `users` | `Meta` | `users_meta` |

Its shape follows the field kind:

```sql
-- slice Tags []string
CREATE TABLE users_tags (Tags TEXT, parent_id INTEGER, UNIQUE(Tags, parent_id));

-- map Meta map[string]string
CREATE TABLE users_meta (k TEXT, v TEXT, parent_id INTEGER, UNIQUE(k, v, parent_id));
```

`parent_id` is named after the primary key column, so a model keyed on `uid`
gets `parent_uid`.

## Opening a database

Import the backend you want and call its constructor. There is no factory and no
driver-name indirection:

```go
import "github.com/Carry-Rao/goutils/database/sql/sqlite"

db, err := sqlite.NewDatabase[User](map[string]string{"path": "./data.db"})
defer db.Close()
```

Configuration is a `map[string]string`, so it drops straight out of a config
file:

```go
db, err := mysql.NewDatabase[User](map[string]string{
	"user": "root", "password": "...", "host": "127.0.0.1",
	"port": "3306", "dbname": "app",
})
```

`T` may be the struct or a pointer to it; both map to the same schema.

`Close() error` is part of the interface, so `defer db.Close()` is checked rather
than ignored.

## Tables

`GetTable` issues `CREATE TABLE IF NOT EXISTS` for the main table and every
companion table, so there is no separate migration step:

```go
tbl, err := db.GetTable("users")
```

`DeleteTable` drops the main table and its companions:

```go
err := db.DeleteTable("users")
```

## CRUD

```go
err := tbl.Ins(User{ID: 1, Name: "alice"}, database.Options{})

users, err := tbl.Get(User{ID: 1}, database.Options{})

err = tbl.Set(User{ID: 1}, User{ID: 1, Name: "bob"}, database.Options{})

err = tbl.Del(User{ID: 1}, database.Options{})
```

`Ins` writes the row and its collections in one transaction, so a failure leaves
nothing behind. `Del` removes the collection rows too.

When `opts.Where` is empty, `Set` and `Del` fall back to the primary key of the
value passed in, which is what the one-line forms above rely on.

## Options

`Options` is the only place filtering, assignment, ordering and paging live. The
zero value means "everything".

```go
type Options struct {
	Where  []Condition
	Values []Value
	Skip   []string
	Order  []Order
	Limit  int
	Offset int
}
```

### Conditions

Conditions are **AND-only**. There is no grouping and no `OR` across fields — use
`ContainsAny` for an OR over a collection, or drop to `Query` for anything more
exotic.

| Constructor | SQL |
|-------------|-----|
| `Eq` `Neq` | `=` `!=` |
| `Gt` `Gte` `Lt` `Lte` | comparisons |
| `Like` `NotLike` | `LIKE` — pass your own `%` |
| `In` `NotIn` | `IN (…)` |
| `Between` | `BETWEEN lo AND hi` |
| `IsNull` `IsNotNull` | `IS NULL` / `IS NOT NULL` |
| `Contains` | slice holds the element; map holds the key |
| `ContainsAll` | every element present |
| `ContainsAny` | at least one element present |
| `ContainsKV` | `map[key] == value` |

```go
users, err := tbl.Get(User{}, database.Options{
	Where: []database.Condition{
		database.Gte("Age", 18),
		database.Contains("Tags", "vip"),
		database.ContainsKV("Meta", "plan", "pro"),
	},
	Order: database.Descending("Age"),
	Limit: 20,
})
```

Collection conditions compile to a correlated `EXISTS` against the companion
table, so they join without loading anything into Go:

```sql
WHERE `Age` >= ?
  AND EXISTS (SELECT 1 FROM `users_tags` c
              WHERE c.`parent_id` = `users`.`ID` AND c.`Tags` = ?)
```

`ContainsAll` emits one `EXISTS` per element ANDed together, since no single row
can hold them all. `ContainsAny` emits a single `EXISTS` with `OR`.

**Every operator is checked against the schema before any SQL runs.** Asking
`Eq("Tags", "x")` on a collection, or `ContainsKV` on a slice, returns an error
immediately rather than generating nonsense SQL. Sort keys are validated for the
same reason — they cannot be bound as parameters, so a caller-supplied string
would otherwise reach `ORDER BY` unescaped.

### Values

`Values` controls what `Set` does to a collection:

| Constructor | Effect |
|-------------|--------|
| `Union` | Merge the given elements in |
| `Difference` | Remove the given elements |
| `Clear` | Empty the collection |
| `Assign` | Replace the column |

```go
tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
	Values: []database.Value{
		database.Union("Tags", []string{"gamma"}),
		database.Clear("Meta"),
	},
})
```

### The `Values` rule

This one is worth stating plainly, because guessing wrong loses data:

- **`Values` empty** — every scalar and every collection is assigned from
  `updated`. This is the plain "replace the record" call.
- **`Values` non-empty** — it becomes the *complete* specification. Only the
  columns it names are touched; everything else is left alone.

```go
// updated carries no Meta, and Values does not mention it, so Meta survives.
tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
	Values: []database.Value{database.Union("Tags", []string{"gamma"})},
})
// Meta is still exactly as it was.
```

`Skip` works alongside this and takes precedence: a column in `Skip` is never
written, even when named in `Values`.

```go
tbl.Set(User{ID: 1}, User{ID: 1, Name: "new", Age: 99}, database.Options{
	Skip: []string{"Age"},
})
```

### Order and paging

```go
database.Asc("Age")            database.Ascending("Age")     // []Order
database.Desc("Age")           database.Descending("Age")   // []Order
```

`Offset` without `Order` is not stable — the row order is the backend's choice.

## Transactions

`Tx` runs a function atomically. Returning an error rolls everything back, so the
error you return is the one that surfaces:

```go
err := db.Tx(func(tx database.Tx[User]) error {
	users, err := tx.GetTable("users")
	if err != nil {
		return err
	}
	if err := users.Ins(User{ID: 2, Name: "bob"}, database.Options{}); err != nil {
		return err
	}
	posts, err := tx.GetTable("posts")
	if err != nil {
		return err
	}
	return posts.Ins(Post{ID: 1, UserID: 2}, database.Options{})
})
```

Tables obtained from `tx` join the open transaction rather than nesting a new
one. Each `Ins`/`Set`/`Del` outside an explicit `Tx` still runs in its own
transaction, so a single write is never half-applied.

`Tx[T]` exposes `GetTable`, `Query` and `Exec` — everything `Database[T]` does
except starting another transaction.

## Raw SQL

For anything the `Options` surface does not cover:

```go
rows, err := db.Query("SELECT * FROM users WHERE Age > ?", 30)
err = db.Exec("UPDATE users SET Name = ? WHERE ID = ?", "renamed", 1)
```

Both are available on `Tx` as well. `Query` scans into `[]T` using the same
schema mapping, so collection fields are simply left empty — use `Get` if you
need them.

## Cache module

`database/cache` addresses records by primary key and nothing else, which is why
its interface is four methods with no options:

```go
type Table[T any] interface {
	Ins(val T) error
	Get(val T) ([]T, error)
	Set(old, updated T) error
	Del(val T) error
}
```

`Set` takes the old and new values separately so a changed primary key **moves**
the record instead of duplicating it.

```go
import "github.com/Carry-Rao/goutils/database/cache/memory"

db, err := memory.NewDatabase[User](nil)
err = db.Create("users", map[string]cache.Config{"id": {PrimaryKey: true}})

tbl, err := db.GetTable("users", User{})
if tbl == nil {
	return errors.New("table users not created, call Create first")
}
```

Cache tables must be declared with `Create` — there is no schema to derive a DDL
from, and no `IF NOT EXISTS` to lean on. `GetTable` returns `(nil, nil)` when the
table does not exist, so **check for a nil table**.

Backends: `memory`, `redis`, `bloom`. `cache.Register` / `cache.Drivers` list
them.

There is no TTL anywhere. If you need expiry, it belongs in the layer above —
Redis and the SQL backends both have native support for it, and burying the
policy in the storage call hides it.

### Bloom filter

Suits a very large key space where false positives are acceptable and values are
not read back — testing whether an ID *may* exist.

```go
import "github.com/Carry-Rao/goutils/database/cache/bloom"

if _, err := tbl.Get(User{ID: id}); err == nil {
	// probably present
} else if errors.Is(err, cache.ErrNotFound) {
	// definitely absent
}
```

A miss is `cache.ErrNotFound` rather than an empty result, so a caller can tell
"absent" from "the filter has false negatives" (it never does). Absent is always
correct; present may be wrong.

### Redis

Redis stores each row as a Hash. Keys are `{table}_{primaryKeyValue}` and fields
are the columns. `Create` must name a primary key first. Values are stored as
strings and converted back per field type on read; a field that fails to convert
keeps its zero value.

## Mixture chain

Chains several stores into one lookup path — the usual shape is filter → cache →
database.

```go
import "github.com/Carry-Rao/goutils/database/mixture"

chain := mixture.New[User]()
chain.Add(bloomTbl, mixture.Continue) // on miss or error, fall through
chain.Add(redisTbl, mixture.Continue)
chain.Add(mysqlTbl, mixture.Return)   // authoritative: fail here

var users cache.Table[User] = chain.Table()
```

Reads return the first non-empty result. A layer that misses *without* an error is
not a hit, so the chain keeps going.

Writes reach **every** layer, not just the first that accepts them — a filter, a
cache and a store all need to stay in sync. `Close` reports every layer's failure
rather than only the first.

All layers must share `T`, since generics allow only one type per chain.

`Get` does not backfill lower layers on a hit; implement write-back in your own
layer if you want it.

## Writing a custom backend

For SQL, embed `database.Backend[T]` and supply a dialect — you get the whole
`Table[T]` for free. For a key-value store, implement `cache.Table[T]`.

```go
type Database[T any] struct {
	database.Backend[T]
}
```

A backend registers its name from `init()` so it shows up in `Drivers()`:

```go
func init() { database.Register("mydb") }
```

`Register` is informational only. It exists so the available backends can be
listed; construction always goes through the backend's own constructor, because
Go has no generic function types and a `func(cfg) (Database[T], error)` cannot be
stored in a registry.

## Reflection overhead

Only `reflect` can read struct tags, so field mapping depends on it. Two
mitigations:

1. Parse results are cached per type in a `sync.Map`, so each type is parsed once.
2. `database.SchemaOf[T]()` returns the cached schema directly.

`Ins`/`Set`/`Get` still use reflection per call to read and write fields — Go
generics cannot expand struct fields at compile time. For SQL backends the cost
is dominated by network I/O and the reflection share is negligible.
