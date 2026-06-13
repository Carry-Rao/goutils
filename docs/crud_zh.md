# CRUD 模块

该模块在数据库模块之上提供了一个**基于结构体的 ORM 风格封装**，使开发者能够使用 Go 结构体而不是原始 map 数据来执行 CRUD 操作。它通过结构体标签自动将结构体字段映射到数据库列，简化了数据访问。

## 核心特性

- **结构体驱动 CRUD**：使用类型化的 Go 结构体操作数据，而非原始 map
- **标签式列映射**：使用 `db` 结构体标签定义列名映射关系
- **无缝集成**：基于数据库模块构建，支持所有数据库类型（PostgreSQL、MySQL、SQLite、Redis、Memory、Bloom、Mixture）
- **统一接口**：无论底层数据库如何，均提供一致的 CRUD 接口

## 快速开始

```go
package main

import (
    "goutils/crud/model"
    "goutils/database/api"
    "goutils/database/sqlite"
)

// 定义数据模型
type User struct {
    ID   int    `db:"id"`
    Name string `db:"name"`
    Age  int    `db:"age"`
}

func main() {
    // 创建数据库实例（支持任意数据库类型）
    db := sqlite.NewDatabase("test.db")
    db.Create("users", map[string]api.Config{
        "id":   {Type: "INTEGER", PrimaryKey: true, Identity: true},
        "name": {Type: "TEXT", NullAble: false},
        "age":  {Type: "INTEGER"},
    })

    // 创建模型
    userModel, _ := model.NewModel("users", User{}, db)

    // 创建记录
    userModel.Create(User{Name: "Alice", Age: 30})

    // 查询记录
    var user User
    userModel.Get(map[string]any{"id": 1}, &user)

    // 更新记录
    userModel.Update(User{ID: 1, Name: "Alice", Age: 31})

    // 删除记录
    userModel.Delete(map[string]any{"id": 1})
}
```

## API 文档

### 类型定义

#### `Model`
```go
type Model struct { ... }
```
模型包装了一个数据库表，并提供类型化的 CRUD 操作。它使用 `db` 结构体标签将结构体字段映射到数据库列。

### 函数

#### `NewModel(tableName string, t any, db api.Database) (*Model, error)`

创建新的 Model 实例。

- `tableName`：要操作的数据库表名
- `t`：用于确定 schema 的结构体实例或结构体指针
- `db`：实现 `api.Database` 接口的数据库实例

如果 `t` 不是结构体或结构体指针，则返回错误。

### 方法

#### `Create(data any) error`

向表中插入一条新记录。

- `data`：带有 `db` 标签的结构体实例。结构体通过 `db` 标签转换为 map，然后传递给底层 `Table.Create()` 执行。

#### `Get(where map[string]any, dest any) error`

查询匹配条件的一条记录。

- `where`：用于过滤的列-值对 map（例如 `{"id": 1}`）
- `dest`：用于存储结果的结构体指针。使用 `db` 标签将数据库列映射到结构体字段。

如果未找到记录则返回 `"no record found"`，如果目标参数不是结构体指针则返回错误。

#### `Update(data any) error`

更新一条已有记录。

- `data`：带有 `db` 标签的结构体实例。结构体转换为 map 后传递给底层 `Table.Set()` 执行。

注意：更新操作通常根据主键或唯一字段匹配并替换整行数据。

#### `Delete(where map[string]any) error`

删除匹配条件的记录。

- `where`：用于过滤的列-值对 map（例如 `{"id": 1}`）。

## 结构体标签约定

字段必须使用 `db` 标签指定对应的列名：

```go
type User struct {
    ID       int    `db:"id"`
    Username string `db:"username"`
    Email    string `db:"email"`
    Age      int    `db:"age"`
}
```

- 没有 `db` 标签的字段在映射时会被**忽略**
- 标签中第一个逗号之前的值用作列名
- 支持结构体和结构体指针两种方式

## 错误处理

- `NewModel`：如果类型参数不是结构体或结构体指针，返回错误
- `Get`：查询结果为空时返回 `"no record found"`；返回数据无法映射时返回 `"invalid data format"`；目标参数无效时返回 `"dest must be non-nil struct pointer"`
- 所有方法都可能返回底层数据库操作的错误