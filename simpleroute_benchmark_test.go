package simpleroute

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkStaticRoute(b *testing.B) {
	r := NewRouter(RouterConfig{})
	r.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err := r.Build(); err != nil {
		b.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}
	})
}

func BenchmarkStaticRouteDeep(b *testing.B) {
	r := NewRouter(RouterConfig{})
	r.Get("/a/b/c/d/e", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err := r.Build(); err != nil {
		b.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/a/b/c/d/e", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}
	})
}

func BenchmarkPathParams(b *testing.B) {
	r := NewRouter(RouterConfig{})
	r.Get("/user/{id}/post/{postId}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err := r.Build(); err != nil {
		b.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/user/42/post/99", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}
	})
}

func BenchmarkNotFound(b *testing.B) {
	r := NewRouter(RouterConfig{})
	r.Get("/existing", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err := r.Build(); err != nil {
		b.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/notfound", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}
	})
}

func BenchmarkCatchAll(b *testing.B) {
	r := NewRouter(RouterConfig{})
	r.Use("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err := r.Build(); err != nil {
		b.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/any-page", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}
	})
}

func BenchmarkMultipleRoutes(b *testing.B) {
	r := NewRouter(RouterConfig{})
	for i := range 1000 {
		path := fmt.Sprintf("/route/%d", i)
		r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	}
	if err := r.Build(); err != nil {
		b.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/route/500", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}
	})
}

func BenchmarkBuild(b *testing.B) {
	for b.Loop() {
		r := NewRouter(RouterConfig{})
		for i := range 1000 {
			path := fmt.Sprintf("/route/%d", i)
			r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			r.Post(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
		}
		if err := r.Build(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMiddlewareChainDepth(b *testing.B) {
	r := NewRouter(RouterConfig{})
	var mws []MiddlewareFunc
	for range 10 {
		mws = append(mws, func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		})
	}
	r.Use("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), mws)
	if err := r.Build(); err != nil {
		b.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}
	})
}

func BenchmarkParamsExtraction(b *testing.B) {
	r := NewRouter(RouterConfig{})
	r.Get("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = Params(r)
		w.WriteHeader(http.StatusOK)
	}))
	if err := r.Build(); err != nil {
		b.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/user/abc123", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}
	})
}

func BenchmarkGroupedRoutes(b *testing.B) {
	r := NewRouter(RouterConfig{})
	r.Group("/api", func(router Router) Router {
		for i := range 100 {
			path := fmt.Sprintf("/resource/%d", i)
			router.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
		}
		return router
	})
	if err := r.Build(); err != nil {
		b.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/resource/50", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
		}
	})
}

func BenchmarkRouteRegistration(b *testing.B) {
	for b.Loop() {
		r := NewRouter(RouterConfig{})
		for i := range 100 {
			path := fmt.Sprintf("/route/%d", i)
			r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
		}
	}
}

func BenchmarkConcurrentServe(b *testing.B) {
	r := NewRouter(RouterConfig{})
	for i := range 100 {
		path := fmt.Sprintf("/route/%d", i)
		r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	}
	r.Get("/user/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = Params(r)
		w.WriteHeader(http.StatusOK)
	}))
	r.Use("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err := r.Build(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/route/50", nil)
			r.ServeHTTP(rec, req)
		}
	})
}
