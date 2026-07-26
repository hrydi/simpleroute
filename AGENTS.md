# AGENTS.md

Lightweight Go HTTP router (`github.com/hrydi/simpleroute`). Go 1.24.4+ required (see `go.mod`).

## Commands

- `make run` — runs the example app (`go run ./example`). Needs `HTTP_PORT` env (default 17881).
- `make build` — production Docker build (`deployments/docker-compose.yml` target `hrydi-simple-route-build`).
- `make compose-run` — dev stack: mounts repo + starts Vite UI container.
- `go build ./...` — compile everything.
- `go vet ./...` — lint.
- `go test ./...` — run all tests (42 tests + 12 benchmarks in `simpleroute_test.go` and `simpleroute_benchmark_test.go`).

## Library entry points

- `NewRouter(config RouterConfig)` → `*routerImpl`. Register routes, then call `.Build() error`.
- `NewHttp(config ServerConfig)` → `*httpServerImpl`. `server.Start(router)` starts serving (router implements `http.Handler`).
- `ServeHTTP` returns `500 "routes not initialize"` if `Build()` was not called first.

## Router API

- `Router` interface: `Get/Post/Put/Patch/Delete/Head(path, args...)`.
- `RouteRegister` adds `Group(path, args...)` and `Use(args...)`.
- `Use` is variadic and polymorphic: accepts `HttpRouter`, `string` (method or pattern), `http.Handler`, `MiddlewareFunc`, `[]MiddlewareFunc`.
- `HttpRouter` interface (`Routes(r RouteRegister)`) is the pattern for grouping route registration (see `example/user.go`).
- Middleware order: global router middlewares run first, then route middlewares, then the handler. `Handle` wraps in reverse so the first middleware is outermost.
- Path parameters (`{param}` in patterns) are injected into request context. Use `simpleroute.Params(r *http.Request) map[string]string` to retrieve them.
- Group middleware is passed as extra args to `Group(path, callback, middleware1, middleware2, ...)`. The callback receives `Router` (no `Use` method).

## Built-in middleware

- `CORS(config CORSConfig)` — configurable CORS with origin validation, preflight, headers, credentials, max-age.
- `RecoverMiddleware` — panic recovery that logs and returns 500.
- `ContentTypeJson` — sets `Content-Type: application/json`.
- `RequestLogger` — logs method, path, and duration.
- `WithContext(name, value)` — injects a value into the request context.

## Dev vs production

- `DEV=development` — the UI is served via a reverse proxy to Vite dev server (default `http://localhost:5566`).
- Otherwise static assets are served from the embedded `vue/dist` (via `go:embed`).

## Memory / leak notes

- `pkg/signal.HandleSignals` spawns a goroutine that loops forever reading from `signal.Notify`; it is never stopped. Acceptable for a process-lifetime signal listener, but do not call it in tests or per-request code.
- Route maps (`routes`, `groups`) are populated at setup and never grow per-request — no leak.
- `routerImpl` holds no per-request state; `ServeHTTP` reads only `r.mux` and `r.routeHandlers` (set once by `Build()` via `sync.Once`).
- Middleware chains are built once at `Build()` time and cached; zero per-request allocations for middleware wrapping.

## Repo layout

- Root `*.go` (`simpleroute.go`, `http.go`, `functions.go`, `middlewares.go`) — the library.
- `pkg/signal/` — signal-to-context bridge.
- `example/` — runnable demo app (Vue frontend via `go:embed`).
- `build/Dockerfile` — multi-stage (Go + Node/Vite).
- `deployments/` — compose + `.env`.
