# 贡献指南

感谢你考虑为 Goutils 贡献代码。本文说明约定与流程。

## 目录

- [提交信息规范](#提交信息规范)
- [分支策略](#分支策略)
- [代码风格](#代码风格)
- [Pull Request 流程](#pull-request-流程)

## 提交信息规范

采用 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>(<scope>): <subject>

<body>

<footer>
```

### 类型

| 类型 | 用途 |
|------|------|
| `feat` | 新功能 |
| `fix` | 修复缺陷 |
| `docs` | 仅文档改动 |
| `style` | 不影响逻辑的格式调整 |
| `refactor` | 重构（非新增功能、非修 bug） |
| `perf` | 性能优化 |
| `test` | 补充或修正测试 |
| `chore` | 构建、依赖、CI、工具链 |
| `revert` | 回滚某次提交 |

### 规则

- subject 不超过 50 字符，动词开头，不加句号。
- body 说明**为什么**要这么改，而不是复述 diff 改了什么。保持简洁。
- footer 用 `Closes #123` 关闭 issue，用 `BREAKING CHANGE: ...` 标注破坏性变更。
- 一次提交只做一件事。既改功能又夹带无关修复的，应当拆开。

本项目使用的 scope：`database`、`router`、`log`、`docs`、`ci`。

### 示例

```
feat(router): 支持 :uuid 路径变量

新增 UUID 格式校验,非法格式不再命中路由,避免把非 UUID 段
误判为合法标识。

Closes #42
```

```
fix(log): 修复缓冲日志缺少换行导致的多条记录粘连

缓冲路径写入时未拷贝换行符,且条长度计算把换行计为 0,导致
所有缓冲记录挤在同一行。改由 addLog 统一追加换行,entryLen 与
实际写入保持一致。
```

### 破坏性变更

任何改动公开 API 的提交都必须显式标注。本项目尚未发布 1.0，破坏性变更会随次版本号一起发布。

## 分支策略

分支统一使用 `类型/简要描述`，全部小写，单词间用短横线连接。

| 前缀 | 用途 |
|------|------|
| `feature/` | 新功能 |
| `fix/` | 缺陷修复 |
| `refactor/` | 不改变行为的重构 |
| `chore/` | 构建、依赖、配置 |
| `docs/` | 文档 |

长期分支：

- `master` —— 随时可发布的稳定状态，只在发版或线上紧急修复时被改动
- `develop` —— 日常开发的汇总分支，功能分支从此切出并合回，合并方式为 `git merge --squash`

只在开始一段独立的新工作时才新建分支。若只是把上游改动同步进当前分支，用 `git pull` / `git rebase` 即可，无需另开分支。

若项目不采用 `develop`，把上文的 `develop` 全部替换为 `master`，流程不变。

## 代码风格

遵循 [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)。CI 还会执行以下静态检查。

### 函数长度

函数长度需在 **4 到 50 行**（含）之间，过短或过长会产生警告。

### 错误处理

- **不要**用 `_` 丢弃 `error`。可以丢弃的是你确实不需要的非 error 返回值。
- 丢弃第一个非 error 返回值是惯用写法：
  ```go
  // 正确：sql.Result（影响行数）不需要，但 error 保留
  _, err := db.Exec(query, args...)
  ```
- 直接吞掉 error 则不可接受：
  ```go
  // 错误：写入失败被丢弃
  _, _ = l.File.Write(buf)
  ```
- 若某处确实无法上报 error（例如 logger 写自身失败），请在该调用点注释说明原因。
- **注意**：`.github/workflows/check.go` 的 AST 检查会把「`_` + 非 `fmt` 调用」一律标记出来，因此 `_, err := db.Exec(...)` 目前属于误报。请按上面的规则逐条判断，不要为了消警告而改写正确的代码。

### 泛型约定

database 模块以模型类型 `T` 为泛型参数，修改时请遵守：

- **不要在模块边界重新引入 `any`**。`Database[T]` 与 `Table[T]` 必须保持泛型；返回位置用 `[]T`，而不是 `[]any` 加调用方断言。
- **用 `database.SchemaOf[T]()` 取 schema**，不要对零值调 `reflect.TypeOf`。`database.TypeOf[T]()` 已解引用指针，再调 `.Elem()` 会 panic —— `Elem()` 只对 `Ptr`、`Array`、`Slice`、`Map`、`Chan` 合法。
- struct tag 读取与字段访问依赖反射是预期内的：Go 泛型无法在编译期展开结构体字段。依靠按类型缓存的 schema 把反射挡在热路径之外。
- **不要引入泛型函数值**（例如 `func() (Database[T], error)` 类型的字段或工厂注册表）。Go 不支持泛型函数类型，因此后端通过各自的 `NewDatabase` 构造，`database.Register` 只记录名字供 `Drivers()` 列出。
- **遵守导入分层**：`database` 不得导入任何内部包；`database/cache` 不得导入 `database`；`database/mixture` 只能导入 `database/cache`。往任一根包加后端导入都会重新引入循环依赖。

### database 模块约定

- **SQL 与缓存是两个独立抽象。** `database` 只做 SQL，`database/cache` 只做键值。不要合并 —— 缓存没有 `WHERE`、没有列级赋值、没有排序，硬合并出来的接口大半都是 `if`。
- **完全没有 TTL。** 过期是策略，应该放在存储调用之上的那一层，而不是 `Ins`/`Get` 的参数。不要重新引入。
- **三个 SQL 后端共用一份实现。** 后端嵌入 `database.Backend[T]` 并给出 `Dialect`；建表、拼条件、集合存储、事务都在 `database.SQLTable[T]` 里。加第四个后端应该是加一个方言，而不是第四份 `Table`。
- `Dialect` 的每个字段只能描述真正的**拼写**差异（引号、占位符、自增、冲突处理）。如果两个方言需要不同的**逻辑**，说明共用那份代码写错了。
- **集合字段就是普通字段。** 标了 `db:",child"` 的切片/数组/映射存在附属表里，由 `Ins`/`Get`/`Set`/`Del` 处理。不要新增 `Children` 类型或 `AddChild`/`RemoveChild` —— 重点正是让调用方不必关心这层拆分。
- 没标 `child` 的切片/映射必须继续在首次使用时 panic。默默丢掉等于丢数据。
- **`Options` 就是全部查询入口。** 过滤、赋值、排序、分页都归它，不放位置参数。`Options.Where` 刻意只支持 AND；需要分组或字段间 `OR` 时用 `db.Query` 兜底。
- 每个运算符都必须在生成 SQL **之前**对 schema 校验。`Order` 与排序字段无法用参数绑定，这个校验是唯一能挡住调用方字符串进入 `ORDER BY` 的东西。
- **`Options.Values` 的语义是要命的。** `Values` 为空表示"全部取 `updated` 覆盖"；非空表示"只动点名的，其余一律不动"。改这条会让第二种用法静默丢数据。
- 所有写路径都走 `TxRunner`。基于 `*sql.DB` 的表每次写开一个事务；`Tx` 里的表通过 `TxTxRunner` 加入调用方的事务。不要在写方法里直接调 `t.exec`。
- `Database.Close() error` 属于接口的一部分；无资源的后端返回 nil，
  复合后端用 `errors.Join` 上报全部失败。

### 路由约定

- 保持路由匹配无副作用。`pathTree.match` 只解析路径，中间件与 handler 之后由 `routeMatch.exec` 执行，不要在遍历过程中执行中间件。
- 新增变量类型需同步修改四处：`Type` 常量、`matchType`、`varPriority` 数组、`parseVarType`（注册用）。漏掉 `varPriority` 会导致该类型无法被匹配到。
- `:string` 是单片段 catch-all，**不得**跨 `/`。本库刻意不提供跨斜杠的通配变量，尾部多段路径请用 `Static` 或 `Sub` 路由。
- 404/405 的判定依赖遍历全部方法的路由树。修改 `defaultMethods` 后请确认 `Allow` 头的内容仍然正确。

### 通用约定

- 提交前执行 `go fmt`。
- 使用有意义的变量与函数命名。
- 保持包职责单一。
- 新功能需补测试。CI 会执行 `go test ./... -race`，改动 `database/` 与 `http/router/` 时须保证既有测试通过，并为新行为补充覆盖。
- 导出的函数、类型、常量使用 Go 风格注释说明。

## Pull Request 流程

1. 从 `develop` 切出功能分支（若项目没有 `develop`，则从 `master` 切出）。
2. 按上述代码风格规范完成改动。
3. 提交粒度要小且单一职责，遵循提交信息规范。
4. 本地跑通测试，并为新行为补充覆盖。
5. 执行 `go fmt ./...`。
6. 向 `develop` 提交 PR，标题沿用与提交相同的 Conventional Commits 格式。
7. PR 描述需包含：变更摘要、变更动机、关联 issue。
8. 分支以 `git merge --squash` 合并，因此会以一条干净提交落到主干。squash 后的提交信息需重写，不要沿用分支上第一条提交的信息。
9. 已合并的远程分支保留到下次发版为止 —— squash 后主干历史不再保留分支节点，PR 记录是主要追溯依据。

## 获取帮助

如有疑问，欢迎提 issue。