package middleware

import (
	"net/http"
)

func Concurrency(next http.HandlerFunc, limit int) http.HandlerFunc {
	if limit <= 0 {
		return next
	}

	sem := make(chan struct{}, limit)

	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			next(w, r)
		default:
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
	}
}
