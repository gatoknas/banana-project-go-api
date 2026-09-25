package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"
	"org.banana.project/api/internal/handlers"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type MockSaleRepo struct {
	CreateSaleFunc           func(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error)
	CreateSaleDetailFunc     func(ctx context.Context, tx *sql.Tx, d *models.SaleDetail) (int64, error)
	GetProductDetailsFunc    func(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error)
	GetRecipeIngredientsFunc func(ctx context.Context, tx *sql.Tx, parentProductID int64) ([]models.ProductRecipe, error)
	DeductStockFunc          func(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error
	ListSalesFunc            func(ctx context.Context, filter models.SaleFilter) ([]models.Sale, models.SalePagination, models.SaleSummaryStats, error)
	GetSaleByIDFunc          func(ctx context.Context, id int64) (*models.Sale, error)
}

func (m *MockSaleRepo) CreateSale(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error) {
	return m.CreateSaleFunc(ctx, tx, s)
}

func (m *MockSaleRepo) CreateSaleDetail(ctx context.Context, tx *sql.Tx, d *models.SaleDetail) (int64, error) {
	return m.CreateSaleDetailFunc(ctx, tx, d)
}

func (m *MockSaleRepo) GetProductDetails(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error) {
	return m.GetProductDetailsFunc(ctx, tx, productID)
}

func (m *MockSaleRepo) GetRecipeIngredients(ctx context.Context, tx *sql.Tx, parentProductID int64) ([]models.ProductRecipe, error) {
	return m.GetRecipeIngredientsFunc(ctx, tx, parentProductID)
}

func (m *MockSaleRepo) DeductStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error {
	return m.DeductStockFunc(ctx, tx, productID, quantity)
}

func (m *MockSaleRepo) ListSales(ctx context.Context, filter models.SaleFilter) ([]models.Sale, models.SalePagination, models.SaleSummaryStats, error) {
	if m.ListSalesFunc != nil {
		return m.ListSalesFunc(ctx, filter)
	}
	return nil, models.SalePagination{}, models.SaleSummaryStats{}, nil
}

func (m *MockSaleRepo) GetSaleByID(ctx context.Context, id int64) (*models.Sale, error) {
	if m.GetSaleByIDFunc != nil {
		return m.GetSaleByIDFunc(ctx, id)
	}
	return nil, nil
}

func TestSaleHandler_Create(t *testing.T) {
	validBody := `{"userId":1,"totalAmount":10,"paymentMethod":"cash","items":[{"productId":1,"quantity":2}]}`

	tests := []struct {
		name       string
		body       string
		repo       *MockSaleRepo
		useDB      bool
		expect     func(mock sqlmock.Sqlmock)
		wantStatus int
	}{
		{
			name: "success",
			body: validBody,
			repo: &MockSaleRepo{
				CreateSaleFunc: func(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error) { return 10, nil },
				GetProductDetailsFunc: func(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error) {
					return 5.0, false, nil
				},
				CreateSaleDetailFunc: func(ctx context.Context, tx *sql.Tx, d *models.SaleDetail) (int64, error) { return 1, nil },
				DeductStockFunc:      func(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error { return nil },
			},
			useDB:      true,
			expect:     func(mock sqlmock.Sqlmock) { mock.ExpectBegin(); mock.ExpectCommit() },
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json",
			body:       `{`,
			repo:       &MockSaleRepo{},
			expect:     func(mock sqlmock.Sqlmock) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty items",
			body:       `{"userId":1,"items":[]}`,
			repo:       &MockSaleRepo{},
			expect:     func(mock sqlmock.Sqlmock) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "product not found",
			body: validBody,
			repo: &MockSaleRepo{
				CreateSaleFunc: func(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error) { return 10, nil },
				GetProductDetailsFunc: func(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error) {
					return 0, false, errors.New("no rows")
				},
			},
			useDB:      true,
			expect:     func(mock sqlmock.Sqlmock) { mock.ExpectBegin(); mock.ExpectRollback() },
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "create sale fails",
			body: validBody,
			repo: &MockSaleRepo{
				CreateSaleFunc: func(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error) {
					return 0, errors.New("db error")
				},
			},
			useDB:      true,
			expect:     func(mock sqlmock.Sqlmock) { mock.ExpectBegin(); mock.ExpectRollback() },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var db *sql.DB
			if tt.useDB {
				var mock sqlmock.Sqlmock
				db, mock = newSQLMock(t)
				defer db.Close()
				tt.expect(mock)
			} else {
				db, _ = newSQLMock(t)
				defer db.Close()
			}

			h := handlers.NewSaleHandler(service.NewSaleService(tt.repo, db), zap.NewNop())
			req := httptest.NewRequest(http.MethodPost, "/api/v1/sales", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.Create(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantStatus == http.StatusCreated {
				var resp handlers.SaleResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.SaleID != 10 {
					t.Errorf("expected saleId 10, got %d", resp.SaleID)
				}
			}
		})
	}
}

func TestSaleHandler_List(t *testing.T) {
	carlos := "Carlos Gómez"
	tests := []struct {
		name       string
		queryURL   string
		repo       *MockSaleRepo
		wantStatus int
		wantItems  int
	}{
		{
			name:     "success default query",
			queryURL: "/api/v1/sales",
			repo: &MockSaleRepo{
				ListSalesFunc: func(ctx context.Context, filter models.SaleFilter) ([]models.Sale, models.SalePagination, models.SaleSummaryStats, error) {
					return []models.Sale{
						{
							ID:            1,
							UserID:        2,
							UserName:      &carlos,
							SaleDate:      time.Now(),
							TotalAmount:   45000,
							PaymentMethod: "Cash",
							ItemsCount:    2,
						},
					}, models.SalePagination{Page: 1, PageSize: 10, TotalItems: 1, TotalPages: 1}, models.SaleSummaryStats{TotalAmount: 45000, TotalCount: 1}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantItems:  1,
		},
		{
			name:     "success with all filters",
			queryURL: "/api/v1/sales?page=2&pageSize=20&fromDate=2026-09-01&toDate=2026-09-30&paymentMethod=Cash&userId=2",
			repo: &MockSaleRepo{
				ListSalesFunc: func(ctx context.Context, filter models.SaleFilter) ([]models.Sale, models.SalePagination, models.SaleSummaryStats, error) {
					if filter.Page != 2 || filter.PageSize != 20 || filter.PaymentMethod != "Cash" || filter.UserID == nil || *filter.UserID != 2 {
						return nil, models.SalePagination{}, models.SaleSummaryStats{}, errors.New("filter mismatch")
					}
					return []models.Sale{}, models.SalePagination{Page: 2, PageSize: 20, TotalItems: 0, TotalPages: 0}, models.SaleSummaryStats{}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantItems:  0,
		},
		{
			name:       "invalid page",
			queryURL:   "/api/v1/sales?page=0",
			repo:       &MockSaleRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid pageSize",
			queryURL:   "/api/v1/sales?pageSize=-5",
			repo:       &MockSaleRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid fromDate",
			queryURL:   "/api/v1/sales?fromDate=invalid-date",
			repo:       &MockSaleRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid toDate",
			queryURL:   "/api/v1/sales?toDate=invalid-date",
			repo:       &MockSaleRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid userId",
			queryURL:   "/api/v1/sales?userId=abc",
			repo:       &MockSaleRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "repo error returns 500",
			queryURL: "/api/v1/sales",
			repo: &MockSaleRepo{
				ListSalesFunc: func(ctx context.Context, filter models.SaleFilter) ([]models.Sale, models.SalePagination, models.SaleSummaryStats, error) {
					return nil, models.SalePagination{}, models.SaleSummaryStats{}, errors.New("internal error")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _ := newSQLMock(t)
			defer db.Close()

			h := handlers.NewSaleHandler(service.NewSaleService(tt.repo, db), zap.NewNop())
			req := httptest.NewRequest(http.MethodGet, tt.queryURL, nil)
			w := httptest.NewRecorder()

			h.List(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantStatus == http.StatusOK {
				var resp models.PaginatedSalesResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(resp.Data) != tt.wantItems {
					t.Errorf("expected %d items, got %d", tt.wantItems, len(resp.Data))
				}
			}
		})
	}
}

func TestSaleHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		pathValue  string
		repo       *MockSaleRepo
		wantStatus int
		wantID     int64
	}{
		{
			name:      "success get sale",
			pathValue: "10",
			repo: &MockSaleRepo{
				GetSaleByIDFunc: func(ctx context.Context, id int64) (*models.Sale, error) {
					return &models.Sale{
						ID:          10,
						TotalAmount: 30000,
						Details: []models.SaleDetail{
							{ID: 1, SaleID: 10, ProductID: 2, Quantity: 1, Subtotal: 30000},
						},
					}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantID:     10,
		},
		{
			name:       "invalid id",
			pathValue:  "abc",
			repo:       &MockSaleRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "not found",
			pathValue: "999",
			repo: &MockSaleRepo{
				GetSaleByIDFunc: func(ctx context.Context, id int64) (*models.Sale, error) {
					return nil, sql.ErrNoRows
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:      "internal server error",
			pathValue: "50",
			repo: &MockSaleRepo{
				GetSaleByIDFunc: func(ctx context.Context, id int64) (*models.Sale, error) {
					return nil, errors.New("database connection broken")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _ := newSQLMock(t)
			defer db.Close()

			h := handlers.NewSaleHandler(service.NewSaleService(tt.repo, db), zap.NewNop())
			req := httptest.NewRequest(http.MethodGet, "/api/v1/sales/"+tt.pathValue, nil)
			req.SetPathValue("id", tt.pathValue)
			w := httptest.NewRecorder()

			h.Get(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantStatus == http.StatusOK {
				var s models.Sale
				if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if s.ID != tt.wantID {
					t.Errorf("expected sale ID %d, got %d", tt.wantID, s.ID)
				}
			}
		})
	}
}

