# Log 模块

> English: [database_en.md](database_en.md) · [http_en.md](http_en.md) · [log_en.md](log_en.md)

带缓冲写入的多级别日志，支持彩色输出与级别过滤。

## 快速开始

```go
package main

import "github.com/Carry-Rao/goutils/log"

func main() {
	log.Console.Info("服务启动") // 输出到 stdout，带颜色

	l := log.NewLogger("/var/log/app.log", 4096)
	defer l.Flush()

	l.Info("写入文件")
	l.Error("出错")
}
```

## 日志级别

| 常量 | 级别 | 方法 | 行为 |
|------|------|------|------|
| `Debug` | 1 | `Debug` | 调试信息 |
| `Info` | 2 | `Info` | 一般信息 |
| `Warn` | 3 | `Warn` | 警告 |
| `Error` | 4 | `Error` | 错误 |
| `Panic` | 5 | `Panic` | 记录后 **flush 并 panic** |
| — | 5 | `Fatal` | 记录后 **flush 并 os.Exit(1)** |

## 级别过滤

`LogLevel` 表示「输出到此级别及以上的所有日志」，数值越大越严重：

```go
l := log.NewLogger("app.log", 4096)
l.LogLevel = log.Warn // 只输出 WARN / ERROR / PANIC

l.Info("不会输出")
l.Warn("会输出")
```

`log.Console` 默认级别为 `Info`。

## 控制台日志器

`log.Console` 是预置的全局日志器，写入 `os.Stdout` 且开启彩色：

```go
log.Console.Debug("...") // 绿色
log.Console.Info("...")  // 蓝色
log.Console.Warn("...")  // 黄色
log.Console.Error("...") // 红色
log.Console.Fatal("...") // 白字红底
```

它是包级变量，可直接调整字段：

```go
log.Console.LogLevel = log.Debug
log.Console.Color = false // 关闭颜色
```

输出格式：

```
2026-09-27 14:30:05 [INFO] 服务启动
```

开启颜色后级别标签带 ANSI 色码。

## 文件日志器

```go
l := log.NewLogger("/var/log/app.log", 4096)
```

- 以 `O_CREATE|O_WRONLY|O_APPEND` 打开，文件不存在则创建
- **打开失败会 panic**，请确保目录存在且可写
- `Color` 为 `false`（文件中不写 ANSI 转义序列）
- 返回 `*Logger`，可直接赋值给结构体字段

## 缓冲写入

`Logger.Buffer` 是固定大小的字节缓冲，`Len` 为其容量。写入时先追加到缓冲区，缓冲区满则整体落盘。

```go
type Logger struct {
	File     *os.File
	LogLevel LogLevel
	Buffer   []byte
	Len      uint32
	Pos      uint32
	Color    bool
}
```

这能显著减少高频日志的 I/O 次数。

### 换行由 logger 负责

每条记录以 `\n` 结尾，由 `addLog` 统一追加。调用 `Info` / `Warn` 等方法时**不要**自己拼 `\n`，否则会出现空行。传入的消息本身可以包含换行（多行日志原样输出）。

### 缓冲区大小

单条日志超过缓冲区容量时会直接写文件，不经过缓冲。缓冲区过小会导致频繁落盘失去意义，过大会增加内存占用与崩溃时丢失的日志量。建议 4KB ~ 64KB，按平均日志长度和写入频率调整。

### 主动 flush

`Flush` 把缓冲区剩余内容落盘。应在正常退出前调用：

```go
l := log.NewLogger("app.log", 4096)
defer l.Flush()
```

`Panic` 与 `Fatal` 内部已自动 flush。

### 并发安全

`Logger` 可被多个 goroutine 并发调用。

实现上用互斥锁保护 `Pos` 与 `Buffer` 内容。这里原本用「先 CAS 占位、再写数据」的原子方案，但并发的 `Flush` 会把已占位却尚未写入的区域刷出去，导致同一条日志被写两次，因此改为整段加锁 —— 相比落盘本身的 I/O 开销，这点锁竞争可以忽略。

首次使用后不要复制 `Logger` 值，请始终通过指针使用。

## 自定义日志器

字段全部导出，可自行构造：

```go
f, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
if err != nil {
	panic(err)
}

l := &log.Logger{
	File:     f,
	LogLevel: log.Debug,
	Buffer:   make([]byte, 8192),
	Len:      8192,
	Color:    false,
}
```

注意自行构造时必须同时设置 `Buffer` 和 `Len`，否则无法正确写入。

## 限制

- **不支持结构化日志**：仅接受字符串，不支持 `key=value` 或 JSON 格式
- **无日志轮转**：文件无限增长，需自行实现轮转或借助 logrotate
- **无 `Printf` 系列**：格式化需自行用 `fmt.Sprintf`
- **`NewLogger` 失败即 panic**：不适合路径可能不可用的场景
- `Fatal` 直接 `os.Exit(1)`，不会执行 `defer`

## 性能

`go test -bench=. -benchmem ./log/` 可运行基准，与标准库 `log` 对比。

缓冲模式下 CPU 开销主要是字符串拼接与 ANSI 转义；`Color: false` 时开销更低。

级别判断发生在方法内部、字符串拼接之前，因此被过滤掉的日志不会产生拼接与时间格式化开销。但拼接发生在**调用方传进来的值**之后 —— 若写成 `l.Info(fmt.Sprintf(...))`，`Sprintf` 的开销无论是否输出都会发生。级别过滤适合配合常量字符串或已缓存的字符串使用。
