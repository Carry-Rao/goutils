# goutils
- 一个 Golang 的工具集
- 让程序员们专注于业务逻辑，而不是重复性工作
- 让代码更加简洁，易读，易维护

## 安装
** 注意：** 请确保您的 Go 版本 >= 1.24.4
首先要开启 Go Modules 功能：
```bash
export GO111MODULE=on
```
其次，初始化项目：
```bash
go mod init xxx # 将xxx替换为项目名称
```
然后在项目文件夹下执行以下命令安装 goutils：
```bash
go get -u github.com/Carry-Rao/goutils
```

## 使用
### Database
#### SQl
支持以下常用SQL数据库：
+ MySQL
+ PostgreSQL
+ SQLite3

在代码中导入：
```go
import "github.com/Carry-Rao/goutils/database/sql"
```
并直接使用
```go
db, err := sql.Open(driver, dsn)
```
即可打开数据库连接。

#### KV对
支持以下常用KV对数据库：
+ Redis
+ golang原生map

在代码中导入：
```go
import "github.com/Carry-Rao/goutils/database/kv"
```
并直接使用
```go
redis, err := NewClient(&kv.Options{
    Addr: address,
    Password: password,
    DB: db,
    Protocol: protocol,
}) // 连接到Redis

mem, err := kv.NewClient(&kv.Options{
    Memory: true,
}) // 连接到内存KV对
```
即可打开KV对连接。

### 日志
支持将日志输出到：
+ 控制台
+ 文件

在代码中导入：
```go
import "github.com/Carry-Rao/goutils/log"
```
并直接使用
```go
logger := log.NewConsoleLogger(false) // 输出到控制台 (stderr)
logger.SetLevel(log.Debug) // 设置日志级别
logger.Debug("debug message") // 输出debug日志
logger.Info("info message") // 输出info日志
logger.Warn("warn message") // 输出warn日志
logger.Error("error message") // 输出error日志
```
即可输出日志。

-------
还有一些功能暂时没有时间写README，后续慢慢补充。