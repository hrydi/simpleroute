package main

import (
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"fmt"

	"github.com/hrydi/simpleroute"
	"github.com/hrydi/simpleroute/example/ui"
	"github.com/hrydi/simpleroute/pkg/signal"
)

func loadEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		if k != "" && os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

type appLogger struct{}

func (appLogger) Errorf(format string, args ...any) { log.Printf("[APP] [ERROR] "+format, args...) }
func (appLogger) Warnf(format string, args ...any)  { log.Printf("[APP] [WARN]  "+format, args...) }
func (appLogger) Infof(format string, args ...any)  { log.Printf("[APP] [INFO]  "+format, args...) }
func (appLogger) Debugf(format string, args ...any) { log.Printf("[APP] [DEBUG] "+format, args...) }

func main() {
	loadEnv(".env")

	sigCh := signal.HandleSignals(os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	ctx, _ := signal.CreateContext(sigCh)

	server := simpleroute.NewHttp(simpleroute.ServerConfig{
		Addr: fmt.Sprintf("0.0.0.0:%s", os.Getenv("HTTP_PORT")),
	})

	router := simpleroute.NewRouter(simpleroute.RouterConfig{
		AssetPath: "/assets/",
		AssetDir:  "vue/dist/assets",
		FS:        ui.UIStaticFS,
		Logger:    appLogger{},
		LogLevel:  simpleroute.LogLevelDebug,
	})

	if os.Getenv("DEV") == "development" {
		host := os.Getenv("VITE_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("VITE_PORT")
		if port == "" {
			port = "5566"
		}
		viteUrl, err := url.Parse(fmt.Sprintf("http://%s:%s", host, port))
		if err == nil {
			router.Use(ui.New(viteUrl))
		}
	} else {
		spa := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			staticFS, err := fs.Sub(ui.UIStaticFS, "vue/dist")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			http.ServeFileFS(w, r, staticFS, "/index.html")
	})

	router.Use(NewUser())
		router.Use("/", spa)
	}

	router.Use(NewUser())

	if err := router.Build(); err != nil {
		log.Fatalf("router build error: %v", err)
	}

	go func() {
		if err := server.Start(router); err != nil {
			log.Fatalf("server start error: %v", err)
		}
	}()

	<-ctx.Done()

	if err := server.Stop(ctx); err != nil {
		log.Printf("error stopping server %v", err)
	}
}
