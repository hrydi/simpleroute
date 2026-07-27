---
title: Context & Logger
layout: default
nav_order: 4
---

# Context & Logger

## Base Context

Set a parent context for all requests. Cancelling the base context cancels all in-flight requests:

```go
ctx, cancel := context.WithCancel(context.Background())

router := simpleroute.NewRouter(simpleroute.RouterConfig{
    BaseContext: ctx,
})

// Cancel all in-flight requests
cancel()
```

Each request derives a new cancellable context from `BaseContext` via `context.WithCancel`. The `cancel()` function is deferred in `ServeHTTP`, so the derived context is always cleaned up.

## Request-Scoped Values

Use `SetCtx` and `GetCtx[T]` for typed values:

```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        r = simpleroute.SetCtx(r, "user", &User{Name: "alice"})
        next.ServeHTTP(w, r)
    })
}

func handler(w http.ResponseWriter, r *http.Request) {
    user, ok := simpleroute.GetCtx[*User](r, "user")
    if ok {
        fmt.Fprintf(w, "Hello, %s", user.Name)
    }
}
```

`SetCtx` is chainable:

```go
r = simpleroute.SetCtx(simpleroute.SetCtx(r, "a", 1), "b", 2)
```

`GetCtx[T]` is generic — the type parameter ensures type safety:

```go
val, ok := simpleroute.GetCtx[string](r, "key")
```

Returns the zero value and `false` if the key is missing or the type doesn't match.

## Path Parameters

Access path parameters as a map:

```go
id := simpleroute.Params(r)["id"]
```

Or with zero-alloc single-parameter access:

```go
id := simpleroute.URLParam(r, "id")
```

> `Params(r)` returns `nil` when no parameters are matched. `URLParam` returns an empty string.

## Logger

### Interface

```go
type Logger interface {
    Errorf(format string, args ...any)
    Warnf(format string, args ...any)
    Infof(format string, args ...any)
    Debugf(format string, args ...any)
}
```

### Configuration

```go
router := simpleroute.NewRouter(simpleroute.RouterConfig{
    Logger:   myLogger{},
    LogLevel: simpleroute.LogLevelDebug,
})
```

| Level | Constant | Value |
|-------|----------|-------|
| Error | `LogLevelError` | `1` |
| Warn | `LogLevelWarn` | `2` |
| Info | `LogLevelInfo` (default) | `3` |
| Debug | `LogLevelDebug` | `4` |

### How It Works

1. `NewRouter` calls `resolveLogger(config)` to create or select the logger
2. When `config.Logger` is set, it syncs to a package-level variable (protected by `sync.RWMutex`)
3. Built-in middleware reads the logger once at setup time and captures it by closure
4. The logger is **never injected into request context** — zero per-request allocations

### GetLogger

Use in custom middleware:

```go
func myMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        simpleroute.GetLogger().Infof("handling %s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}
```

### Explicit Logger (Zero Global Dependency)

Pass a logger directly to middleware to avoid the package-level variable entirely:

```go
router.Use(simpleroute.RequestLogger(handler, myLogger))
```

### Example: zerolog

```go
type zeroLogger struct {
    l zerolog.Logger
}

func (z *zeroLogger) Errorf(format string, args ...any) { z.l.Error().Msgf(format, args...) }
func (z *zeroLogger) Warnf(format string, args ...any)  { z.l.Warn().Msgf(format, args...) }
func (z *zeroLogger) Infof(format string, args ...any)  { z.l.Info().Msgf(format, args...) }
func (z *zeroLogger) Debugf(format string, args ...any) { z.l.Debug().Msgf(format, args...) }
```

### Default Logger

When no logger is configured, output goes through `log.Printf` with `[simpleroute] [LEVEL]` prefix at `INFO` level:

```
2026/07/26 12:34:56 [simpleroute] [INFO]  GET /hello 1.2ms
```
