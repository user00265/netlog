package auth

import (
	"context"

	"netlog/internal/models"
)

type ctxKey int

const userKey ctxKey = iota

// WithUser returns a context carrying the authenticated user.
func WithUser(ctx context.Context, u models.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFrom extracts the authenticated user from the context, if present.
func UserFrom(ctx context.Context) (models.User, bool) {
	u, ok := ctx.Value(userKey).(models.User)
	return u, ok
}
