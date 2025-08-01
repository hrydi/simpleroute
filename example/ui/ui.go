package ui

import (
	"context"
	"embed"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/hrydi/simpleroute"
)

//go:embed vue/dist
var UIStaticFS embed.FS

type ui struct {
	proxy *httputil.ReverseProxy
}

// Routes implements simpleroute.HttpRouter.
func (u *ui) Routes(r simpleroute.RouteRegister) {
	r.Use("/", 
		func() http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				isProxy := simpleroute.ContextKey("isProxy")
				ctx := context.WithValue(r.Context(), isProxy, "true")
				req := r.WithContext(ctx)
				u.proxy.ServeHTTP(w, req)
			})
		}(),
	)
}

func New(url *url.URL) *ui {
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ModifyResponse = func(r *http.Response) error {
		r.Header.Set("X-Ui-Proxy", "true")
		return nil
	}
	return &ui{
		proxy: proxy,
	}
}

var _ simpleroute.HttpRouter = (*ui)(nil)
