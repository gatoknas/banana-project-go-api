package auth

import "context"

type AyuramiContextKey string

const UserClaimsKey AyuramiContextKey = "ayuramiUserClaims"

// WithAyuramiUser injects user claims into the context
func WithAyuramiUser(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, UserClaimsKey, claims)
}

// GetAyuramiClaims extracts user claims from the context
func GetAyuramiClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(UserClaimsKey).(*Claims)
	return claims, ok
}
