package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
)

type accessTokenPayload struct {
	jwt.RegisteredClaims
	Scope             string   `json:"scp"`
	Roles             []string `json:"roles"`
	Email             string   `json:"email"`
	PreferredUsername string   `json:"preferred_username"`
	UPN               string   `json:"upn"`
}

type Authenticator struct {
	issuer   string
	audience string
	keySet   oidc.KeySet
}

func NewAuthenticator(ctx context.Context, tenantID, audience string) (*Authenticator, error) {
	if audience == "" {
		return nil, fmt.Errorf("AZURE_AUDIENCE is required")
	}

	issuer := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("create oidc provider: %w", err)
	}

	return &Authenticator{
		issuer:   issuer,
		audience: audience,
		keySet:   provider.RemoteKeySet(),
	}, nil
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawToken, err := extractBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		claims, err := a.verifyAccessToken(r.Context(), rawToken)
		if err != nil {
			http.Error(w, "invalid or expired access token", http.StatusUnauthorized)
			return
		}

		ctx := withUserClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Authenticator) verifyAccessToken(ctx context.Context, rawToken string) (*UserClaims, error) {
	payload, err := a.keySet.VerifySignature(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("verify token signature: %w", err)
	}

	var tokenClaims accessTokenPayload
	if err := json.Unmarshal(payload, &tokenClaims); err != nil {
		return nil, fmt.Errorf("parse token claims: %w", err)
	}

	if err := validateRegisteredClaims(tokenClaims.RegisteredClaims, a.issuer, a.audience); err != nil {
		return nil, err
	}

	email := firstNonEmpty(tokenClaims.Email, tokenClaims.PreferredUsername, tokenClaims.UPN)
	if email == "" && tokenClaims.Subject != "" {
		email = tokenClaims.Subject
	}
	if email == "" {
		return nil, fmt.Errorf("token missing user identity")
	}

	return &UserClaims{
		Subject: tokenClaims.Subject,
		Email:   email,
		Roles:   tokenClaims.Roles,
		Scopes:  parseScopes(tokenClaims.Scope),
	}, nil
}

func validateRegisteredClaims(claims jwt.RegisteredClaims, issuer, audience string) error {
	if !claims.VerifyIssuer(issuer, true) {
		return fmt.Errorf("invalid issuer")
	}
	if !claims.VerifyAudience(audience) {
		return fmt.Errorf("invalid audience")
	}
	if !claims.VerifyExpiresAt(time.Now(), true) {
		return fmt.Errorf("token expired")
	}
	return nil
}

func parseScopes(scope string) []string {
	if strings.TrimSpace(scope) == "" {
		return nil
	}
	return strings.Fields(scope)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
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
