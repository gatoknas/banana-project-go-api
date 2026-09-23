package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"org.banana.project/api/internal/handlers"
)

func TestHelloHandler_Hello(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedType   string
		bodyContains   string
	}{
		{
			name:           "GET returns the hello html page",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedType:   "text/html; charset=utf-8",
			bodyContains:   "Hecho en Colombia",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.NewHelloHandler(zap.NewNop())
			req := httptest.NewRequest(tt.method, "/hello", nil)
			w := httptest.NewRecorder()

			h.Hello(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if ct := w.Header().Get("Content-Type"); ct != tt.expectedType {
				t.Errorf("expected Content-Type %q, got %q", tt.expectedType, ct)
			}

			if !strings.Contains(w.Body.String(), tt.bodyContains) {
				t.Errorf("expected body to contain %q", tt.bodyContains)
			}
		})
	}
}

func TestHelloHandler_Logo(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "GET returns the embedded png logo",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedType:   "image/png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.NewHelloHandler(zap.NewNop())
			req := httptest.NewRequest(tt.method, "/hello/logo.png", nil)
			w := httptest.NewRecorder()

			h.Logo(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if ct := w.Header().Get("Content-Type"); ct != tt.expectedType {
				t.Errorf("expected Content-Type %q, got %q", tt.expectedType, ct)
			}

			if w.Body.Len() == 0 {
				t.Error("expected a non-empty logo body")
			}
		})
	}
}
