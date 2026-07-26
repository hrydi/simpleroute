# Context & Logger

## Base Context

Set a parent context for all requests. Cancelling the base context cancels all in-flight requests:

```go
ctx, cancel := context.WithCancel(context.Background())

router := simpleroute.NewRouter(simpleroute.RouterConfig{
    BaseContext: ctx,
})

// Later: cancel all in-flight requests
cancel()
```

Each request derives a new cancellable context from the base context (`defer cancel()` in `ServeHTTP`).

## Request-Scoped Values

Use `SetCtx` and `GetCtx[T]` for typed request-scoped values:

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

## Path Parameters

```go
id := simpleroute.Params(r)["id"]
```

Returns `nil` if no parameters were matched.

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

Set the logger and level via `RouterConfig`:

```go
router := simpleroute.NewRouter(simpleroute.RouterConfig{
    Logger:   myLogger{},
    LogLevel: simpleroute.LogLevelDebug,
})
```

Levels: `LogLevelError` (1) → `LogLevelWarn` (2) → `LogLevelInfo` (3, default) → `LogLevelDebug` (4).

### How It Works

- `NewRouter` syncs the config logger to a package-level variable (`sync.RWMutex`-protected)
- Built-in middleware (`RecoverMiddleware`, `RequestLogger`) read the logger safely at setup time
- The logger is **not** injected into request context — no per-request allocations

### GetLogger

Use `GetLogger()` in custom middleware:

```go
func myMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        simpleroute.GetLogger().Infof("handling %s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}
```

### Explicit Logger (zero global dependency)

Pass a logger directly to `RequestLogger` to avoid the global entirely:

```go
router.Use(simpleroute.RequestLogger(handler, myLogger))
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

### Default

When no logger is configured, output goes through `log.Printf` with `[simpleroute] [LEVEL]` prefix at `INFO` level.
