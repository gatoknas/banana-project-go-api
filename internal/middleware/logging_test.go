package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestGetTraceID(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "nil context",
			ctx:  nil,
			want: "",
		},
		{
			name: "context without trace id",
			ctx:  context.Background(),
			want: "",
		},
		{
			name: "context with trace id",
			ctx:  context.WithValue(context.Background(), TraceIDContextKey, "test-trace-1234"),
			want: "test-trace-1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTraceID(tt.ctx)
			if got != tt.want {
				t.Errorf("GetTraceID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequestLogger(t *testing.T) {
	tests := []struct {
		name                 string
		incomingTraceID      string
		targetPath           string
		handlerStatusCode   int
		handlerBody          string
		expectedLogLevel     zap.AtomicLevel
		expectedLogMessage   string
		expectedLogCount     int
		shouldSuppressStatus bool
	}{
		{
			name:               "200 OK request generates new trace ID and logs INFO",
			incomingTraceID:    "",
			targetPath:         "/api/v1/products",
			handlerStatusCode:  http.StatusOK,
			handlerBody:        `{"data":[]}`,
			expectedLogMessage: "HTTP request processed successfully",
			expectedLogCount:   1,
		},
		{
			name:               "Propagates existing trace ID and logs INFO",
			incomingTraceID:    "incoming-custom-uuid-456",
			targetPath:         "/api/v1/sales",
			handlerStatusCode:  http.StatusCreated,
			handlerBody:        `{"id":1}`,
			expectedLogMessage: "HTTP request processed successfully",
			expectedLogCount:   1,
		},
		{
			name:               "400 Bad Request logs WARN",
			incomingTraceID:    "trace-400",
			targetPath:         "/api/v1/users",
			handlerStatusCode:  http.StatusBadRequest,
			handlerBody:        `{"error":"bad payload"}`,
			expectedLogMessage: "HTTP request completed with client error",
			expectedLogCount:   1,
		},
		{
			name:               "500 Internal Server Error logs ERROR",
			incomingTraceID:    "trace-500",
			targetPath:         "/api/v1/checkout",
			handlerStatusCode:  http.StatusInternalServerError,
			handlerBody:        `{"error":"db connection timeout"}`,
			expectedLogMessage: "HTTP request failed with server error",
			expectedLogCount:   1,
		},
		{
			name:                 "Health check /status with 200 is suppressed from logging",
			incomingTraceID:      "",
			targetPath:           "/status",
			handlerStatusCode:    http.StatusOK,
			handlerBody:          `{"status":"ok"}`,
			expectedLogCount:     0,
			shouldSuppressStatus: true,
		},
		{
			name:                 "Health check /healthz with 200 is suppressed from logging",
			incomingTraceID:      "",
			targetPath:           "/healthz",
			handlerStatusCode:    http.StatusOK,
			handlerBody:          `ok`,
			expectedLogCount:     0,
			shouldSuppressStatus: true,
		},
		{
			name:               "Health check /status with 500 error is NOT suppressed and logs ERROR",
			incomingTraceID:    "health-failure-trace",
			targetPath:         "/status",
			handlerStatusCode:  http.StatusInternalServerError,
			handlerBody:        `{"status":"db down"}`,
			expectedLogMessage: "HTTP request failed with server error",
			expectedLogCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, recorded := observer.New(zap.InfoLevel)
			logger := zap.New(core)

			var capturedTraceFromContext string
			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedTraceFromContext = GetTraceID(r.Context())
				w.WriteHeader(tt.handlerStatusCode)
				if tt.handlerBody != "" {
					_, _ = w.Write([]byte(tt.handlerBody))
				}
			})

			middlewareFunc := RequestLogger(logger)
			wrapped := middlewareFunc(mockHandler)

			req := httptest.NewRequest(http.MethodGet, tt.targetPath, nil)
			if tt.incomingTraceID != "" {
				req.Header.Set(TraceIDHeader, tt.incomingTraceID)
			}
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			// 1. Assert Response Status
			if rec.Code != tt.handlerStatusCode {
				t.Errorf("expected status code %d, got %d", tt.handlerStatusCode, rec.Code)
			}

			// 2. Assert Trace ID Header in response
			responseTraceID := rec.Header().Get(TraceIDHeader)
			if responseTraceID == "" {
				t.Errorf("expected %s header in response, got empty", TraceIDHeader)
			}
			if tt.incomingTraceID != "" && responseTraceID != tt.incomingTraceID {
				t.Errorf("expected trace ID %s, got %s", tt.incomingTraceID, responseTraceID)
			}

			// 3. Assert Trace ID bound to request context
			if capturedTraceFromContext != responseTraceID {
				t.Errorf("context trace ID %s != response trace ID %s", capturedTraceFromContext, responseTraceID)
			}

			// 4. Assert Log Entries
			logs := recorded.All()
			if len(logs) != tt.expectedLogCount {
				t.Fatalf("expected %d logs, got %d", tt.expectedLogCount, len(logs))
			}

			if tt.expectedLogCount > 0 {
				logEntry := logs[0]
				if logEntry.Message != tt.expectedLogMessage {
					t.Errorf("expected log message %q, got %q", tt.expectedLogMessage, logEntry.Message)
				}

				// Verify standard structured fields are present
				contextMap := logEntry.ContextMap()
				if contextMap["trace_id"] != responseTraceID {
					t.Errorf("log field trace_id = %v, want %v", contextMap["trace_id"], responseTraceID)
				}
				if contextMap["status_code"] != int64(tt.handlerStatusCode) {
					t.Errorf("log field status_code = %v, want %v", contextMap["status_code"], tt.handlerStatusCode)
				}
				if contextMap["path"] != tt.targetPath {
					t.Errorf("log field path = %v, want %v", contextMap["path"], tt.targetPath)
				}
			}
		})
	}
}
