package middleware

import (
	"context"

	"github.com/google/uuid"
)

// WithTestAuth sets authentication values in the context for testing.
// This is the only way to inject auth context from outside the middleware package
// since the context key type is unexported.
func WithTestAuth(ctx context.Context, userID, providerID uuid.UUID, role string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, providerIDKey, providerID)
	ctx = context.WithValue(ctx, roleKey, role)
	return ctx
}
