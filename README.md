# simpleroute

[![Go Reference](https://pkg.go.dev/badge/github.com/hrydi/simpleroute)](https://pkg.go.dev/github.com/hrydi/simpleroute)
[![Go Report Card](https://goreportcard.com/badge/github.com/hrydi/simpleroute)](https://goreportcard.com/report/github.com/hrydi/simpleroute)

Lightweight, zero-dependency HTTP router for Go 1.24+.

**[Documentation →](docs/index.md)**

## Features

- **Zero external dependencies** — pure stdlib
- **Path parameters** — `{param}` patterns injected into request context
- **Group routing** — namespaced routes with shared middleware
- **Middleware chain** — global, group, and route-level middleware
- **Polymorphic `Use`** — accepts `HttpRouter`, `http.Handler`, `MiddlewareFunc`, method/pattern strings
- **Static file serving** — supports both `embed.FS` and `os.DirFS`
- **Built-in middleware** — CORS, panic recovery, request logging, request ID, gzip, rate limiter, metrics, context injection
- **HEAD auto-routing** — HEAD requests fall back to GET handlers, body stripped automatically
- **Custom 404/405 handlers** — plug your own handlers via `RouterConfig`
- **Subtree mount** — `router.Mount("/prefix", subHandler)` for all methods
- **Query helpers** — `Query`, `QueryInt`, `QueryFloat`, `QueryBool` with defaults
- **Response helpers** — `JSON`, `WriteError`, `Text`
- **Concurrent-safe** — `sync.Once` build, no per-request locks, per-router logger
- **Production-ready server** — configurable timeouts (10s read, 10s write, 60s idle by default)

## Installation

```bash
go get github.com/hrydi/simpleroute
```

Requires Go 1.24.4+.

## Quick Start

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/hrydi/simpleroute"
)

func main() {
	router := simpleroute.NewRouter(simpleroute.RouterConfig{})

	router.Get("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, World!")
	}))

	if err := router.Build(); err != nil {
		panic(err)
	}

	server := simpleroute.NewHttp(simpleroute.ServerConfig{Addr: ":8080"})
	if err := server.Start(router); err != nil {
		panic(err)
	}
}
```

## API

### Router

`args ...any` accepts: `http.Handler`, `MiddlewareFunc`, `[]MiddlewareFunc` (in any order).

| Method | Description |
|--------|-------------|
| `Get(path, args...)` | Register GET handler |
| `Post(path, args...)` | Register POST handler |
| `Put(path, args...)` | Register PUT handler |
| `Patch(path, args...)` | Register PATCH handler |
| `Delete(path, args...)` | Register DELETE handler |
| `Head(path, args...)` | Register HEAD handler |
| `Logger()` | Return the router's Logger instance |

### RouteRegister

| Method | Args types | Description |
|--------|------------|-------------|
| `Group(path, callback, args...)` | `func(Router) Router`, `MiddlewareFunc`, `[]MiddlewareFunc` | Namespaced route group |
| `Use(args...)` | `HttpRouter`, `string` (method/pattern), `http.Handler`, `MiddlewareFunc`, `[]MiddlewareFunc` | Register middleware, handlers, or routes |
| `Mount(path, handler)` | — | Register subtree handler for all HTTP methods |

### Middleware

| Function | Description |
|----------|-------------|
| `CORS(config)` | Configurable CORS with preflight |
| `RecoverMiddleware(handler, stackTrace...)` | Panic recovery (returns 500), optional stack trace |
| `ContentTypeJson(handler)` | Sets `Content-Type: application/json` |
| `RequestLogger(handler, logger...)` | Logs method, path, and duration. Optional custom logger. |
| `WithContext(key, val)(handler)` | Injects value into request context |
| `RequestID(handler)` | Injects/preserves `X-Request-ID` header + context |
| `Gzip(handler)` | Transparent gzip compression |
| `RateLimiter(config)` | Token bucket rate limiter (returns 429) |
| `Metrics(recorder)` | Atomic counters for total/active requests and cumulative duration |

### Utilities

| Function | Description |
|----------|-------------|
| `Params(r) map[string]string` | Extract path parameters from context as a map |
| `URLParam(r, key) string` | Extract a single path parameter by name (zero-alloc) |
| `JSON(w, code, data)` | Write JSON response with content-type |
| `WriteError(w, code, msg)` | Write plain-text error response |
| `Text(w, code, msg)` | Write plain-text response |
| `Handle(middlewares, handler)` | Build middleware chain |
| `SetCtx(r, key, val) *http.Request` | Store value in request context (chainable) |
| `GetCtx[T](r, key) (T, bool)` | Retrieve typed value from request context |
| `Query(r, key)` | Get query parameter value |
| `QueryInt(r, key, default)` | Get query parameter as int |
| `QueryFloat(r, key, default)` | Get query parameter as float64 |
| `QueryBool(r, key, default)` | Get query parameter as bool |


### Server

| Type | Description |
|------|-------------|
| `ServerConfig` | Config with `Addr`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` |
| `NewHttp(config)` | Create HTTP server with production-ready defaults |
| `server.Start(router)` | Start serving (returns error, filters `ErrServerClosed`) |
| `server.Stop(ctx)` | Graceful shutdown |

## Context

### Base context

Set a parent context for all requests via `RouterConfig`:

```go
ctx, cancel := context.WithCancel(context.Background())

r := simpleroute.NewRouter(simpleroute.RouterConfig{
    BaseContext: ctx,  // cancel ctx → all in-flight requests cancelled
})
```

### Request-scoped values

```go
func handler(w http.ResponseWriter, r *http.Request) {
    r = simpleroute.SetCtx(r, "user", user)
    // ... later or in middleware:
    user, ok := simpleroute.GetCtx[*User](r, "user")
}
```

`SetCtx` is chainable:
```go
r = simpleroute.SetCtx(simpleroute.SetCtx(r, "a", 1), "b", 2)
```

### Path parameters

```go
id := simpleroute.Params(r)["id"]
```

## Pluggable Logger

Set the logger once via `RouterConfig` — it applies to both router internals and custom middleware:

```go
r := simpleroute.NewRouter(simpleroute.RouterConfig{
    Logger:   myLogger{},
    LogLevel: simpleroute.LogLevelDebug,
})
```

Levels: `LogLevelError` → `LogLevelWarn` → `LogLevelInfo` (default) → `LogLevelDebug`.

Access the logger from any code that holds a `Router`:

```go
router.Logger().Infof("handling request")
```

The `RequestLogger` middleware accepts an optional logger parameter:

```go
router.Use(simpleroute.RequestLogger(handler, myLogger))
```

### Interface

```go
type Logger interface {
    Errorf(format string, args ...any)
    Warnf(format string, args ...any)
    Infof(format string, args ...any)
    Debugf(format string, args ...any)
}
```

### Example with zerolog

```go
type zeroLogger struct {
    l zerolog.Logger
}

func (z *zeroLogger) Errorf(format string, args ...any) { z.l.Error().Msgf(format, args...) }
func (z *zeroLogger) Warnf(format string, args ...any)  { z.l.Warn().Msgf(format, args...) }
func (z *zeroLogger) Infof(format string, args ...any)  { z.l.Info().Msgf(format, args...) }
func (z *zeroLogger) Debugf(format string, args ...any) { z.l.Debug().Msgf(format, args...) }
```

Defaults to `[simpleroute] [INFO/ERROR/...]` prefixed output via `log.Printf`.

## Path Parameters

Use `{name}` in route patterns:

```go
router.Get("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    id := simpleroute.Params(r)["id"]
    fmt.Fprintf(w, "User: %s", id)
}))
```

## Route Groups

Group routes under a common prefix with optional shared middleware:

```go
router.Group("/api", func(router simpleroute.Router) simpleroute.Router {
    router.Logger().Infof("setting up /api routes")
    return router.
        Get("/users", listUsers).
        Post("/users", createUser)
}, authMiddleware, loggerMiddleware)
```

## Middleware Order

Global router middlewares → group middlewares → route middlewares → handler.

The first middleware in the chain wraps the outermost layer:

```go
router.Use(requestLogger)          // outermost
router.Get("/data", handler, auth) // auth wraps handler
// Order: requestLogger → auth → handler
```

## RecoverMiddleware

Panic recovery with optional stack trace:

```go
// Default (no stack trace)
router.Use("/api", apiHandler, simpleroute.RecoverMiddleware)

// With stack trace logged at ERROR level
router.Use("/api", apiHandler, simpleroute.RecoverMiddleware)
```

`RecoverMiddleware` is **automatically** applied to all requests by `NewHttp`/`server.Start()`. Only use it explicitly if you need a custom panic boundary.

## CORS Example

```go
router.Use("/api", apiHandler, simpleroute.CORS(simpleroute.CORSConfig{
    AllowedOrigins:   []string{"https://example.com"},
    AllowedMethods:   []string{"GET", "POST"},
    AllowedHeaders:   []string{"Content-Type"},
    AllowCredentials: true,
    MaxAge:           3600,
}))
```

## Rate Limiter

Token bucket rate limiter:

```go
router.Use("/api", simpleroute.RateLimiter(simpleroute.RateLimiterConfig{
    RequestsPerSecond: 10,
    Burst:             20,
}))
```

## Metrics

Atomic request metrics collector:

```go
metrics := &simpleroute.MetricsRecorder{}
router.Use(simpleroute.Metrics(metrics))
go func() {
    for range time.Tick(10 * time.Second) {
        snap := metrics.Snapshot()
        fmt.Printf("requests: %d, active: %d, avg_dur: %dns\n",
            snap["total_requests"], snap["active_requests"], snap["avg_duration_ns"])
    }
}()
```

## Subtree Mount

Mount an `http.Handler` as a subtree for all HTTP methods:

```go
router.Mount("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./public"))))
```

## Static Files

Serve embedded or on-disk static assets:

```go
//go:embed static/*
var staticFS embed.FS

router := simpleroute.NewRouter(simpleroute.RouterConfig{
    AssetPath: "/assets/",
    AssetDir:  "static",
    FS:        staticFS,
})
```

## Custom 404/405 Handlers

```go
router := simpleroute.NewRouter(simpleroute.RouterConfig{
    NotFoundHandler:         http.HandlerFunc(custom404),
    MethodNotAllowedHandler: http.HandlerFunc(custom405),
})
```

## HEAD Auto-Routing

HEAD requests automatically fall back to GET handlers when no explicit HEAD handler is registered. The response body is stripped — headers and status code are preserved:

```go
router.Get("/data", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Length", "5")
    fmt.Fprint(w, "hello")
}))

// HEAD /data → 200, Content-Length: 5, body: ""
```

## Built-in HTTP Server

```go
server := simpleroute.NewHttp(simpleroute.ServerConfig{
    Addr: ":8080",
    // ReadTimeout, WriteTimeout, IdleTimeout default to 10s/10s/60s
})
go func() {
    if err := server.Start(router); err != nil {
        log.Fatal(err)
    }
}()
// ... later
server.Stop(ctx)  // graceful shutdown
```

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

## Development

```bash
make run          # run example app (needs Vite running)
make compose-run  # full dev stack via Docker Compose
make build        # production Docker build
```

Run tests:

```bash
go test ./... -v
go test -race ./...
go test -bench=. -benchmem ./...
```

## Documentation

Full docs are in [`docs/`](docs/index.md):

| Page | Description |
|------|-------------|
| [Getting Started](docs/getting-started.md) | Install, quick start, lifecycle |
| [Routing](docs/routing.md) | Methods, path params, groups, mount, static files, HEAD auto-routing, custom 404/405 |
| [Middleware](docs/middleware.md) | Built-in middleware, custom middleware, ordering |
| [Context & Logger](docs/context-logger.md) | Base context, SetCtx/GetCtx, Params, logger interface |
| [Configuration](docs/configuration.md) | RouterConfig, ServerConfig, polymorphic Use, benchmarks |

## License

MIT
