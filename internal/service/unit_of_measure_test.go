package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type MockUnitOfMeasureRepository struct {
	ListFunc    func(ctx context.Context) ([]models.UnitOfMeasure, error)
	GetByIDFunc func(ctx context.Context, id int64) (*models.UnitOfMeasure, error)
}

func (m *MockUnitOfMeasureRepository) List(ctx context.Context) ([]models.UnitOfMeasure, error) {
	return m.ListFunc(ctx)
}

func (m *MockUnitOfMeasureRepository) GetByID(ctx context.Context, id int64) (*models.UnitOfMeasure, error) {
	return m.GetByIDFunc(ctx, id)
}

func TestListUnits(t *testing.T) {
	now := time.Now()
	sampleUnits := []models.UnitOfMeasure{
		{ID: 1, Name: "Unidad", Abbreviation: "und", CreatedAt: now},
		{ID: 2, Name: "Kilo", Abbreviation: "kg", CreatedAt: now},
		{ID: 3, Name: "Bulto", Abbreviation: "bl", CreatedAt: now},
	}

	tests := []struct {
		name        string
		mockSetup   func(m *MockUnitOfMeasureRepository)
		expectedLen int
		expectErr   bool
	}{
		{
			name: "Success with units",
			mockSetup: func(m *MockUnitOfMeasureRepository) {
				m.ListFunc = func(ctx context.Context) ([]models.UnitOfMeasure, error) {
					return sampleUnits, nil
				}
			},
			expectedLen: 3,
			expectErr:   false,
		},
		{
			name: "Success with empty list",
			mockSetup: func(m *MockUnitOfMeasureRepository) {
				m.ListFunc = func(ctx context.Context) ([]models.UnitOfMeasure, error) {
					return []models.UnitOfMeasure{}, nil
				}
			},
			expectedLen: 0,
			expectErr:   false,
		},
		{
			name: "Repository error",
			mockSetup: func(m *MockUnitOfMeasureRepository) {
				m.ListFunc = func(ctx context.Context) ([]models.UnitOfMeasure, error) {
					return nil, errors.New("database connection error")
				}
			},
			expectedLen: 0,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockUnitOfMeasureRepository{}
			tt.mockSetup(mockRepo)

			svc := service.NewUnitOfMeasureService(mockRepo)
			units, err := svc.ListUnits(context.Background())

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && len(units) != tt.expectedLen {
				t.Errorf("expected %d units, got %d", tt.expectedLen, len(units))
			}
		})
	}
}

func TestGetUnitByID(t *testing.T) {
	now := time.Now()
	sampleUnit := &models.UnitOfMeasure{ID: 1, Name: "Kilo", Abbreviation: "kg", CreatedAt: now}

	tests := []struct {
		name      string
		id        int64
		mockSetup func(m *MockUnitOfMeasureRepository)
		expected  *models.UnitOfMeasure
		expectErr bool
	}{
		{
			name: "Success found",
			id:   1,
			mockSetup: func(m *MockUnitOfMeasureRepository) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.UnitOfMeasure, error) {
					return sampleUnit, nil
				}
			},
			expected:  sampleUnit,
			expectErr: false,
		},
		{
			name: "Not found error",
			id:   999,
			mockSetup: func(m *MockUnitOfMeasureRepository) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.UnitOfMeasure, error) {
					return nil, errors.New("not found")
				}
			},
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockUnitOfMeasureRepository{}
			tt.mockSetup(mockRepo)

			svc := service.NewUnitOfMeasureService(mockRepo)
			unit, err := svc.GetUnitByID(context.Background(), tt.id)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && (unit == nil || unit.Name != tt.expected.Name) {
				t.Errorf("expected unit %v, got %v", tt.expected, unit)
			}
		})
	}
}
