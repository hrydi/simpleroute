package simpleroute

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// HttpServer wraps http.Server with graceful Start/Stop lifecycle.
type HttpServer interface {
	Start(router http.Handler) error
	Stop(ctx context.Context) error
}

// HttpRouter is implemented by types that register routes onto a RouteRegister.
// Use it to encapsulate route groups in separate types (see example/user.go).
type HttpRouter interface {
	Routes(r RouteRegister)
}

// ServerConfig configures the HTTP server created by NewHttp.
// Zero values are replaced with sensible production defaults
// (10s ReadTimeout, 10s WriteTimeout, 60s IdleTimeout).
type ServerConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func (c *ServerConfig) defaults() {
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 10 * time.Second
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 10 * time.Second
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = 60 * time.Second
	}
}

type httpServerImpl struct {
	app http.Server
}

func (h *httpServerImpl) Start(router http.Handler) error {
	h.app.Handler = RecoverMiddleware(router)
	if err := h.app.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
	return nil
}

func (h *httpServerImpl) Stop(ctx context.Context) error {
	return h.app.Shutdown(ctx)
}

// NewHttp creates a new HTTP server for the given config.
// Production-ready timeouts are set by default (configurable via ServerConfig).
func NewHttp(config ServerConfig) *httpServerImpl {
	config.defaults()
	return &httpServerImpl{
		app: http.Server{
			Addr:         config.Addr,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			IdleTimeout:  config.IdleTimeout,
		},
	}
}

var _ HttpServer = (*httpServerImpl)(nil)

// Params extracts all path parameters from the request context as a map.
// Returns nil if no parameters were matched.
func Params(r *http.Request) map[string]string {
	params, _ := r.Context().Value(ParamsContextKey).([]Param)
	if params == nil {
		return nil
	}
	m := make(map[string]string, len(params))
	for _, p := range params {
		m[p.Key] = p.Value
	}
	return m
}

// URLParam returns the value of a single path parameter by name.
// Returns empty string if the parameter is not found.
func URLParam(r *http.Request, key string) string {
	params, _ := r.Context().Value(ParamsContextKey).([]Param)
	for _, p := range params {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

// JSON writes data as JSON with the given status code.
// Sets Content-Type to application/json automatically.
func JSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

// SetCtx stores a value in the request context and returns the modified request.
// Chainable: r = SetCtx(SetCtx(r, "a", 1), "b", 2).
func SetCtx(r *http.Request, key, value any) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), key, value))
}

// GetCtx retrieves a typed value from the request context.
// Returns the zero value and false if the key is missing or the type doesn't match.
func GetCtx[T any](r *http.Request, key any) (T, bool) {
	v := r.Context().Value(key)
	if v == nil {
		var zero T
		return zero, false
	}
	val, ok := v.(T)
	return val, ok
}

// GetLogger returns the package-level logger, which is synced
// from the most recently created router's RouterConfig.Logger.
func GetLogger() Logger {
	return getPkgLogger()
}
