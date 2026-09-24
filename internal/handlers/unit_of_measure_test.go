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

type MockUnitRepo struct {
	ListFunc    func(ctx context.Context) ([]models.UnitOfMeasure, error)
	GetByIDFunc func(ctx context.Context, id int64) (*models.UnitOfMeasure, error)
}

func (m *MockUnitRepo) List(ctx context.Context) ([]models.UnitOfMeasure, error) {
	return m.ListFunc(ctx)
}

func (m *MockUnitRepo) GetByID(ctx context.Context, id int64) (*models.UnitOfMeasure, error) {
	return m.GetByIDFunc(ctx, id)
}

func TestUnitOfMeasureHandler_List(t *testing.T) {
	logger := zap.NewNop()
	now := time.Now()

	tests := []struct {
		name           string
		mockRepo       *MockUnitRepo
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "Success Returns Units",
			mockRepo: &MockUnitRepo{
				ListFunc: func(ctx context.Context) ([]models.UnitOfMeasure, error) {
					return []models.UnitOfMeasure{
						{ID: 1, Name: "Bulto", Abbreviation: "bl", CreatedAt: now},
						{ID: 2, Name: "Kilo", Abbreviation: "kg", CreatedAt: now},
						{ID: 3, Name: "Unidad", Abbreviation: "und", CreatedAt: now},
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name: "Internal Server Error",
			mockRepo: &MockUnitRepo{
				ListFunc: func(ctx context.Context) ([]models.UnitOfMeasure, error) {
					return nil, errors.New("database failure")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewUnitOfMeasureService(tt.mockRepo)
			handler := handlers.NewUnitOfMeasureHandler(svc, logger)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/units-of-measure", nil)
			w := httptest.NewRecorder()

			handler.List(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp []models.UnitOfMeasure
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
