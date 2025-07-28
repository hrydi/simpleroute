package simpleroute

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func remap(mapRoute *routerImpl) []route {
	keys := make(map[string]bool)
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

			key := fmt.Sprintf("%s-%s", pattern, router.method)
			if _, exists := keys[key]; !exists {
				keys[key] = true
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
	return  err == nil
}

func Handle(handlers []MiddlewareFunc, handler http.Handler) http.Handler {
	if handler == nil {
		handler = http.NewServeMux()
	}

	for i := range handlers {
		handler = handlers[len(handlers) - 1 - i](handler)
	}

	return handler
}

func responseHandler(w http.ResponseWriter, code int, response HttpResponse) {
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

func SuccessResponseHandler(w http.ResponseWriter, code int, message string, data any) {
	responseHandler(w, code, HttpResponse{
		Message: message,
		Error:   false,
		Data:    data,
	})
}

func ErrorResponseHandler(w http.ResponseWriter, code int, message string) {
	responseHandler(w, code, HttpResponse{
		Message: message,
		Error:   true,
	})
}

func GetFullURL(r *http.Request) string {
	// Default to http or https based on TLS
	scheme := "http"
	if r.Header.Get("X-Forwarded-Proto") != "" {
		scheme = r.Header.Get("X-Forwarded-Proto")
	} else if r.TLS != nil {
		scheme = "https"
	}

	// Use X-Forwarded-Host if available, otherwise fallback to r.Host
	host := r.Host
	if xfHost := r.Header.Get("X-Forwarded-Host"); xfHost != "" {
		host = xfHost
	}

	// Combine to get full URL
	return fmt.Sprintf("%s://%s%s", scheme, host, r.RequestURI)
}