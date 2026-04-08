package api

import (
	"crypto/subtle"
	"log"
	"net/http"
)

// adminAuthMiddleware checks for a valid admin secret in:
//   - Cookie: admin_token=<secret>
//   - Query param: ?token=<secret>
//   - Authorization header: "Bearer <secret>"
func (s *Server) adminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := s.cfg.AdminSecret

		// Check cookie
		if c, err := r.Cookie("admin_token"); err == nil && subtle.ConstantTimeCompare([]byte(c.Value), []byte(secret)) == 1 {
			next.ServeHTTP(w, r)
			return
		}

		// Check query param
		if token := r.URL.Query().Get("token"); subtle.ConstantTimeCompare([]byte(token), []byte(secret)) == 1 {
			next.ServeHTTP(w, r)
			return
		}

		// Check Authorization header
		if token := r.Header.Get("Authorization"); subtle.ConstantTimeCompare([]byte(token), []byte("Bearer "+secret)) == 1 {
			next.ServeHTTP(w, r)
			return
		}

		// Not authenticated — show login form
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`<!DOCTYPE html><html><body>
<h2>Admin Login</h2>
<form method="GET" action="/admin">
  <input type="password" name="token" placeholder="Admin secret" />
  <button type="submit">Login</button>
</form></body></html>`))
	})
}

// sdkAdminGateMiddleware blocks access to admin routes when the
// admin_panel_enabled license field is explicitly set to "false".
// Fail-open: if SDK is unavailable or field is absent, access is allowed.
func (s *Server) sdkAdminGateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.sdk.IsFeatureEnabled(r.Context(), "admin_panel_enabled") {
			log.Printf("admin: access denied by license entitlement (admin_panel_enabled=false)")
			http.Error(w, "Admin panel disabled by license", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
