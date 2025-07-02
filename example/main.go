package main

import (
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

	fs := http.FileServer(http.Dir("example/static"))
	
	server := sr.NewHttp("0.0.0.0:17881")
	router := sr.NewRouter(sr.RouterConfig{ Spa: true }) // SPA=true, for assets to be load
	router.Get("/assets/", http.StripPrefix("/assets/", fs))
	router.Use(NewUser())
	
	go server.Start(router)

	<-ctx.Done()

	if err := server.Stop(ctx); err != nil {
		log.Printf("stopped server %v", err)
	}
}