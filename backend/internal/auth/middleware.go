package auth

import (
	"context"
	"net/http"
	"strings"

	"darkhold/internal/httpx"
)

type ctxKey struct{}

// Middleware requires a valid "Authorization: Bearer <token>" header.
func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			token, found := strings.CutPrefix(h, "Bearer ")
			if !found || token == "" {
				httpx.WriteError(w, http.StatusUnauthorized, "missing token")
				return
			}
			uid, err := ParseToken(token, secret)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxKey{}).(string)
	return id, ok && id != ""
}