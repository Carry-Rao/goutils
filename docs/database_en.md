# Database Module Documentation

## Quick Start

Use `database.NewDatabase()` to create a database instance. Specify the database type via the `api.DbType` enum:

```go
import (
    "github.com/Carry-Rao/goutils/database"
    "github.com/Carry-Rao/goutils/database/api"
)

// Create a MySQL instance
db, _ := database.NewDatabase(api.MySQL, map[string]string{
    "user":     "root",
    "password": "123456",
    "host":     "127.0.0.1",
    "port":     "3306",
    "dbname":   "test",
})

// Create a table
db.Create("users", map[string]api.Config{
    "id":   {Type: "INT", PrimaryKey: true, Identity: true},
    "name": {Type: "VARCHAR(255)", NullAble: false},
})

// Get a table handler (pass a struct example for reflection)
table, _ := db.GetTable("users", &User{})

// CRUD operations (using struct and field name lists)
table.Ins(&User{Name: "Alice", Email: "alice@example.com"}, 0)                // Insert
result, _ := table.Get(&User{Name: "Alice"}, []string{"Name"}, 0)                // Query
table.Set(&User{Name: "Alice", Email: "alice@new.com"}, []string{"Name"}, 0)     // Update
table.Del(&User{Name: "Bob"}, []string{"Name"}, 0)                            // Delete

// Delete the table
db.DeleteTable("users")
```

### Type-safe Generic Usage

Use `typed.TypedTable[T]` for compile-time type safety:

```go
import "github.com/Carry-Rao/goutils/database/typed"

rawTable, _ := db.GetTable("users", &User{})
table := typed.NewTable[User](rawTable)

// Compile-time type safety
table.Ins(User{Name: "Alice"}, 0)
users, _ := table.Get(User{}, []string{"name"}, 0) // Returns []User, not []any
```

## Core Concepts

The database module uses a **struct-driven** design pattern:

- Each database table maps to a Go struct with `db` tags for field names and constraints
- `Insert`/`Get`/`Set`/`Delete` all take struct instances as parameters
- Use field name lists to specify WHERE conditions, avoiding raw SQL
- `GetTable` automatically creates the table from the struct (skips if already exists), no manual DDL needed

### Struct Definition Example

```go
type User struct {
    ID    int    `db:"id,primary,autoinc"`   // Auto-increment primary key
    Name  string `db:"name"`                 // Regular field
    Email string `db:"email"`
    Age   int    `db:"age"`
}
```

### TTL (Time To Live)

All `Table` methods accept a `time.Duration` as the last parameter for data expiration:

- **Relational databases** (sqlite / mysql / postgresql): TTL is ignored, data never expires
- **Memory database** (memory): expired entries are automatically cleaned up
- **Redis**: TTL works with `EXPIRE` command
- `0` means no expiration

---

## Interface Definition

**Database interface** (database-level operations):

| Method | Description |
|--------|-------------|
| `Create(tableName, config)` | Create a table; config maps field names to field properties |
| `GetTable(tableName, example)` | Get a table handler; example is used for struct reflection |
| `DeleteTable(tableName)` | Drop a table |

**Table interface** (table-level CRUD):

| Method | Description |
|--------|-------------|
| `Ins(example, ttl)` | Insert a record; fields tagged `autoinc` are skipped |
| `Get(example, whereFields, ttl)` | Query by conditions (returns the first match) |
| `Set(example, whereFields, ttl)` | Update by conditions; `whereFields` are not updated |
| `Del(example, whereFields, ttl)` | Delete by conditions |

**Config field properties**:

| Field | Description |
|-------|-------------|
| `Type` | Field type (e.g. INT, VARCHAR(255)) |
| `NullAble` | Whether the field can be null |
| `Identity` | Whether the field is auto-increment |
| `PrimaryKey` | Whether the field is a primary key |
| `Unique` | Whether the field is unique |

---

## Database Configuration

### Relational Databases

| Type | Config keys | Description |
|------|-------------|-------------|
| `mysql` | `user`, `password`, `host`, `port`, `dbname` | MySQL connection parameters |
| `postgresql` | `user`, `password`, `host`, `port`, `dbname` | PostgreSQL connection parameters |
| `sqlite` | `path` | SQLite file path |

### Cache-based Simulated Databases

| Type | Config keys | Description |
|------|-------------|-------------|
| `redis` | `addr`, `password` | Redis address and password (simulated table structure) |
| `memory` | none required | In-memory storage with TTL auto-expiration |
| `bloom` | none required | Bloom filter for fast existence checks |

> Cache-based simulated databases wrap cache components into the standard Database/Table interface. When calling `Create`, you must specify a `PrimaryKey` field in Config as the data identifier.

---

## Mixture Database

Mixture lets you combine multiple databases into a data link chain, with custom error handling strategies for each layer.

### Setup

```go
import (
    "github.com/Carry-Rao/goutils/database"
    "github.com/Carry-Rao/goutils/database/api"
    "github.com/Carry-Rao/goutils/database/mixture"
)

// Create underlying databases
redisDB, _ := database.NewDatabase(api.Redis, map[string]string{"addr": "127.0.0.1:6379"})
mysqlDB, _ := database.NewDatabase(api.MySQL, map[string]string{/* ... */})

// Build chain: Redis → MySQL, continue on Redis error, return on MySQL error
mix := &mixture.Database{}
mix.Add(redisDB, mixture.Continue)
mix.Add(mysqlDB, mixture.Return)
```

### Error Handling Strategies

| Strategy | Value | Behavior |
|----------|-------|----------|
| `Continue` | 0 | On error, continue to the next layer |
| `Return` | 1 | On error, return immediately |

### Link Behavior by Method

| Method | Behavior |
|--------|----------|
| `Ins` | Executes on all layers in order; failures are handled per strategy |
| `Get` | Queries from the first layer; returns on first hit; returns the last error if all fail |
| `Set` | Executes on all layers in order; failures are handled per strategy |
| `Del` | Executes on all layers in order; failures are handled per strategy |

### Typical Scenario: Multi-tier Read/Write

```go
// Bloom filter → Redis cache → MySQL database
mix := &mixture.Database{}
mix.Add(bloomDB, mixture.Continue)
mix.Add(redisDB, mixture.Continue)
mix.Add(mysqlDB, mixture.Return)

// Upper business code is unaware of the link structure — the interface is identical
table, _ := mix.GetTable("users", &User{})
table.Ins(&User{Name: "Alice"}, 0)
```

---

## Registering Custom Databases

Use `api.RegisterFactory` to register custom database backends. Implement the `api.Database` interface:

```go
package mydb

import (
    "github.com/Carry-Rao/goutils/database/api"
)

type Database struct {
    // your database connection
}

func (d *Database) Create(tableName string, config map[string]api.Config) error {
    return nil
}

func (d *Database) GetTable(tableName string, example any) (api.Table, error) {
    return &Table{}, nil
}

func (d *Database) DeleteTable(tableName string) error {
    return nil
}

func init() {
    api.RegisterFactory("mydb", func(config map[string]string) (api.Database, error) {
        return &Database{}, nil
    })
}
```

Create via `NewDatabaseByName`:

```go
import "github.com/Carry-Rao/goutils/database"

db, _ := database.NewDatabaseByName("mydb", map[string]string{
    "host": "127.0.0.1",
})
```

Register aliases:

```go
api.Register("pg", api.PostgreSQL)
db, _ := database.NewDatabaseByName("pg", config)
```