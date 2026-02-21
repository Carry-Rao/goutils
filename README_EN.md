# goutils
- A comprehensive utility toolkit for Golang
- Enables developers to focus on business logic rather than repetitive tasks
- Makes code more concise, readable, and maintainable

## Installation
** Notice: ** You must have Go 1.24.4 or later installed on your system.
First, enable Go Modules:
```bash
export GO111MODULE=on
```
Second, initialize your project:
```bash
go mod init xxx # Replace xxx with your project name
```
Then execute the following command in your project directory to install goutils:
```bash
go get -u github.com/Carry-Rao/goutils
```

## Usage
### Database
#### SQL
Supports the following commonly used SQL databases:
+ MySQL
+ PostgreSQL
+ SQLite3

Import in your code:
```go
import "github.com/Carry-Rao/goutils/database/sql"
```
And use directly:
```go
db, err := sql.Open(driver, dsn)
```
to open a database connection.

#### Key-Value (KV)
Supports the following commonly used KV databases:
+ Redis
+ Golang native map

Import in your code:
```go
import "github.com/Carry-Rao/goutils/database/kv"
```
And use directly:
```go
redis, err := NewClient(&kv.Options{
    Addr:     address,
    Password: password,
    DB:       db,
    Protocol: protocol,
}) // Connect to Redis

mem, err := kv.NewClient(&kv.Options{
    Memory: true,
}) // Connect to in-memory KV store
```
to open a KV store connection.

### Logging
Supports logging output to:
+ Console
+ File

Import in your code:
```go
import "github.com/Carry-Rao/goutils/log"
```
And use directly:
```go
logger := log.NewConsoleLogger(false) // Output to console (stderr)
logger.SetLevel(log.Debug)            // Set log level
logger.Debug("debug message")         // Output debug log
logger.Info("info message")           // Output info log
logger.Warn("warn message")           // Output warn log
logger.Error("error message")         // Output error log
```
to output logs.

-------
There are additional features that I haven't had time to document in the README yet, which will be added gradually.
