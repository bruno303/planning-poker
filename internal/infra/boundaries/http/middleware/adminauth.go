package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type AdminMiddleware struct {
	apiKey string
}

func NewAdminMiddleware(apiKey string) AdminMiddleware {
	return AdminMiddleware{
		apiKey: apiKey,
	}
}

func (m *AdminMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")

		apiKey := strings.TrimPrefix(authorization, "Bearer ")

		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(m.apiKey)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
