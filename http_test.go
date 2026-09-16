package simpleroute

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServerConfigDefaults(t *testing.T) {
	t.Parallel()
	c := ServerConfig{}
	c.defaults()

	if c.ReadTimeout != 10*time.Second {
		t.Errorf("expected ReadTimeout 10s, got %s", c.ReadTimeout)
	}
	if c.WriteTimeout != 10*time.Second {
		t.Errorf("expected WriteTimeout 10s, got %s", c.WriteTimeout)
	}
	if c.IdleTimeout != 60*time.Second {
		t.Errorf("expected IdleTimeout 60s, got %s", c.IdleTimeout)
	}
}

func TestServerConfigKeepsExplicitValues(t *testing.T) {
	t.Parallel()
	c := ServerConfig{
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 2 * time.Second,
		IdleTimeout:  3 * time.Second,
	}
	c.defaults()

	if c.ReadTimeout != 1*time.Second || c.WriteTimeout != 2*time.Second || c.IdleTimeout != 3*time.Second {
		t.Errorf("defaults() overwrote explicit values: %+v", c)
	}
}

func TestNewHttpAppliesConfig(t *testing.T) {
	t.Parallel()
	s := NewHttp(ServerConfig{Addr: "127.0.0.1:1234", ReadTimeout: 5 * time.Second})

	if s.app.Addr != "127.0.0.1:1234" {
		t.Errorf("expected addr 127.0.0.1:1234, got %s", s.app.Addr)
	}
	if s.app.ReadTimeout != 5*time.Second {
		t.Errorf("expected ReadTimeout 5s, got %s", s.app.ReadTimeout)
	}
	if s.app.WriteTimeout != 10*time.Second {
		t.Errorf("expected defaulted WriteTimeout 10s, got %s", s.app.WriteTimeout)
	}
	if s.app.IdleTimeout != 60*time.Second {
		t.Errorf("expected defaulted IdleTimeout 60s, got %s", s.app.IdleTimeout)
	}
}

func TestServerStartServesAndStopsGracefully(t *testing.T) {
	t.Parallel()
	addr := freeAddr(t)
	server := NewHttp(ServerConfig{Addr: addr})

	r := NewRouter(RouterConfig{})
	r.Get("/ping", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "pong")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- server.Start(r) }()
	waitForServer(t, addr)

	body, code := get(t, "http://"+addr+"/ping")
	if code != http.StatusOK {
		t.Errorf("expected 200, got %d", code)
	}
	if body != "pong" {
		t.Errorf("expected 'pong', got '%s'", body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected nil after graceful shutdown (ErrServerClosed is filtered), got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after Stop")
	}
}

func TestServerStartRecoversHandlerPanic(t *testing.T) {
	t.Parallel()
	addr := freeAddr(t)
	server := NewHttp(ServerConfig{Addr: addr})

	r := NewRouter(RouterConfig{})
	r.Get("/boom", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("handler exploded")
	}))
	r.Get("/ok", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "still alive")
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	go server.Start(r)
	waitForServer(t, addr)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Stop(ctx)
	})

	// Start wraps the router in RecoverMiddleware, so a panicking handler
	// must surface as 500 rather than killing the server.
	if _, code := get(t, "http://"+addr+"/boom"); code != http.StatusInternalServerError {
		t.Errorf("expected 500 from recovered panic, got %d", code)
	}

	body, code := get(t, "http://"+addr+"/ok")
	if code != http.StatusOK || body != "still alive" {
		t.Errorf("server did not survive panic: got %d '%s'", code, body)
	}
}

func TestServerStartReturnsListenError(t *testing.T) {
	t.Parallel()

	// Hold the port so ListenAndServe cannot bind it.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	r := NewRouter(RouterConfig{})
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	server := NewHttp(ServerConfig{Addr: l.Addr().String()})
	if err := server.Start(r); err == nil {
		t.Error("expected a listen error for an occupied port, got nil")
	}
}

func TestURLParam(t *testing.T) {
	t.Parallel()
	r := NewRouter(RouterConfig{})
	r.Get("/user/{id}/post/{postID}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s|%s|%q", URLParam(r, "id"), URLParam(r, "postID"), URLParam(r, "missing"))
	}))
	if err := r.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/user/7/post/99", nil))

	if rec.Body.String() != `7|99|""` {
		t.Errorf(`expected '7|99|""', got '%s'`, rec.Body.String())
	}
}

func TestSetCtxAndGetCtx(t *testing.T) {
	t.Parallel()
	type user struct{ Name string }

	req := httptest.NewRequest("GET", "/", nil)
	req = SetCtx(SetCtx(req, "user", &user{Name: "alice"}), "count", 3)

	got, ok := GetCtx[*user](req, "user")
	if !ok || got.Name != "alice" {
		t.Errorf("expected user alice, got %+v (ok=%v)", got, ok)
	}

	count, ok := GetCtx[int](req, "count")
	if !ok || count != 3 {
		t.Errorf("expected count 3, got %d (ok=%v)", count, ok)
	}
}

func TestGetCtxMissingAndWrongType(t *testing.T) {
	t.Parallel()
	req := SetCtx(httptest.NewRequest("GET", "/", nil), "count", 3)

	if v, ok := GetCtx[int](req, "nope"); ok || v != 0 {
		t.Errorf("expected zero value and false for a missing key, got %d (ok=%v)", v, ok)
	}
	if v, ok := GetCtx[string](req, "count"); ok || v != "" {
		t.Errorf("expected zero value and false on type mismatch, got %q (ok=%v)", v, ok)
	}
}

// helpers

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

func waitForServer(t *testing.T, addr string) {
	t.Helper()
	for range 100 {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server at %s never became reachable", addr)
}

// get uses a keep-alive-free client so connections never linger and delay
// the graceful shutdown these tests assert on.
func get(t *testing.T, url string) (string, int) {
	t.Helper()
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading %s: %v", url, err)
	}
	return string(body), resp.StatusCode
}
