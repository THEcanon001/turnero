package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	tjwt "github.com/THEcanon001/turnero/pkg/jwt"
)

type authContextKey string

const (
	userIDKey     authContextKey = "user_id"
	providerIDKey authContextKey = "provider_id"
	roleKey       authContextKey = "role"
)

// Auth returns middleware that validates JWT access tokens.
func Auth(jwtManager *tjwt.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization format")
				return
			}

			claims, err := jwtManager.ValidateToken(parts[1])
			if err != nil {
				if strings.Contains(err.Error(), "expired") {
					writeAuthError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Token has expired")
					return
				}
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token")
				return
			}

			if claims.TokenType != "access" {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token type")
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, userIDKey, claims.UserID)
			ctx = context.WithValue(ctx, providerIDKey, claims.ProviderID)
			ctx = context.WithValue(ctx, roleKey, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID returns the authenticated user's ID from the context.
func GetUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(userIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// GetProviderID returns the authenticated user's provider ID from the context.
func GetProviderID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(providerIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// GetRole returns the authenticated user's role from the context.
func GetRole(ctx context.Context) string {
	if role, ok := ctx.Value(roleKey).(string); ok {
		return role
	}
	return ""
}

// RequireRole returns middleware that checks the user has the required role.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRole(r.Context())
			if !roleSet[role] {
				writeAuthError(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  code,
	})
}
