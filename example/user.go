package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/hrydi/simpleroute"
)

type userImpl struct{}

func helloMiddleware() simpleroute.MiddlewareFunc {
	return func (next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("calling hello middleware")
			next.ServeHTTP(w, r)
		})
	}
}

// Routes implements http.HttpRouter.
func (u *userImpl) Routes(r simpleroute.RouteRegister) {
	
	r.Group("/user", func(router simpleroute.Router) simpleroute.Router {
		return router.
		Get("/", 
			func() http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprintf(w, "Hellooo")
				})
			}(),
			helloMiddleware(),
		).
		Get("/profile", 
			func() http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					json.NewEncoder(w).Encode(map[string]any{
						"name": "Username",
						"age": 19,
					})
				})
			}(),
		)
	})
}

func NewUser() *userImpl {
	return &userImpl{}
}

var _ simpleroute.HttpRouter = (*userImpl)(nil)
