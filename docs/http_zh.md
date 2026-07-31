# HTTP 路由模块

该模块提供了一个基于**前缀树（radix tree）** 的高性能 HTTP 路由器，支持静态路径、带类型路径变量、中间件与 CORS 配置等特性。

## 核心特性

- **类型化路径变量**：支持 `:int`（整数）与 `:string`（字符串）两种路径变量，`int` 优先匹配
- **方法路由**：内置 GET、POST、PUT、DELETE、PATCH、HEAD、OPTIONS 方法快捷注册
- **多方法注册**：`All` 方法可为同一路径同时注册所有 HTTP 方法
- **中间件**：支持链式中间件，返回 `false` 即可终止请求
- **CORS 支持**：可为路径模式配置跨域策略，自动响应预检（OPTIONS）请求
- **自定义错误处理**：可替换默认的 404 / 400 响应处理器

## 快速开始

```go
package main

import (
    "net/http"
    "github.com/Carry-Rao/goutils/http/router"
)

func main() {
    r := router.New()

    // 静态路径
    r.GET("/", func(w http.ResponseWriter, r *http.Request, params []string) {
        w.Write([]byte("hello"))
    })

    // 类型化路径变量，params 按声明顺序返回匹配值
    r.GET("/users/:int", func(w http.ResponseWriter, r *http.Request, params []string) {
        // params[0] 为整数形式的用户 ID
        w.Write([]byte("user " + params[0]))
    })

    r.GET("/files/:string", func(w http.ResponseWriter, r *http.Request, params []string) {
        w.Write([]byte("file " + params[0]))
    })

    r.ListenAndServe(":8080")
}
```

## 路径变量

路径变量以 `:` 开头，当前支持两种类型：

| 变量 | 匹配规则 | 优先级 |
|------|----------|--------|
| `:int` | 整数（含负数），如 `-42`、`123` | 高于 `:string` |
| `:string` | 任意非空字符串 | 低于 `:int` |

```go
r.GET("/items/:int", func(w http.ResponseWriter, r *http.Request, params []string) {
    // 访问 /items/42 → params = ["42"]
})
```

多个变量按声明顺序依次填入 `params`：

```go
r.GET("/users/:int/posts/:string", func(w http.ResponseWriter, r *http.Request, params []string) {
    // 访问 /users/10/posts/hello → params = ["10", "hello"]
})
```

## 方法注册

```go
r.GET("/path", handler)
r.POST("/path", handler)
r.PUT("/path", handler)
r.DELETE("/path", handler)
r.PATCH("/path", handler)
r.HEAD("/path", handler)
r.OPTIONS("/path", handler)
```

所有 HTTP 方法注册同一路径处理器：

```go
r.All("/path", handler)
// OPTIONS 自动返回 Allow: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
```

## 中间件

使用 `Medium` 注册中间件，按注册顺序依次执行。返回 `false` 时终止请求，不再执行后续中间件与处理器：

```go
r.Medium(func(w http.ResponseWriter, r *http.Request, params []string) bool {
    // 鉴权、日志等逻辑
    return true // 返回 false 终止请求
})

r.Medium(func(w http.ResponseWriter, r *http.Request, params []string) bool {
    return true
})

r.GET("/secret", func(w http.ResponseWriter, r *http.Request, params []string) {
    // 通过中间件后才执行
})
```

## CORS 配置

使用 `Option` 创建跨域配置，通过链式调用设置属性，最后调用 `Enable` 生效：

```go
r.Option("/api/:string").
    Enable().
    Origin("https://example.com", "https://app.example.com").
    Methods("GET,POST").
    Headers("Content-Type,Authorization").
    Credentials(true).
    MaxAge(3600)
```

CORS 配置方法：

| 方法 | 说明 |
|------|------|
| `Enable()` | 启用该 CORS 配置 |
| `Origin(origins...)` | 设置允许的来源 |
| `Methods(methods...)` | 设置允许的请求方法 |
| `Headers(headers...)` | 设置允许的请求头 |
| `Credentials(allow)` | 是否允许携带凭证（Access-Control-Allow-Credentials） |
| `MaxAge(seconds)` | 预检请求缓存时间（Access-Control-Max-Age） |

命中 CORS 配置后，若请求为 `OPTIONS` 预检请求，将直接返回 204 并设置跨域响应头。

> CORS 路径模式同样支持 `:int` / `:string` 类型变量。

## 自定义错误处理

默认的 404 与 400 响应可通过替换包级变量自定义：

```go
router.NotFound = func(w http.ResponseWriter, r *http.Request, params []string) {
    http.Error(w, "resource not found", http.StatusNotFound)
}

router.BadRequest = func(w http.ResponseWriter, r *http.Request, params []string) {
    http.Error(w, "method not allowed", http.StatusBadRequest)
}
```

- **404**：路径未匹配到任何路由
- **400**：请求方法未注册（如仅注册了 GET，却收到 POST）

## 服务启动

```go
r.ListenAndServe(":8080")                // HTTP
r.ListenAndServeTLS(":8443", "cert.pem", "key.pem") // HTTPS
```

`Router` 实现了 `http.Handler` 接口，可直接作为参数传给标准库服务。
