# HTTP Router 模块

> English: [database_en.md](database_en.md) · [http_en.md](http_en.md) · [log_en.md](log_en.md)

基于前缀树（prefix tree）的路由匹配。相比 `http.ServeMux`，提供类型化路径变量、中间件、per-pattern CORS 与正确的 404/405 区分。

## 快速开始

```go
r := router.New()

r.GET("/users/:int", func(w http.ResponseWriter, req *http.Request, params []string) {
	w.Write([]byte("User " + params[0]))
})

if err := r.ListenAndServe(":8080"); err != nil {
	panic(err)
}
```

## Handler 签名

```go
func(http.ResponseWriter, *http.Request, []string)
```

第三个参数是按路由声明顺序排列的路径变量值。例如 `/users/:int/posts/:string` 匹配 `/users/1/posts/hello` 时，`params` 为 `["1", "hello"]`。

`params` 会被依次传给该节点的每一层中间件，因此中间件也能读到变量。

## 注册方法

```go
r.Handle(method, pattern, handler) // 任意方法
r.GET(pattern, handler)
r.POST(pattern, handler)
r.PUT(pattern, handler)
r.DELETE(pattern, handler)
r.PATCH(pattern, handler)
r.HEAD(pattern, handler)
r.OPTIONS(pattern, handler)
r.All(pattern, handler) // 全部方法
```

`All` 会额外为 `OPTIONS` 注册一个返回 `Allow` 头与 `204` 的处理器。

## 路径变量

| 写法 | 匹配内容 | 示例 |
|------|----------|------|
| `:int` | 可选负号 + 数字 | `123`、`-42` |
| `:float` | 可选负号 + 数字，且**必须含小数点** | `3.14`、`-0.5` |
| `:alpha` | 仅字母 | `abc`、`AbC` |
| `:alphanum` | 字母与数字 | `abc123`、`ABC` |
| `:uuid` | 标准 8-4-4-4-12 十六进制 | `550e8400-e29b-41d4-a716-446655440000` |
| `:string` | 任意非空片段（**不跨 `/`**） | `abc`、`a-b_c` |

注意几个易错点：

- `:float` 要求必须出现小数点，`3` 不会匹配 `:float`（它匹配 `:int`）
- `:uuid` 大小写均接受，非十六进制字符或分隔符位置不对则不匹配
- `:string` 是**单片段** catch-all，不跨 `/`。`/path/one/two` 不会匹配 `/path/:string`

需要匹配「剩余全部路径」时本库**不提供**通配变量。静态文件用 `Static` 挂载；
其他情况请用 `Sub` 划出前缀后，在该前缀的 handler 内自行读取剩余路径。

### 匹配优先级

同一节点下多个变量类型同时存在时，按固定顺序尝试，**越具体越优先**：

```
字面量路径  >  :int  >  :float  >  :uuid  >  :alpha  >  :alphanum  >  :string
```

因此同时注册 `/items/:int` 与 `/items/:string` 时，`/items/42` 命中 `:int`，`/items/hello` 命中 `:string`。

字面量路径优先级最高：`/users/profile` 会先于 `/users/:string` 被匹配。

## 中间件

中间件签名为 `func(http.ResponseWriter, *http.Request, []string) bool`，返回 `false` 即中断请求，handler 不会执行。

```go
r.Medium(func(w http.ResponseWriter, req *http.Request, params []string) bool {
	if req.Header.Get("Authorization") == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return false // 中断
	}
	return true // 继续
})
```

`Medium` 作用于该 `Router` 的全部路由。要为部分路由单独挂中间件，用 `Sub` 划分子路由：

```go
r.Medium(globalMiddleware) // 全局

api := r.Sub("/api")
authMiddleware // 仅 /api/*

admin := r.Sub("/admin")
admin.Medium(adminMiddleware) // 仅 /admin/*
```

执行顺序为**从根到叶**：全局中间件先于子路由中间件，子路由中间件先于 handler。

由于中间件在路由匹配成功后才执行，未匹配任何路由的请求（404/405）不会触发中间件。

## CORS

按 pattern 配置，链式调用：

```go
r.Option("/api/:string").
	Enable().
	Origin("https://example.com", "https://app.example.com").
	Methods("GET", "POST", "PUT").
	Headers("Content-Type", "Authorization").
	Credentials(true).
	MaxAge(3600)
```

- pattern 支持全部路径变量类型
- 非 `OPTIONS` 请求也会带上响应头
- `OPTIONS` 预检请求返回 `204` 并立即结束
- 未调用 `Enable()` 的配置不生效
- `Origin` 未设置时不下发 `Access-Control-Allow-Origin`

## 404 与 405

路由按方法各有一棵树。请求到来时：

1. 在当前方法的树中匹配，命中则执行中间件与 handler
2. 未命中则检查**所有方法的树**
   - 该路径在其他方法下存在 → `405 Method Not Allowed`，并下发 `Allow` 头列出可用方法
   - 所有方法下都不存在 → `404 Not Found`

```go
r.GET("/users", h)  // 只有 GET

// POST /users  =>  405, Allow: GET
// GET  /nope   =>  404
```

这解决了标准库 `ServeMux` 返回 405 但不带 `Allow`、以及部分场景误报 404 的问题。

## 自定义错误响应

`NotFound` 与 `MethodNotAllowed` 是可替换的包级变量：

```go
router.NotFound = func(w http.ResponseWriter, r *http.Request, _ []string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error":"not found"}`))
}

router.MethodNotAllowed = func(w http.ResponseWriter, r *http.Request, _ []string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Write([]byte(`{"error":"method not allowed"}`))
}
```

默认实现是纯文本的 `not found (404)` 与 `method not allowed (405)`。

包级变量是全局的，替换会影响进程内所有 Router；如需按 Router 定制，请在创建时替换。

## panic 恢复

```go
r := router.New()
r.Recover = true
```

或直接用构造函数：

```go
r := router.NewRecovery()
```

开启后 handler 中的 panic 会被捕获并返回 `500 internal server error (500)`，不会中断进程。响应体固定，如需自定义请替换包级变量或在 handler 内部自行处理。

> `Recover` 需在注册路由前后任意时机设置，只影响 `ServeHTTP` 的请求处理，不覆盖 `ListenAndServe` 本身的错误。

## 静态文件

```go
r.Static("/static", "./public")           // /static/css/app.css → ./public/css/app.css
r.Static("/", "./public")                 // 挂到根

api := r.Sub("/api")
api.Static("/assets", "./public")         // /api/assets/x → ./public/x
```

挂载点会被**剥离**后再查文件，所以 `/static/css/app.css` 对应 `./public/css/app.css`，
而不是 `./public/static/css/app.css`。`Sub` 的前缀会自动累加。

前缀之下的所有路径都会交给该处理器。同一前缀下的显式路由优先级更高：

```go
r.Static("/static", "./public")
r.GET("/static/config", handler) // /static/config 命中这里，而不是静态文件
```

只注册 `GET`（`HEAD` 不会命中）。中间件照常生效，因为挂载点仍在路由树上。

底层是 `http.FileServer`，`Range`、`ETag`、目录索引、路径穿越防护都由 `net/http`
提供。注意它会把 `/index.html` 重定向到 `./`（标准行为），目录请求则在存在
`index.html` 时直接返回其内容。

## 启动服务

```go
r.ListenAndServe(":8080")
r.ListenAndServeTLS(":8443", "cert.pem", "key.pem")
```

两者均等价于 `net/http` 的对应函数。`Router` 实现了 `http.Handler`，可直接交给 `http.Server` 以便自定义超时与连接参数：

```go
srv := &http.Server{
	Addr:              ":8080",
	Handler:           r,
	ReadHeaderTimeout: 5 * time.Second,
}
log.Fatal(srv.ListenAndServe())
```

## 性能

匹配过程为 O(路径深度)，每层一次 map 查找加至多一次类型判定。

- 路径字面量优先，未命中才走变量判定
- 变量类型按固定顺序探测，且仅在对应子节点存在时才调用判定函数
- 中间件只在匹配成功后执行，不匹配请求无额外开销
- 匹配与执行分离：`match` 无副作用，`exec` 才跑中间件与 handler

`405` 探测会遍历其他方法的树，**仅在当前方法未匹配时触发**（正常命中的请求无此开销）。

仓库内 `bench_test.go` 提供了与 `http.ServeMux` 的对比基准，可用 `go test -bench=. -benchmem ./http/router/` 运行。

## 破坏性变更

- `BadRequest` 更名为 `MethodNotAllowed`，状态码由 `400` 改为 `405`
- 中间件不再对未匹配的请求（404/405）执行
- 中间件收到的 `params` 为完整变量列表，而非匹配到该层时的前缀
