# Contributing to Goutils

Thank you for considering contributing to Goutils. This document outlines the conventions and guidelines to help you get started.

## Table of Contents

- [Commit Message Convention](#commit-message-convention)
- [Branch Strategy](#branch-strategy)
- [Code Style](#code-style)
- [Pull Request Process](#pull-request-process)
- [Getting Help](#getting-help)

## Commit Message Convention

Commits follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

| Type | Use for |
|------|---------|
| `feat` | New functionality |
| `fix` | Bug fixes |
| `docs` | Documentation only |
| `style` | Formatting changes that do not affect logic |
| `refactor` | Restructuring without adding behaviour or fixing bugs |
| `perf` | Performance work |
| `test` | Adding or correcting tests |
| `chore` | Build, dependencies, CI, tooling |
| `revert` | Reverting an earlier commit |

### Rules

- The subject stays under 50 characters, starts with a verb, and has no trailing period.
- The body explains **why** the change was needed, not a restatement of the diff. Keep it concise.
- The footer closes issues with `Closes #123`, or flags a breaking change with `BREAKING CHANGE: ...`.
- One commit does one thing. A commit that touches both a feature and an unrelated fix should be split.

Scopes in use: `database`, `router`, `log`, `docs`, `ci`.

### Examples

```
feat(router): 支持 :uuid 路径变量

新增 UUID 格式校验,非法格式不再命中路由,避免把非 UUID 段
误判为合法标识。

Closes #42
```

```
fix(log): 修复缓冲日志缺少换行导致的多条记录粘连

缓冲路径写入时未拷贝换行符,且条的长度计算把换行计为 0,导致
所有缓冲记录挤在同一行。改由 addLog 统一追加换行,entryLen 与
实际写入保持一致。
```

### Breaking changes

Anything that changes the public API must be called out. This project is pre-1.0, so breaking changes land in a minor bump.

## Branch Strategy

Branches use `type/short-description`, all lowercase with hyphens.

| Prefix | Use for |
|--------|---------|
| `feature/` | New functionality |
| `fix/` | Bug fixes |
| `refactor/` | Restructuring with no behaviour change |
| `chore/` | Build, dependencies, configuration |
| `docs/` | Documentation |

Long-lived branches:

- `master` — always releasable, only touched by a release or an urgent fix
- `develop` — where day-to-day work lands; branches are cut from here and merged back here with `git merge --squash`

Create a branch only when starting a distinct piece of work. To pull in upstream changes mid-task, `git pull` / `git rebase` on the current branch is enough.

If the project does not adopt `develop`, substitute `master` throughout; the flow is otherwise unchanged.

## Code Style

This project uses **Go** and follows the [official Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) guidelines. Additionally, the CI pipeline performs static analysis with the following rules:

### Function Length

- Functions must be between **4 and 50 lines** (inclusive).
- A function shorter than 4 lines will trigger a CI warning.
- A function longer than 50 lines will trigger a CI warning.

### Error Handling

- **Do not** discard an `error` with a blank identifier. Acceptable discards are non-error results you genuinely do not need.
- Discarding the non-error first result is idiomatic:
  ```go
  // OK: sql.Result (rows affected) is unused, the error is kept
  _, err := db.Exec(query, args...)
  ```
- Swallowing an error outright is not:
  ```go
  // BAD: the write failure is lost
  _, _ = l.File.Write(buf)
  ```
- If an error genuinely cannot be reported — a logger writing its own failure, for example — say why in a comment at the call site.
- **Note**: the AST checker in `.github/workflows/check.go` flags any blank identifier paired with a non-`fmt` call, so `_, err := db.Exec(...)` currently produces a false positive. Treat those warnings as advisory and judge each case against the rules above, rather than rewriting correct code to satisfy the checker.

### Generics

The database module is generic over the model type `T`. Follow these rules when touching it:

- **Never reintroduce `any` at the module boundary.** `Database[T]` and `Table[T]` must stay generic. Use `[]T` in return positions rather than `[]any` plus caller-side assertions.
- **Obtain the schema via `database.SchemaOf[T]()**, not `reflect.TypeOf` on a zero value. `database.TypeOf[T]()` already dereferences pointers — calling `.Elem()` on its result panics, because `Elem()` is only valid on `Ptr`, `Array`, `Slice`, `Map`, and `Chan`.
- Reflection is expected for struct tag reading and field access; Go generics cannot expand struct fields at compile time. Keep reflection out of hot paths by relying on the per-type schema cache.
- Do not add generic function values (e.g. a `func() (Database[T], error)` field or a factory registry). Go does not support generic function types, so backends are constructed through their own `NewDatabase` and `database.Register` only records a name for `Drivers()`.
- Respect the import layering: `database` must not import any internal package, `database/cache` must not import `database`, and `database/mixture` must import only `database/cache`. Adding a backend import to either root package reintroduces an import cycle.

### Database conventions

- **SQL and cache are separate abstractions.** `database` is SQL only; `database/cache` is key-value only. Do not merge them — a cache has no `WHERE`, no column-level assignment and no ordering, and a combined interface is mostly `if`.
- **No TTL.** Expiry is a policy decision that belongs in the layer above the storage call, not a parameter on `Ins`/`Get`. Do not reintroduce one.
- **The three SQL backends share one implementation.** Backends embed `database.Backend[T]` and supply a `Dialect`; DDL generation, condition rendering, collection storage and transactions live in `database.SQLTable[T]`. Adding a fourth backend should mean adding a dialect, not a fourth `Table`.
- A `Dialect` field may only describe a genuine spelling difference (quoting, placeholders, auto-increment, insert-conflict). If two dialects need different *logic*, the shared code is wrong.
- **Collection fields are ordinary fields.** A slice, array or map marked `db:",child"` is stored in a companion table and handled by `Ins`/`Get`/`Set`/`Del`. Do not add a `Children` type, `AddChild`, or `RemoveChild` — the whole point is that callers do not think about the split.
- An unmarked slice or map must keep panicking at first use. Silently dropping it loses data.
- **`Options` is the whole query surface.** Filtering, assignment, ordering and paging belong there, not in positional arguments. `Options.Where` is AND-only by design; `db.Query` is the escape hatch for anything needing grouping or `OR` across fields.
- Every operator must be validated against the schema *before* SQL is generated. `Order` columns and sort keys cannot be bound as parameters, so that check is the only thing stopping a caller string from reaching `ORDER BY`.
- **`Options.Values` semantics are load-bearing.** Empty `Values` means "assign everything from `updated`"; non-empty means "assign exactly these, touch nothing else". Changing this silently destroys data in the second form.
- Every write path goes through `TxRunner`. A table over a `*sql.DB` opens a transaction per write; a table inside `Tx` joins the caller's transaction via `TxTxRunner`. Do not call `t.exec` directly from a write method.
- `Database.Close() error` is part of the interface; a composite reports every layer's failure via `errors.Join`.
- **`Options.SkipSet` and `Options.ValueFor` are exported on purpose.** The SQL and JSON backends must resolve `Skip` and `Values` to columns identically, so that resolution lives in one place. Keep them exported and shared rather than reimplementing it per backend.

### Backends

- **Never hardcode a TLS setting.** `sslmode` and `tls` are read from the config map and omitted when unset, so the driver's own default applies. Baking in `sslmode=disable` silently downgrades every production connection; baking in `verify-full` breaks local servers with no TLS.
- Build DSNs with `net/url` or the driver's own config type. Hand-interpolating credentials with `fmt.Sprintf` breaks on any password containing `@`, `/`, `?` or `#`.
- Any config key the backend does not recognise is forwarded to the driver as a parameter. Consumed keys must be excluded, or they leak back into the query string.
- Watch for `url.URL.Query()`: it returns a **copy**, so parameters set on it are dropped unless written back to `RawQuery`.
- The JSON backend rewrites the whole file per write, through a temp file plus `rename`. Keep it that way — rename is atomic within a filesystem, so a crash cannot leave truncated JSON. Reads work on an immutable snapshot because `update` clones first and swaps the pointer last, which is why handing `s.data` to a reader is safe.
- A transaction gets a store with `transient` set, which makes `update` edit the shared draft **in place**. Cloning again there would strand the changes in a copy nothing commits.
- `Exec` on the JSON backend must decode every row into `T` before writing anything, so one malformed row leaves the table untouched instead of half-replaced.
- Reflection helpers that produce Go values need `reflect.New(t).Elem()`, not `reflect.MakeSlice`/`MakeMapWithSize` — the latter return values a later `Set` will panic on.

### Routing

- Keep route matching side-effect free. `pathTree.match` resolves a path only; middleware and handlers run afterwards in `routeMatch.exec`. Do not execute middleware during traversal.
- Any new variable type must be added in four places: the `Type` constants, `matchType`, the `varPriority` array, and `parseVarType` for registration. Omitting `varPriority` makes the type unreachable.
- `:string` is the single-segment catch-all and must not span `/`. There is deliberately no multi-segment wildcard; use `Static` or a `Sub` router for path tails.
- **A 405 stub must never shadow a real route.** `pathTree.resolve` picks the literal before the variables and then `varPriority`, but any candidate holding a real handler wins over one holding a stub. Dropping that tie-break lets the stub planted for `/user/:int` hide a real `POST /user/:string`, since `Int` outranks `String`. `http/router/method_test.go` pins this down.
- Registering a route must clear `MethodNotAllowed` on that node. `All` relies on it: each method overwrites the stub the previous registration planted. Leaving the flag set makes a real handler report 405.
- `Router.New` pre-creates a tree per `defaultMethods` entry, because `plantStubs` needs somewhere to plant. Adding a method to `defaultMethods` gives it the same treatment; `OPTIONS` is absent on purpose and still falls back to probing.
- `pathExists` deliberately ignores stubs — a 405 stub is not evidence that a path is allowed, and counting it would put every method in `Allow`.

### General Go Conventions

- Run `go fmt` before committing to ensure consistent formatting.
- Use meaningful variable and function names.
- Keep packages focused and single-purpose.
- Write unit tests for new functionality. CI runs `go test ./... -race`; changes to `database/` and `http/router/` must keep the existing tests green and should add coverage for new behavior.
- Document exported functions, types, and constants with Go-style comments.

## Pull Request Process

1. Create a branch from `develop` (or `master` if the project has no `develop`).
2. Make your changes following the code style guidelines above.
3. Keep commits small and single-purpose, following the commit convention.
4. Ensure tests pass locally and add coverage for new behaviour.
5. Run `go fmt ./...`.
6. Open a pull request against `develop`. The PR title uses the same Conventional Commits format as the commit.
7. In the PR description, include:
   - A summary of the changes
   - The motivation for the change
   - Any relevant issue numbers
8. Branches are merged with `git merge --squash`, so the PR lands as one tidy commit. Rewrite the squashed message rather than reusing the branch's first commit message.
9. Keep the merged remote branch until the next release; after a squash the branch node is no longer in the trunk history, so the PR record is the main way to trace it.

## Getting Help

If you have questions or need help, feel free to open an issue on GitHub.
