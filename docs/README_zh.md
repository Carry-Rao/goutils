# Goutils

[English](../README.md)

一套用于数据库抽象、HTTP 路由与日志的 Go 工具库。

- **需要 Go 1.24+**（使用泛型）

```bash
go get github.com/Carry-Rao/goutils
```

## 模块

| 模块 | 说明 | 文档 |
|------|------|------|
| Database | SQL 的 `Database[T]` / `Table[T]`，支持 MySQL、PostgreSQL、SQLite，另有文件型 JSON 后端。含条件、事务，切片/映射字段 | [中文](database_zh.md) · [English](database_en.md) |
| Cache | 键值 `Table[T]`，支持内存、Redis、布隆过滤器，可用 `mixture` 串接 | [中文](database_zh.md#cache-模块) · [English](database_en.md#cache-module) |
| HTTP Router | 前缀树路由，支持类型化路径变量、中间件、按 pattern 配置 CORS，以及正确的 404/405 区分 | [中文](http_zh.md) · [English](http_en.md) |
| Log | 带缓冲的多级别日志，支持彩色输出 | [中文](log_zh.md) · [English](log_en.md) |

贡献规范见 [CONTRIBUTION_zh.md](CONTRIBUTION_zh.md)。

## Database 速览

```go
import "github.com/Carry-Rao/goutils/database/sql/sqlite"

type User struct {
	ID   int      `db:",primary"`
	Name string
	Tags []string `db:",child"` // 存进 `users_tags`，读回时自动填好
}

db, err := sqlite.NewDatabase[User](map[string]string{"path": "./data.db"})
if err != nil {
	panic(err)
}
defer db.Close()

// 表和附属表不存在时会被创建。
tbl, err := db.GetTable("users")
if err != nil {
	panic(err)
}

if err := tbl.Ins(User{ID: 1, Name: "Alice", Tags: []string{"vip"}},
	database.Options{}); err != nil {
	panic(err)
}

// 返回 []User 而非 []any；条件写在 Options 里，Tags 会被填好。
users, err := tbl.Get(User{ID: 1}, database.Options{
	Where: []database.Condition{database.Contains("Tags", "vip")},
	Order: database.Ascending("Name"),
})

// WHERE 取自旧值，赋值取自新值。Values 为空表示"整条替换"。
err = tbl.Set(User{ID: 1}, User{ID: 1, Name: "Bob"}, database.Options{})

// 多表写入是原子的。
err = db.Tx(func(tx database.Tx[User]) error {
	tt, err := tx.GetTable("users")
	if err != nil {
		return err
	}
	return tt.Del(User{ID: 1}, database.Options{})
})
```

三个 SQL 后端都是同一份实现的薄封装：按需 import，然后调用它的 `NewDatabase`。
没有 factory，也没有任何 TTL。

## 相对上一版的破坏性变更

- **SQL 与缓存拆成两个包。** `database` 只做 SQL（`GetTable(name)`、`Query`、
  `Exec`、`Tx`、`Close`）；`database/cache` 只做键值（`Create`、
  `GetTable(name, example)`，只按主键寻址）。
- **完全没有 TTL。** 过期策略放在上一层做。
- **过滤与赋值全部收进 `Options`**，取代 `whereFields` / `setFields` 和 `ttl`
  参数：
  - `Table[T]`：`Ins(val, Options)`、`Get(val, Options) ([]T, error)`、
    `Set(old, updated, Options)`、`Del(val, Options)`
  - `cache.Table[T]`：`Ins(val)`、`Get(val) ([]T, error)`、`Set(old, updated)`、
    `Del(val)`
- **切片和映射就是普通字段。** 标上 `db:",child"`，`Ins`/`Get`/`Set`/`Del` 会
  自动处理；没有 `Children` 类型，也没有 `AddChild` / `RemoveChild`。
- **查询不再需要写原生 SQL。** `Options.Where` 覆盖比较、`IN`、`BETWEEN`、
  `LIKE`、`IS NULL` 以及集合成员判断；`Options.Values` 提供
  `Union` / `Difference` / `Clear`；`Options.Skip` 排除列。
- 新增 **JSON 后端**：一个文件装下整个数据库，先按表名再按主键索引，集合嵌套在行
  对象里。非主键条件是在 Go 里线性扫描。TLS 设置不再硬编码 —— `sslmode` / `tls`
  从配置 map 读取，不传就省略。
- 405 改为由注册时种下的桩应答，不再遍历所有方法的树；真处理器始终优先于桩。
- 移除了 `database/factory`，后端直接 import。
- `router.BadRequest` 改为 `router.MethodNotAllowed`，状态码 400 → 405
- 路由移除 `:any`，只有 `:string` 是通配且不跨 `/`

## 状态

有端到端测试覆盖的：**SQLite**（含事务、集合字段与集合运算）和 **JSON**（含
原子写入、回滚与线性扫描过滤）。MySQL 与 PostgreSQL 的连接串有字符串级测试覆盖，
但真实连接没有，因为它们需要外部服务。内存、布隆、复合后端也有覆盖。

没有自动化测试覆盖：MySQL、PostgreSQL、Redis 的真实连接。未经自身验证前不建议用于
生产环境。