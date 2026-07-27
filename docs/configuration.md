---
title: Configuration
layout: default
nav_order: 5
---

# Configuration

## RouterConfig

```go
type RouterConfig struct {
    AssetPath               string
    AssetDir                string
    FS                      fs.FS
    BaseContext             context.Context
    Logger                  Logger
    LogLevel                LogLevel
    NotFoundHandler         http.Handler
    MethodNotAllowedHandler http.Handler
}
```

| Field | Type | Description |
|-------|------|-------------|
| `AssetPath` | `string` | URL prefix for static assets (e.g., `/assets/`) |
| `AssetDir` | `string` | Directory or embed path for static assets |
| `FS` | `fs.FS` | Embedded filesystem. Uses `os.DirFS(AssetDir)` when nil |
| `BaseContext` | `context.Context` | Parent context for all requests. Cancel → all in-flight cancelled |
| `Logger` | `Logger` | Custom logger (nil → `log.Printf` with `[simpleroute]` prefix) |
| `LogLevel` | `LogLevel` | Minimum log level to emit (`LogLevelInfo` default) |
| `NotFoundHandler` | `http.Handler` | Custom 404 handler (nil → `http.Error("page not found", 404)`) |
| `MethodNotAllowedHandler` | `http.Handler` | Custom 405 handler (nil → `http.Error("method not allowed", 405)`) |

## ServerConfig

```go
type ServerConfig struct {
    Addr         string
    ReadTimeout  time.Duration  // default: 10s
    WriteTimeout time.Duration  // default: 10s
    IdleTimeout  time.Duration  // default: 60s
}
```

Zero values are replaced with production defaults:

```go
server := simpleroute.NewHttp(simpleroute.ServerConfig{
    Addr:         ":8080",
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  120 * time.Second,
})
```

## HttpServer Interface

```go
type HttpServer interface {
    Start(router http.Handler) error
    Stop(ctx context.Context) error
}
```

- `Start` wraps the router in `RecoverMiddleware` and calls `ListenAndServe`
- Returns `nil` on graceful shutdown (`http.ErrServerClosed` is filtered)
- `Stop` delegates to `http.Server.Shutdown`

## Polymorphic Use

`Use(args ...any)` accepts the following types (mixed in any order):

| Type | Behavior | Example |
|------|----------|---------|
| `HttpRouter` | Calls `Routes(r)` to register routes | `NewUser()` |
| `string` | HTTP method or URL pattern | `"GET"`, `"/api"` |
| `http.Handler` | Final route handler | `myHandler` |
| `MiddlewareFunc` | Middleware wrapping the handler | `authMiddleware` |
| `[]MiddlewareFunc` | Batch middleware | `[]MiddlewareFunc{m1, m2}` |

Route methods (`Get`, `Post`, etc.) accept a subset: `http.Handler`, `MiddlewareFunc`, `[]MiddlewareFunc`. See [Routing](routing.md) for details.

Example — registers `GET /api` with the middleware chain:

```go
router.Use(
    simpleroute.RequestLogger,  // global middleware
    NewUser(),                   // HttpRouter — registers routes
    "/api",                      // pattern
    authMiddleware,              // route middleware
    apiHandler,                  // final handler
)
```

> Multiple `string` arguments: the first uppercase string is treated as the HTTP method, the rest as the path.

---

## Benchmarks

```
goos: linux
goarch: amd64
cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz
BenchmarkStaticRoute-8              10,124,955    127 ns/op    242 B/op     5 allocs/op
BenchmarkStaticRouteDeep-8           8,391,156    151 ns/op    240 B/op     5 allocs/op
BenchmarkPathParams-8                3,818,419    404 ns/op    776 B/op    12 allocs/op
BenchmarkNotFound-8                  2,643,332    589 ns/op   1234 B/op    21 allocs/op
BenchmarkCatchAll-8                  4,942,819    260 ns/op    288 B/op     8 allocs/op
BenchmarkMultipleRoutes-8            5,115,830    282 ns/op    240 B/op     5 allocs/op
BenchmarkBuild-8                           222  5.71 ms/op  4.48 MB/op    40k allocs/op
BenchmarkMiddlewareChainDepth-8      8,036,977    161 ns/op    240 B/op     5 allocs/op
BenchmarkParamsExtraction-8          3,903,521    423 ns/op   1016 B/op    12 allocs/op
BenchmarkGroupedRoutes-8             8,266,606    149 ns/op    240 B/op     5 allocs/op
BenchmarkRouteRegistration-8            28,312 36.3 μs/op 44.3 kB/op     513 allocs/op
BenchmarkConcurrentServe-8             730,724  1.73 μs/op 5.35 kB/op      14 allocs/op
```
