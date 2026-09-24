package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

// Recoverer catches any unhandled panics in HTTP handlers, logs the stack trace with Zap,
// and returns a clean HTTP 500 response, preventing socket hang-ups.
func Recoverer(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered in HTTP handler",
						zap.Any("error", rec),
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
						zap.String("stack", string(debug.Stack())),
					)
					http.Error(w, fmt.Sprintf("Internal server error: %v", rec), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
