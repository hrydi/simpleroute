package simpleroute

import (
	"fmt"
	"net/http"
)

// WriteError writes a plain-text error response with the given status code.
func WriteError(w http.ResponseWriter, code int, msg string) {
	http.Error(w, msg, code)
}

// Text writes a plain-text response with the given status code.
func Text(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprint(w, msg)
}
