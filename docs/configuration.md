# Configuration

## RouterConfig

```go
type RouterConfig struct {
    AssetPath              string
    AssetDir               string
    FS                     fs.FS
    BaseContext            context.Context
    Logger                 Logger
    LogLevel               LogLevel
    NotFoundHandler        http.Handler
    MethodNotAllowedHandler http.Handler
}
```

| Field | Type | Description |
|-------|------|-------------|
| `AssetPath` | `string` | URL prefix for static assets (e.g., `/assets/`) |
| `AssetDir` | `string` | Directory or embed path for static assets |
| `FS` | `fs.FS` | Embedded filesystem (optional, uses `os.DirFS` when nil) |
| `BaseContext` | `context.Context` | Parent context for all requests. Cancel → all in-flight requests cancelled |
| `Logger` | `Logger` | Custom logger (defaults to `log.Printf` with `[simpleroute]` prefix) |
| `LogLevel` | `LogLevel` | Minimum log level (`LogLevelError`, `LogLevelWarn`, `LogLevelInfo`, `LogLevelDebug`) |
| `NotFoundHandler` | `http.Handler` | Custom handler for 404 responses |
| `MethodNotAllowedHandler` | `http.Handler` | Custom handler for 405 responses |

## ServerConfig

```go
type ServerConfig struct {
    Addr         string
    ReadTimeout  time.Duration  // default: 10s
    WriteTimeout time.Duration  // default: 10s
    IdleTimeout  time.Duration  // default: 60s
}
```

```go
server := simpleroute.NewHttp(simpleroute.ServerConfig{
    Addr:         ":8080",
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  120 * time.Second,
})
```

Zero values are replaced with production defaults (10s read, 10s write, 60s idle).

## HttpServer Interface

```go
type HttpServer interface {
    Start(router http.Handler) error
    Stop(ctx context.Context) error
}
```

- `Start` wraps the router in `RecoverMiddleware` and calls `ListenAndServe`
- `Start` returns `nil` on graceful shutdown (`http.ErrServerClosed` is filtered)
- `Stop` delegates to `http.Server.Shutdown`

## Polymorphic Use

The `Use` method accepts any combination of:

| Type | Behavior |
|------|----------|
| `HttpRouter` | Calls `Routes(r)` to register routes |
| `string` | Method (e.g., `"GET"`) or URL pattern |
| `http.Handler` | Final handler for the route |
| `MiddlewareFunc` | Middleware wrapping the handler |
| `[]MiddlewareFunc` | Batch middleware registration |

Example:

```go
router.Use(
    simpleroute.RequestLogger,      // MiddlewareFunc (global)
    NewUser(),                       // HttpRouter (registers routes)
    "/api",                          // string pattern
    authMiddleware,                  // MiddlewareFunc (route-level)
    apiHandler,                      // http.Handler (final)
)
```

This registers `GET /api` with `requestLogger → authMiddleware → apiHandler`.

## Benchmarks

```
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
