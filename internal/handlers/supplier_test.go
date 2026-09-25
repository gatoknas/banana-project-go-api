package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
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

func strPtr(s string) *string {
	return &s
}

type MockSupplierRepoForHandler struct {
	CreateFunc       func(ctx context.Context, tx *sql.Tx, s *models.Supplier) (int64, error)
	GetByIDFunc      func(ctx context.Context, id int64) (*models.Supplier, error)
	GetByTaxIDFunc   func(ctx context.Context, taxID string) (*models.Supplier, error)
	ListFunc         func(ctx context.Context, searchQuery string) ([]models.Supplier, error)
	UpdateFunc       func(ctx context.Context, s *models.Supplier) error
	DeleteFunc       func(ctx context.Context, tx *sql.Tx, id int64) error
	HasPurchasesFunc func(ctx context.Context, id int64) (bool, error)
}

func (m *MockSupplierRepoForHandler) Create(ctx context.Context, tx *sql.Tx, s *models.Supplier) (int64, error) {
	return m.CreateFunc(ctx, tx, s)
}

func (m *MockSupplierRepoForHandler) GetByID(ctx context.Context, id int64) (*models.Supplier, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *MockSupplierRepoForHandler) GetByTaxID(ctx context.Context, taxID string) (*models.Supplier, error) {
	return m.GetByTaxIDFunc(ctx, taxID)
}

func (m *MockSupplierRepoForHandler) List(ctx context.Context, searchQuery string) ([]models.Supplier, error) {
	return m.ListFunc(ctx, searchQuery)
}

func (m *MockSupplierRepoForHandler) Update(ctx context.Context, s *models.Supplier) error {
	return m.UpdateFunc(ctx, s)
}

func (m *MockSupplierRepoForHandler) Delete(ctx context.Context, tx *sql.Tx, id int64) error {
	return m.DeleteFunc(ctx, tx, id)
}

func (m *MockSupplierRepoForHandler) HasPurchases(ctx context.Context, id int64) (bool, error) {
	return m.HasPurchasesFunc(ctx, id)
}

func TestSupplierHandler_Create(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		body           string
		mockSetup      func(m *MockSupplierRepoForHandler)
		expectedStatus int
	}{
		{
			name: "Success 201 Created with Tax ID and Phone",
			body: `{"taxId": "900123", "companyName": "Fruit Express", "phone": "+57 300 123"}`,
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.GetByTaxIDFunc = func(ctx context.Context, taxID string) (*models.Supplier, error) {
					return nil, sql.ErrNoRows
				}
				m.CreateFunc = func(ctx context.Context, tx *sql.Tx, s *models.Supplier) (int64, error) {
					return 10, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Success 201 Created without Tax ID (Optional)",
			body: `{"companyName": "Fruit Express", "phone": "+57 300 123"}`,
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.CreateFunc = func(ctx context.Context, tx *sql.Tx, s *models.Supplier) (int64, error) {
					return 10, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Success 201 Created with Description",
			body: `{"taxId": "900123", "companyName": "Fruit Express", "phone": "+57 300 123", "description": "Preferred fruit distributor in Medellín"}`,
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.GetByTaxIDFunc = func(ctx context.Context, taxID string) (*models.Supplier, error) {
					return nil, sql.ErrNoRows
				}
				m.CreateFunc = func(ctx context.Context, tx *sql.Tx, s *models.Supplier) (int64, error) {
					if s.Description == nil || *s.Description != "Preferred fruit distributor in Medellín" {
						t.Errorf("expected description to be passed, got %v", s.Description)
					}
					return 10, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Invalid JSON payload 400",
			body:           `{invalid-json}`,
			mockSetup:      func(m *MockSupplierRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error missing company name 400",
			body: `{"phone": "+57 300 123", "companyName": ""}`,
			mockSetup: func(m *MockSupplierRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error missing phone 400",
			body: `{"companyName": "Fruit Express"}`,
			mockSetup: func(m *MockSupplierRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Conflict tax_id already registered 409",
			body: `{"taxId": "900123", "companyName": "Fruit Express", "phone": "+57 300 123"}`,
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.GetByTaxIDFunc = func(ctx context.Context, taxID string) (*models.Supplier, error) {
					return &models.Supplier{ID: 1, TaxID: strPtr("900123")}, nil
				}
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockSupplierRepoForHandler{}
			tt.mockSetup(mock)

			svc := service.NewSupplierService(mock, nil)
			handler := handlers.NewSupplierHandler(svc, logger)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/suppliers", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			handler.Create(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d (body: %s)", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestSupplierHandler_List(t *testing.T) {
	logger := zap.NewNop()
	now := time.Now()

	tests := []struct {
		name           string
		mockSetup      func(m *MockSupplierRepoForHandler)
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "Success Returns Suppliers",
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.ListFunc = func(ctx context.Context, searchQuery string) ([]models.Supplier, error) {
					return []models.Supplier{
						{ID: 1, TaxID: strPtr("111"), CompanyName: "Supplier A", CreatedAt: now},
						{ID: 2, TaxID: strPtr("222"), CompanyName: "Supplier B", CreatedAt: now},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name: "Database error 500",
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.ListFunc = func(ctx context.Context, searchQuery string) ([]models.Supplier, error) {
					return nil, errors.New("db error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockSupplierRepoForHandler{}
			tt.mockSetup(mock)

			svc := service.NewSupplierService(mock, nil)
			handler := handlers.NewSupplierHandler(svc, logger)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/suppliers", nil)
			w := httptest.NewRecorder()

			handler.List(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedStatus == http.StatusOK {
				var list []models.Supplier
				if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(list) != tt.expectedCount {
					t.Errorf("expected %d items, got %d", tt.expectedCount, len(list))
				}
			}
		})
	}
}

func TestSupplierHandler_Get(t *testing.T) {
	logger := zap.NewNop()
	sample := &models.Supplier{ID: 1, TaxID: strPtr("111"), CompanyName: "Supplier A"}

	tests := []struct {
		name           string
		id             string
		mockSetup      func(m *MockSupplierRepoForHandler)
		expectedStatus int
	}{
		{
			name: "Success 200",
			id:   "1",
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return sample, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid ID 400",
			id:             "abc",
			mockSetup:      func(m *MockSupplierRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Not Found 404",
			id:   "999",
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return nil, sql.ErrNoRows
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockSupplierRepoForHandler{}
			tt.mockSetup(mock)

			svc := service.NewSupplierService(mock, nil)
			handler := handlers.NewSupplierHandler(svc, logger)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/suppliers/"+tt.id, nil)
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			handler.Get(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestSupplierHandler_Update(t *testing.T) {
	logger := zap.NewNop()
	existing := &models.Supplier{ID: 1, TaxID: strPtr("111"), CompanyName: "Supplier A", Phone: strPtr("+57 300 123")}

	tests := []struct {
		name           string
		id             string
		body           string
		mockSetup      func(m *MockSupplierRepoForHandler)
		expectedStatus int
	}{
		{
			name: "Success 200",
			id:   "1",
			body: `{"taxId": "111", "companyName": "Updated Supplier", "phone": "+57 300 123"}`,
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return existing, nil
				}
				m.UpdateFunc = func(ctx context.Context, s *models.Supplier) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Success 200 with Description",
			id:   "1",
			body: `{"taxId": "111", "companyName": "Updated Supplier", "phone": "+57 300 123", "description": "Updated vendor notes"}`,
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return existing, nil
				}
				m.UpdateFunc = func(ctx context.Context, s *models.Supplier) error {
					if s.Description == nil || *s.Description != "Updated vendor notes" {
						t.Errorf("expected description to be passed, got %v", s.Description)
					}
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid ID 400",
			id:             "0",
			body:           `{"taxId": "111", "companyName": "Updated", "phone": "+57 300 123"}`,
			mockSetup:      func(m *MockSupplierRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing Phone 400",
			id:             "1",
			body:           `{"taxId": "111", "companyName": "Updated"}`,
			mockSetup:      func(m *MockSupplierRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Not Found 404",
			id:   "99",
			body: `{"taxId": "111", "companyName": "Updated", "phone": "+57 300 123"}`,
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return nil, sql.ErrNoRows
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockSupplierRepoForHandler{}
			tt.mockSetup(mock)

			svc := service.NewSupplierService(mock, nil)
			handler := handlers.NewSupplierHandler(svc, logger)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/suppliers/"+tt.id, bytes.NewBufferString(tt.body))
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			handler.Update(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestSupplierHandler_Delete(t *testing.T) {
	logger := zap.NewNop()
	existing := &models.Supplier{ID: 1, TaxID: strPtr("111"), CompanyName: "Supplier A"}

	tests := []struct {
		name           string
		id             string
		mockSetup      func(m *MockSupplierRepoForHandler)
		expectedStatus int
	}{
		{
			name: "Success 200",
			id:   "1",
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return existing, nil
				}
				m.HasPurchasesFunc = func(ctx context.Context, id int64) (bool, error) {
					return false, nil
				}
				m.DeleteFunc = func(ctx context.Context, tx *sql.Tx, id int64) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Conflict has purchases 409",
			id:   "1",
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return existing, nil
				}
				m.HasPurchasesFunc = func(ctx context.Context, id int64) (bool, error) {
					return true, nil
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "Not Found 404",
			id:   "99",
			mockSetup: func(m *MockSupplierRepoForHandler) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return nil, sql.ErrNoRows
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockSupplierRepoForHandler{}
			tt.mockSetup(mock)

			svc := service.NewSupplierService(mock, nil)
			handler := handlers.NewSupplierHandler(svc, logger)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/suppliers/"+tt.id, nil)
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			handler.Delete(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
