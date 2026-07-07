package middleware

import (
	"context"
	"net/http"
	"strings"

	"image-ai/backend/internal/httpx"
	"image-ai/backend/internal/security"
)

type contextKey string

const claimsKey contextKey = "claims"

func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, origin := range origins {
		allowed[strings.TrimSpace(origin)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowed["*"] || allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, err := security.BearerToken(r.Header.Get("Authorization"))
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "请先登录")
				return
			}
			claims, err := security.VerifyToken(secret, raw)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "登录状态已失效")
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (security.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(security.Claims)
	return claims, ok
}

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || claims.Role != "admin" {
			httpx.Error(w, http.StatusForbidden, "需要管理员权限")
			return
		}
		next.ServeHTTP(w, r)
	})
}
