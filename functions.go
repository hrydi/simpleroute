package simpleroute

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Param represents a single key-value path parameter.
type Param struct {
	Key   string
	Value string
}

func parseSegments(pattern string) []segment {
	segs := make([]segment, 0, 4)
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '/' {
			continue
		}
		start := i
		for i < len(pattern) && pattern[i] != '/' {
			i++
		}
		s := pattern[start:i]
		if len(s) >= 3 && s[0] == '{' && s[len(s)-1] == '}' {
			segs = append(segs, segment{isParam: true, val: s[1 : len(s)-1]})
		} else {
			segs = append(segs, segment{isParam: false, val: s})
		}
	}
	return segs
}

func matchRoute(segs []segment, path string) ([]Param, bool) {
	var params []Param
	ppi := 0
	ppl := len(path)

	for si := 0; si < len(segs); si++ {
		for ppi < ppl && path[ppi] == '/' {
			ppi++
		}
		if ppi >= ppl {
			return nil, false
		}

		uStart := ppi
		for ppi < ppl && path[ppi] != '/' {
			ppi++
		}
		uSeg := path[uStart:ppi]

		if segs[si].isParam {
			params = append(params, Param{segs[si].val, uSeg})
		} else if segs[si].val != uSeg {
			return nil, false
		}
	}

	for ppi < ppl && path[ppi] == '/' {
		ppi++
	}
	if ppi < ppl {
		return nil, false
	}

	return params, true
}

func chainMiddleware(slices ...[]MiddlewareFunc) []MiddlewareFunc {
	n := 0
	for _, s := range slices {
		n += len(s)
	}
	mws := make([]MiddlewareFunc, 0, n)
	for _, s := range slices {
		mws = append(mws, s...)
	}
	return mws
}

func matchPath(pattern, path string) ([]Param, bool) {
	pi := 0
	ppi := 0
	pl := len(pattern)
	ppl := len(path)

	var params []Param

	for {
		for pi < pl && pattern[pi] == '/' {
			pi++
		}
		for ppi < ppl && path[ppi] == '/' {
			ppi++
		}

		if pi >= pl && ppi >= ppl {
			return params, true
		}
		if pi >= pl || ppi >= ppl {
			return nil, false
		}

		pStart := pi
		for pi < pl && pattern[pi] != '/' {
			pi++
		}
		pSeg := pattern[pStart:pi]

		uStart := ppi
		for ppi < ppl && path[ppi] != '/' {
			ppi++
		}
		uSeg := path[uStart:ppi]

		if len(pSeg) >= 3 && pSeg[0] == '{' && pSeg[len(pSeg)-1] == '}' {
			params = append(params, Param{pSeg[1 : len(pSeg)-1], uSeg})
		} else if pSeg != uSeg {
			return nil, false
		}
	}
}

func existsInStatic(uri_path, asset_path, asset_dir string, embedFS fs.FS) bool {

	file_path := strings.ReplaceAll(uri_path, asset_path, "")
	fullPath := filepath.Join(asset_dir, file_path)

	if embedFS != nil {
		_, err := fs.Stat(embedFS, fullPath)
		return err == nil
	}

	_, err := os.Stat(fullPath)
	return err == nil
}

// Handle builds a middleware chain around the given handler.
// Middleware order: the first middleware in the slice is the outermost wrapper.
// If handler is nil, a default http.NewServeMux is used as the base.
func Handle(handlers []MiddlewareFunc, handler http.Handler) http.Handler {
	if handler == nil {
		handler = http.NewServeMux()
	}

	for i := range handlers {
		handler = handlers[len(handlers)-1-i](handler)
	}

	return handler
}
