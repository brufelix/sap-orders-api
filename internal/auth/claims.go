package auth

import (
	"context"
	"slices"
)

type contextKey string

const userClaimsKey contextKey = "userClaims"

// UserClaims holds validated access token identity and permissions.
type UserClaims struct {
	Subject string
	Email   string
	Roles   []string
	Scopes  []string
}

func (c *UserClaims) HasScope(scope string) bool {
	if slices.Contains(c.Scopes, scope) {
		return true
	}

	for _, role := range c.Roles {
		for _, mapped := range roleScopes[role] {
			if mapped == scope {
				return true
			}
		}
	}

	return false
}

func withUserClaims(ctx context.Context, claims *UserClaims) context.Context {
	return context.WithValue(ctx, userClaimsKey, claims)
}

func UserClaimsFromContext(ctx context.Context) (*UserClaims, bool) {
	claims, ok := ctx.Value(userClaimsKey).(*UserClaims)
	return claims, ok
}

func UserEmailFromContext(ctx context.Context) string {
	claims, ok := UserClaimsFromContext(ctx)
	if !ok {
		return ""
	}
	return claims.Email
}
