# 性能基准测试

本页汇总各模块的基准测试结果，帮助开发者了解各模块的性能量级，便于选型与容量评估。

> 基准测试运行环境：Linux / amd64，CPU：Intel Core i7-6600U @ 2.60GHz，Go 1.24。
> 测试命令：`go test -bench=. -benchmem -benchtime=1000000x -run=^$ ./<模块>/`
>
> 对照表格中，指标列数值格式为 `标准库 / 本模块`。

## HTTP 路由模块

与标准库 `net/http.ServeMux` 对比，路由器基于前缀树实现，静态路径与多路由场景下均优于标准库：

| stdlib | benchmark | ns/op | B/op | allocs/op |
|--------|-----------|-------|------|-----------|
| `ServeMux_Static` | `Router_Static` | 133.9 / 52.45 | 0 / 0 | 0 / 0 |
| `ServeMux_Var` | `Router_StringVar` | 289.8 / 143.6 | 16 / 16 | 1 / 1 |
| `ServeMux_Var` | `Router_IntVar` | 289.8 / 147.7 | 16 / 16 | 1 / 1 |
| `ServeMux_Deep` | `Router_Deep` | 446.9 / 290.5 | 0 / 0 | 0 / 0 |
| `ServeMux_NotFound` | `Router_NotFound` | 4026 / 117.2 | 208 / 16 | 12 / 1 |
| `ServeMux_ManyRoutes` | `Router_ManyRoutes` | 141.6 / 59.27 | 0 / 0 | 0 / 0 |
| — | `Router_NoMiddleware` | — / 56.74 | — / 0 | — / 0 |
| — | `Router_TwoMiddleware` | — / 69.17 | — / 0 | — / 0 |
| — | `Router_CORS` | — / 349.0 | — / 16 | — / 1 |
| — | `Router_CORSPreflight` | — / 269.9 | — / 16 | — / 1 |
| — | `Router_MixedVar` | — / 465.9 | — / 112 | — / 3 |

要点：

- 静态路径路由约 **52 ns/op**，为标准库的 **2.5 倍**吞吐
- 99 条静态路由共存时约 **59 ns/op**，前缀树结构下路由数量对查找性能影响极小
- 未命中路径（404）约 **117 ns/op**，远优于标准库（~4 µs），抗恶意扫描能力强
- 带路径变量、中间件、CORS 时开销略有增加，但仍保持在亚微秒级别

## 日志模块

单条日志写入，与标准库 `log` 包对比：

| stdlib | benchmark | ns/op | B/op | allocs/op |
|--------|-----------|-------|------|-----------|
| `log.StdlibLog` | `log.Debug` | 1693 / 379.4 | 0 / 24 | 0 / 1 |
| `log.StdlibLog` | `log.Info` | 1693 / 370.2 | 0 / 24 | 0 / 1 |
| `log.StdlibLog` | `log.Error` | 1693 / 366.2 | 0 / 24 | 0 / 1 |
| `log.StdlibLog` | `log.InfoColor` | 1693 / 425.3 | 0 / 72 | 0 / 2 |
| `log.StdlibLog` | `log.InfoDiscard` | 1693 / 360.6 | 0 / 24 | 0 / 1 |

要点：

- 单条日志写入约 **360~430 ns/op**，比标准库 `log`（~1.7 µs/op）快约 **4 倍**
- 本模块使用内存缓冲区，缓冲区未满时不落盘；标准库 `log.Print` 每次直接写文件（OS 缓冲），故耗时更高而零分配
- 颜色输出（ANSI 转义码）额外带来约 50 ns 与 1 次分配的开销
- 各日志级别性能接近，级别过滤逻辑本身开销可忽略

## 数据库模块

### Memory（内存缓存）

基于 `sync.RWMutex` 的哈希表实现，与原生 `map` 基线对比：

| stdlib | benchmark | ns/op | B/op | allocs/op |
|--------|-----------|-------|------|-----------|
| `StdlibMap_Ins` | `Ins` | 1153 / 1335 | 223 / 270 | 3 / 4 |
| `StdlibMap_Get` | `Get` | 586.2 / 666.0 | 23 / 71 | 1 / 3 |
| `StdlibMap_Set` | `Set` | 593.2 / 706.6 | 55 / 103 | 2 / 3 |
| `StdlibMap_Del` | `Del` | 571.6 / 693.6 | 23 / 55 | 1 / 2 |
| — | `GetMiss` | — / 494.5 | — / 56 | — / 3 |

要点：

- 读操作约 **0.7 µs/op**，写操作约 **0.7~1.3 µs/op**
- 相比原生 map 慢约 **1.1~1.2 倍**，主要开销来自接口层的反射（结构体字段解析）与键拼接、过期检查等封装能力
- 换取的是统一 Database/Table 接口与 TTL 过期管理等附加能力

### Bloom（布隆过滤器）

基于 1024 位布隆过滤器 + FNV 双哈希，用于快速判存，与原生 `map` 判存对比：

| stdlib | benchmark | ns/op | B/op | allocs/op |
|--------|-----------|-------|------|-----------|
| — | `Ins` | — / 1453 | — / 223 | — / 4 |
| — | `GetHit` | — / 832.2 | — / 56 | — / 3 |
| — | `GetMiss` | — / 683.1 | — / 40 | — / 3 |
| `StdlibMap_Contains` | `Contains` | 28.57 / 37.43 | 0 / 0 | 0 / 0 |
| — | `Add` | — / 75.90 | — / 0 | — / 0 |

要点：

- 纯哈希判存 `Contains` 约 **37 ns/op**，零分配，与原生 map 判存（~29 ns）处于同一量级
- 布隆过滤器优势在于**固定内存**（1024 位）与判存不随数据量增长，劣势是有误判率；原生 map 判存更快但内存随元素数线性增长
- 走完整 Table 接口（含反射与锁）时约 0.7~1.5 µs/op

### Mixture（融合链路）

组合 Bloom → Memory 两层（均使用 `Continue` 策略）的基准，与原生 `map` 读取对比：

| stdlib | benchmark | ns/op | B/op | allocs/op |
|--------|-----------|-------|------|-----------|
| — | `Ins` | — / 1302 | — / 223 | — / 4 |
| `StdlibMap_Get` | `GetHit` | 397.4 / 848.9 | 7 / 56 | 0 / 3 |
| — | `Set` | — / 849.9 | — / 55 | — / 3 |
| — | `Del` | — / 742.1 | — / 40 | — / 2 |

要点：

- 链路首层（Bloom）即处理成功时，整体开销与单层 Bloom 相当，**约 0.7~1.3 µs/op**
- 相比原生 map 读取（~0.4 µs）多出约 0.4 µs，是多层链路（Bloom 判存 + Memory 存储）与统一接口的代价
- 说明多层级联时，命中首层的场景几乎无额外损耗

## 如何复现

```bash
# 运行全部模块基准测试
go test -bench=. -benchmem -benchtime=1000000x -run=^$ ./http/router/ ./log/ ./database/memory/ ./database/bloom/ ./database/mixture/
```
