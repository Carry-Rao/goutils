# Database 模块

> English: [database_en.md](database_en.md) · [http_zh.md](http_zh.md) · [log_zh.md](log_zh.md)

两个互相独立的抽象：

- **`database`** —— 只做 SQL。表、列、条件、排序、分页、事务。
- **`database/cache`** —— 只做键值。只能按主键寻址，别的都不做。

刻意分开是因为缓存没有 `WHERE`、没有列级赋值、没有排序，硬塞进一个接口里
大半都是 `if`。拆开之后各自的接口才诚实。

## 目录结构

```
database/                     SQL：接口、schema、方言、Options、共用的 Table
database/sql/mysql/           薄后端，只嵌入 database.Backend[T]
database/sql/postgresql/
database/sql/sqlite/
database/cache/               键值：自己的 Table、Schema、Config
database/cache/memory/
database/cache/bloom/
database/cache/redis/
database/mixture/             把多个 cache table 串成一条查找链
```

没有循环依赖：`database` 不引用任何内部包；每个后端只引用根包；`mixture` 只
引用 `database/cache`。

三个 SQL 后端各自只有约 30 行。共用的部分——建表、拼条件、集合存储、事务——
只写了一份，在 `database.SQLTable[T]` 里。后端只提供一个 `*sql.DB` 和一个方言：

```go
type Database[T any] struct {
	database.Backend[T]
}

func NewDatabase[T any](cfg map[string]string) (*Database[T], error) {
	db, err := sql.Open("sqlite3", cfg["path"])
	// ...
	return &Database[T]{Backend[T]{DB: db, Dialect: database.SQLiteDialect}}, nil
}
```

## 模型定义

```go
type User struct {
	ID    int               `db:",primary"`
	Name  string
	Email string            `db:"email_address"`
	Age   int               `db:",null"`
	Tags  []string          `db:",child"`
	Meta  map[string]string `db:",child"`
}
```

tag 语法是 `column[,constraint]...`。**第一段可以留空**，留空表示"列名跟字段名
一致"，其后每一段都是约束：

| tag | 含义 |
|-----|------|
| *（不写）* | 列名同字段名，NOT NULL |
| `db:"email_address"` | 指定列名 |
| `db:",primary"` | 主键 |
| `db:",autoinc"` | 自增，`INSERT` 时不写入 |
| `db:",unique"` | 唯一索引 |
| `db:",null"` | 可空 |
| `db:",child"` | 切片/映射，存到附属表 |
| `db:",child=tag_links"` | 显式指定附属表名 |

必须且只能有一个 `primary` 字段；没有主键的模型在打开表时就会被拒绝。目前只
支持单列主键。

### 集合就是普通字段

标了 `child` 的切片、数组或映射存进自己的表，但读写方式和普通字段完全一样——
没有单独的 `Children` 类型，没有 `AddChild`，也不用记得去多调一个函数：

```go
u := User{ID: 1, Name: "alice", Tags: []string{"vip", "beta"}}
tbl.Ins(u, database.Options{})

got, _ := tbl.Get(User{ID: 1}, database.Options{})
got[0].Tags // []string{"vip", "beta"} —— 自动填好
```

`Get` 对每个集合字段多发一条查询（`WHERE parent_id IN (...)`）并回填。没有元素
时切片是**非 nil 的空切片**，所以 `len(u.Tags) == 0` 可以直接用，不必判空。

没标 `child` 的切片或映射会在**首次使用时 panic**。默默丢掉等于丢数据，所以
这里选择把错误吵出来。

元素类型必须能通过文本列往返，即 `string`、各整数与浮点类型、或 `bool`。
`[]SomeStruct` 这样的子字段会在 schema 阶段被拒绝，而不是等到回填时才 panic。
映射的键同样会被检查。

### 表名

附属表默认是主表名加小写列名：

| 主表 | 字段 | 附属表 |
|------|------|--------|
| `user` | `Tags` | `user_tags` |
| `users` | `Meta` | `users_meta` |

表结构随字段类型变化：

```sql
-- 切片 Tags []string
CREATE TABLE users_tags (Tags TEXT, parent_id INTEGER, UNIQUE(Tags, parent_id));

-- 映射 Meta map[string]string
CREATE TABLE users_meta (k TEXT, v TEXT, parent_id INTEGER, UNIQUE(k, v, parent_id));
```

`parent_id` 取自主键列名，所以主键叫 `uid` 时附属列就是 `parent_uid`。

## 打开数据库

直接 import 需要的后端调用构造函数，没有 factory，也没有按名字驱动的中间层：

```go
import "github.com/Carry-Rao/goutils/database/sql/sqlite"

db, err := sqlite.NewDatabase[User](map[string]string{"path": "./data.db"})
defer db.Close()
```

配置是 `map[string]string`，可以直接从配置文件里取出来用：

```go
db, err := mysql.NewDatabase[User](map[string]string{
	"user": "root", "password": "...", "host": "127.0.0.1",
	"port": "3306", "dbname": "app",
})
```

`T` 可以是结构体，也可以是指向它的指针，两者共用同一份 schema。

`Close() error` 在接口里，所以 `defer db.Close()` 是被检查的，不会被静默忽略。

## 表

`GetTable` 会对主表和每个附属表执行 `CREATE TABLE IF NOT EXISTS`，所以不需要
单独的迁移步骤：

```go
tbl, err := db.GetTable("users")
```

`DeleteTable` 会连同附属表一起删掉：

```go
err := db.DeleteTable("users")
```

## 增删改查

```go
err := tbl.Ins(User{ID: 1, Name: "alice"}, database.Options{})

users, err := tbl.Get(User{ID: 1}, database.Options{})

err = tbl.Set(User{ID: 1}, User{ID: 1, Name: "bob"}, database.Options{})

err = tbl.Del(User{ID: 1}, database.Options{})
```

`Ins` 在一个事务里写入主行和集合行，失败时不会留下半截数据。`Del` 会一并删除
集合行。

`opts.Where` 为空时，`Set` 和 `Del` 退回到传入值的主键，上面这种一行写法就靠
这个。

## Options

过滤、赋值、排序、分页全都收在 `Options` 里，零值表示"全部"。

```go
type Options struct {
	Where  []Condition
	Values []Value
	Skip   []string
	Order  []Order
	Limit  int
	Offset int
}
```

### 条件

条件**只有 AND**，没有分组，字段之间也没有 `OR`——集合上的或用 `ContainsAny`，
更复杂的情况直接用 `Query`。

| 构造函数 | SQL |
|----------|-----|
| `Eq` `Neq` | `=` `!=` |
| `Gt` `Gte` `Lt` `Lte` | 比较 |
| `Like` `NotLike` | `LIKE`，`%` 自己写 |
| `In` `NotIn` | `IN (…)` |
| `Between` | `BETWEEN lo AND hi` |
| `IsNull` `IsNotNull` | `IS NULL` / `IS NOT NULL` |
| `Contains` | 切片含该元素；映射含该键 |
| `ContainsAll` | 每个元素都要在 |
| `ContainsAny` | 至少有一个元素在 |
| `ContainsKV` | `map[key] == value` |

```go
users, err := tbl.Get(User{}, database.Options{
	Where: []database.Condition{
		database.Gte("Age", 18),
		database.Contains("Tags", "vip"),
		database.ContainsKV("Meta", "plan", "pro"),
	},
	Order: database.Descending("Age"),
	Limit: 20,
})
```

集合条件会编译成对附属表的相关 `EXISTS`，在 SQL 层完成关联，不把数据捞到 Go
里再比：

```sql
WHERE `Age` >= ?
  AND EXISTS (SELECT 1 FROM `users_tags` c
              WHERE c.`parent_id` = `users`.`ID` AND c.`Tags` = ?)
```

`ContainsAll` 为每个元素各生成一条 `EXISTS` 再用 `AND` 连起来，因为一行装不下
全部元素；`ContainsAny` 则是一条 `EXISTS` 内部用 `OR`。

**每个运算符都会先对着 schema 校验，然后才生成 SQL。** 对集合用 `Eq("Tags", "x")`、
对切片用 `ContainsKV`，都会立刻返回错误，而不是拼出无意义的 SQL。排序字段同理
——它没法作为参数绑定，调用方给的字符串必须先校验，否则就会原样进到 `ORDER BY`
里。

### 赋值

`Values` 决定 `Set` 怎么改集合：

| 构造函数 | 效果 |
|----------|------|
| `Union` | 合并进去 |
| `Difference` | 移除 |
| `Clear` | 清空 |
| `Assign` | 替换该列 |

```go
tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
	Values: []database.Value{
		database.Union("Tags", []string{"gamma"}),
		database.Clear("Meta"),
	},
})
```

### `Values` 的规则

这条要单独讲清楚，猜错会丢数据：

- **`Values` 为空** —— 所有标量列和集合列都取 `updated` 的值覆盖。这就是平常
  的"整条替换"。
- **`Values` 非空** —— 它就是**完整的**指定范围。只动它点名的列，其余一律不动。

```go
// updated 里没有 Meta，Values 也没提到它，所以 Meta 保留原样。
tbl.Set(User{ID: 1}, User{ID: 1}, database.Options{
	Values: []database.Value{database.Union("Tags", []string{"gamma"})},
})
// Meta 仍然是原来的内容。
```

`Skip` 与之并存且优先级更高：出现在 `Skip` 里的列永远不会被写，即使它在
`Values` 里被点名。

```go
tbl.Set(User{ID: 1}, User{ID: 1, Name: "new", Age: 99}, database.Options{
	Skip: []string{"Age"},
})
```

### 排序与分页

```go
database.Asc("Age")            database.Ascending("Age")     // []Order
database.Desc("Age")           database.Descending("Age")   // []Order
```

只有 `Offset` 没有 `Order` 时结果不稳定——行序由后端决定。

## 事务

`Tx` 原子地执行一个函数。返回 error 就整体回滚，并且你返回的这个 error 就是外面
拿到的那个：

```go
err := db.Tx(func(tx database.Tx[User]) error {
	users, err := tx.GetTable("users")
	if err != nil {
		return err
	}
	if err := users.Ins(User{ID: 2, Name: "bob"}, database.Options{}); err != nil {
		return err
	}
	posts, err := tx.GetTable("posts")
	if err != nil {
		return err
	}
	return posts.Ins(Post{ID: 1, UserID: 2}, database.Options{})
})
```

从 `tx` 拿到的表会加入当前事务，不会再开一个嵌套的。没有显式 `Tx` 时，每次
`Ins`/`Set`/`Del` 自己也跑在事务里，所以单次写入不会出现写一半。

`Tx[T]` 提供 `GetTable`、`Query`、`Exec`——除了不能再开事务，和 `Database[T]`
一样。

## 原生 SQL

`Options` 覆盖不到的情况直接写 SQL：

```go
rows, err := db.Query("SELECT * FROM users WHERE Age > ?", 30)
err = db.Exec("UPDATE users SET Name = ? WHERE ID = ?", "renamed", 1)
```

`Tx` 上也有这两个方法。`Query` 用同一套 schema 映射扫成 `[]T`，集合字段留空，
需要集合内容请用 `Get`。

## Cache 模块

`database/cache` 只能按主键寻址，所以接口就是四个方法、没有 options：

```go
type Table[T any] interface {
	Ins(val T) error
	Get(val T) ([]T, error)
	Set(old, updated T) error
	Del(val T) error
}
```

`Set` 分开接收新旧两个值，这样主键变了是**移动**记录，而不是多出一条。

```go
import "github.com/Carry-Rao/goutils/database/cache/memory"

db, err := memory.NewDatabase[User](nil)
err = db.Create("users", map[string]cache.Config{"id": {PrimaryKey: true}})

tbl, err := db.GetTable("users", User{})
if tbl == nil {
	return errors.New("table users not created, call Create first")
}
```

缓存表必须先用 `Create` 声明——没有 schema 可以反推 DDL，也就没有 `IF NOT
EXISTS` 可以倚靠。表不存在时 `GetTable` 返回 `(nil, nil)`，所以**要判断表是否
为 nil**。

后端有 `memory`、`redis`、`bloom`，`cache.Register` / `cache.Drivers` 可以列出
它们。

**这里没有 TTL。** 需要过期就放在上一层做：Redis 和 SQL 后端都有原生支持，把
策略埋进存储调用只会让它看不见。

### Bloom 过滤器

适合键空间极大、能接受误判、也不需要把值读回来的场景——比如判断某个 ID *是否
可能*存在。

```go
import "github.com/Carry-Rao/goutils/database/cache/bloom"

if _, err := tbl.Get(User{ID: id}); err == nil {
	// 大概率存在
} else if errors.Is(err, cache.ErrNotFound) {
	// 一定不存在
}
```

未命中是 `cache.ErrNotFound` 而不是空结果，这样调用方能区分"不存在"和"过滤器
有漏判"（它不会漏）。不存在一定正确；存在可能不准。

### Redis

Redis 用 Hash 存每一行，键是 `{table}_{主键值}`，字段是各列。`Create` 必须先指定
主键。值按字符串存，读的时候按字段类型转回去；转换失败的字段保留零值。

## Mixture 串接

把多个存储串成一条查找链，常见形态是 过滤器 → 缓存 → 数据库。

```go
import "github.com/Carry-Rao/goutils/database/mixture"

chain := mixture.New[User]()
chain.Add(bloomTbl, mixture.Continue) // 未命中或出错就往后走
chain.Add(redisTbl, mixture.Continue)
chain.Add(mysqlTbl, mixture.Return)   // 权威存储：错在这里就返回

var users cache.Table[User] = chain.Table()
```

读返回第一个非空结果。**没报错但没数据**不算命中，所以链会继续往下走。

写会到达**每一层**，而不是第一个接受的就停——过滤器、缓存、存储都得保持同步。
`Close` 会汇总每层的错误，而不是遇到第一个就返回。

泛型只允许一种类型，所以所有层必须共用 `T`。

命中时 `Get` **不会**回填下层；需要回填请在自己的层里实现。

## 编写自定义后端

SQL 后端嵌入 `database.Backend[T]` 并给出方言，`Table[T]` 就都有了。键值存储
则实现 `cache.Table[T]`。

```go
type Database[T any] struct {
	database.Backend[T]
}
```

后端可以在 `init()` 里注册名字，好让它出现在 `Drivers()` 里：

```go
func init() { database.Register("mydb") }
```

`Register` 只是信息性的，方便列出可用后端。构造一律走后端自己的构造函数，
因为 Go 没有泛型函数类型，`func(cfg) (Database[T], error)` 没法存进注册表。

## 反射开销

只有 `reflect` 能读结构体 tag，所以字段映射离不开它。有两点缓解：

1. 解析结果按类型缓存在 `sync.Map` 里，每种类型只解析一次。
2. `database.SchemaOf[T]()` 直接返回缓存好的 schema。

`Ins`/`Set`/`Get` 每次调用仍然要用反射读写字段——泛型没法在编译期展开结构体
字段。对 SQL 后端来说开销主要在网络 I/O 上，反射那部分可以忽略。
