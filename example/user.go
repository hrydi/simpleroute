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
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("calling hello middleware")
			next.ServeHTTP(w, r)
		})
	}
}

func (u *userImpl) Routes(r simpleroute.RouteRegister) {
	r.Group("/user", func(router simpleroute.Router) simpleroute.Router {
		return router.
			Get("/", helloMiddleware(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "Hellooo")
			})).
			Get("/profile", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				log := simpleroute.GetLogger()
				log.Infof("user profile page")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{
					"name": "Username",
					"age":  19,
				})
			})).
			Get("/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				params := simpleroute.Params(r)
				fmt.Fprintf(w, "user id: %s", params["id"])
			}))
	})
}

func NewUser() *userImpl {
	return &userImpl{}
}

var _ simpleroute.HttpRouter = (*userImpl)(nil)
