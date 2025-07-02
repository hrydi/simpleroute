package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"

	sr "github.com/hrydi/simpleroute/pkg/http"
	"github.com/hrydi/simpleroute/pkg/signal"
)

func main() {
	sigCh := signal.HandleSignals(os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	ctx, _ := signal.CreateContext(sigCh)

	server := sr.NewHttp("0.0.0.0:17881")
	router := sr.NewRouter(sr.RouterConfig{
		AssetPath: "/assets",
		AssetDir: "example/static",
	})
	
	router.Use("/", func() http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "index page")
		})
	}())
	router.Use(NewUser())
	
	go server.Start(router)

	<-ctx.Done()

	if err := server.Stop(ctx); err != nil {
		log.Printf("error stopping server %v", err)
	}
}