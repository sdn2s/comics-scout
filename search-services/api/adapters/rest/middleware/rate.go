package middleware

import (
	"net/http"
	"time"
)

func Rate(next http.HandlerFunc, rps int) http.HandlerFunc {
	if rps <= 0 {
		return next
	}

	interval := time.Second / time.Duration(rps)
	if interval <= 0 {
		interval = time.Second
	}

	ticker := time.NewTicker(interval)

	return func(w http.ResponseWriter, r *http.Request) {
		<-ticker.C
		next(w, r)
	}
}
