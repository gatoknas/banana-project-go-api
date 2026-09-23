package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/handlers"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type MockCategoryRepo struct {
	ListFunc    func(ctx context.Context) ([]models.Category, error)
	GetByIDFunc func(ctx context.Context, id int64) (*models.Category, error)
}

func (m *MockCategoryRepo) List(ctx context.Context) ([]models.Category, error) {
	return m.ListFunc(ctx)
}

func (m *MockCategoryRepo) GetByID(ctx context.Context, id int64) (*models.Category, error) {
	return m.GetByIDFunc(ctx, id)
}

func TestCategoryHandler_List(t *testing.T) {
	logger := zap.NewNop()
	now := time.Now()

	tests := []struct {
		name           string
		mockRepo       *MockCategoryRepo
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "Success Returns Categories",
			mockRepo: &MockCategoryRepo{
				ListFunc: func(ctx context.Context) ([]models.Category, error) {
					return []models.Category{
						{ID: 1, Name: "Pasabocas y Golosinas", CreatedAt: now},
						{ID: 9, Name: "Cafeteria", CreatedAt: now},
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name: "Internal Server Error",
			mockRepo: &MockCategoryRepo{
				ListFunc: func(ctx context.Context) ([]models.Category, error) {
					return nil, errors.New("db error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewCategoryService(tt.mockRepo)
			handler := handlers.NewCategoryHandler(svc, logger)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
			w := httptest.NewRecorder()

			handler.List(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp []models.Category
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(resp) != tt.expectedCount {
					t.Fatalf("expected %d items, got %d", tt.expectedCount, len(resp))
				}
			}
		})
	}
}
