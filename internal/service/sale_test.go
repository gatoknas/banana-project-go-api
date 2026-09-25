package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type MockSaleRepository struct {
	CreateSaleFunc           func(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error)
	CreateSaleDetailFunc     func(ctx context.Context, tx *sql.Tx, d *models.SaleDetail) (int64, error)
	GetProductDetailsFunc    func(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error)
	GetRecipeIngredientsFunc func(ctx context.Context, tx *sql.Tx, parentProductID int64) ([]models.ProductRecipe, error)
	DeductStockFunc          func(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error
	ListSalesFunc            func(ctx context.Context, filter models.SaleFilter) ([]models.Sale, models.SalePagination, models.SaleSummaryStats, error)
	GetSaleByIDFunc          func(ctx context.Context, id int64) (*models.Sale, error)
}

func (m *MockSaleRepository) CreateSale(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error) {
	if m.CreateSaleFunc != nil {
		return m.CreateSaleFunc(ctx, tx, s)
	}
	return 0, nil
}

func (m *MockSaleRepository) CreateSaleDetail(ctx context.Context, tx *sql.Tx, d *models.SaleDetail) (int64, error) {
	if m.CreateSaleDetailFunc != nil {
		return m.CreateSaleDetailFunc(ctx, tx, d)
	}
	return 0, nil
}

func (m *MockSaleRepository) GetProductDetails(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error) {
	if m.GetProductDetailsFunc != nil {
		return m.GetProductDetailsFunc(ctx, tx, productID)
	}
	return 0, false, nil
}

func (m *MockSaleRepository) GetRecipeIngredients(ctx context.Context, tx *sql.Tx, parentProductID int64) ([]models.ProductRecipe, error) {
	if m.GetRecipeIngredientsFunc != nil {
		return m.GetRecipeIngredientsFunc(ctx, tx, parentProductID)
	}
	return nil, nil
}

func (m *MockSaleRepository) DeductStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error {
	if m.DeductStockFunc != nil {
		return m.DeductStockFunc(ctx, tx, productID, quantity)
	}
	return nil
}

func (m *MockSaleRepository) ListSales(ctx context.Context, filter models.SaleFilter) ([]models.Sale, models.SalePagination, models.SaleSummaryStats, error) {
	if m.ListSalesFunc != nil {
		return m.ListSalesFunc(ctx, filter)
	}
	return nil, models.SalePagination{}, models.SaleSummaryStats{}, nil
}

func (m *MockSaleRepository) GetSaleByID(ctx context.Context, id int64) (*models.Sale, error) {
	if m.GetSaleByIDFunc != nil {
		return m.GetSaleByIDFunc(ctx, id)
	}
	return nil, nil
}

func TestSaleService_CreateSale(t *testing.T) {
	tests := []struct {
		name       string
		req        service.SaleRequest
		mockSetup  func(mock sqlmock.Sqlmock)
		repo       *MockSaleRepository
		wantErr    bool
		expectedID int64
	}{
		{
			name: "success with standard product",
			req: service.SaleRequest{
				UserID:        1,
				TotalAmount:   20000,
				PaymentMethod: "Cash",
				Items: []service.SaleItemRequest{
					{ProductID: 10, Quantity: 2},
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			repo: &MockSaleRepository{
				CreateSaleFunc: func(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error) {
					return 101, nil
				},
				GetProductDetailsFunc: func(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error) {
					return 10000, false, nil
				},
				CreateSaleDetailFunc: func(ctx context.Context, tx *sql.Tx, d *models.SaleDetail) (int64, error) {
					return 1, nil
				},
				DeductStockFunc: func(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error {
					return nil
				},
			},
			wantErr:    false,
			expectedID: 101,
		},
		{
			name: "success with recipe product",
			req: service.SaleRequest{
				UserID:        1,
				TotalAmount:   15000,
				PaymentMethod: "Card",
				Items: []service.SaleItemRequest{
					{ProductID: 20, Quantity: 1},
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			repo: &MockSaleRepository{
				CreateSaleFunc: func(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error) {
					return 102, nil
				},
				GetProductDetailsFunc: func(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error) {
					return 15000, true, nil
				},
				CreateSaleDetailFunc: func(ctx context.Context, tx *sql.Tx, d *models.SaleDetail) (int64, error) {
					return 2, nil
				},
				GetRecipeIngredientsFunc: func(ctx context.Context, tx *sql.Tx, parentProductID int64) ([]models.ProductRecipe, error) {
					return []models.ProductRecipe{
						{ParentProductID: 20, ChildProductID: 30, Quantity: 0.5},
					}, nil
				},
				DeductStockFunc: func(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error {
					return nil
				},
			},
			wantErr:    false,
			expectedID: 102,
		},
		{
			name: "error empty items",
			req: service.SaleRequest{
				UserID: 1,
				Items:  []service.SaleItemRequest{},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {},
			repo:      &MockSaleRepository{},
			wantErr:   true,
		},
		{
			name: "error begin transaction fails",
			req: service.SaleRequest{
				UserID: 1,
				Items:  []service.SaleItemRequest{{ProductID: 1, Quantity: 1}},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errors.New("db connection failure"))
			},
			repo:    &MockSaleRepository{},
			wantErr: true,
		},
		{
			name: "error product details fails",
			req: service.SaleRequest{
				UserID: 1,
				Items:  []service.SaleItemRequest{{ProductID: 99, Quantity: 1}},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			repo: &MockSaleRepository{
				CreateSaleFunc: func(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error) {
					return 103, nil
				},
				GetProductDetailsFunc: func(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error) {
					return 0, false, errors.New("product 99 not found")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to open sqlmock: %v", err)
			}
			defer db.Close()

			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			svc := service.NewSaleService(tt.repo, db)
			id, err := svc.CreateSale(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSale() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && id != tt.expectedID {
				t.Errorf("CreateSale() id = %v, expected %v", id, tt.expectedID)
			}
		})
	}
}

func TestSaleService_ListSales(t *testing.T) {
	carlos := "Carlos Gómez"
	tests := []struct {
		name       string
		filter     models.SaleFilter
		repo       *MockSaleRepository
		wantErr    bool
		wantCount  int
		wantAmount float64
	}{
		{
			name: "success listing sales",
			filter: models.SaleFilter{
				Page:     1,
				PageSize: 10,
			},
			repo: &MockSaleRepository{
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
			wantErr:    false,
			wantCount:  1,
			wantAmount: 45000,
		},
		{
			name: "error from repository",
			filter: models.SaleFilter{
				Page: 1,
			},
			repo: &MockSaleRepository{
				ListSalesFunc: func(ctx context.Context, filter models.SaleFilter) ([]models.Sale, models.SalePagination, models.SaleSummaryStats, error) {
					return nil, models.SalePagination{}, models.SaleSummaryStats{}, errors.New("db error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewSaleService(tt.repo, nil)
			resp, err := svc.ListSales(context.Background(), tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListSales() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(resp.Data) != tt.wantCount {
					t.Errorf("ListSales() count = %d, want %d", len(resp.Data), tt.wantCount)
				}
				if resp.Summary.TotalAmount != tt.wantAmount {
					t.Errorf("ListSales() totalAmount = %v, want %v", resp.Summary.TotalAmount, tt.wantAmount)
				}
			}
		})
	}
}

func TestSaleService_GetSaleByID(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		repo    *MockSaleRepository
		wantErr bool
		wantID  int64
	}{
		{
			name: "success found",
			id:   5,
			repo: &MockSaleRepository{
				GetSaleByIDFunc: func(ctx context.Context, id int64) (*models.Sale, error) {
					return &models.Sale{ID: 5, TotalAmount: 25000}, nil
				},
			},
			wantErr: false,
			wantID:  5,
		},
		{
			name:    "error invalid id",
			id:      0,
			repo:    &MockSaleRepository{},
			wantErr: true,
		},
		{
			name: "error repo failure",
			id:   99,
			repo: &MockSaleRepository{
				GetSaleByIDFunc: func(ctx context.Context, id int64) (*models.Sale, error) {
					return nil, errors.New("not found")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewSaleService(tt.repo, nil)
			s, err := svc.GetSaleByID(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSaleByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && (s == nil || s.ID != tt.wantID) {
				t.Errorf("GetSaleByID() = %v, want ID %v", s, tt.wantID)
			}
		})
	}
}
