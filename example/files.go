package main

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/hrydi/simpleroute"
)

//go:embed public
var publicFS embed.FS

// filesImpl demonstrates simpleroute's wildcard/catch-all path parameter
// ({name...}): a single route serves every file under public/, no matter
// how deeply nested, instead of one route per file.
type filesImpl struct {
	fsys  fs.FS
	index []byte // precomputed listFiles response body
}

func (f *filesImpl) Routes(r simpleroute.RouteRegister) {
	r.Get("/files/{path...}", http.HandlerFunc(f.serve))
}

func (f *filesImpl) serve(w http.ResponseWriter, r *http.Request) {
	path := simpleroute.URLParam(r, "path")
	if path == "" {
		f.listFiles(w)
		return
	}

	// fs.FS rejects paths containing ".." (fs.ValidPath), so this can't
	// escape the public/ directory regardless of what the client sends.
	if info, err := fs.Stat(f.fsys, path); err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	http.ServeFileFS(w, r, f.fsys, path)
}

// listFiles serves the index built once in NewFiles, so GET /files/ is
// useful on its own instead of a bare 404 without re-walking the tree
// (and doing per-request work an attacker could trigger for free) on every
// request.
func (f *filesImpl) listFiles(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(f.index)
}

// buildIndex walks fsys once and renders the listFiles response body.
// Safe to compute eagerly here because fsys is an embed.FS: its contents
// are fixed at compile time, so this never needs to be recomputed.
func buildIndex(fsys fs.FS) []byte {
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "Try one of these (all served by the same GET /files/{path...} route):")
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			fmt.Fprintf(&buf, "  GET /files/%s\n", path)
		}
		return nil
	})
	return buf.Bytes()
}

func NewFiles() *filesImpl {
	sub, err := fs.Sub(publicFS, "public")
	if err != nil {
		panic(err)
	}
	return &filesImpl{fsys: sub, index: buildIndex(sub)}
}

var _ simpleroute.HttpRouter = (*filesImpl)(nil)
