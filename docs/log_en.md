# Log Module

> 中文文档：[database_zh.md](database_zh.md) · [http_zh.md](http_zh.md) · [log_zh.md](log_zh.md)

Buffered multi-level logging with color output.

## Quick start

```go
package main

import "github.com/Carry-Rao/goutils/log"

func main() {
	log.Console.Info("service started") // stdout, colorized

	l := log.NewLogger("/var/log/app.log", 4096)
	defer l.Flush()

	l.Info("written to file")
	l.Error("something failed")
}
```

## Levels

| Constant | Value | Method | Behaviour |
|----------|-------|--------|-----------|
| `Debug` | 1 | `Debug` | Debug information |
| `Info` | 2 | `Info` | General information |
| `Warn` | 3 | `Warn` | Warning |
| `Error` | 4 | `Error` | Error |
| `Panic` | 5 | `Panic` | Log, then **flush and panic** |
| — | 5 | `Fatal` | Log, then **flush and os.Exit(1)** |

## Level filtering

`LogLevel` means "emit this level and everything more severe"; higher values are more severe:

```go
l := log.NewLogger("app.log", 4096)
l.LogLevel = log.Warn // only WARN and above

l.Info("not emitted")
l.Warn("emitted")
```

`log.Console` defaults to `Info`.

## Console logger

`log.Console` is a ready-made global logger writing to `os.Stdout` with color enabled:

```go
log.Console.Debug("...") // green
log.Console.Info("...")  // blue
log.Console.Warn("...")  // yellow
log.Console.Error("...") // red
log.Console.Fatal("...") // white on red
```

It is a package-level variable, so its fields can be adjusted directly:

```go
log.Console.LogLevel = log.Debug
log.Console.Color = false
```

Output format:

```
2026-09-27 14:30:05 [INFO] service started
```

With color enabled the level tag is wrapped in ANSI escapes.

## File logger

```go
l := log.NewLogger("/var/log/app.log", 4096)
```

- Opens with `O_CREATE|O_WRONLY|O_APPEND`, creating the file if absent
- **Panics if the file cannot be opened**, so ensure the directory exists and is writable
- `Color` is `false` (no ANSI escapes in the file)
- Returns a `*Logger`, which can be assigned directly to a struct field

## Buffered writes

`Logger.Buffer` is a fixed-size byte buffer and `Len` is its capacity. Entries are appended to the buffer and flushed to disk when it fills.

```go
type Logger struct {
	File     *os.File
	LogLevel LogLevel
	Buffer   []byte
	Len      uint32
	Pos      uint32
	Color    bool
}
```

This substantially reduces the number of write syscalls under high log volume.

### The logger owns the newline

Every entry is terminated with `\n`, appended by `addLog`. Do **not** append `\n` yourself when calling `Info`, `Warn`, and friends, or you will get blank lines. A message passed in may itself contain newlines; multi-line entries are written through as given.

### Buffer size

An entry larger than the buffer is written straight to the file, bypassing it. A buffer that is too small flushes so often that buffering loses its purpose; too large wastes memory and increases the amount of log lost on a crash. Somewhere between 4KB and 64KB is a reasonable starting point, tuned to your average entry size and write rate.

### Explicit flushing

`Flush` writes whatever remains in the buffer. Call it before a normal exit:

```go
l := log.NewLogger("app.log", 4096)
defer l.Flush()
```

`Panic` and `Fatal` already flush internally.

### Concurrency

A `Logger` may be used from multiple goroutines concurrently.

A mutex guards `Pos` and the contents of `Buffer`. This was previously an atomic fast path that claimed space with a CAS and then wrote the bytes, but a concurrent `Flush` could write a region that was claimed and not yet filled, emitting the same entry twice. Locking the whole append keeps the two consistent; against the I/O a flush already performs, the contention is not worth the corruption risk.

Do not copy a `Logger` by value once it is in use.

## Custom logger

Every field is exported, so you can construct one directly:

```go
f, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
if err != nil {
	panic(err)
}

l := &log.Logger{
	File:     f,
	LogLevel: log.Debug,
	Buffer:   make([]byte, 8192),
	Len:      8192,
	Color:    false,
}
```

When constructing one yourself you must set both `Buffer` and `Len`, or writes will not behave correctly.

## Limitations

- **No structured logging**: accepts strings only, with no `key=value` or JSON output
- **No rotation**: files grow without bound; bring your own rotation or use logrotate
- **No `Printf` family**: format with `fmt.Sprintf` yourself
- **`NewLogger` panics on failure**, which does not suit paths that may be unavailable
- `Fatal` calls `os.Exit(1)` directly, so deferred functions do not run

## Performance

Run the benchmarks, including the comparison against the standard library's `log`, with:

```bash
go test -bench=. -benchmem ./log/
```

With buffering on, the CPU cost is dominated by string concatenation and ANSI escapes; `Color: false` is cheaper. The level check happens before any concatenation, so filtered-out entries cost no formatting work — but that check happens *after* the caller's value is built. Writing `l.Info(fmt.Sprintf(...))` pays for `Sprintf` whether or not the entry is emitted, so level filtering pairs best with constant or pre-computed strings.