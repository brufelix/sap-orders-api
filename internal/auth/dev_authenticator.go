package auth

import "net/http"

// DevAuthenticator allows local development without Microsoft Entra ID.
// Enable only with ENV=development and DEV_BYPASS_AUTH=true.
type DevAuthenticator struct {
	email string
}

func NewDevAuthenticator(email string) *DevAuthenticator {
	return &DevAuthenticator{email: email}
}

func (a *DevAuthenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &UserClaims{
			Subject: a.email,
			Email:   a.email,
			Scopes:  []string{ScopeOrdersRead, ScopeOrdersWrite},
			Roles:   []string{RoleOrdersAdmin},
		}
		ctx := withUserClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
