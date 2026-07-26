---
title: Getting Started
layout: default
nav_order: 1
---

# Getting Started

## Installation

```bash
go get github.com/hrydi/simpleroute
```

Requires **Go 1.24.4+**.

## Minimal Server

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

    server := simpleroute.NewHttp(simpleroute.ServerConfig{
        Addr: ":8080",
    })
    if err := server.Start(router); err != nil {
        panic(err)
    }
}
```

> `server.Start` returns `nil` on graceful shutdown (`http.ErrServerClosed` is filtered).

## Lifecycle

| Phase | Call | Description |
|-------|------|-------------|
| **1. Create** | `NewRouter(config)` | Create router with optional `RouterConfig` |
| **2. Register** | `Get()`, `Post()`, `Use()`, `Group()`, `Mount()` | Register routes and middleware |
| **3. Build** | `router.Build()` | Compile routes into handler tree. Returns error on conflicts |
| **4. Serve** | `router.ServeHTTP(w, r)` or `server.Start(router)` | Router implements `http.Handler` |

`Build()` is idempotent and concurrent-safe (`sync.Once`).

## Running the Example

The repo includes a full example app with Vue frontend, CORS, and custom logging:

```bash
make run
```

See the [example directory](https://github.com/hrydi/simpleroute/tree/main/example) for the complete source.
