package simpleroute

import (
	"net/http"
	"strconv"
)

// Query returns the first value of the named query parameter, or empty string.
func Query(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// QueryInt returns the first value of the named query parameter as int,
// or the default value if the parameter is missing or not a valid integer.
func QueryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

// QueryFloat returns the first value of the named query parameter as float64,
// or the default value if the parameter is missing or not a valid number.
func QueryFloat(r *http.Request, key string, defaultVal float64) float64 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return defaultVal
	}
	return n
}

// QueryBool returns the first value of the named query parameter as bool.
// Accepts "1", "t", "T", "true", "TRUE", "True" as true.
// Returns the default value if the parameter is missing.
func QueryBool(r *http.Request, key string, defaultVal bool) bool {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}
