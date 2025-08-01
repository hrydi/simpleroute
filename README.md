### Using embed fs
```golang
//go:embed static/*
var staticFS embed.FS
```

### Initiate mux simple route with static assets
```golang
server := simpleroute.NewHttp("0.0.0.0:17881")
router := simpleroute.NewRouter(simpleroute.RouterConfig{
    AssetPath: "/assets/",
    AssetDir:  "static",
    FS:        staticFS,
})
```

### Using as group routes (with implementation of simpleroute.HttpRouter)
```golang
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
```

### How to use
#### you could run make command for this example, for development use, run this command and then attach to each of running containers and start each services from their console :D
```bash
make compose-run
```

### Pre-Production build
#### Run this command to preview production build
```bash
make build
```