# 数据库模块文档

## 快速开始

使用 `database.NewDatabase()` 创建数据库实例，通过 `type` 字段指定数据库类型：

```go
import "github.com/Carry-Rao/goutils/database"

// 创建 MySQL 实例
db, _ := database.NewDatabase(map[string]string{
    "type":     "mysql",
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

// 获取表操作对象
table, _ := db.GetTable("users")

// CRUD 操作
table.Create(map[string]any{"id": 1, "name": "Alice"})
result, _ := table.Get(map[string]any{"id": 1})
table.Set(map[string]any{"id": 1, "name": "Bob"})
table.Delete(map[string]any{"id": 1})

// 删除表
db.DeleteTable("users")
```

## 接口定义

**Database 接口**（数据库级操作）：

| 方法 | 说明 |
|------|------|
| `Create(tableName, config)` | 创建表，config 定义字段名 → 字段属性 |
| `GetTable(tableName)` | 获取表操作对象 |
| `DeleteTable(tableName)` | 删除表 |

**Table 接口**（表级 CRUD）：

| 方法 | 说明 |
|------|------|
| `Create(data)` | 插入数据 |
| `Get(where)` | 条件查询（返回第一条） |
| `Set(data)` | 更新数据（需包含 id 字段） |
| `Delete(where)` | 条件删除 |

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
| `memory` | 无需配置 | 内存存储，进程级缓存 |
| `bloom` | 无需配置 | 布隆过滤器，用于快速判存 |

> 缓存模拟数据库将缓存组件包装为标准 Database/Table 接口，`Create` 时需在 Config 中指定一个 `PrimaryKey` 字段作为数据标识。

---

## 融合数据库（Mixture）

Mixture 支持将多个数据库组合成一条数据链路，并自定义各层级的异常处理策略。

### 配置

```go
import "github.com/Carry-Rao/goutils/database/mixture"

// 创建底层数据库
redisDB, _ := database.NewDatabase(map[string]string{"type": "redis", "addr": "127.0.0.1:6379"})
mysqlDB, _ := database.NewDatabase(map[string]string{"type": "mysql", /* ... */})

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
| `Create` | 所有层级依次执行，某一层失败按策略处理 |
| `Get` | 从第一层开始查询，查到结果即返回；全部失败则返回最后一个错误 |
| `Set` | 所有层级依次执行，某一层失败按策略处理 |
| `Delete` | 所有层级依次执行，某一层失败按策略处理 |

### 典型场景：多级读写

```go
// Bloom 过滤器 → Redis 缓存 → MySQL 数据库
mix := &mixture.Database{}
mix.Add(bloomDB, mixture.Continue)
mix.Add(redisDB, mixture.Continue)
mix.Add(mysqlDB, mixture.Return)

// 上层业务代码无需感知链路结构，接口完全一致
table, _ := mix.GetTable("users")
table.Create(map[string]any{"id": 1, "name": "Alice"})