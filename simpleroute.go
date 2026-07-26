package simpleroute

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"slices"
	"strings"
	"sync"
)

var METHODS = []string{
	"GET",
	"HEAD",
	"POST",
	"PUT",
	"DELETE",
	"PATCH",
	"OPTIONS",
}

type RouterConfig struct {
	AssetDir  string
	AssetPath string
	FS        fs.FS
}

type Router interface {
	Get(path string, args ...any) Router
	Post(path string, args ...any) Router
	Put(path string, args ...any) Router
	Patch(path string, args ...any) Router
	Delete(path string, args ...any) Router
	Head(path string, args ...any) Router
}

type RouteRegister interface {
	Router
	Group(path string, args ...any) Router
	Use(args ...any) RouteRegister
}

type MiddlewareFunc = func(http.Handler) http.Handler

type RouterAction = func(router Router) Router

type ContextKey string

const ParamsContextKey ContextKey = "route_params"

type route struct {
	method      string
	pattern     string
	handler     http.Handler
	middlewares []MiddlewareFunc
	wrapped     http.Handler
}

type routerImpl struct {
	config        RouterConfig
	group         string
	groups        map[string]Router
	routes        map[string][]route
	middlewares   []MiddlewareFunc
	routeHandlers []route
	mux           *http.ServeMux

	once      sync.Once
	built     bool
	buildErr  error
}

func (r *routerImpl) Get(path string, args ...any) Router {
	return r.Handle("GET", path, args...)
}

func (r *routerImpl) Delete(path string, args ...any) Router {
	return r.Handle("DELETE", path, args...)
}

func (r *routerImpl) Patch(path string, args ...any) Router {
	return r.Handle("PATCH", path, args...)
}

func (r *routerImpl) Post(path string, args ...any) Router {
	return r.Handle("POST", path, args...)
}

func (r *routerImpl) Put(path string, args ...any) Router {
	return r.Handle("PUT", path, args...)
}

func (r *routerImpl) Head(path string, args ...any) Router {
	return r.Handle("HEAD", path, args...)
}

func (r *routerImpl) Group(path string, args ...any) Router {
	var callbackRoute RouterAction
	var middlewares []MiddlewareFunc

	for i := range args {
		switch arg := args[i].(type) {
		case RouterAction:
			callbackRoute = arg
		case MiddlewareFunc:
			middlewares = append(middlewares, arg)
		case []MiddlewareFunc:
			middlewares = append(middlewares, arg...)
		}
	}

	if callbackRoute == nil {
		return r
	}

	router := &routerImpl{
		group:       path,
		routes:      make(map[string][]route),
		middlewares: middlewares,
	}

	r.groups[path] = callbackRoute(router)
	return r
}

func (r *routerImpl) Use(args ...any) RouteRegister {
	var method = "GET"
	var pattern string
	var handler http.Handler
	var middlewares []MiddlewareFunc

	for i := range args {
		switch arg := args[i].(type) {
		case HttpRouter:
			arg.Routes(r)
		case string:
			upper := strings.ToUpper(arg)
			if slices.Contains(METHODS, upper) {
				method = upper
			} else {
				pattern = arg
			}
		case http.Handler:
			handler = arg
		case MiddlewareFunc:
			middlewares = append(middlewares, arg)
		case []MiddlewareFunc:
			middlewares = append(middlewares, arg...)
		}
	}

	if method != "" && pattern != "" && handler != nil {
		r.routes[r.group] = append(r.routes[r.group], route{
			method:      method,
			pattern:     pattern,
			handler:     handler,
			middlewares: middlewares,
		})
	} else if len(middlewares) > 0 {
		r.middlewares = append(r.middlewares, middlewares...)
	}

	return r
}

func (r *routerImpl) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if !r.built {
		http.Error(res, "routes not initialize", http.StatusInternalServerError)
		return
	}

	_, pattern := r.mux.Handler(req)
	patterns := strings.Split(pattern, " ")

	if len(patterns) == 2 {
		pathPattern := patterns[1]
		if pathPattern == "/" {
			// Subtree catch-all — check parameterized routes first
			for _, mh := range r.routeHandlers {
				params, ok := r.matchPath(mh.pattern, req.URL.Path)
				if !ok {
					continue
				}
				if mh.method != req.Method {
					http.Error(res, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				if len(params) > 0 {
					ctx := context.WithValue(req.Context(), ParamsContextKey, params)
					req = req.WithContext(ctx)
				}
				mh.wrapped.ServeHTTP(res, req)
				return
			}
			r.mux.ServeHTTP(res, req)
			return
		}
		if params, ok := r.matchPath(pathPattern, req.URL.Path); ok {
			if len(params) > 0 {
				ctx := context.WithValue(req.Context(), ParamsContextKey, params)
				req = req.WithContext(ctx)
			}
			r.mux.ServeHTTP(res, req)
			return
		}
		http.Error(res, "page not found", http.StatusNotFound)
		return
	}

	for _, mh := range r.routeHandlers {
		params, ok := r.matchPath(mh.pattern, req.URL.Path)
		if !ok {
			continue
		}
		if mh.method != req.Method {
			http.Error(res, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if len(params) > 0 {
			ctx := context.WithValue(req.Context(), ParamsContextKey, params)
			req = req.WithContext(ctx)
		}
		mh.wrapped.ServeHTTP(res, req)
		return
	}

	http.Error(res, "page not found", http.StatusNotFound)
}

func (r *routerImpl) Handle(method, path string, args ...any) *routerImpl {
	params := make([]any, 0, 2+len(args))
	params = append(params, method, path)
	params = append(params, args...)
	return r.Use(params...).(*routerImpl)
}

func (r *routerImpl) Build() error {
	r.once.Do(func() {
		mux, handlers, err := r.setupRoutes()
		if err != nil {
			r.buildErr = err
			return
		}
		r.mux = mux
		r.routeHandlers = handlers
		r.built = true
	})
	return r.buildErr
}

func (r *routerImpl) setupRoutes() (*http.ServeMux, []route, error) {
	mux := http.NewServeMux()

	if r.config.AssetPath != "" && r.config.AssetDir != "" {
		var fsHandler http.Handler = http.FileServer(http.Dir(r.config.AssetDir))

		if r.config.FS != nil {
			content, err := fs.Sub(r.config.FS, r.config.AssetDir)
			if err != nil {
				return nil, nil, fmt.Errorf("embedded fs: %w", err)
			}
			fsHandler = http.FileServer(http.FS(content))
		}

		r.Get(r.config.AssetPath, http.StripPrefix(r.config.AssetPath, fsHandler))
	}

	var allRoutes []route
	seen := make(map[string]bool)

	rootRoutes := remap(r)
	for _, rt := range rootRoutes {
		pat := fmt.Sprintf("%s %s", rt.method, rt.pattern)
		if seen[pat] {
			continue
		}
		seen[pat] = true
		wrapped := Handle(rt.middlewares, rt.handler)
		rt.wrapped = wrapped
		mux.Handle(pat, wrapped)
		allRoutes = append(allRoutes, rt)
	}

	for _, group := range r.groups {
		g, ok := group.(*routerImpl)
		if !ok {
			continue
		}

		for _, rt := range remap(g) {
			var handlers []MiddlewareFunc
			handlers = append(handlers, r.middlewares...)
			handlers = append(handlers, rt.middlewares...)
			pat := fmt.Sprintf("%s %s", rt.method, rt.pattern)
			if seen[pat] {
				continue
			}
			seen[pat] = true
			wrapped := Handle(handlers, rt.handler)
			mux.Handle(pat, wrapped)
			allRoutes = append(allRoutes, route{
				method:      rt.method,
				pattern:     rt.pattern,
				handler:     rt.handler,
				middlewares: handlers,
				wrapped:     wrapped,
			})
		}
	}

	return mux, allRoutes, nil
}

func (r *routerImpl) matchPath(pattern, path string) (map[string]string, bool) {
	if strings.Contains(pattern, r.config.AssetPath) && existsInStatic(path, r.config.AssetPath, r.config.AssetDir, r.config.FS) {
		return nil, true
	}
	return matchPath(pattern, path)
}

func NewRouter(config RouterConfig) *routerImpl {
	return &routerImpl{
		config:      config,
		routes:      make(map[string][]route),
		groups:      make(map[string]Router),
		middlewares: make([]MiddlewareFunc, 0),
	}
}

var _ Router = (*routerImpl)(nil)
var _ RouteRegister = (*routerImpl)(nil)
var _ http.Handler = (*routerImpl)(nil)
