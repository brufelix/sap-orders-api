package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type contextKey string

const UserEmailKey contextKey = "userEmail"

type Authenticator struct {
	verifier *oidc.IDTokenVerifier
}

func NewAuthenticator(ctx context.Context, tenantID, clientID, audience string) (*Authenticator, error) {
	issuer := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("create oidc provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: audience,
	})

	return &Authenticator{verifier: verifier}, nil
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := extractBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		idToken, err := a.verifier.Verify(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		var claims struct {
			Email string `json:"email"`
			UPN   string `json:"preferred_username"`
		}
		if err := idToken.Claims(&claims); err != nil {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		email := claims.Email
		if email == "" {
			email = claims.UPN
		}
		if email == "" {
			http.Error(w, "token missing user identity", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserEmailKey, email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserEmailFromContext(ctx context.Context) string {
	email, _ := ctx.Value(UserEmailKey).(string)
	return email
}

func extractBearerToken(header string) (string, error) {
	if header == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", fmt.Errorf("invalid authorization header")
	}

	return parts[1], nil
}
