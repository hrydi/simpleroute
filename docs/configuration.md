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

`Use` accepts any combination of types in a single call:

| Type | Behavior | Example |
|------|----------|---------|
| `HttpRouter` | Calls `Routes(r)` to register routes | `NewUser()` |
| `string` | HTTP method or URL pattern | `"GET"`, `"/api"` |
| `http.Handler` | Final route handler | `myHandler` |
| `MiddlewareFunc` | Middleware wrapping the handler | `authMiddleware` |
| `[]MiddlewareFunc` | Batch middleware | `[]MiddlewareFunc{m1, m2}` |

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
BenchmarkStaticRoute-8               3,582,957    340 ns/op    450 B/op     7 allocs/op
BenchmarkStaticRouteDeep-8           2,096,168    578 ns/op    720 B/op    11 allocs/op
BenchmarkPathParams-8                1,456,713    832 ns/op   1448 B/op    18 allocs/op
BenchmarkNotFound-8                  1,245,278    993 ns/op   1584 B/op    27 allocs/op
BenchmarkCatchAll-8                  1,926,688    593 ns/op    688 B/op    17 allocs/op
BenchmarkMultipleRoutes-8            2,110,357    529 ns/op    624 B/op    11 allocs/op
BenchmarkBuild-8                           222  5.42 ms/op  4.59 MB/op    38k allocs/op
BenchmarkMiddlewareChainDepth-8      3,263,679    372 ns/op    450 B/op     7 allocs/op
BenchmarkParamsExtraction-8          1,740,031    726 ns/op   1312 B/op    16 allocs/op
BenchmarkGroupedRoutes-8             1,862,180    633 ns/op    664 B/op    11 allocs/op
BenchmarkRouteRegistration-8            28,404 36.3 μs/op 36.2 kB/op     513 allocs/op
BenchmarkConcurrentServe-8             705,944  1.69 μs/op 5.74 kB/op      20 allocs/op
```
