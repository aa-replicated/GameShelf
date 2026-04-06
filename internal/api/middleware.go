package api

import (
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
		if c, err := r.Cookie("admin_token"); err == nil && c.Value == secret {
			next.ServeHTTP(w, r)
			return
		}

		// Check query param
		if r.URL.Query().Get("token") == secret {
			next.ServeHTTP(w, r)
			return
		}

		// Check Authorization header
		if r.Header.Get("Authorization") == "Bearer "+secret {
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
