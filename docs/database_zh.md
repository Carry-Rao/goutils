# 数据库模块文档

## 快速开始

使用 `database.NewDatabase()` 创建数据库实例，通过 `api.DbType` 枚举指定数据库类型：

```go
import (
    "github.com/Carry-Rao/goutils/database"
    "github.com/Carry-Rao/goutils/database/api"
)

// 创建 MySQL 实例
db, _ := database.NewDatabase(api.MySQL, map[string]string{
    "user":     "root",
    "password": "123456",
    "host":     "127.0.0.1",
    "port":     "3306",
    "dbname":   "test",
})

// 创建表
db.Create("users", map[string]api.Config{
    "id":   {Type: "INT", PrimaryKey: true, Identity: true},
    "name": {Type: "VARCHAR(255)", NullAble: false},
})

// 获取表操作对象（需传入 struct 示例用于反射）
table, _ := db.GetTable("users", &User{})

// CRUD 操作（使用 struct 和字段名列表）
table.Ins(&User{Name: "Alice", Email: "alice@example.com"}, 0)           // 插入
result, _ := table.Get(&User{Name: "Alice"}, []string{"Name"}, 0)         // 查询
table.Set(&User{Name: "Alice", Email: "alice@new.com"}, []string{"Name"}, 0) // 更新
table.Del(&User{Name: "Bob"}, []string{"Name"}, 0)                     // 删除

// 删除表
db.DeleteTable("users")
```

### 类型安全的泛型用法

使用 `typed.TypedTable[T]` 获得编译期类型检查：

```go
import "github.com/Carry-Rao/goutils/database/typed"

rawTable, _ := db.GetTable("users", &User{})
table := typed.NewTable[User](rawTable)

// 编译期类型安全
table.Ins(User{Name: "Alice"}, 0)
users, _ := table.Get(User{}, []string{"name"}, 0) // 返回 []User，不是 []any
```

## 核心概念

数据库模块采用 **结构体驱动** 的设计模式：

- 每个数据库表对应一个 Go 结构体，通过 `db` tag 标记字段名和约束
- `Insert`/`Get`/`Set`/`Delete` 均以结构体实例作为参数
- 使用字段名列表指定 WHERE 条件，避免手写 SQL
- `GetTable` 自动根据结构体创建表（如表已存在则跳过），无需手动建表

### 结构体定义示例

```go
type User struct {
    ID    int    `db:"id,primary,autoinc"`   // 自增主键
    Name  string `db:"name"`                 // 普通字段
    Email string `db:"email"`
    Age   int    `db:"age"`
}
```

### TTL（生存时间）

所有 `Table` 方法都接受 `time.Duration` 作为最后一个参数，用于控制数据的生存时间：

- **关系型数据库**（sqlite / mysql / postgresql）：TTL 被忽略，数据永不过期
- **内存数据库**（memory）：到达 TTL 后自动删除过期条目
- **Redis**：可配合 `EXPIRE` 命令使用
- `0` 表示永不过期

---

## 接口定义

**Database 接口**（数据库级操作）：

| 方法 | 说明 |
|------|------|
| `Create(tableName, config)` | 创建表，config 定义字段名 → 字段属性 |
| `GetTable(tableName, example)` | 获取表操作对象，example 用于反射表结构 |
| `DeleteTable(tableName)` | 删除表 |

**Table 接口**（表级 CRUD）：

| 方法 | 说明 |
|------|------|
| `Ins(example, ttl)` | 插入一条记录，跳过标记为 autoinc 的字段 |
| `Get(example, whereFields, ttl)` | 条件查询（返回第一条），whereFields 指定条件字段 |
| `Set(example, whereFields, ttl)` | 按条件更新，whereFields 中的字段不参与更新 |
| `Del(example, whereFields, ttl)` | 条件删除 |

**Config 字段属性**：

| 字段 | 说明 |
|------|------|
| `Type` | 字段类型（如 INT, VARCHAR(255)） |
| `NullAble` | 是否允许为空 |
| `Identity` | 是否自增 |
| `PrimaryKey` | 是否主键 |
| `Unique` | 是否唯一 |

---

## 各数据库配置说明

### 关系型数据库

| 类型(type) | 配置 key | 说明 |
|------------|----------|------|
| `mysql` | `user`, `password`, `host`, `port`, `dbname` | MySQL 连接参数 |
| `postgresql` | `user`, `password`, `host`, `port`, `dbname` | PostgreSQL 连接参数 |
| `sqlite` | `path` | SQLite 文件路径 |

### 缓存模拟数据库

| 类型(type) | 配置 key | 说明 |
|------------|----------|------|
| `redis` | `addr`, `password` | Redis 连接地址和密码（模拟表结构） |
| `memory` | 无需配置 | 内存存储，支持 TTL 自动过期 |
| `bloom` | 无需配置 | 布隆过滤器，用于快速判存 |

> 缓存模拟数据库将缓存组件包装为标准 Database/Table 接口，`Create` 时需在 Config 中指定一个 `PrimaryKey` 字段作为数据标识。

---

## 融合数据库（Mixture）

Mixture 支持将多个数据库组合成一条数据链路，并自定义各层级的异常处理策略。

### 配置

```go
import (
    "github.com/Carry-Rao/goutils/database"
    "github.com/Carry-Rao/goutils/database/api"
    "github.com/Carry-Rao/goutils/database/mixture"
)

// 创建底层数据库
redisDB, _ := database.NewDatabase(api.Redis, map[string]string{"addr": "127.0.0.1:6379"})
mysqlDB, _ := database.NewDatabase(api.MySQL, map[string]string{/* ... */})

// 构建链路：Redis → MySQL，Redis 出错时继续，MySQL 出错时返回
mix := &mixture.Database{}
mix.Add(redisDB, mixture.Continue)  // 继续
mix.Add(mysqlDB, mixture.Return)     // 返回错误
```

### 错误处理策略

| 策略 | 值 | 行为 |
|------|----|------|
| `Continue` | 0 | 当前层级出错时继续执行下一层级 |
| `Return` | 1 | 当前层级出错时立即返回错误 |

### 各方法的链路行为

| 方法 | 行为 |
|------|------|
| `Ins` | 所有层级依次执行，某一层失败按策略处理 |
| `Get` | 从第一层开始查询，查到结果即返回；全部失败则返回最后一个错误 |
| `Set` | 所有层级依次执行，某一层失败按策略处理 |
| `Del` | 所有层级依次执行，某一层失败按策略处理 |

### 典型场景：多级读写

```go
// Bloom 过滤器 → Redis 缓存 → MySQL 数据库
mix := &mixture.Database{}
mix.Add(bloomDB, mixture.Continue)
mix.Add(redisDB, mixture.Continue)
mix.Add(mysqlDB, mixture.Return)

// 上层业务代码无需感知链路结构，接口完全一致
table, _ := mix.GetTable("users", &User{})
table.Ins(&User{Name: "Alice"}, 0)
```

---

## 注册自定义数据库

通过 `api.RegisterFactory` 注册自定义数据库后端，实现 `api.Database` 接口即可：

```go
package mydb

import (
    "github.com/Carry-Rao/goutils/database/api"
)

type Database struct {
    // 你的数据库连接
}

func (d *Database) Create(tableName string, config map[string]api.Config) error {
    // 实现建表逻辑
    return nil
}

func (d *Database) GetTable(tableName string, example any) (api.Table, error) {
    // 返回实现了 api.Table 的对象
    return &Table{}, nil
}

func (d *Database) DeleteTable(tableName string) error {
    return nil
}

func init() {
    api.RegisterFactory("mydb", func(config map[string]string) (api.Database, error) {
        return &Database{}, nil
    })
}
```

使用时通过 `NewDatabaseByName` 创建：

```go
import "github.com/Carry-Rao/goutils/database"

db, _ := database.NewDatabaseByName("mydb", map[string]string{
    "host": "127.0.0.1",
})
```

也可以注册别名：

```go
api.Register("pg", api.PostgreSQL)
db, _ := database.NewDatabaseByName("pg", config)
```