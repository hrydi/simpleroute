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

// METHODS lists all HTTP methods the router supports.
var METHODS = []string{
	"GET",
	"HEAD",
	"POST",
	"PUT",
	"DELETE",
	"PATCH",
	"OPTIONS",
}

// RouterConfig configures a new router instance.
type RouterConfig struct {
	AssetDir               string
	AssetPath              string
	FS                     fs.FS
	Logger                 Logger
	LogLevel               LogLevel
	BaseContext            context.Context
	NotFoundHandler        http.Handler
	MethodNotAllowedHandler http.Handler
}

// Router defines HTTP method handlers for route registration.
type Router interface {
	Get(path string, args ...any) Router
	Post(path string, args ...any) Router
	Put(path string, args ...any) Router
	Patch(path string, args ...any) Router
	Delete(path string, args ...any) Router
	Head(path string, args ...any) Router
	Mount(path string, sub http.Handler) Router
}

// RouteRegister extends Router with group and middleware registration.
type RouteRegister interface {
	Router
	Group(path string, args ...any) Router
	Use(args ...any) RouteRegister
}

// MiddlewareFunc wraps an http.Handler to add cross-cutting behavior.
type MiddlewareFunc = func(http.Handler) http.Handler

// RouterAction is a callback that receives a Router and returns it.
type RouterAction = func(router Router) Router

// ContextKey is used for request context value keys.
type ContextKey string

// ParamsContextKey is the context key for path parameters.
const ParamsContextKey ContextKey = "route_params"

type segment struct {
	isParam bool
	val     string
}

type route struct {
	method      string
	pattern     string
	handler     http.Handler
	middlewares []MiddlewareFunc
	wrapped     http.Handler
	segments    []segment
}

type routerImpl struct {
	config        RouterConfig
	log           Logger
	group         string
	groups        map[string]Router
	routes        map[string][]route
	middlewares   []MiddlewareFunc
	routeHandlers []route
	allowMethods  map[string][]string
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

// Mount attaches a sub-handler under the given path prefix.
// All requests to path/* are forwarded to the sub-handler.
func (r *routerImpl) Mount(path string, sub http.Handler) Router {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	for _, m := range METHODS {
		r.Use(m, path, sub)
	}
	return r
}

// Group creates a route group under the given path prefix.
// Extra args are treated as group-level middleware (MiddlewareFunc or []MiddlewareFunc).
// The last func(Router) Router argument is the callback that receives a Router with no Use method.
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

// Use is the polymorphic registration entry point.
// It accepts: HttpRouter, string (method or pattern), http.Handler,
// MiddlewareFunc, and []MiddlewareFunc in any order.
// If method+pattern+handler are present, a route is registered.
// Otherwise middleware args are appended to the global middleware list.
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

// headResponseWriter discards the body on write so HEAD handlers
// can reuse GET handlers without sending a response body.
type headResponseWriter struct {
	http.ResponseWriter
}

func (h *headResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

// ServeHTTP implements http.Handler. Returns 500 if Build() was not called.
func (r *routerImpl) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if !r.built {
		r.log.Errorf("routes not initialized")
		http.Error(res, "routes not initialize", http.StatusInternalServerError)
		return
	}

	if r.config.BaseContext != nil {
		ctx, cancel := context.WithCancel(r.config.BaseContext)
		defer cancel()
		req = req.WithContext(ctx)
	}

	if req.Method == http.MethodHead {
		res = &headResponseWriter{ResponseWriter: res}
	}

	if r.dispatchMuxRoute(res, req) {
		return
	}

	for _, mh := range r.routeHandlers {
		if r.tryRoute(mh, res, req) {
			return
		}
	}

	r.writeNotFound(res, req)
}

// dispatchMuxRoute attempts to dispatch via mux.Handler pattern routing.
// Returns true if the request was handled (or a response was written).
func (r *routerImpl) dispatchMuxRoute(res http.ResponseWriter, req *http.Request) bool {
	muxHandler, pattern := r.mux.Handler(req)
	patterns := strings.Split(pattern, " ")
	if len(patterns) != 2 {
		return false
	}

	pathPattern := patterns[1]
	if strings.HasSuffix(pathPattern, "/") {
		if pathPattern == "/" {
			for _, mh := range r.routeHandlers {
				if r.tryRoute(mh, res, req) {
					return true
				}
			}
		}
		muxHandler.ServeHTTP(res, req)
		return true
	}

	if !strings.ContainsRune(pathPattern, '{') {
		muxHandler.ServeHTTP(res, req)
		return true
	}

	if params, ok := r.matchPath(pathPattern, req.URL.Path); ok {
		if len(params) > 0 {
			ctx := context.WithValue(req.Context(), ParamsContextKey, params)
			req = req.WithContext(ctx)
		}
		muxHandler.ServeHTTP(res, req)
		return true
	}

	r.writeNotFound(res, req)
	return true
}

// tryRoute attempts to match and dispatch a single parameterized route.
// Returns true if the route matched (regardless of whether it was dispatched
// or rejected with a method-not-allowed response).
func (r *routerImpl) tryRoute(mh route, res http.ResponseWriter, req *http.Request) bool {
	var params []Param
	var ok bool
	if mh.segments != nil && r.config.AssetPath == "" {
		params, ok = matchRoute(mh.segments, req.URL.Path)
	} else {
		params, ok = r.matchPath(mh.pattern, req.URL.Path)
	}
	if !ok {
		return false
	}
	rw, ok := r.matchMethod(mh, res, req)
	if !ok {
		return true
	}
	if len(params) > 0 {
		ctx := context.WithValue(req.Context(), ParamsContextKey, params)
		req = req.WithContext(ctx)
	}
	mh.wrapped.ServeHTTP(rw, req)
	return true
}

// matchMethod checks whether the request method matches the route.
// HEAD requests are allowed to match GET routes (body is already stripped
// by the headResponseWriter installed at the top of ServeHTTP).
func (r *routerImpl) matchMethod(mh route, res http.ResponseWriter, req *http.Request) (http.ResponseWriter, bool) {
	if mh.method == req.Method {
		return res, true
	}
	if req.Method == http.MethodHead && mh.method == http.MethodGet {
		return res, true
	}
	r.writeMethodNotAllowed(res, req, mh.pattern)
	return res, false
}

func (r *routerImpl) writeMethodNotAllowed(res http.ResponseWriter, req *http.Request, pattern string) {
	methods := r.allowMethods[pattern]
	if len(methods) > 0 {
		res.Header().Set("Allow", strings.Join(methods, ", "))
	}
	if req.Method == http.MethodOptions {
		res.WriteHeader(http.StatusNoContent)
		return
	}
	if r.config.MethodNotAllowedHandler != nil {
		r.config.MethodNotAllowedHandler.ServeHTTP(res, req)
		return
	}
	http.Error(res, "method not allowed", http.StatusMethodNotAllowed)
}

func (r *routerImpl) writeNotFound(res http.ResponseWriter, req *http.Request) {
	if r.config.NotFoundHandler != nil {
		r.config.NotFoundHandler.ServeHTTP(res, req)
		return
	}
	http.Error(res, "page not found", http.StatusNotFound)
}

// Handle is the internal route registration method used by Get/Post/etc.
// It collects method, path, and optional middleware into a single call to Use.
func (r *routerImpl) Handle(method, path string, args ...any) *routerImpl {
	params := make([]any, 0, 2+len(args))
	params = append(params, method, path)
	params = append(params, args...)
	return r.Use(params...).(*routerImpl)
}

// Build compiles all registered routes and middlewares into the final handler tree.
// Must be called before the router can ServeHTTP. Idempotent and concurrent-safe.
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

	for _, routes := range r.routes {
		for _, rt := range routes {
			mws := chainMiddleware(r.middlewares, rt.middlewares)
			if err := r.registerRoute(mux, seen, &allRoutes, rt, mws); err != nil {
				return nil, nil, err
			}
		}
	}

	for _, group := range r.groups {
		g, ok := group.(*routerImpl)
		if !ok {
			continue
		}

		for _, routes := range g.routes {
			for _, rt := range routes {
				pattern := rt.pattern
				if g.group != "" {
					if pattern == "/" {
						pattern = ""
					}
					pattern = g.group + pattern
				}

				mws := chainMiddleware(r.middlewares, g.middlewares, rt.middlewares)
				grt := route{
					method:  rt.method,
					pattern: pattern,
					handler: rt.handler,
				}
				if err := r.registerRoute(mux, seen, &allRoutes, grt, mws); err != nil {
					return nil, nil, err
				}
			}
		}
	}

	allow := make(map[string][]string)
	for _, rt := range allRoutes {
		allow[rt.pattern] = append(allow[rt.pattern], rt.method)
	}
	for p := range allow {
		slices.Sort(allow[p])
	}
	r.allowMethods = allow

	return mux, allRoutes, nil
}

func (r *routerImpl) registerRoute(mux *http.ServeMux, seen map[string]bool, allRoutes *[]route, rt route, middlewares []MiddlewareFunc) error {
	pat := fmt.Sprintf("%s %s", rt.method, rt.pattern)
	if seen[pat] {
		return fmt.Errorf("route conflict: %s", pat)
	}
	seen[pat] = true
	wrapped := Handle(middlewares, rt.handler)
	mux.Handle(pat, wrapped)
	*allRoutes = append(*allRoutes, route{
		method:      rt.method,
		pattern:     rt.pattern,
		handler:     rt.handler,
		middlewares: middlewares,
		wrapped:     wrapped,
		segments:    parseSegments(rt.pattern),
	})
	return nil
}

func (r *routerImpl) matchPath(pattern, path string) ([]Param, bool) {
	if r.config.AssetPath != "" && strings.Contains(pattern, r.config.AssetPath) && existsInStatic(path, r.config.AssetPath, r.config.AssetDir, r.config.FS) {
		return nil, true
	}
	return matchPath(pattern, path)
}

// NewRouter creates a new router with the given configuration.
func NewRouter(config RouterConfig) *routerImpl {
	l := resolveLogger(config)
	if config.Logger != nil {
		setPkgLogger(l)
	}
	return &routerImpl{
		config:      config,
		log:         l,
		routes:      make(map[string][]route),
		groups:      make(map[string]Router),
		middlewares: make([]MiddlewareFunc, 0),
	}
}

var _ Router = (*routerImpl)(nil)
var _ RouteRegister = (*routerImpl)(nil)
var _ http.Handler = (*routerImpl)(nil)
