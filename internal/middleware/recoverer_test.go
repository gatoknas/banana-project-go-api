package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"org.banana.project/api/internal/middleware"
)

func TestRecoverer(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
	}{
		{
			name: "Normal handler executes without panic",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Panicking handler is recovered with 500",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("simulated fatal error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := middleware.Recoverer(logger)(tt.handler)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			rec.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Recoverer() status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
