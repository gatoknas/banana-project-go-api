package middleware

import (
	"net/http"

	"org.banana.project/api/internal/auth"
)

// RequireRoles returns a middleware that checks if the request's authenticated user
// has one of the specified roles. If not, it returns 403 Forbidden.
func RequireRoles(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.GetAyuramiClaims(r.Context())
			if !ok {
				http.Error(w, "Forbidden: authentication context missing", http.StatusForbidden)
				return
			}

			roleAllowed := false
			for _, role := range allowedRoles {
				if claims.Role == role {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
