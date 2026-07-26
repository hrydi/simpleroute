package simpleroute

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func remap(mapRoute *routerImpl) []route {
	res := make([]route, 0)

	mapRoutes := mapRoute.routes
	for name, routes := range mapRoutes {
		for _, router := range routes {
			pattern := router.pattern
			if name != "" {
				if pattern == "/" {
					pattern = ""
				}

				pattern = fmt.Sprintf("%s%s", name, pattern)
			}

			middlewares := make([]MiddlewareFunc, 0)
			middlewares = append(middlewares, mapRoute.middlewares...)
			middlewares = append(middlewares, router.middlewares...)
			res = append(res, route{
				method:      router.method,
				pattern:     pattern,
				handler:     router.handler,
				middlewares: middlewares,
			})
		}
	}

	return res
}

func matchPath(pattern, path string) (map[string]string, bool) {
	pSegs := strings.Split(strings.Trim(pattern, "/"), "/")
	pathSegs := strings.Split(strings.Trim(path, "/"), "/")

	if len(pSegs) != len(pathSegs) {
		return nil, false
	}

	params := make(map[string]string)
	for i := range pSegs {
		if strings.HasPrefix(pSegs[i], "{") && strings.HasSuffix(pSegs[i], "}") {
			key := pSegs[i][1 : len(pSegs[i])-1]
			params[key] = pathSegs[i]
		} else if pSegs[i] != pathSegs[i] {
			return nil, false
		}
	}
	return params, true
}

func existsInStatic(uri_path, asset_path, asset_dir string, embedFS fs.FS) bool {

	file_path := strings.ReplaceAll(uri_path, asset_path, "")
	fullPath := filepath.Join(asset_dir, file_path)

	if embedFS != nil {
		_, err := fs.Stat(embedFS, fullPath)
		return err == nil
	}

	_, err := os.Stat(fullPath)
	return err == nil
}

// Handle builds a middleware chain around the given handler.
// Middleware order: the first middleware in the slice is the outermost wrapper.
// If handler is nil, a default http.NewServeMux is used as the base.
func Handle(handlers []MiddlewareFunc, handler http.Handler) http.Handler {
	if handler == nil {
		handler = http.NewServeMux()
	}

	for i := range handlers {
		handler = handlers[len(handlers)-1-i](handler)
	}

	return handler
}
