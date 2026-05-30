# Log Module

This module provides a unified logging capability, supporting multi-level logging, buffered writes, and colored output.

## Core Features

- **Multi-level Logging**: Supports five log levels: DEBUG, INFO, WARN, ERROR, PANIC
- **Buffered Write**: Buffer mechanism to reduce frequent I/O operations and improve performance
- **Color Output**: Supports terminal color output for easy log level identification
- **Auto Flush**: PANIC and FATAL level logs automatically flush the buffer and terminate the program
- **Level Filtering**: Control log output granularity by setting the log level

## Log Levels

| Level | Value | Color   | Description                     |
|-------|-------|---------|---------------------------------|
| DEBUG | 1     | Green   | Debugging information           |
| INFO  | 2     | Blue    | General information             |
| WARN  | 3     | Yellow  | Warning information             |
| ERROR | 4     | Red     | Error information               |
| PANIC | 5     | White on Red | Fatal error, triggers panic |

## Quick Start

```go
package main

import (
    "os"
    "goutils/log"
)

func main() {
    // Create a log file
    file, _ := os.Create("app.log")
    defer file.Close()

    // Initialize the logger
    logger := &log.Logger{
        File:     file,           // Output file
        LogLevel: log.Debug,      // Log level (lowest level, outputs all logs)
        Len:      1024,           // Buffer size (bytes)
        Color:    true,           // Enable color output
    }

    // Record logs at different levels
    logger.Debug("This is a debug message")
    logger.Info("This is an info message")
    logger.Warn("This is a warning message")
    logger.Error("This is an error message")

    // Manually flush the buffer
    logger.Flush()
}
```

## API Documentation

### Type Definitions

#### `LogLevel`
```go
type LogLevel int
```
Log level enum type, with values:
- `Debug = 1`
- `Info  = 2`
- `Warn  = 3`
- `Error = 4`
- `Panic = 5`

#### `Log`
```go
type Log struct {
    LogLevel LogLevel
    LogInfo  string
}
```
Represents a log entry, containing the log level and log content.

#### `Logger`
```go
type Logger struct {
    File     *os.File  // Log output file
    LogLevel LogLevel  // Log level filter
    Buffer   string    // Log buffer
    Len      uint      // Buffer size (bytes)
    Color    bool      // Whether to enable color output
}
```
Logger instance for recording and managing logs.

### Methods

#### `Debug(info string)`
Logs a DEBUG level message. Only outputs when the log level >= Debug.

#### `Info(info string)`
Logs an INFO level message. Only outputs when the log level >= Info.

#### `Warn(info string)`
Logs a WARN level message. Only outputs when the log level >= Warn.

#### `Error(info string)`
Logs an ERROR level message. Only outputs when the log level >= Error.

#### `Panic(info string)`
Logs a PANIC level message and triggers a panic. After writing the log, it calls `Flush()` to flush the buffer, then executes `panic(info)` to terminate the program. Only triggers when the log level >= Panic.

#### `Fatal(info string)`
Logs a FATAL level message and exits the program. After writing the log, it calls `Flush()` to flush the buffer, then executes `os.Exit(1)` to exit the program. Only triggers when the log level >= Panic (FATAL uses the PANIC level).

#### `Flush()`
Manually flushes the buffer, writing all buffered logs to the file.

## Buffer Mechanism

The logger uses a buffered write mechanism to reduce disk I/O:
1. Each log entry is first written to the in-memory buffer
2. When the buffer size reaches `Len` bytes, its contents are automatically written to the file
3. Before the program ends, `Flush()` should be called to ensure all buffered logs are written to the file
4. `Panic` and `Fatal` methods automatically call `Flush()`, so no manual flushing is needed

## Level Filtering

By setting `Logger.LogLevel`, you can control the granularity of log output. Only logs with a level greater than or equal to the set value will be recorded.

For example, when `LogLevel = Warn` is set, only WARN, ERROR, and PANIC level logs will be output; DEBUG and INFO level logs will be ignored.

## Color Output

Setting `Logger.Color = true` enables terminal color output. Different levels use different colors for easy identification:

- **DEBUG**: `\033[32m` (Green)
- **INFO**: `\033[34m` (Blue)
- **WARN**: `\033[33m` (Yellow)
- **ERROR**: `\033[31m` (Red)
- **PANIC/FATAL**: `\033[41;37m` (White on Red)

Color output is only visible in the terminal. It is recommended to disable color output when writing to a file.