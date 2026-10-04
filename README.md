# Goutils

[中文](docs/README_zh.md)

A Go utility library for database abstraction, HTTP routing, and logging.

- **Requires Go 1.24+** (uses generics)

```bash
go get github.com/Carry-Rao/goutils
```

## Modules

| Module | Description | Docs |
|--------|-------------|------|
| Database | SQL `Database[T]` / `Table[T]` over MySQL, PostgreSQL, and SQLite, plus a file-backed JSON backend. Conditions, transactions, and slice/map fields | [EN](docs/database_en.md) · [中文](docs/database_zh.md) |
| Cache | Key-value `Table[T]` over memory, Redis, and bloom filter, chainable via `mixture` | [EN](docs/database_en.md#cache-module) · [中文](docs/database_zh.md#cache-模块) |
| HTTP Router | Prefix-tree router with typed path variables, middleware, per-pattern CORS, and correct 404/405 handling | [EN](docs/http_en.md) · [中文](docs/http_zh.md) |
| Log | Buffered multi-level logging with color output | [EN](docs/log_en.md) · [中文](docs/log_zh.md) |

Contributing: [CONTRIBUTION.md](CONTRIBUTION.md) · [中文](docs/CONTRIBUTION_zh.md)

## Database in one snippet

```go
import "github.com/Carry-Rao/goutils/database/sql/sqlite"

type User struct {
	ID   int      `db:",primary"`
	Name string
	Tags []string `db:",child"` // stored in `users_tags`, read back automatically
}

db, err := sqlite.NewDatabase[User](map[string]string{"path": "./data.db"})
if err != nil {
	panic(err)
}
defer db.Close()

// Creates the table and the companion table if they are missing.
tbl, err := db.GetTable("users")
if err != nil {
	panic(err)
}

if err := tbl.Ins(User{ID: 1, Name: "Alice", Tags: []string{"vip"}},
	database.Options{}); err != nil {
	panic(err)
}

// []User, not []any. Conditions live in Options, and Tags comes back filled in.
users, err := tbl.Get(User{ID: 1}, database.Options{
	Where: []database.Condition{database.Contains("Tags", "vip")},
	Order: database.Ascending("Name"),
})

// WHERE from the old value, assignment from the new one. An empty Values means
// "replace the whole record".
err = tbl.Set(User{ID: 1}, User{ID: 1, Name: "Bob"}, database.Options{})

// Multi-table writes are atomic.
err = db.Tx(func(tx database.Tx[User]) error {
	tt, err := tx.GetTable("users")
	if err != nil {
		return err
	}
	return tt.Del(User{ID: 1}, database.Options{})
})
```

The three SQL backends are thin wrappers over a shared implementation: import the
one you want and call its `NewDatabase`. There is no factory and no TTL.

## Breaking changes from the previous release

- **SQL and cache are now separate packages.** `database` is SQL only
  (`GetTable(name)`, `Query`, `Exec`, `Tx`, `Close`); `database/cache` is
  key-value only (`Create`, `GetTable(name, example)`, primary-key addressing).
- **No TTL anywhere.** Expiry belongs in the layer above.
- **All filtering and assignment moved into `Options`**, which replaces
  `whereFields` / `setFields` and the `ttl` argument:
  - `Table[T]`: `Ins(val, Options)`, `Get(val, Options) ([]T, error)`,
    `Set(old, updated, Options)`, `Del(val, Options)`
  - `cache.Table[T]`: `Ins(val)`, `Get(val) ([]T, error)`, `Set(old, updated)`,
    `Del(val)`
- **Slice and map fields are ordinary fields.** Mark them `db:",child"` and
  `Ins`/`Get`/`Set`/`Del` handle them; there is no `Children` type and no
  `AddChild` / `RemoveChild`.
- **Queries no longer need raw SQL.** `Options.Where` covers comparisons, `IN`,
  `BETWEEN`, `LIKE`, `IS NULL`, and collection membership; `Options.Values` adds
  `Union` / `Difference` / `Clear`; `Options.Skip` excludes columns.
- A **JSON backend** was added: one file holds the whole database, keyed by table
  and then by primary key, with collections nested inside each row. Non-key
  conditions are a linear scan in Go. TLS settings are no longer hardcoded —
  `sslmode` / `tls` come from the config map and are omitted when unset.
- 405 is now answered from a stub planted at registration time instead of by
  probing every method tree; a real handler always outranks a stub.
- `database/factory` was removed; backends are imported directly.
- `router.BadRequest` became `router.MethodNotAllowed`; status code 400 → 405
- Router `:any` was removed; `:string` is the only catch-all and does not span `/`

## Status

Covered by the test suite end to end: **SQLite** (including transactions,
collection fields and set operations) and **JSON** (including atomic writes,
rollback and linear-scan filtering). Connection strings for MySQL and PostgreSQL
are covered by string-level tests; the live connections are not, since they need
external services. The memory, bloom and mixture backends are covered too.

Not covered by automated tests: live MySQL, PostgreSQL and Redis connections, as
they require external services. Not recommended for production use without your
own testing.