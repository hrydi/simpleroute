package simpleroute

import (
	"context"
	"encoding/json"
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

func Params(r *http.Request) map[string]string {
	params, _ := r.Context().Value(ParamsContextKey).(map[string]string)
	return params
}

func JSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

type httpServerImpl struct {
	app    http.Server
}

func (h *httpServerImpl) Start(router http.Handler) {
	h.app.Handler = RecoverMiddleware(router)
	if err := h.app.ListenAndServe(); err != nil {
		log.Println("http listen and serve error", err)
	}
}

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
