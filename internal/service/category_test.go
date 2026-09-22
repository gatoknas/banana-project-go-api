package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type MockCategoryRepository struct {
	ListFunc    func(ctx context.Context) ([]models.Category, error)
	GetByIDFunc func(ctx context.Context, id int64) (*models.Category, error)
}

func (m *MockCategoryRepository) List(ctx context.Context) ([]models.Category, error) {
	return m.ListFunc(ctx)
}

func (m *MockCategoryRepository) GetByID(ctx context.Context, id int64) (*models.Category, error) {
	return m.GetByIDFunc(ctx, id)
}

func TestListCategories(t *testing.T) {
	now := time.Now()
	sampleCategories := []models.Category{
		{ID: 1, Name: "Pasabocas y Golosinas", CreatedAt: now},
		{ID: 2, Name: "Frutas", CreatedAt: now},
		{ID: 9, Name: "Cafeteria", CreatedAt: now},
	}

	tests := []struct {
		name        string
		mockSetup   func(m *MockCategoryRepository)
		expectedLen int
		expectErr   bool
	}{
		{
			name: "Success with categories",
			mockSetup: func(m *MockCategoryRepository) {
				m.ListFunc = func(ctx context.Context) ([]models.Category, error) {
					return sampleCategories, nil
				}
			},
			expectedLen: 3,
			expectErr:   false,
		},
		{
			name: "Success with empty list",
			mockSetup: func(m *MockCategoryRepository) {
				m.ListFunc = func(ctx context.Context) ([]models.Category, error) {
					return []models.Category{}, nil
				}
			},
			expectedLen: 0,
			expectErr:   false,
		},
		{
			name: "Repository error",
			mockSetup: func(m *MockCategoryRepository) {
				m.ListFunc = func(ctx context.Context) ([]models.Category, error) {
					return nil, errors.New("database connection failed")
				}
			},
			expectedLen: 0,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			svc := service.NewCategoryService(mockRepo)
			categories, err := svc.ListCategories(context.Background())

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && len(categories) != tt.expectedLen {
				t.Errorf("expected %d categories, got %d", tt.expectedLen, len(categories))
			}
		})
	}
}

func TestGetCategoryByID(t *testing.T) {
	now := time.Now()
	sampleCategory := &models.Category{ID: 9, Name: "Cafeteria", CreatedAt: now}

	tests := []struct {
		name        string
		id          int64
		mockSetup   func(m *MockCategoryRepository)
		expected    *models.Category
		expectErr   bool
	}{
		{
			name: "Success found",
			id:   9,
			mockSetup: func(m *MockCategoryRepository) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Category, error) {
					return sampleCategory, nil
				}
			},
			expected:  sampleCategory,
			expectErr: false,
		},
		{
			name: "Not found",
			id:   999,
			mockSetup: func(m *MockCategoryRepository) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Category, error) {
					return nil, errors.New("not found")
				}
			},
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			svc := service.NewCategoryService(mockRepo)
			category, err := svc.GetCategoryByID(context.Background(), tt.id)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && (category == nil || category.Name != tt.expected.Name) {
				t.Errorf("expected category %v, got %v", tt.expected, category)
			}
		})
	}
}
