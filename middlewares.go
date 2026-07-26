package simpleroute

import (
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// WithContext injects a key-value pair into the request context.
func WithContext(name string, value any) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nameCtx := ContextKey(name)
			ctx := context.WithValue(r.Context(), nameCtx, value)
			req := r.WithContext(ctx)
			next.ServeHTTP(w, req)
		})
	}
}

// ContentTypeJson sets Content-Type: application/json on every response.
func ContentTypeJson(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// RecoverMiddleware catches panics in the handler chain, logs the error,
// and returns a 500 Internal Server Error to the client.
// When stackTrace is true, the full stack trace is logged at error level.
func RecoverMiddleware(next http.Handler, stackTrace ...bool) http.Handler {
	withStack := len(stackTrace) > 0 && stackTrace[0]
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				l := getPkgLogger()
				if withStack {
					buf := make([]byte, 4096)
					n := runtime.Stack(buf, false)
					l.Errorf("recover from panic %v\n%s", err, buf[:n])
				} else {
					l.Errorf("recover from panic %v", err)
				}
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORSConfig configures CORS behavior for the CORS middleware.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// CORS returns a middleware that handles cross-origin requests.
// Preflight (OPTIONS) requests return 204 without calling the next handler.
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			allowed := false
			for _, o := range config.AllowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if len(config.ExposedHeaders) > 0 {
				for _, h := range config.ExposedHeaders {
					w.Header().Add("Access-Control-Expose-Headers", h)
				}
			}

			if r.Method == http.MethodOptions {
				if len(config.AllowedMethods) > 0 {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				}
				if len(config.AllowedHeaders) > 0 {
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				}
				if config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger logs the HTTP method, path, and duration of each request.
// An optional logger can be provided; otherwise the package-level logger is used.
func RequestLogger(next http.Handler, logger ...Logger) http.Handler {
	l := getPkgLogger()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		l.Infof("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// RequestID injects or preserves an X-Request-ID header in both the response
// and the request context. If the incoming request has no X-Request-ID, a
// unique ID is generated using the current timestamp.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Request-ID", id)
		r = SetCtx(r, ContextKey("request_id"), id)
		next.ServeHTTP(w, r)
	})
}

// gzipResponseWriter wraps http.ResponseWriter, compressing the body with gzip.
type gzipResponseWriter struct {
	http.ResponseWriter
	zw *gzip.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.zw.Write(b)
}

// Gzip returns a middleware that compresses responses with gzip when the
// client sends Accept-Encoding: gzip.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		zw := gzip.NewWriter(w)
		defer zw.Close()
		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, zw: zw}, r)
	})
}

// RateLimiterConfig configures the rate limiter middleware.
type RateLimiterConfig struct {
	RequestsPerSecond int
	Burst             int
}

// RateLimiter returns a middleware that limits request rates using a token
// bucket algorithm. It returns 429 Too Many Requests when the limit is exceeded.
func RateLimiter(config RateLimiterConfig) func(http.Handler) http.Handler {
	var mu sync.Mutex
	tokens := float64(config.Burst)
	lastRefill := time.Now()
	maxTokens := float64(config.Burst)
	if maxTokens <= 0 {
		maxTokens = 1
	}
	if tokens <= 0 {
		tokens = maxTokens
	}
	if maxTokens <= 0 {
		maxTokens = float64(config.RequestsPerSecond)
	}
	rate := float64(config.RequestsPerSecond)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			now := time.Now()
			elapsed := now.Sub(lastRefill).Seconds()
			tokens += elapsed * rate
			if tokens > maxTokens {
				tokens = maxTokens
			}
			if tokens < 1 {
				mu.Unlock()
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			tokens--
			lastRefill = now
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

// MetricsRecorder records HTTP request metrics.
type MetricsRecorder struct {
	TotalRequests  atomic.Int64
	ActiveRequests atomic.Int64
	TotalDuration  atomic.Int64
}

// Snapshot returns a point-in-time snapshot of the metrics.
func (m *MetricsRecorder) Snapshot() map[string]any {
	return map[string]any{
		"total_requests":   m.TotalRequests.Load(),
		"active_requests":  m.ActiveRequests.Load(),
		"avg_duration_ns":  m.avgDuration(),
	}
}

func (m *MetricsRecorder) avgDuration() int64 {
	total := m.TotalRequests.Load()
	if total == 0 {
		return 0
	}
	return m.TotalDuration.Load() / total
}

// Metrics returns a middleware that records request count, concurrency, and
// cumulative duration. Pass a shared *MetricsRecorder to collect data.
func Metrics(recorder *MetricsRecorder) func(http.Handler) http.Handler {
	if recorder == nil {
		recorder = new(MetricsRecorder)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder.ActiveRequests.Add(1)
			recorder.TotalRequests.Add(1)
			start := time.Now()
			next.ServeHTTP(w, r)
			recorder.TotalDuration.Add(time.Since(start).Nanoseconds())
			recorder.ActiveRequests.Add(-1)
		})
	}
}
