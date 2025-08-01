package simpleroute

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"slices"
	"strings"
)

var METHODS []string = []string{
	"GET",
	"POST",
	"PUT",
	"DELETE",
	"PATCH",
	"OPTION",
}

type RouterConfig struct {
	AssetDir  string
	AssetPath string
	FS        fs.FS
	UseProxy  bool
}

type Router interface {
	Get(path string, args ...any) Router
	Post(path string, args ...any) Router
	Put(path string, args ...any) Router
	Patch(path string, args ...any) Router
	Delete(path string, args ...any) Router
}

type RouteRegister interface {
	Router
	Group(path string, args ...any) Router
	Use(args ...any) RouteRegister
}

type MiddlewareFunc = func(http.Handler) http.Handler

type RouterAction = func(router Router) Router

type ContextKey string

type route struct {
	method      string
	pattern     string
	handler     http.Handler
	middlewares []MiddlewareFunc
}

type routerImpl struct {
	config        RouterConfig
	group         string
	groups        map[string]Router
	routes        map[string][]route
	middlewares   []MiddlewareFunc
	routeHandlers []route
	mux           *http.ServeMux
}

// Get implements Router.
func (r *routerImpl) Get(path string, args ...any) Router {
	return r.Handle("GET", path, args...)
}

// Delete implements Router.
func (r *routerImpl) Delete(path string, args ...any) Router {
	return r.Handle("DELETE", path, args...)
}

// Patch implements Router.
func (r *routerImpl) Patch(path string, args ...any) Router {
	return r.Handle("PATCH", path, args...)
}

// Post implements Router.
func (r *routerImpl) Post(path string, args ...any) Router {
	return r.Handle("POST", path, args...)
}

// Put implements Router.
func (r *routerImpl) Put(path string, args ...any) Router {
	return r.Handle("PUT", path, args...)
}

// Group implements RouteRegister
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

	router := &routerImpl{
		group:       path,
		routes:      make(map[string][]route),
		middlewares: middlewares,
	}

	r.groups[path] = callbackRoute(router)
	return r
}

// Use implements RouteRegister
func (r *routerImpl) Use(args ...any) RouteRegister {

	method := "GET"
	keys := make(map[string]bool)
	var pattern string
	var handler http.Handler
	var middlewares []MiddlewareFunc = make([]MiddlewareFunc, 0)

	key := fmt.Sprintf("%s-%s", method, pattern)

	for i := range args {

		switch arg := args[i].(type) {
		case HttpRouter:
			arg.Routes(r)
		case string:
			if slices.Contains(METHODS, strings.ToUpper(arg)) {
				method = strings.ToUpper(arg)
			}

			pattern = arg
		case http.Handler:
			handler = arg
		case MiddlewareFunc:
			middlewares = append(middlewares, arg)
		case []MiddlewareFunc:
			middlewares = append(middlewares, arg...)
		}
	}

	if method != "" && pattern != "" && handler != nil {
		if _, exists := keys[key]; !exists {
			keys[key] = true
			r.routes[r.group] = append(r.routes[r.group], route{
				method:      method,
				pattern:     pattern,
				handler:     handler,
				middlewares: middlewares,
			})
		}
	} else if len(middlewares) > 0 {
		r.middlewares = append(r.middlewares, middlewares...)
	}

	return r
}

// ServeHTTP implements http.Handler.
func (r *routerImpl) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	if r.mux == nil || len(r.routeHandlers) <= 0 {
		http.Error(res, "routes not initialize", http.StatusInternalServerError)
		return
	}

	mux := r.mux
	handlers := r.routeHandlers

	_, pattern := mux.Handler(req)
	patterns := strings.Split(pattern, " ")

	if len(patterns) == 2 {

		_, ok := r.matchPath(patterns[1], req.URL.Path)

		if !ok && !r.config.UseProxy {
			http.Error(res, "page not found", http.StatusNotFound)
			return
		}

	} else {
		matchError := 0
		for _, mh := range handlers {
			_, ok := r.matchPath(mh.pattern, req.URL.Path)

			if ok && mh.method != req.Method {
				http.Error(res, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			matchError++
		}

		if matchError == len(handlers) {
			http.Error(res, "page not found", http.StatusNotFound)
			return
		}

	}

	mux.ServeHTTP(res, req)
}

func (r *routerImpl) Handle(method, path string, args ...any) *routerImpl {
	params := make([]any, 0)
	params = append(params, method)
	params = append(params, path)
	params = append(params, args...)

	use := r.Use(params...)

	route, ok := use.(*routerImpl)

	if !ok {
		log.Fatal("invalid type")
	}

	return route
}

func (r *routerImpl) Build() *routerImpl {
	mux, handlers := r.setupRoutes()
	r.mux = mux
	r.routeHandlers = handlers
	return r
}

func (r *routerImpl) setupRoutes() (*http.ServeMux, []route) {

	mux := http.NewServeMux()

	if r.config.AssetPath != "" && r.config.AssetDir != "" {

		var fsHandler http.Handler = http.FileServer(http.Dir(r.config.AssetDir))

		if r.config.FS != nil {
			content, err := fs.Sub(r.config.FS, r.config.AssetDir)
			if err != nil {
				log.Fatalf("use embedded fs, error: %v", err)
			}

			fsHandler = http.FileServer(http.FS(content))
		}

		r.Get(r.config.AssetPath, http.StripPrefix(r.config.AssetPath, fsHandler))
	}

	routes := remap(r)

	for _, group := range r.groups {
		g, ok := group.(*routerImpl)
		if !ok {
			continue
		}

		groupRoutes := remap(g)
		routes = append(routes, groupRoutes...)
	}

	for _, route := range routes {
		var handlers []MiddlewareFunc = make([]MiddlewareFunc, 0)
		handlers = append(handlers, r.middlewares...)
		handlers = append(handlers, route.middlewares...)
		pattern := fmt.Sprintf("%s %s", route.method, route.pattern)

		mux.Handle(pattern, Handle(handlers, route.handler))
	}

	return mux, routes
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
		routes:      make(map[string][]route, 0),
		groups:      make(map[string]Router),
		middlewares: make([]MiddlewareFunc, 0),
	}
}

var _ Router = (*routerImpl)(nil)
var _ RouteRegister = (*routerImpl)(nil)
var _ http.Handler = (*routerImpl)(nil)
