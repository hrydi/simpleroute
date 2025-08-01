package main

import (
	// "embed"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"fmt"

	"github.com/hrydi/simpleroute"
	"github.com/hrydi/simpleroute/example/ui"
	"github.com/hrydi/simpleroute/pkg/signal"
)

// //go:embed static/*
// var staticFS embed.FS

func main() {
	sigCh := signal.HandleSignals(os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	ctx, _ := signal.CreateContext(sigCh)

	server := simpleroute.NewHttp(fmt.Sprintf("0.0.0.0:%s", os.Getenv("HTTP_PORT")))

	routerConfig := simpleroute.RouterConfig{
		AssetPath: "/assets/",
		AssetDir:  "vue/dist/assets",
		FS:        ui.UIStaticFS,
	}

	if os.Getenv("DEV") == "development" {
		routerConfig = simpleroute.RouterConfig{UseProxy: true}
	}

	router := simpleroute.NewRouter(routerConfig)
	
	if os.Getenv("DEV") == "development" {
		viteUrl, err := url.Parse("http://hrydi-simple-ui:6668")
		if err == nil {
			router.Use(ui.New(viteUrl))
		}
	}else{
		router.Use("/", func() http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				staticFS, err := fs.Sub(ui.UIStaticFS, "vue/dist")
				if err != nil {
					http.NotFound(w, r)
					return
				}

				http.ServeFileFS(w, r, staticFS, "/index.html")
			})
		}())
	}
	

	router.Use(NewUser())
	
	go server.Start(router.Build())

	<-ctx.Done()

	if err := server.Stop(ctx); err != nil {
		log.Printf("error stopping server %v", err)
	}
}
