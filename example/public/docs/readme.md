# Wildcard route demo

This file lives at `public/docs/readme.md` and is reachable at
`GET /files/docs/readme.md`.

The route is registered once as:

```go
r.Get("/files/{path...}", handler)
```

`{path...}` captures everything after `/files/`, slashes included, so any
file under `public/` — no matter how deeply nested — is served without
registering a route per file.
