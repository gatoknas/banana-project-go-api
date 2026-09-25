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

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"
	"org.banana.project/api/internal/handlers"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type MockPurchaseRepoForHandler struct {
	CreatePurchaseFunc           func(ctx context.Context, tx *sql.Tx, p *models.Purchase) (int64, error)
	CreatePurchaseDetailFunc     func(ctx context.Context, tx *sql.Tx, pd *models.PurchaseDetail) (int64, error)
	GetProductStockAndCostFunc   func(ctx context.Context, tx *sql.Tx, productID int64) (float64, float64, error)
	UpdateProductAverageCostFunc func(ctx context.Context, tx *sql.Tx, productID int64, newAverageCost float64) error
	AddStockFunc                 func(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error
	ListPurchasesFunc            func(ctx context.Context, supplierID *int64, fromDate, toDate *time.Time) ([]models.Purchase, error)
	GetPurchaseByIDFunc          func(ctx context.Context, id int64) (*models.Purchase, error)
	GetPurchaseDetailsFunc       func(ctx context.Context, purchaseID int64) ([]models.PurchaseDetail, error)
	UpdatePurchaseFunc           func(ctx context.Context, tx *sql.Tx, p *models.Purchase) error
	DeletePurchaseDetailsFunc    func(ctx context.Context, tx *sql.Tx, purchaseID int64) error
	DeductStockFunc              func(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error
}

func (m *MockPurchaseRepoForHandler) CreatePurchase(ctx context.Context, tx *sql.Tx, p *models.Purchase) (int64, error) {
	return m.CreatePurchaseFunc(ctx, tx, p)
}

func (m *MockPurchaseRepoForHandler) CreatePurchaseDetail(ctx context.Context, tx *sql.Tx, pd *models.PurchaseDetail) (int64, error) {
	return m.CreatePurchaseDetailFunc(ctx, tx, pd)
}

func (m *MockPurchaseRepoForHandler) GetProductStockAndCost(ctx context.Context, tx *sql.Tx, productID int64) (float64, float64, error) {
	return m.GetProductStockAndCostFunc(ctx, tx, productID)
}

func (m *MockPurchaseRepoForHandler) UpdateProductAverageCost(ctx context.Context, tx *sql.Tx, productID int64, newAverageCost float64) error {
	return m.UpdateProductAverageCostFunc(ctx, tx, productID, newAverageCost)
}

func (m *MockPurchaseRepoForHandler) AddStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error {
	return m.AddStockFunc(ctx, tx, productID, quantity)
}

func (m *MockPurchaseRepoForHandler) ListPurchases(ctx context.Context, supplierID *int64, fromDate, toDate *time.Time) ([]models.Purchase, error) {
	return m.ListPurchasesFunc(ctx, supplierID, fromDate, toDate)
}

func (m *MockPurchaseRepoForHandler) GetPurchaseByID(ctx context.Context, id int64) (*models.Purchase, error) {
	return m.GetPurchaseByIDFunc(ctx, id)
}

func (m *MockPurchaseRepoForHandler) GetPurchaseDetails(ctx context.Context, purchaseID int64) ([]models.PurchaseDetail, error) {
	return m.GetPurchaseDetailsFunc(ctx, purchaseID)
}

func (m *MockPurchaseRepoForHandler) UpdatePurchase(ctx context.Context, tx *sql.Tx, p *models.Purchase) error {
	if m.UpdatePurchaseFunc != nil {
		return m.UpdatePurchaseFunc(ctx, tx, p)
	}
	return nil
}

func (m *MockPurchaseRepoForHandler) DeletePurchaseDetails(ctx context.Context, tx *sql.Tx, purchaseID int64) error {
	if m.DeletePurchaseDetailsFunc != nil {
		return m.DeletePurchaseDetailsFunc(ctx, tx, purchaseID)
	}
	return nil
}

func (m *MockPurchaseRepoForHandler) DeductStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error {
	if m.DeductStockFunc != nil {
		return m.DeductStockFunc(ctx, tx, productID, quantity)
	}
	return nil
}

func TestPurchaseHandler_Create(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		body           string
		mockSetup      func(m *MockPurchaseRepoForHandler)
		expectedStatus int
	}{
		{
			name: "Success 201 Created",
			body: `{"supplierId": 1, "items": [{"productId": 1, "quantityPurchased": 10, "unitCost": 5000}]}`,
			mockSetup: func(m *MockPurchaseRepoForHandler) {
				m.CreatePurchaseFunc = func(ctx context.Context, tx *sql.Tx, p *models.Purchase) (int64, error) {
					return 100, nil
				}
				m.CreatePurchaseDetailFunc = func(ctx context.Context, tx *sql.Tx, pd *models.PurchaseDetail) (int64, error) {
					return 1, nil
				}
				m.GetProductStockAndCostFunc = func(ctx context.Context, tx *sql.Tx, productID int64) (float64, float64, error) {
					return 0, 0, nil
				}
				m.UpdateProductAverageCostFunc = func(ctx context.Context, tx *sql.Tx, productID int64, newCost float64) error {
					return nil
				}
				m.AddStockFunc = func(ctx context.Context, tx *sql.Tx, productID int64, qty float64) error {
					return nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Invalid JSON payload 400",
			body:           `{not-valid-json`,
			mockSetup:      func(m *MockPurchaseRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error empty items 400",
			body: `{"supplierId": 1, "items": []}`,
			mockSetup: func(m *MockPurchaseRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error zero supplier 400",
			body: `{"supplierId": 0, "items": [{"productId": 1, "quantityPurchased": 10, "unitCost": 5000}]}`,
			mockSetup: func(m *MockPurchaseRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mockSQL, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			if tt.expectedStatus == http.StatusCreated {
				mockSQL.ExpectBegin()
				mockSQL.ExpectCommit()
			}

			mock := &MockPurchaseRepoForHandler{}
			tt.mockSetup(mock)

			svc := service.NewPurchaseService(mock, db)
			handler := handlers.NewPurchaseHandler(svc, logger)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			handler.Create(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d (body: %s)", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestPurchaseHandler_List(t *testing.T) {
	logger := zap.NewNop()
	now := time.Now()
	sample := []models.Purchase{
		{ID: 1, SupplierID: 1, TotalAmount: 50000, PurchaseDate: now},
	}

	tests := []struct {
		name           string
		queryURL       string
		mockSetup      func(m *MockPurchaseRepoForHandler)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:     "Success 200 without filters",
			queryURL: "/api/v1/purchases",
			mockSetup: func(m *MockPurchaseRepoForHandler) {
				m.ListPurchasesFunc = func(ctx context.Context, supplierID *int64, fromDate, toDate *time.Time) ([]models.Purchase, error) {
					return sample, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:     "Success 200 with valid filters",
			queryURL: "/api/v1/purchases?supplierId=1&fromDate=2026-09-01&toDate=2026-09-30",
			mockSetup: func(m *MockPurchaseRepoForHandler) {
				m.ListPurchasesFunc = func(ctx context.Context, supplierID *int64, fromDate, toDate *time.Time) ([]models.Purchase, error) {
					return sample, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "Invalid supplierId query 400",
			queryURL:       "/api/v1/purchases?supplierId=abc",
			mockSetup:      func(m *MockPurchaseRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:           "Invalid fromDate query 400",
			queryURL:       "/api/v1/purchases?fromDate=invalid-date",
			mockSetup:      func(m *MockPurchaseRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:     "Database error 500",
			queryURL: "/api/v1/purchases",
			mockSetup: func(m *MockPurchaseRepoForHandler) {
				m.ListPurchasesFunc = func(ctx context.Context, supplierID *int64, fromDate, toDate *time.Time) ([]models.Purchase, error) {
					return nil, errors.New("db error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockPurchaseRepoForHandler{}
			tt.mockSetup(mock)

			svc := service.NewPurchaseService(mock, nil)
			handler := handlers.NewPurchaseHandler(svc, logger)

			req := httptest.NewRequest(http.MethodGet, tt.queryURL, nil)
			w := httptest.NewRecorder()

			handler.List(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedStatus == http.StatusOK {
				var resp []models.Purchase
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(resp) != tt.expectedCount {
					t.Errorf("expected %d purchases, got %d", tt.expectedCount, len(resp))
				}
			}
		})
	}
}

func TestPurchaseHandler_Get(t *testing.T) {
	logger := zap.NewNop()
	sample := &models.Purchase{
		ID:          1,
		SupplierID:  1,
		TotalAmount: 50000,
	}

	tests := []struct {
		name           string
		id             string
		mockSetup      func(m *MockPurchaseRepoForHandler)
		expectedStatus int
	}{
		{
			name: "Success 200",
			id:   "1",
			mockSetup: func(m *MockPurchaseRepoForHandler) {
				m.GetPurchaseByIDFunc = func(ctx context.Context, id int64) (*models.Purchase, error) {
					return sample, nil
				}
				m.GetPurchaseDetailsFunc = func(ctx context.Context, purchaseID int64) ([]models.PurchaseDetail, error) {
					return []models.PurchaseDetail{}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid ID 400",
			id:             "xyz",
			mockSetup:      func(m *MockPurchaseRepoForHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Not Found 404",
			id:   "999",
			mockSetup: func(m *MockPurchaseRepoForHandler) {
				m.GetPurchaseByIDFunc = func(ctx context.Context, id int64) (*models.Purchase, error) {
					return nil, sql.ErrNoRows
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockPurchaseRepoForHandler{}
			tt.mockSetup(mock)

			svc := service.NewPurchaseService(mock, nil)
			handler := handlers.NewPurchaseHandler(svc, logger)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/purchases/"+tt.id, nil)
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			handler.Get(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestPurchaseHandler_Update(t *testing.T) {
	logger := zap.NewNop()
	sample := &models.Purchase{ID: 1, SupplierID: 1, TotalAmount: 5000}

	tests := []struct {
		name           string
		id             string
		body           string
		mockSetup      func(m *MockPurchaseRepoForHandler, mock sqlmock.Sqlmock)
		expectedStatus int
	}{
		{
			name: "Success 200 OK",
			id:   "1",
			body: `{"supplierId": 1, "items": [{"productId": 1, "quantityPurchased": 10, "unitCost": 500}]}`,
			mockSetup: func(m *MockPurchaseRepoForHandler, mock sqlmock.Sqlmock) {
				m.GetPurchaseByIDFunc = func(ctx context.Context, id int64) (*models.Purchase, error) {
					return sample, nil
				}
				m.GetPurchaseDetailsFunc = func(ctx context.Context, purchaseID int64) ([]models.PurchaseDetail, error) {
					return []models.PurchaseDetail{}, nil
				}
				mock.ExpectBegin()
				m.DeletePurchaseDetailsFunc = func(ctx context.Context, tx *sql.Tx, purchaseID int64) error {
					return nil
				}
				m.UpdatePurchaseFunc = func(ctx context.Context, tx *sql.Tx, p *models.Purchase) error {
					return nil
				}
				m.CreatePurchaseDetailFunc = func(ctx context.Context, tx *sql.Tx, pd *models.PurchaseDetail) (int64, error) {
					return 1, nil
				}
				m.GetProductStockAndCostFunc = func(ctx context.Context, tx *sql.Tx, productID int64) (float64, float64, error) {
					return 0, 0, nil
				}
				m.UpdateProductAverageCostFunc = func(ctx context.Context, tx *sql.Tx, productID int64, newCost float64) error {
					return nil
				}
				m.AddStockFunc = func(ctx context.Context, tx *sql.Tx, productID int64, qty float64) error {
					return nil
				}
				mock.ExpectCommit()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid ID 400",
			id:             "xyz",
			body:           `{"supplierId": 1, "items": [{"productId": 1, "quantityPurchased": 10, "unitCost": 500}]}`,
			mockSetup:      func(m *MockPurchaseRepoForHandler, mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON payload 400",
			id:             "1",
			body:           `{invalid-json`,
			mockSetup:      func(m *MockPurchaseRepoForHandler, mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Validation error empty items 400",
			id:             "1",
			body:           `{"supplierId": 1, "items": []}`,
			mockSetup:      func(m *MockPurchaseRepoForHandler, mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Purchase not found 404",
			id:   "999",
			body: `{"supplierId": 1, "items": [{"productId": 1, "quantityPurchased": 10, "unitCost": 500}]}`,
			mockSetup: func(m *MockPurchaseRepoForHandler, mock sqlmock.Sqlmock) {
				m.GetPurchaseByIDFunc = func(ctx context.Context, id int64) (*models.Purchase, error) {
					return nil, sql.ErrNoRows
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Internal server error 500",
			id:   "1",
			body: `{"supplierId": 1, "items": [{"productId": 1, "quantityPurchased": 10, "unitCost": 500}]}`,
			mockSetup: func(m *MockPurchaseRepoForHandler, mock sqlmock.Sqlmock) {
				m.GetPurchaseByIDFunc = func(ctx context.Context, id int64) (*models.Purchase, error) {
					return nil, errors.New("db disk failure")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mockSQL, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			mock := &MockPurchaseRepoForHandler{}
			tt.mockSetup(mock, mockSQL)

			svc := service.NewPurchaseService(mock, db)
			handler := handlers.NewPurchaseHandler(svc, logger)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/purchases/"+tt.id, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			handler.Update(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

