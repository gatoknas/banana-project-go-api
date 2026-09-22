package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/auth"
	"org.banana.project/api/internal/handlers"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

func TestAuthHandler_Login(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-12345")
	logger := zap.NewNop()
	hashedPassword, _ := auth.HashPassword("correctpass")
	now := time.Now()

	tests := []struct {
		name           string
		body           string
		mockRepo       *MockUserRepo
		expectedStatus int
	}{
		{
			name: "Success Login",
			body: `{"username":"admin","password":"correctpass"}`,
			mockRepo: &MockUserRepo{
				GetByUsernameFunc: func(ctx context.Context, username string) (*models.User, error) {
					return &models.User{
						ID:           1,
						Username:     "admin",
						PasswordHash: string(hashedPassword),
						Role:         "admin",
						IsActive:     true,
						CreatedAt:    now,
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid Password",
			body: `{"username":"admin","password":"wrongpass"}`,
			mockRepo: &MockUserRepo{
				GetByUsernameFunc: func(ctx context.Context, username string) (*models.User, error) {
					return &models.User{
						ID:           1,
						Username:     "admin",
						PasswordHash: string(hashedPassword),
						Role:         "admin",
						IsActive:     true,
					}, nil
				},
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "User Inactive",
			body: `{"username":"admin","password":"correctpass"}`,
			mockRepo: &MockUserRepo{
				GetByUsernameFunc: func(ctx context.Context, username string) (*models.User, error) {
					return &models.User{
						ID:           1,
						Username:     "admin",
						PasswordHash: string(hashedPassword),
						Role:         "admin",
						IsActive:     false,
					}, nil
				},
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "User Not Found",
			body: `{"username":"nonexistent","password":"password"}`,
			mockRepo: &MockUserRepo{
				GetByUsernameFunc: func(ctx context.Context, username string) (*models.User, error) {
					return nil, sql.ErrNoRows
				},
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid JSON payload",
			body:           `{invalid-json}`,
			mockRepo:       &MockUserRepo{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewUserService(tt.mockRepo, nil)
			handler := handlers.NewAuthHandler(svc, logger)

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Login(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp handlers.LoginResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode login response: %v", err)
				}
				if resp.Token == "" || resp.RefreshToken == "" {
					t.Errorf("expected non-empty tokens, got token=%q, refreshToken=%q", resp.Token, resp.RefreshToken)
				}
			}
		})
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-12345")
	logger := zap.NewNop()
	validRefresh, _ := auth.GenerateRefreshToken(1, "admin", "admin")

	tests := []struct {
		name           string
		body           string
		mockRepo       *MockUserRepo
		expectedStatus int
	}{
		{
			name: "Valid Refresh Token",
			body: `{"refreshToken":"` + validRefresh + `"}`,
			mockRepo: &MockUserRepo{
				GetByUsernameFunc: func(ctx context.Context, username string) (*models.User, error) {
					return &models.User{
						ID:       1,
						Username: "admin",
						Role:     "admin",
						IsActive: true,
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid Refresh Token",
			body:           `{"refreshToken":"invalid.token.here"}`,
			mockRepo:       &MockUserRepo{},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Malformed JSON",
			body:           `{bad-json}`,
			mockRepo:       &MockUserRepo{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewUserService(tt.mockRepo, nil)
			handler := handlers.NewAuthHandler(svc, logger)

			req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Refresh(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
