# Getting Started

## Installation

```bash
go get github.com/hrydi/simpleroute
```

Requires Go 1.24.4+.

## Minimal Example

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

## Lifecycle

1. **Create** — `NewRouter(config)` with optional `RouterConfig`
2. **Register** — `Get`, `Post`, `Put`, `Patch`, `Delete`, `Head`, `Group`, `Use`, `Mount`
3. **Build** — `router.Build()` compiles routes into the final handler tree. Returns error on conflicts.
4. **Serve** — Router implements `http.Handler`. Pass to any HTTP server or call `ServeHTTP` directly.

## Running

```bash
go run main.go
```

See the [example app](../example/) for a full demo with embedded UI, CORS, and custom logging.
