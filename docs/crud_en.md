# CRUD Module

This module provides a **struct-based ORM-like wrapper** over the database module, enabling developers to perform CRUD operations using Go structs instead of raw map data. It simplifies data access by automatically mapping struct fields to database columns via struct tags.

## Core Features

- **Struct-driven CRUD**: Operate on typed Go structs instead of raw maps
- **Tag-based Column Mapping**: Use `db` struct tags to define column names
- **Seamless Integration**: Built on top of the database module, supports all database types (PostgreSQL, MySQL, SQLite, Redis, Memory, Bloom, Mixture)
- **Unified Interface**: Same CRUD interface regardless of the underlying database

## Quick Start

```go
package main

import (
    "goutils/crud/model"
    "goutils/database/sqlite"
)

// Define your data model
type User struct {
    ID   int    `db:"id"`
    Name string `db:"name"`
    Age  int    `db:"age"`
}

func main() {
    // Create a database instance (any database type works)
    db := sqlite.NewDatabase("test.db")
    db.Create("users", map[string]api.Config{
        "id":   {Type: "INTEGER", PrimaryKey: true, Identity: true},
        "name": {Type: "TEXT", NullAble: false},
        "age":  {Type: "INTEGER"},
    })

    // Create a model
    userModel, _ := model.NewModel("users", User{}, db)

    // Create a record
    userModel.Create(User{Name: "Alice", Age: 30})

    // Get a record
    var user User
    userModel.Get(map[string]any{"id": 1}, &user)

    // Update a record
    userModel.Update(User{ID: 1, Name: "Alice", Age: 31})

    // Delete a record
    userModel.Delete(map[string]any{"id": 1})
}
```

## API Documentation

### Type Definitions

#### `Model`
```go
type Model struct { ... }
```
A model wraps a database table and provides typed CRUD operations. It maps struct fields to database columns using `db` struct tags.

### Functions

#### `NewModel(tableName string, t any, db api.Database) (*Model, error)`

Creates a new Model instance.

- `tableName`: The name of the database table to operate on
- `t`: An example struct instance or struct pointer used to determine the schema
- `db`: The database instance implementing `api.Database` interface

Returns an error if `t` is not a struct or struct pointer.

### Methods

#### `Create(data any) error`

Inserts a new record into the table.

- `data`: A struct instance with `db` tags. The struct is converted to a map using `db` tags as column names, then passed to the underlying `Table.Create()`.

#### `Get(where map[string]any, dest any) error`

Retrieves a single record matching the given conditions.

- `where`: A map of column-value pairs for filtering (e.g., `{"id": 1}`)
- `dest`: A pointer to a struct where the result will be stored. Uses `db` tags to map database columns to struct fields.

Returns an error if no record is found or if the destination is not a struct pointer.

#### `Update(data any) error`

Updates an existing record.

- `data`: A struct instance with `db` tags. The struct is converted to a map and passed to the underlying `Table.Set()`.

Note: The update operation typically replaces the entire row data matched by primary key or unique fields.

#### `Delete(where map[string]any) error`

Deletes records matching the given conditions.

- `where`: A map of column-value pairs for filtering (e.g., `{"id": 1}`).

## Struct Tag Convention

Fields must use the `db` tag to specify their corresponding column name:

```go
type User struct {
    ID       int    `db:"id"`
    Username string `db:"username"`
    Email    string `db:"email"`
    Age      int    `db:"age"`
}
```

- Fields without a `db` tag are **ignored** during mapping
- The tag value before the first comma is used as the column name
- Both structs and struct pointers are supported

## Error Handling

- `NewModel`: Returns error if the type parameter is not a struct or struct pointer
- `Get`: Returns `"no record found"` if the query returns empty results; returns `"invalid data format"` if the returned data cannot be mapped; returns `"dest must be non-nil struct pointer"` if the destination parameter is invalid
- All methods may return errors from the underlying database operations