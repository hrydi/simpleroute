---
title: Middleware
layout: default
nav_order: 3
---

# Middleware

## Ordering

```
global router middleware → group middleware → route middleware → handler
```

The first middleware registered is outermost in the chain:

```go
router.Use(requestLogger)          // outermost (runs first, last to complete)
router.Get("/data", handler, auth) // auth wraps handler
// Request flow:  requestLogger → auth → handler → auth → requestLogger
```

## Registration

| Scope | Method | Example |
|-------|--------|---------|
| **Global** | `router.Use(mw)` | Applies to all routes |
| **Route** | `router.Get("/x", mw, handler)` | Applies to single route |
| **Group** | `router.Group("/x", fn, mw)` | Shared by all routes in group |

## Built-in Middleware

### RecoverMiddleware

Catches panics and returns `500 Internal Server Error`. Optionally logs the full stack trace:

```go
// No stack trace
router.Use(simpleroute.RecoverMiddleware)

// With stack trace at ERROR level
router.Use(simpleroute.RecoverMiddleware)
```

> `RecoverMiddleware` is automatically applied by `NewHttp`/`server.Start()`. Use explicitly only for custom panic boundaries.

### RequestLogger

Logs HTTP method, path, and duration. Accepts an optional logger:

```go
// Uses package-level logger
router.Use(simpleroute.RequestLogger)

// Uses explicit logger (no global dependency)
router.Use(simpleroute.RequestLogger(myLogger))
```

### CORS

Configurable cross-origin support with preflight handling:

```go
router.Use("/api", simpleroute.CORS(simpleroute.CORSConfig{
    AllowedOrigins:   []string{"https://example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    ExposedHeaders:   []string{"X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           3600,
}))
```

Preflight (OPTIONS) returns `204 No Content` without calling the next handler.

### RequestID

Injects or preserves `X-Request-ID` in response headers and request context:

```go
router.Use(simpleroute.RequestID)
```

Access the ID in handlers:

```go
id, ok := simpleroute.GetCtx[string](r, "request_id")
```

Generates a unique ID from `time.Now().UnixNano()` when the incoming request has no `X-Request-ID` header.

### Gzip

Transparent gzip compression when the client sends `Accept-Encoding: gzip`:

```go
router.Use(simpleroute.Gzip)
```

### RateLimiter

Token bucket rate limiter. Returns `429 Too Many Requests` when the limit is exceeded:

```go
router.Use("/api", simpleroute.RateLimiter(simpleroute.RateLimiterConfig{
    RequestsPerSecond: 10,
    Burst:             20,
}))
```

### Metrics

Atomic counters for total requests, active requests, and cumulative duration:

```go
metrics := &simpleroute.MetricsRecorder{}
router.Use(simpleroute.Metrics(metrics))

// Periodic snapshot
go func() {
    for range time.Tick(10 * time.Second) {
        snap := metrics.Snapshot()
        log.Printf("total=%d active=%d avg_ns=%d",
            snap["total_requests"],
            snap["active_requests"],
            snap["avg_duration_ns"])
    }
}()
```

### ContentTypeJson

Sets `Content-Type: application/json` on every response:

```go
router.Use(simpleroute.ContentTypeJson)
```

### WithContext

Injects a static value into the request context:

```go
router.Use(simpleroute.WithContext("db", dbPool))
```

## Custom Middleware

Implement `MiddlewareFunc` (`func(http.Handler) http.Handler`):

```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

For logging, use `GetLogger()` or capture a logger explicitly:

```go
func myMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        simpleroute.GetLogger().Infof("processing %s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}
```
