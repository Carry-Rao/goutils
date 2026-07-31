# 性能基准测试

本页汇总各模块的基准测试结果，帮助开发者了解各模块的性能量级，便于选型与容量评估。

> 基准测试运行环境：Linux / amd64，CPU：Intel Core i7-6600U @ 2.60GHz，Go 1.24。
> 测试命令：`go test -bench=. -benchmem -benchtime=1000000x -run=^$ ./<模块>/`

## HTTP 路由模块

与标准库 `net/http.ServeMux` 对比，路由器基于前缀树实现，静态路径与多路由场景下均优于标准库：

| 基准测试 | 耗时/op | 内存/op | 分配/op |
|----------|---------|---------|---------|
| `ServeMux_Static` | 133.9 ns | 0 B | 0 |
| `Router_Static` | 52.45 ns | 0 B | 0 |
| `ServeMux_Var` | 289.8 ns | 16 B | 1 |
| `Router_StringVar` | 143.6 ns | 16 B | 1 |
| `Router_IntVar` | 147.7 ns | 16 B | 1 |
| `ServeMux_Deep` | 446.9 ns | 0 B | 0 |
| `Router_Deep` | 290.5 ns | 0 B | 0 |
| `ServeMux_NotFound` | 4026 ns | 208 B | 12 |
| `Router_NotFound` | 117.2 ns | 16 B | 1 |
| `ServeMux_ManyRoutes` | 141.6 ns | 0 B | 0 |
| `Router_ManyRoutes` | 59.27 ns | 0 B | 0 |
| `Router_NoMiddleware` | 56.74 ns | 0 B | 0 |
| `Router_TwoMiddleware` | 69.17 ns | 0 B | 0 |
| `Router_CORS` | 349.0 ns | 16 B | 1 |
| `Router_CORSPreflight` | 269.9 ns | 16 B | 1 |
| `Router_MixedVar` | 465.9 ns | 112 B | 3 |

要点：

- 静态路径路由约 **52 ns/op**，为标准库的 **2.5 倍**吞吐
- 99 条静态路由共存时约 **59 ns/op**，前缀树结构下路由数量对查找性能影响极小
- 未命中路径（404）约 **117 ns/op**，远优于标准库（~4 µs），抗恶意扫描能力强
- 带路径变量、中间件、CORS 时开销略有增加，但仍保持在亚微秒级别

## 日志模块

单条日志写入（缓冲区未满时不落盘，仅写入内存缓冲区）：

| 基准测试 | 耗时/op | 内存/op | 分配/op |
|----------|---------|---------|---------|
| `Debug` | 383.7 ns | 24 B | 1 |
| `Info` | 408.3 ns | 24 B | 1 |
| `Error` | 396.1 ns | 24 B | 1 |
| `InfoColor` | 436.3 ns | 72 B | 2 |
| `InfoDiscard` | 370.2 ns | 24 B | 1 |

要点：

- 单条日志写入约 **380~440 ns/op**，缓冲区机制避免了频繁磁盘 IO
- 颜色输出（ANSI 转义码）额外带来约 30 ns 与 1 次分配的开销
- 各日志级别性能接近，级别过滤逻辑本身开销可忽略

## 数据库 - Memory（内存缓存）

基于 `sync.RWMutex` 的哈希表实现，CRUD 基准：

| 基准测试 | 耗时/op | 内存/op | 分配/op |
|----------|---------|---------|---------|
| `Ins` | 1709 ns | 279 B | 5 |
| `Get`（命中） | 807.3 ns | 47 B | 2 |
| `Set` | 966.6 ns | 112 B | 4 |
| `Del` | 753.9 ns | 48 B | 2 |
| `GetMiss`（未命中） | 646.2 ns | 48 B | 3 |

要点：

- 读操作约 **0.8 µs/op**，写操作约 **1~1.7 µs/op**
- 主要开销来自接口层的反射（结构体字段解析）与 `fmt.Sprintf` 拼接主键

## 数据库 - Bloom（布隆过滤器）

基于 1024 位布隆过滤器 + FNV 双哈希，用于快速判存：

| 基准测试 | 耗时/op | 内存/op | 分配/op |
|----------|---------|---------|---------|
| `Ins` | 1218 ns | 223 B | 4 |
| `GetHit`（命中） | 745.5 ns | 56 B | 3 |
| `GetMiss`（未命中） | 681.4 ns | 40 B | 3 |
| `Contains` | 39.02 ns | 0 B | 0 |
| `Add` | 99.88 ns | 0 B | 0 |

要点：

- 纯哈希判存 `Contains` 仅约 **39 ns/op**，零分配，适合高频过滤场景
- 走完整 Table 接口（含反射与锁）时约 0.7~1.2 µs/op

## 数据库 - Mixture（融合链路）

组合 Bloom → Memory 两层（均使用 `Continue` 策略）的基准：

| 基准测试 | 耗时/op | 内存/op | 分配/op |
|----------|---------|---------|---------|
| `Ins` | 1229 ns | 223 B | 4 |
| `GetHit`（命中） | 798.0 ns | 56 B | 4 |
| `Set` | 759.7 ns | 55 B | 3 |
| `Del` | 709.8 ns | 40 B | 2 |

要点：

- 链路首层（Bloom）即处理成功时，整体开销与单层 Bloom 相当，**约 0.7~1.2 µs/op**
- 说明多层级联时，命中首层的场景几乎无额外损耗

## 如何复现

```bash
# 运行全部模块基准测试
go test -bench=. -benchmem -benchtime=1000000x -run=^$ ./http/router/ ./log/ ./database/memory/ ./database/bloom/ ./database/mixture/
```
