package auth

import "net/http"

func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := UserClaimsFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if !claims.HasScope(scope) {
				http.Error(w, "forbidden: missing required scope", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
