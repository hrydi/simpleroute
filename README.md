# simpleroute

Lightweight, zero-dependency HTTP router for Go 1.24+.

## Features

- **Zero external dependencies** — pure stdlib
- **Path parameters** — `{param}` patterns injected into request context
- **Group routing** — namespaced routes with shared middleware
- **Middleware chain** — global, group, and route-level middleware
- **Polymorphic `Use`** — accepts `HttpRouter`, `http.Handler`, `MiddlewareFunc`, method/pattern strings
- **Static file serving** — supports both `embed.FS` and `os.DirFS`
- **Built-in middleware** — CORS, panic recovery, request logging, context injection
- **Concurrent-safe** — `sync.Once` build, no per-request locks

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

	server := simpleroute.NewHttp(":8080")
	if err := server.Start(router); err != nil {
		panic(err)
	}
}
```

## API

### Router

| Method | Description |
|--------|-------------|
| `Get(path, args...)` | Register GET handler |
| `Post(path, args...)` | Register POST handler |
| `Put(path, args...)` | Register PUT handler |
| `Patch(path, args...)` | Register PATCH handler |
| `Delete(path, args...)` | Register DELETE handler |
| `Head(path, args...)` | Register HEAD handler |

### RouteRegister

| Method | Description |
|--------|-------------|
| `Group(path, callback, middlewares...)` | Namespaced route group |
| `Use(args...)` | Register middleware, handlers, or HttpRouter |

### Middleware

| Function | Description |
|----------|-------------|
| `CORS(config)` | Configurable CORS with preflight |
| `RecoverMiddleware(handler)` | Panic recovery (returns 500) |
| `ContentTypeJson(handler)` | Sets `Content-Type: application/json` |
| `RequestLogger(handler)` | Logs method, path, and duration |
| `WithContext(key, val)(handler)` | Injects value into request context |

### Utilities

| Function | Description |
|----------|-------------|
| `Params(r) map[string]string` | Extract path parameters from context |
| `JSON(w, code, data)` | Write JSON response with content-type |
| `Handle(middlewares, handler)` | Build middleware chain |
| `SetCtx(r, key, val) *http.Request` | Store value in request context (chainable) |
| `GetCtx[T](r, key) (T, bool)` | Retrieve typed value from request context |

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

Set the logger once via `RouterConfig` — it applies to both router internals and middleware:

```go
r := simpleroute.NewRouter(simpleroute.RouterConfig{
    Logger:   myLogger{},
    LogLevel: simpleroute.LogLevelDebug,
})
```

Levels: `LogLevelError` → `LogLevelWarn` → `LogLevelInfo` (default) → `LogLevelDebug`.

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

## Built-in HTTP Server

```go
server := simpleroute.NewHttp(":8080")
go func() {
	if err := server.Start(router); err != nil {
		log.Fatal(err)
	}
}()
// ... later
server.Stop(ctx)  // graceful shutdown
```

## CORS Example

```go
router.Use("/api", apiHandler, simpleroute.CORS(simpleroute.CORSConfig{
	AllowedOrigins: []string{"https://example.com"},
	AllowedMethods: []string{"GET", "POST"},
	AllowedHeaders: []string{"Content-Type"},
	AllowCredentials: true,
	MaxAge: 3600,
}))
```

## Benchmarks

```
goos: linux
goarch: amd64
cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz
BenchmarkStaticRoute-8               3,312,459    352 ns/op    450 B/op     7 allocs/op
BenchmarkStaticRouteDeep-8           1,974,855    612 ns/op    720 B/op    11 allocs/op
BenchmarkPathParams-8                1,369,901    893 ns/op   1448 B/op    18 allocs/op
BenchmarkNotFound-8                  1,000,000  1,043 ns/op   1584 B/op    27 allocs/op
BenchmarkCatchAll-8                  1,885,642    659 ns/op    688 B/op    17 allocs/op
BenchmarkMultipleRoutes (1000)-8     1,985,503    599 ns/op    624 B/op    11 allocs/op
BenchmarkBuild (1000 routes)-8             206  5.95 ms/op  4.67 MB/op    42k allocs/op
BenchmarkMiddlewareChainDepth (10)-8 2,928,586    413 ns/op    450 B/op     7 allocs/op
BenchmarkParamsExtraction-8          1,394,835    829 ns/op   1312 B/op    16 allocs/op
BenchmarkGroupedRoutes-8             1,834,677    655 ns/op    664 B/op    11 allocs/op
BenchmarkRouteRegistration (100)-8      33,114 36.4 μs/op 36.2 kB/op     513 allocs/op
BenchmarkConcurrentServe-8             586,178  2.06 μs/op 5.74 kB/op      20 allocs/op
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

## License

MIT
