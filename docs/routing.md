# Routing

## Method Routers

| Method | Usage |
|--------|-------|
| `Get(path, args...)` | `router.Get("/users", handler)` |
| `Post(path, args...)` | `router.Post("/users", handler)` |
| `Put(path, args...)` | `router.Put("/users/{id}", handler)` |
| `Patch(path, args...)` | `router.Patch("/users/{id}", handler)` |
| `Delete(path, args...)` | `router.Delete("/users/{id}", handler)` |
| `Head(path, args...)` | `router.Head("/health", handler)` |

Each accepts an optional middleware list as trailing arguments.

## Path Parameters

Use `{name}` in patterns. Extracted via `Params(r)`:

```go
router.Get("/users/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    id := simpleroute.Params(r)["id"]
    fmt.Fprintf(w, "User: %s", id)
}))

router.Get("/files/{dir}/{name}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    params := simpleroute.Params(r)
    fmt.Fprintf(w, "dir=%s, name=%s", params["dir"], params["name"])
}))
```

## Route Groups

Group routes under a prefix with shared middleware:

```go
router.Group("/api", func(router simpleroute.Router) simpleroute.Router {
    return router.
        Get("/users", listUsers).
        Post("/users", createUser).
        Get("/users/{id}", getUser)
}, authMiddleware, loggerMiddleware)
```

The callback receives a `Router` (no `Use` method). Group middleware is passed as extra args.

## Subtree Mount

Mount any `http.Handler` to handle all methods under a prefix:

```go
router.Mount("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./public"))))
```

`Mount` registers the handler for all standard HTTP methods (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`).

## Static File Serving

Serve embedded or on-disk files:

```go
//go:embed static/*
var staticFS embed.FS

router := simpleroute.NewRouter(simpleroute.RouterConfig{
    AssetPath: "/assets/",
    AssetDir:  "static",
    FS:        staticFS,
})
```

## HEAD Auto-Routing

HEAD requests without an explicit HEAD handler automatically fall back to GET. The response body is stripped — headers and status code are preserved:

```go
router.Get("/data", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Length", "5")
    fmt.Fprint(w, "hello")
}))

// HEAD /data → 200 OK, Content-Length: 5, body: ""
```

## Custom 404/405

```go
router := simpleroute.NewRouter(simpleroute.RouterConfig{
    NotFoundHandler:         http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
    }),
    MethodNotAllowedHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
    }),
})
```

Route conflicts return an error from `Build()`:

```go
router.Get("/same", handlerA)
router.Get("/same", handlerB)
err := router.Build()  // returns error: route conflict: GET /same
```
