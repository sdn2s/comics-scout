package middleware

import (
	"net/http"
	"strings"
)

type TokenVerifier interface {
	Verify(token string) error
}

func Auth(next http.HandlerFunc, verifier TokenVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Token "

		header := r.Header.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, prefix) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		token := strings.TrimSpace(header[len(prefix):])
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if err := verifier.Verify(token); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
