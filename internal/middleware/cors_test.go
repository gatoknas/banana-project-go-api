package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	// A simple mock handler to verify if CORS passes the request to the next handler
	mockNext := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted) // Status 202
	})

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectNextCall bool
	}{
		{
			name:           "OPTIONS request preflight",
			method:         http.MethodOptions,
			expectedStatus: http.StatusOK,
			expectNextCall: false,
		},
		{
			name:           "GET request passes through",
			method:         http.MethodGet,
			expectedStatus: http.StatusAccepted,
			expectNextCall: true,
		},
		{
			name:           "POST request passes through",
			method:         http.MethodPost,
			expectedStatus: http.StatusAccepted,
			expectNextCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://localhost/test", nil)
			rr := httptest.NewRecorder()

			handler := CORS(mockNext)
			handler.ServeHTTP(rr, req)

			// Assert Status Code
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status code %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Assert CORS Headers
			headers := []string{
				"Access-Control-Allow-Origin",
				"Access-Control-Allow-Methods",
				"Access-Control-Allow-Headers",
			}

			for _, h := range headers {
				if rr.Header().Get(h) == "" {
					t.Errorf("expected header %s to be set", h)
				}
			}
		})
	}
}
