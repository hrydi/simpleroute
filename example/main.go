package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"

	sr "github.com/hrydi/simpleroute/pkg/http"
	"github.com/hrydi/simpleroute/pkg/signal"
)

//go:embed static/*
var staticFS embed.FS

func main() {
	sigCh := signal.HandleSignals(os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	ctx, _ := signal.CreateContext(sigCh)

	server := sr.NewHttp("0.0.0.0:17881")
	router := sr.NewRouter(sr.RouterConfig{
		AssetPath: "/assets/",
		AssetDir:  "static",
		FS:        staticFS,
	})

	router.Use("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
		<html>
			<head>
				<title>Index page</title>
			</head>
			<body>
				<p>Welcome (: </p>
			<script src="/assets/index.js"></script>
			</body>
		</html>`)
	}))
	router.Use(NewUser())
	
	go server.Start(router.Build())

	<-ctx.Done()

	if err := server.Stop(ctx); err != nil {
		log.Printf("error stopping server %v", err)
	}
}
