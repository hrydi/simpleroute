---
title: Routing
layout: default
nav_order: 2
---

# Routing

## Method Routers

| Method | Signature | Example |
|--------|-----------|---------|
| `Get` | `Get(path string, args ...any)` | `router.Get("/users", handler)` |
| `Post` | `Post(path string, args ...any)` | `router.Post("/users", handler)` |
| `Put` | `Put(path string, args ...any)` | `router.Put("/users/{id}", handler)` |
| `Patch` | `Patch(path string, args ...any)` | `router.Patch("/users/{id}", handler)` |
| `Delete` | `Delete(path string, args ...any)` | `router.Delete("/users/{id}", handler)` |
| `Head` | `Head(path string, args ...any)` | `router.Head("/health", handler)` |

Each method accepts optional middleware as trailing arguments:

```go
router.Get("/admin", authMiddleware, adminHandler)
```

## Path Parameters

Use `{name}` in route patterns. Extract via `Params(r)`:

```go
router.Get("/users/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    id := simpleroute.Params(r)["id"]
    fmt.Fprintf(w, "User: %s", id)
}))
```

Multiple parameters:

```go
router.Get("/files/{dir}/{name}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    params := simpleroute.Params(r)
    fmt.Fprintf(w, "dir=%s, name=%s", params["dir"], params["name"])
}))
```

> `Params(r)` returns `nil` when no parameters are matched.

## Route Groups

Group routes under a common prefix with shared middleware:

```go
router.Group("/api", func(router simpleroute.Router) simpleroute.Router {
    return router.
        Get("/users", listUsers).
        Post("/users", createUser).
        Get("/users/{id}", getUser)
}, authMiddleware, loggerMiddleware)
```

The callback receives a `Router` (no `Use` method). Group middleware is passed as extra arguments after the callback.

## Subtree Mount

Register a handler for all standard methods (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`) under a prefix:

```go
router.Mount("/static/", http.StripPrefix("/static/",
    http.FileServer(http.Dir("./public"))))
```

## Static File Serving

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

When `FS` is nil, `AssetDir` is served from the local filesystem via `os.DirFS`.

## HEAD Auto-Routing

HEAD requests automatically fall back to GET handlers when no explicit HEAD handler is registered. The response body is stripped; headers and status code are preserved.

```go
router.Get("/data", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Length", "5")
    fmt.Fprint(w, "hello")
}))
```

| Request | Status | Body | Headers |
|---------|--------|------|---------|
| `GET /data` | `200` | `hello` | `Content-Length: 5` |
| `HEAD /data` | `200` | _(empty)_ | `Content-Length: 5` |

## Custom 404/405 Handlers

```go
router := simpleroute.NewRouter(simpleroute.RouterConfig{
    NotFoundHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
    }),
    MethodNotAllowedHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
    }),
})
```

The `Allow` header is automatically set on 405 responses. OPTIONS requests return `204 No Content` with `Allow`.

## Route Conflict Detection

Duplicate routes return an error from `Build()`:

```go
router.Get("/same", handlerA)
router.Get("/same", handlerB)

err := router.Build()
// err.Error() → "route conflict: GET /same"
```

> This uses `seen` map deduplication in `setupRoutes`. The first registration wins in the map, but `Build()` still returns an error.
