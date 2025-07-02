package main

import (
	"fmt"
	"log"
	"net/http"

	sr "github.com/hrydi/simpleroute/pkg/http"
)

type userImpl struct{}

func helloMiddleware() sr.MiddlewareFunc {
	return func (next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("calling hello middleware")
			next.ServeHTTP(w, r)
		})
	}
}

// Routes implements http.HttpRouter.
func (u *userImpl) Routes(r sr.RouteRegister) {
	
	r.Group("/user", func(router sr.Router) sr.Router {
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
					fmt.Fprintf(w, "Profle")
				})
			}(),
		)
	})
}

func NewUser() *userImpl {
	return &userImpl{}
}

var _ sr.HttpRouter = (*userImpl)(nil)
