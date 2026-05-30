# 日志模块

该模块提供了统一的日志记录能力，支持多级别日志、缓冲写入、颜色输出等特性。

## 核心特性

- **多级别日志**：支持 DEBUG、INFO、WARN、ERROR、PANIC 五个日志级别
- **缓冲写入**：支持缓冲区机制，减少频繁 IO 写入，提升性能
- **颜色输出**：支持终端颜色输出，便于日志分级查看
- **自动刷新**：PANIC 与 FATAL 级别日志自动刷新缓冲区并终止程序
- **级别过滤**：通过设置日志级别，灵活控制日志输出粒度

## 日志级别

| 级别 | 值 | 颜色 | 说明 |
|------|-----|------|------|
| DEBUG | 1 | 绿色 | 调试信息 |
| INFO | 2 | 蓝色 | 常规信息 |
| WARN | 3 | 黄色 | 警告信息 |
| ERROR | 4 | 红色 | 错误信息 |
| PANIC | 5 | 白底红字 | 严重错误，触发 panic |

## 快速开始

```go
package main

import (
    "os"
    "github.com/Carry-Rao/goutils/log"
)

func main() {
    // 创建日志文件
    file, _ := os.Create("app.log")
    defer file.Close()

    // 初始化日志器
    logger := &log.Logger{
        File:     file,           // 输出文件
        LogLevel: log.Debug,      // 日志级别（最低级别，输出所有日志）
        Len:      1024,           // 缓冲区大小（字节）
        Color:    true,           // 启用颜色输出
    }

    // 记录不同级别的日志
    logger.Debug("这是一条调试信息")
    logger.Info("这是一条常规信息")
    logger.Warn("这是一条警告信息")
    logger.Error("这是一条错误信息")

    // 手动刷新缓冲区
    logger.Flush()
}
```

## API 文档

### 类型定义

#### `LogLevel`
```go
type LogLevel int
```
日志级别枚举类型，值为：
- `Debug = 1`
- `Info  = 2`
- `Warn  = 3`
- `Error = 4`
- `Panic = 5`

#### `Log`
```go
type Log struct {
    LogLevel LogLevel
    LogInfo  string
}
```
表示一条日志记录，包含日志级别和日志内容。

#### `Logger`
```go
type Logger struct {
    File     *os.File  // 日志输出文件
    LogLevel LogLevel  // 日志级别过滤
    Buffer   string    // 日志缓冲区
    Len      uint      // 缓冲区大小（字节）
    Color    bool      // 是否启用颜色输出
}
```
日志器实例，用于记录和管理日志。

### 方法

#### `Debug(info string)`
记录 DEBUG 级别日志。仅当日志级别 >= Debug 时输出。

#### `Info(info string)`
记录 INFO 级别日志。仅当日志级别 >= Info 时输出。

#### `Warn(info string)`
记录 WARN 级别日志。仅当日志级别 >= Warn 时输出。

#### `Error(info string)`
记录 ERROR 级别日志。仅当日志级别 >= Error 时输出。

#### `Panic(info string)`
记录 PANIC 级别日志并触发 panic。日志写入后会调用 `Flush()` 刷新缓冲区，然后执行 `panic(info)` 终止程序。仅当日志级别 >= Panic 时触发。

#### `Fatal(info string)`
记录 FATAL 级别日志并退出程序。日志写入后会调用 `Flush()` 刷新缓冲区，然后执行 `os.Exit(1)` 退出程序。仅当日志级别 >= Panic 时触发（FATAL 使用 PANIC 级别）。

#### `Flush()`
手动刷新缓冲区，将缓冲区中所有日志写入文件。

## 缓冲区机制

日志器采用缓冲写入机制以减少磁盘 IO：
1. 每条日志先写入内存缓冲区
2. 当缓冲区大小达到 `Len` 字节时，自动将缓冲区内容写入文件
3. 程序结束前应调用 `Flush()` 确保缓冲区中剩余日志全部写入文件
4. `Panic` 和 `Fatal` 方法会自动调用 `Flush()`，无需手动刷新

## 级别过滤

通过设置 `Logger.LogLevel` 可以控制日志输出粒度。只有级别大于等于设定值的日志才会被记录。

例如，设置 `LogLevel = Warn` 时，仅输出 WARN、ERROR、PANIC 级别的日志，DEBUG 和 INFO 级别的日志将被忽略。

## 颜色输出

设置 `Logger.Color = true` 可启用终端颜色输出，不同级别使用不同颜色以便区分：

- **DEBUG**：`\033[32m`（绿色）
- **INFO**：`\033[34m`（蓝色）
- **WARN**：`\033[33m`（黄色）
- **ERROR**：`\033[31m`（红色）
- **PANIC/FATAL**：`\033[41;37m`（白底红字）

颜色输出仅在终端中可见，写入文件时建议关闭颜色输出。