package simpleroute

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestNewRouter(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	if r == nil {
		t.Fatal("NewRouter returned nil")
	}
}

func TestBuildNotCalled(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestGetRoute(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "world")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "world" {
		t.Errorf("expected 'world', got '%s'", rec.Body.String())
	}
}

func TestPostRoute(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Post("/data", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "created")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/data", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "created" {
		t.Errorf("expected 'created', got '%s'", rec.Body.String())
	}
}

func TestHeadAutoRoute(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	var gotHeaders bool
	r.Get("/data", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "val")
		gotHeaders = true
		fmt.Fprint(w, "body content")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("HEAD", "/data", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Custom") != "val" {
		t.Errorf("expected X-Custom header 'val', got '%s'", rec.Header().Get("X-Custom"))
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body for HEAD, got '%s'", rec.Body.String())
	}
	if !gotHeaders {
		t.Error("GET handler was not called")
	}
}

func TestHeadAutoRouteWithParams(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/users/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := Params(r)["id"]
		fmt.Fprint(w, id)
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("HEAD", "/users/42", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body for HEAD, got '%s'", rec.Body.String())
	}
}

func TestHeadAutoRouteNotAllowed(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Post("/data", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "post only")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("HEAD", "/data", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHeadRoute(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Head("/check", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "val")
		fmt.Fprint(w, "body")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("HEAD", "/check", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Custom") != "val" {
		t.Errorf("expected X-Custom header 'val', got '%s'", rec.Header().Get("X-Custom"))
	}
}

func TestNotFound(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/exists", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/resource", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/resource", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestPathParams(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := Params(r)
		fmt.Fprintf(w, "user %s", params["id"])
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/user/42", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "user 42" {
		t.Errorf("expected 'user 42', got '%s'", rec.Body.String())
	}
}

func TestPathParamsMultiple(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/{a}/x/{b}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := Params(r)
		fmt.Fprintf(w, "%s-%s", params["a"], params["b"])
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/foo/x/bar", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "foo-bar" {
		t.Errorf("expected 'foo-bar', got '%s'", rec.Body.String())
	}
}

func TestParamsReturnsNil(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/plain", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := Params(r)
		if p == nil {
			fmt.Fprint(w, "no params")
		}
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/plain", nil)
	r.ServeHTTP(rec, req)

	if rec.Body.String() != "no params" {
		t.Errorf("expected 'no params', got '%s'", rec.Body.String())
	}
}

func TestGroupRoutes(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Group("/api", func(router Router) Router {
		return router.Get("/ping", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "pong")
		}))
	})
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/ping", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "pong" {
		t.Errorf("expected 'pong', got '%s'", rec.Body.String())
	}
}

func TestGroupMiddleware(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Use(middleware("root"))
	r.Group("/g", func(router Router) Router {
		return router.Get("/r", h)
	}, middleware("group"))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/g/r", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRouteConflictDetection(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/same", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "first")
	}))
	r.Get("/same", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "second")
	}))
	if err := r.Build(); err == nil {
		t.Fatal("expected conflict error, got nil")
	}
}

func TestGlobalMiddlewareOrder(t *testing.T) {
	t.Parallel()
	var order []string
	mu := &sync.Mutex{}
	record := func(s string) { mu.Lock(); order = append(order, s); mu.Unlock() }

	r := NewRouter(RouterConfig{})
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			record("root")
			next.ServeHTTP(w, r)
		})
	})
	r.Group("/g", func(router Router) Router {
		return router.Get("/r", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			record("handler")
			fmt.Fprint(w, "ok")
		}))
	}, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			record("group")
			next.ServeHTTP(w, r)
		})
	})
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/g/r", nil)
	r.ServeHTTP(rec, req)

	if len(order) != 3 {
		t.Fatalf("expected 3 middleware calls, got %d: %v", len(order), order)
	}
	if order[0] != "root" {
		t.Errorf("expected root first, got %s", order[0])
	}
	if order[1] != "group" {
		t.Errorf("expected group second, got %s", order[1])
	}
	if order[2] != "handler" {
		t.Errorf("expected handler last, got %s", order[2])
	}
}

func TestMiddlewareVaryingArgs(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	mw := middleware("mw")

	r.Use(mw, mw)
	r.Get("/a", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}), mw)
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/a", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestBuildIdempotent(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/a", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "a")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}
	if err := r.Build(); err != nil {
		t.Fatal("second Build should not error")
	}
	if r.buildErr != nil {
		t.Fatal("buildErr should be nil after successful build")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/a", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 after second Build, got %d", rec.Code)
	}
}

func TestBuildConcurrent(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/a", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "a")
	}))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := r.Build(); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/a", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 after concurrent Build, got %d", rec.Code)
	}
}

func TestUseWithHttpRouter(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Use(httpRouterFunc(func(rr RouteRegister) {
		rr.Get("/from-router", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "called")
		}))
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/from-router", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "called" {
		t.Errorf("expected 'called', got '%s'", rec.Body.String())
	}
}

func TestCatchAllWithParams(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Use("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "catch-all")
	}))
	r.Get("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "user %s", Params(r)["id"])
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	// Param route should take priority over catch-all
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/user/42", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "user 42" {
		t.Errorf("expected 'user 42', got '%s'", rec.Body.String())
	}

	// Unknown path falls back to catch-all
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/some-page", nil)
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200 from catch-all, got %d", rec2.Code)
	}
	if rec2.Body.String() != "catch-all" {
		t.Errorf("expected 'catch-all', got '%s'", rec2.Body.String())
	}
}

func TestMethodOverrideViaUse(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Use("PUT", "/item", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "updated")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/item", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "updated" {
		t.Errorf("expected 'updated', got '%s'", rec.Body.String())
	}

	// GET should 405
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/item", nil)
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for wrong method, got %d", rec2.Code)
	}
}

func TestMultipleRoutesDifferentMethods(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/resource", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "read")
	}))
	r.Post("/resource", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "create")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, httptest.NewRequest("GET", "/resource", nil))
	if rec1.Body.String() != "read" {
		t.Errorf("expected 'read', got '%s'", rec1.Body.String())
	}

	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, httptest.NewRequest("POST", "/resource", nil))
	if rec2.Body.String() != "create" {
		t.Errorf("expected 'create', got '%s'", rec2.Body.String())
	}
}

func TestRecoverMiddleware(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/panic", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/panic", nil)
	RecoverMiddleware(r).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestCatchAllMethodNotAllowed(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Use("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "catch-all")
	}))
	r.Post("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "post user")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	// GET /user/42 with only POST handler should 405
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/user/42", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for method mismatch with catch-all, got %d", rec.Code)
	}
}

func TestCORSOriginAllowed(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{
		AllowedOrigins: []string{"http://example.com"},
	}
	var called bool
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler not called")
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("expected CORS origin header, got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSOriginDenied(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{
		AllowedOrigins: []string{"http://allowed.com"},
	}
	var called bool
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://evil.com")
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should still be called for denied origins")
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no CORS header for denied origin, got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSPreflight(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         3600,
	}
	var called bool
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	handler.ServeHTTP(rec, req)

	if called {
		t.Error("handler should not be called for preflight")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") != "GET, POST" {
		t.Errorf("unexpected Allow-Methods: '%s'", rec.Header().Get("Access-Control-Allow-Methods"))
	}
	if rec.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Errorf("unexpected Max-Age: '%s'", rec.Header().Get("Access-Control-Max-Age"))
	}
}

func TestCORSNoOrigin(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{
		AllowedOrigins: []string{"http://example.com"},
	}
	var called bool
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	// No Origin header
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should be called when no Origin")
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no CORS header without Origin")
	}
}

func TestContentTypeJsonMiddleware(t *testing.T) {
	t.Parallel()
	handler := ContentTypeJson(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected application/json, got '%s'", rec.Header().Get("Content-Type"))
	}
}

func TestWithContextMiddleware(t *testing.T) {
	t.Parallel()
	handler := WithContext("user", "alice")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Context().Value(ContextKey("user")); v != "alice" {
			t.Errorf("expected 'alice', got '%v'", v)
		}
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec, req)
}

func TestHandleFunc(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/data", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"key": "value"})
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/data", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected application/json, got '%s'", rec.Header().Get("Content-Type"))
	}
}

func TestEmptyRouterServeHTTP(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/anything", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for empty router, got %d", rec.Code)
	}
}

func TestMethodNotAllowedGroup(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Group("/api", func(router Router) Router {
		return router.Post("/data", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "created")
		}))
	})
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestJSONWriter(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusCreated, map[string]string{"id": "abc"})
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected application/json, got '%s'", rec.Header().Get("Content-Type"))
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["id"] != "abc" {
		t.Errorf("expected 'abc', got '%s'", body["id"])
	}
}

// helpers

func middleware(name string) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}

var h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
})

type nopLogger struct{}

func (nopLogger) Errorf(string, ...any) {}
func (nopLogger) Warnf(string, ...any)  {}
func (nopLogger) Infof(string, ...any)  {}
func (nopLogger) Debugf(string, ...any) {}

func TestParallelLoggerRace(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = NewRouter(RouterConfig{
				Logger: nopLogger{},
			})
			_ = GetLogger()
		}()
	}
	wg.Wait()
}

func TestRateLimiterAllowsRequestsWithinBurst(t *testing.T) {
	t.Parallel()
	handler := RateLimiter(RateLimiterConfig{
		RequestsPerSecond: 0,
		Burst:             3,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimiterBlocksExcessRequests(t *testing.T) {
	t.Parallel()
	handler := RateLimiter(RateLimiterConfig{
		RequestsPerSecond: 0,
		Burst:             2,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimiterZeroBurstFallsBack(t *testing.T) {
	t.Parallel()
	handler := RateLimiter(RateLimiterConfig{
		RequestsPerSecond: 5,
		Burst:             0,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimiterAllZeroDefaultsToOne(t *testing.T) {
	t.Parallel()
	handler := RateLimiter(RateLimiterConfig{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}
}

type httpRouterFunc func(RouteRegister)

func (f httpRouterFunc) Routes(r RouteRegister) { f(r) }
