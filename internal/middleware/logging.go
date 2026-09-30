package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const (
	TraceIDHeader                = "X-Trace-ID"
	TraceIDContextKey contextKey = "trace_id"
)

// responseWriterInterceptor intercepts WriteHeader and Write to record status code and response size.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func newResponseWriterInterceptor(w http.ResponseWriter) *responseWriterInterceptor {
	return &responseWriterInterceptor{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status code if WriteHeader is not explicitly called
	}
}

func (rwi *responseWriterInterceptor) WriteHeader(code int) {
	rwi.statusCode = code
	rwi.ResponseWriter.WriteHeader(code)
}

func (rwi *responseWriterInterceptor) Write(b []byte) (int, error) {
	n, err := rwi.ResponseWriter.Write(b)
	rwi.bytesWritten += int64(n)
	return n, err
}

// GetTraceID extracts the trace ID from the given context, or returns an empty string.
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(TraceIDContextKey).(string); ok {
		return traceID
	}
	return ""
}

// RequestLogger returns an HTTP middleware that extracts or generates an X-Trace-ID,
// injects it into request context and response headers, measures execution latency,
// and logs request details via Zap in accordance with our Grafana / Loki observability standard.
func RequestLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Extract or generate Trace ID
			traceID := strings.TrimSpace(r.Header.Get(TraceIDHeader))
			if traceID == "" {
				traceID = uuid.NewString()
			}

			// Propagate trace ID in response headers
			w.Header().Set(TraceIDHeader, traceID)

			// Bind trace ID into request context
			ctx := context.WithValue(r.Context(), TraceIDContextKey, traceID)
			r = r.WithContext(ctx)

			// Intercept response to capture status code and size
			interceptor := newResponseWriterInterceptor(w)

			// Execute handler chain
			next.ServeHTTP(interceptor, r)

			duration := time.Since(start)
			path := r.URL.Path

			// Quiet health check endpoints from spamming production Loki logs unless they return an error
			if (path == "/status" || path == "/healthz") && interceptor.statusCode < 400 {
				return
			}

			fields := []zap.Field{
				zap.String("trace_id", traceID),
				zap.String("method", r.Method),
				zap.String("path", path),
				zap.Int("status_code", interceptor.statusCode),
				zap.Int64("duration_ms", duration.Milliseconds()),
				zap.Int64("bytes_written", interceptor.bytesWritten),
				zap.String("client_ip", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
			}

			// Map HTTP status code to structured log levels
			switch {
			case interceptor.statusCode >= 500:
				logger.Error("HTTP request failed with server error", fields...)
			case interceptor.statusCode >= 400:
				logger.Warn("HTTP request completed with client error", fields...)
			default:
				logger.Info("HTTP request processed successfully", fields...)
			}
		})
	}
}
