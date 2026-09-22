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
