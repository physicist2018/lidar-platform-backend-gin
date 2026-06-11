package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kshmirko/lidar-platform-go/internal/utils/auth"
)

type claimsCtxKey struct{}

func ClaimsToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
			return
		}

		claims, err := auth.ParseToken(secretFromCtx(r), parts[1])
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		ctx := context.WithValue(r.Context(), claimsCtxKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminOnly restricts to admin role.
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil || claims.Role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AdminOrManager allows both admin and manager roles.
func AdminOrManager(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
			return
		}
		if claims.Role != "admin" && claims.Role != "manager" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin or manager role required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware returns middleware that extracts JWT claims into context.
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
				return
			}

			claims, err := auth.ParseToken(secret, parts[1])
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
				return
			}

			ctx := context.WithValue(r.Context(), claimsCtxKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetClaims(r *http.Request) *auth.Claims {
	claims, _ := r.Context().Value(claimsCtxKey{}).(*auth.Claims)
	return claims
}

func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

// secretFromCtx retrieves the JWT secret stored in context by SecretMiddleware.
func secretFromCtx(r *http.Request) string {
	s, _ := r.Context().Value(secretCtxKey{}).(string)
	return s
}

type secretCtxKey struct{}

// SecretMiddleware stores the JWT secret in request context.
func SecretMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), secretCtxKey{}, secret)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
