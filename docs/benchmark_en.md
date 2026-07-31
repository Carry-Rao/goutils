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

Per-entry writes (buffer not full; data stays in the in-memory buffer, no disk I/O):

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| `Debug` | 383.7 | 24 B | 1 |
| `Info` | 408.3 | 24 B | 1 |
| `Error` | 396.1 | 24 B | 1 |
| `InfoColor` | 436.3 | 72 B | 2 |
| `InfoDiscard` | 370.2 | 24 B | 1 |

Highlights:

- Single entry write at ~**380–440 ns/op**; the buffer mechanism avoids frequent disk I/O
- Color output (ANSI escape codes) adds ~30 ns and one extra allocation
- All log levels perform similarly; the level-filter logic itself is negligible

## Database - Memory (in-memory cache)

Hash-map implementation backed by `sync.RWMutex`:

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| `Ins` | 1709 | 279 B | 5 |
| `Get` (hit) | 807.3 | 47 B | 2 |
| `Set` | 966.6 | 112 B | 4 |
| `Del` | 753.9 | 48 B | 2 |
| `GetMiss` | 646.2 | 48 B | 3 |

Highlights:

- Reads at ~**0.8 µs/op**, writes at ~**1–1.7 µs/op**
- Main overhead comes from interface-layer reflection (struct field parsing) and `fmt.Sprintf` key concatenation

## Database - Bloom Filter

1024-bit bloom filter with double FNV hashing for fast existence checks:

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| `Ins` | 1218 | 223 B | 4 |
| `GetHit` | 745.5 | 56 B | 3 |
| `GetMiss` | 681.4 | 40 B | 3 |
| `Contains` | 39.02 | 0 B | 0 |
| `Add` | 99.88 | 0 B | 0 |

Highlights:

- Pure hashing check `Contains` at only ~**39 ns/op** with zero allocations, ideal for high-frequency filtering
- The full Table interface path (reflection + locks) runs at ~0.7–1.2 µs/op

## Database - Mixture (chained databases)

Bloom → Memory two-layer chain, both using the `Continue` strategy:

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| `Ins` | 1229 | 223 B | 4 |
| `GetHit` | 798.0 | 56 B | 4 |
| `Set` | 759.7 | 55 B | 3 |
| `Del` | 709.8 | 40 B | 2 |

Highlights:

- When the first layer (Bloom) handles the request successfully, overall overhead matches a single Bloom layer, ~**0.7–1.2 µs/op**
- Multi-layer chaining adds almost no overhead for first-layer hits

## How to Reproduce

```bash
# Run benchmarks for all modules
go test -bench=. -benchmem -benchtime=1000000x -run=^$ ./http/router/ ./log/ ./database/memory/ ./database/bloom/ ./database/mixture/
```
