# Performance Benchmarks

This page summarizes benchmark results across modules to help you understand each module's performance magnitude for selection and capacity planning.

> Benchmark environment: Linux / amd64, CPU: Intel Core i7-6600U @ 2.60GHz, Go 1.24.
> Command: `go test -bench=. -benchmem -benchtime=1000000x -run=^$ ./<module>/`

## HTTP Router Module

Compared against the standard library `net/http.ServeMux`. The router is built on a prefix tree and outperforms the standard library for both static paths and many-route scenarios:

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| `ServeMux_Static` | 133.9 | 0 B | 0 |
| `Router_Static` | 52.45 | 0 B | 0 |
| `ServeMux_Var` | 289.8 | 16 B | 1 |
| `Router_StringVar` | 143.6 | 16 B | 1 |
| `Router_IntVar` | 147.7 | 16 B | 1 |
| `ServeMux_Deep` | 446.9 | 0 B | 0 |
| `Router_Deep` | 290.5 | 0 B | 0 |
| `ServeMux_NotFound` | 4026 | 208 B | 12 |
| `Router_NotFound` | 117.2 | 16 B | 1 |
| `ServeMux_ManyRoutes` | 141.6 | 0 B | 0 |
| `Router_ManyRoutes` | 59.27 | 0 B | 0 |
| `Router_NoMiddleware` | 56.74 | 0 B | 0 |
| `Router_TwoMiddleware` | 69.17 | 0 B | 0 |
| `Router_CORS` | 349.0 | 16 B | 1 |
| `Router_CORSPreflight` | 269.9 | 16 B | 1 |
| `Router_MixedVar` | 465.9 | 112 B | 3 |

Highlights:

- Static path routing at ~**52 ns/op**, roughly **2.5x** the throughput of the standard library
- With 99 static routes registered, ~**59 ns/op** — the prefix tree keeps lookup cost nearly independent of route count
- Not-found (404) handling at ~**117 ns/op**, far ahead of the standard library (~4 µs), good against path-scanning attacks
- Path variables, middleware, and CORS add some overhead but stay in the sub-microsecond range

## Log Module

Per-entry writes, compared against the standard library `log` package:

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| `log.Debug` | 379.4 | 24 B | 1 |
| `log.Info` | 370.2 | 24 B | 1 |
| `log.Error` | 366.2 | 24 B | 1 |
| `log.InfoColor` | 425.3 | 72 B | 2 |
| `log.InfoDiscard` | 360.6 | 24 B | 1 |
| `log.StdlibLog` (stdlib) | 1693 | 0 B | 0 |

Highlights:

- Single entry write at ~**360–430 ns/op**, roughly **4x faster** than stdlib `log` (~1.7 µs/op)
- This module buffers in memory and does not flush until the buffer is full; stdlib `log.Print` writes to the file on every call (OS-buffered), hence the higher cost and zero allocations
- Color output (ANSI escape codes) adds ~50 ns and one extra allocation
- All log levels perform similarly; the level-filter logic itself is negligible

## Database - Memory (in-memory cache)

Hash-map implementation backed by `sync.RWMutex`, compared against a raw `map` baseline:

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| `Ins` | 1395 | 279 B | 5 |
| `StdlibMap_Ins` (raw map) | 1076 | 223 B | 3 |
| `Get` (hit) | 781.7 | 48 B | 2 |
| `StdlibMap_Get` (raw map) | 451.5 | 23 B | 1 |
| `Set` | 985.8 | 112 B | 4 |
| `StdlibMap_Set` (raw map) | 622.0 | 55 B | 2 |
| `Del` | 708.6 | 48 B | 2 |
| `StdlibMap_Del` (raw map) | 523.2 | 23 B | 1 |
| `GetMiss` | 656.1 | 48 B | 3 |

Highlights:

- Reads at ~**0.8 µs/op**, writes at ~**1–1.4 µs/op**
- Roughly **1.3–1.7x slower** than a raw map; overhead comes from interface-layer reflection (struct field parsing), `fmt.Sprintf` key concatenation, and expiration checks
- In exchange you get the unified Database/Table interface and TTL expiration support

## Database - Bloom Filter

1024-bit bloom filter with double FNV hashing for fast existence checks, compared against raw `map` membership:

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| `Ins` | 1453 | 223 B | 4 |
| `GetHit` | 832.2 | 56 B | 3 |
| `GetMiss` | 683.1 | 40 B | 3 |
| `Contains` | 37.43 | 0 B | 0 |
| `StdlibMap_Contains` (raw map) | 28.57 | 0 B | 0 |
| `Add` | 75.90 | 0 B | 0 |

Highlights:

- Pure hashing check `Contains` at ~**37 ns/op** with zero allocations, same order of magnitude as raw map membership (~29 ns)
- The bloom filter's advantage is **fixed memory** (1024 bits) and lookups that do not grow with data size; the trade-off is a false-positive rate. A raw map is faster but memory grows linearly with element count
- The full Table interface path (reflection + locks) runs at ~0.7–1.5 µs/op

## Database - Mixture (chained databases)

Bloom → Memory two-layer chain, both using the `Continue` strategy, compared against raw `map` reads:

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| `Ins` | 1302 | 223 B | 4 |
| `GetHit` | 848.9 | 56 B | 3 |
| `StdlibMap_Get` (raw map) | 397.4 | 7 B | 0 |
| `Set` | 849.9 | 55 B | 3 |
| `Del` | 742.1 | 40 B | 2 |

Highlights:

- When the first layer (Bloom) handles the request successfully, overall overhead matches a single Bloom layer, ~**0.7–1.3 µs/op**
- ~0.4 µs slower than raw map reads (~0.4 µs), the cost of the multi-layer chain (Bloom existence check + Memory storage) and the unified interface
- Multi-layer chaining adds almost no overhead for first-layer hits

## How to Reproduce

```bash
# Run benchmarks for all modules
go test -bench=. -benchmem -benchtime=1000000x -run=^$ ./http/router/ ./log/ ./database/memory/ ./database/bloom/ ./database/mixture/
```
