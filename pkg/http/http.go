package http

import (
	"context"
	"log"
	"net/http"
)

type HttpServer interface {
	Start(router http.Handler)
	Stop(ctx context.Context) error
}

type HttpRouter interface {
	Routes(r RouteRegister)
}

type HttpResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type httpServerImpl struct {
	app    http.Server
}

// Start implements HttpServer.
func (h *httpServerImpl) Start(router http.Handler) {
	h.app.Handler = RecoverMiddleware(router)
	if err := h.app.ListenAndServe(); err != nil {
		log.Println("http listen and serve error", err)
	}
}

// Stop implements HttpServer.
func (h *httpServerImpl) Stop(ctx context.Context) error {
	return h.app.Shutdown(ctx)
}

func NewHttp(address string) *httpServerImpl {
	return &httpServerImpl{
		app: http.Server{
			Addr: address,
		},
	}
}

var _ HttpServer = (*httpServerImpl)(nil)
