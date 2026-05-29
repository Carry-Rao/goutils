# Database Module Documentation

## Quick Start

Use `database.NewDatabase()` to create a database instance. Specify the database type via the `type` field:

```go
import "github.com/Carry-Rao/goutils/database"

// Create a MySQL instance
db, _ := database.NewDatabase(map[string]string{
    "type":     "mysql",
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

// Get a table handler
table, _ := db.GetTable("users")

// CRUD operations
table.Create(map[string]any{"id": 1, "name": "Alice"})
result, _ := table.Get(map[string]any{"id": 1})
table.Set(map[string]any{"id": 1, "name": "Bob"})
table.Delete(map[string]any{"id": 1})

// Delete the table
db.DeleteTable("users")
```

## Interface Definition

**Database interface** (database-level operations):

| Method | Description |
|--------|-------------|
| `Create(tableName, config)` | Create a table; config maps field names to field properties |
| `GetTable(tableName)` | Get a table handler |
| `DeleteTable(tableName)` | Drop a table |

**Table interface** (table-level CRUD):

| Method | Description |
|--------|-------------|
| `Create(data)` | Insert data |
| `Get(where)` | Query by conditions (returns the first match) |
| `Set(data)` | Update data (must include an `id` field) |
| `Delete(where)` | Delete by conditions |

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
| `memory` | none required | In-memory storage, process-level cache |
| `bloom` | none required | Bloom filter for fast existence checks |

> Cache-based simulated databases wrap cache components into the standard Database/Table interface. When calling `Create`, you must specify a `PrimaryKey` field in Config as the data identifier.

---

## Mixture Database

Mixture lets you combine multiple databases into a data link chain, with custom error handling strategies for each layer.

### Setup

```go
import "github.com/Carry-Rao/goutils/database/mixture"

// Create underlying databases
redisDB, _ := database.NewDatabase(map[string]string{"type": "redis", "addr": "127.0.0.1:6379"})
mysqlDB, _ := database.NewDatabase(map[string]string{"type": "mysql", /* ... */})

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
| `Create` | Executes on all layers in order; failures are handled per strategy |
| `Get` | Queries from the first layer; returns on first hit; returns the last error if all fail |
| `Set` | Executes on all layers in order; failures are handled per strategy |
| `Delete` | Executes on all layers in order; failures are handled per strategy |

### Typical Scenario: Multi-tier Read/Write

```go
// Bloom filter → Redis cache → MySQL database
mix := &mixture.Database{}
mix.Add(bloomDB, mixture.Continue)
mix.Add(redisDB, mixture.Continue)
mix.Add(mysqlDB, mixture.Return)

// Upper business code is unaware of the link structure — the interface is identical
table, _ := mix.GetTable("users")
table.Create(map[string]any{"id": 1, "name": "Alice"})