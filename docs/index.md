---
title: simpleroute
layout: default
---

[![Go Reference](https://pkg.go.dev/badge/github.com/hrydi/simpleroute)](https://pkg.go.dev/github.com/hrydi/simpleroute)
[![Go Report Card](https://goreportcard.com/badge/github.com/hrydi/simpleroute)](https://goreportcard.com/report/github.com/hrydi/simpleroute)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue)](https://golang.org/dl/)

**simpleroute** is a lightweight, zero-dependency HTTP router for Go 1.24+. Built entirely on the standard library.

```go
import "github.com/hrydi/simpleroute"
```

---

## Quick Start

```go
router := simpleroute.NewRouter(simpleroute.RouterConfig{})
router.Get("/hello", handler)
router.Build()
server := simpleroute.NewHttp(simpleroute.ServerConfig{Addr: ":8080"})
server.Start(router)
```

---

## Documentation

| Section | Description |
|---------|-------------|
| [Getting Started](getting-started) | Installation, quick start, full lifecycle |
| [Routing](routing) | Method routers, path params, groups, mount, static files, HEAD auto-routing |
| [Middleware](middleware) | Built-in middleware, custom middleware, ordering |
| [Context & Logger](context-logger) | Base context, SetCtx/GetCtx, path params, logger interface |
| [Configuration](configuration) | RouterConfig, ServerConfig, polymorphic Use, benchmarks |

---

## Features

- **Zero external dependencies** — pure stdlib
- **Path parameters** — `{param}` patterns injected into request context
- **Group routing** — namespaced routes with shared middleware
- **Middleware chain** — global, group, and route-level
- **Polymorphic `Use`** — accepts `HttpRouter`, `http.Handler`, `MiddlewareFunc`, method/pattern strings
- **Built-in middleware** — CORS, panic recovery, request logging, request ID, gzip, rate limiter, metrics
- **HEAD auto-routing** — HEAD → GET fallback with body stripping
- **Custom 404/405 handlers** — plug your own
- **Subtree mount** — all-methods handler registration
- **Concurrent-safe** — `sync.Once` build, `sync.RWMutex` logger, no per-request locks
- **Production-ready server** — configurable timeouts (10s read / 10s write / 60s idle)

---

## Benchmarks

```
BenchmarkStaticRoute-8         10,124,955   127 ns/op   242 B/op   5 allocs/op
BenchmarkPathParams-8           3,818,419   404 ns/op   776 B/op  12 allocs/op
BenchmarkBuild-8                      222  5.71 ms/op  4.48 MB/op  40k allocs/op
BenchmarkConcurrentServe-8        730,724  1.73 μs/op 5.35 kB/op  14 allocs/op
```

[Full benchmark table →](configuration#benchmarks)
