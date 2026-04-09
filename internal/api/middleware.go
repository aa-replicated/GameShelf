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

// sdkAdminGateMiddleware checks the admin_panel_enabled license entitlement
// before any auth check. Fail-open: SDK unavailable or field absent = allow.
func (s *Server) sdkAdminGateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enabled := s.sdk.IsFeatureEnabled(r.Context(), "admin_panel_enabled")
		log.Printf("sdk: admin_panel_enabled check result: %v", enabled)
		if !enabled {
			http.Error(w, "This feature requires an upgraded license", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
