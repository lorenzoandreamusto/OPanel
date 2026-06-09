package middleware

import (
	"context"
	"net/http"
	"strings"

	opaneljwt "opanel/internal/jwt"
)

type contextKey string

const ClaimsKey contextKey = "claims"

func Auth(secret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := ""

		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			}
		}

		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if token == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		claims, err := opaneljwt.ValidateToken(token, secret)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func GetClaims(r *http.Request) *opaneljwt.Claims {
	claims, _ := r.Context().Value(ClaimsKey).(*opaneljwt.Claims)
	return claims
}

func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil || claims.Role != "admin" {
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}
