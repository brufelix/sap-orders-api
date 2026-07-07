package auth

import "net/http"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy(r.URL.Path))
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func contentSecurityPolicy(path string) string {
	if path == "/swagger" {
		return "default-src 'none'; " +
			"script-src 'self' 'unsafe-inline' https://unpkg.com; " +
			"style-src 'self' 'unsafe-inline' https://unpkg.com; " +
			"connect-src 'self'; " +
			"img-src 'self' data:; " +
			"frame-ancestors 'none'"
	}
	return "default-src 'none'; frame-ancestors 'none'"
}

func HTTPSRedirect(enabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if enabled && r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
				target := "https://" + r.Host + r.URL.RequestURI()
				http.Redirect(w, r, target, http.StatusPermanentRedirect)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
