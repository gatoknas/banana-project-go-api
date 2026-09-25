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

type MockPurchaseRepository struct {
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

func (m *MockPurchaseRepository) CreatePurchase(ctx context.Context, tx *sql.Tx, p *models.Purchase) (int64, error) {
	return m.CreatePurchaseFunc(ctx, tx, p)
}

func (m *MockPurchaseRepository) CreatePurchaseDetail(ctx context.Context, tx *sql.Tx, pd *models.PurchaseDetail) (int64, error) {
	return m.CreatePurchaseDetailFunc(ctx, tx, pd)
}

func (m *MockPurchaseRepository) GetProductStockAndCost(ctx context.Context, tx *sql.Tx, productID int64) (float64, float64, error) {
	return m.GetProductStockAndCostFunc(ctx, tx, productID)
}

func (m *MockPurchaseRepository) UpdateProductAverageCost(ctx context.Context, tx *sql.Tx, productID int64, newAverageCost float64) error {
	return m.UpdateProductAverageCostFunc(ctx, tx, productID, newAverageCost)
}

func (m *MockPurchaseRepository) AddStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error {
	return m.AddStockFunc(ctx, tx, productID, quantity)
}

func (m *MockPurchaseRepository) ListPurchases(ctx context.Context, supplierID *int64, fromDate, toDate *time.Time) ([]models.Purchase, error) {
	return m.ListPurchasesFunc(ctx, supplierID, fromDate, toDate)
}

func (m *MockPurchaseRepository) GetPurchaseByID(ctx context.Context, id int64) (*models.Purchase, error) {
	return m.GetPurchaseByIDFunc(ctx, id)
}

func (m *MockPurchaseRepository) GetPurchaseDetails(ctx context.Context, purchaseID int64) ([]models.PurchaseDetail, error) {
	return m.GetPurchaseDetailsFunc(ctx, purchaseID)
}

func (m *MockPurchaseRepository) UpdatePurchase(ctx context.Context, tx *sql.Tx, p *models.Purchase) error {
	if m.UpdatePurchaseFunc != nil {
		return m.UpdatePurchaseFunc(ctx, tx, p)
	}
	return nil
}

func (m *MockPurchaseRepository) DeletePurchaseDetails(ctx context.Context, tx *sql.Tx, purchaseID int64) error {
	if m.DeletePurchaseDetailsFunc != nil {
		return m.DeletePurchaseDetailsFunc(ctx, tx, purchaseID)
	}
	return nil
}

func (m *MockPurchaseRepository) DeductStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error {
	if m.DeductStockFunc != nil {
		return m.DeductStockFunc(ctx, tx, productID, quantity)
	}
	return nil
}

func TestCreatePurchase(t *testing.T) {
	now := time.Now()
	invoice := "INV-1001"
	unitID := int64(3)

	tests := []struct {
		name           string
		req            service.PurchaseRequest
		mockSetup      func(m *MockPurchaseRepository, mock sqlmock.Sqlmock)
		expectErr      error
		expectedID     int64
		checkCostStock func(t *testing.T, recordedCost, recordedStock float64)
	}{
		{
			name: "Success with conversion factor and average cost calculation",
			req: service.PurchaseRequest{
				SupplierID:    1,
				PurchaseDate:  &now,
				InvoiceNumber: &invoice,
				Items: []service.PurchaseItemRequest{
					{
						ProductID:         1,
						PurchaseUnitID:    &unitID,
						QuantityPurchased: 2,       // 2 Bultos
						UnitCost:          100000,  // $100,000 COP per Bulto
						ConversionFactor:  50,      // 50 kg per Bulto -> 100 kg total, cost = $2,000/kg
					},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				m.CreatePurchaseFunc = func(ctx context.Context, tx *sql.Tx, p *models.Purchase) (int64, error) {
					if p.TotalAmount != 200000 {
						t.Errorf("expected total amount 200000, got %f", p.TotalAmount)
					}
					return 10, nil
				}
				m.CreatePurchaseDetailFunc = func(ctx context.Context, tx *sql.Tx, pd *models.PurchaseDetail) (int64, error) {
					return 1, nil
				}
				// Initial stock: 100 kg at $1,000/kg
				m.GetProductStockAndCostFunc = func(ctx context.Context, tx *sql.Tx, productID int64) (float64, float64, error) {
					return 100, 1000, nil
				}
				// New cost: ((100 * 1000) + (100 * 2000)) / (100 + 100) = (100000 + 200000) / 200 = 1500
				m.UpdateProductAverageCostFunc = func(ctx context.Context, tx *sql.Tx, productID int64, newCost float64) error {
					if newCost != 1500 {
						t.Errorf("expected new weighted average cost 1500, got %f", newCost)
					}
					return nil
				}
				// Stock added: 2 * 50 = 100
				m.AddStockFunc = func(ctx context.Context, tx *sql.Tx, productID int64, qty float64) error {
					if qty != 100 {
						t.Errorf("expected stock added 100, got %f", qty)
					}
					return nil
				}
				mock.ExpectCommit()
			},
			expectErr:  nil,
			expectedID: 10,
		},
		{
			name: "Success with initial zero stock sets cost directly",
			req: service.PurchaseRequest{
				SupplierID: 1,
				Items: []service.PurchaseItemRequest{
					{
						ProductID:         2,
						QuantityPurchased: 10,
						UnitCost:          500,
						ConversionFactor:  1.0,
					},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				m.CreatePurchaseFunc = func(ctx context.Context, tx *sql.Tx, p *models.Purchase) (int64, error) {
					return 11, nil
				}
				m.CreatePurchaseDetailFunc = func(ctx context.Context, tx *sql.Tx, pd *models.PurchaseDetail) (int64, error) {
					return 2, nil
				}
				m.GetProductStockAndCostFunc = func(ctx context.Context, tx *sql.Tx, productID int64) (float64, float64, error) {
					return 0, 0, nil
				}
				m.UpdateProductAverageCostFunc = func(ctx context.Context, tx *sql.Tx, productID int64, newCost float64) error {
					if newCost != 500 {
						t.Errorf("expected cost 500, got %f", newCost)
					}
					return nil
				}
				m.AddStockFunc = func(ctx context.Context, tx *sql.Tx, productID int64, qty float64) error {
					return nil
				}
				mock.ExpectCommit()
			},
			expectErr:  nil,
			expectedID: 11,
		},
		{
			name: "Missing supplier ID",
			req: service.PurchaseRequest{
				SupplierID: 0,
				Items: []service.PurchaseItemRequest{
					{ProductID: 1, QuantityPurchased: 1, UnitCost: 10},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {},
			expectErr: service.ErrSupplierIDRequired,
		},
		{
			name: "Empty items list",
			req: service.PurchaseRequest{
				SupplierID: 1,
				Items:      []service.PurchaseItemRequest{},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {},
			expectErr: service.ErrItemsRequired,
		},
		{
			name: "Invalid quantity",
			req: service.PurchaseRequest{
				SupplierID: 1,
				Items: []service.PurchaseItemRequest{
					{ProductID: 1, QuantityPurchased: 0, UnitCost: 10},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {},
			expectErr: service.ErrInvalidQuantity,
		},
		{
			name: "Negative unit cost",
			req: service.PurchaseRequest{
				SupplierID: 1,
				Items: []service.PurchaseItemRequest{
					{ProductID: 1, QuantityPurchased: 5, UnitCost: -100},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {},
			expectErr: service.ErrInvalidUnitCost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			repo := &MockPurchaseRepository{}
			tt.mockSetup(repo, mock)

			svc := service.NewPurchaseService(repo, db)
			id, err := svc.CreatePurchase(context.Background(), tt.req)

			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("expected error %v, got %v", tt.expectErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if id != tt.expectedID {
					t.Errorf("expected id %d, got %d", tt.expectedID, id)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func TestListPurchases(t *testing.T) {
	now := time.Now()
	supplierName := "Fruit Supplier"
	samplePurchases := []models.Purchase{
		{
			ID:           1,
			SupplierID:   1,
			SupplierName: &supplierName,
			TotalAmount:  150000,
			PurchaseDate: now,
		},
	}

	tests := []struct {
		name        string
		mockSetup   func(m *MockPurchaseRepository)
		expectedLen int
		expectErr   bool
	}{
		{
			name: "Success with purchases",
			mockSetup: func(m *MockPurchaseRepository) {
				m.ListPurchasesFunc = func(ctx context.Context, supplierID *int64, fromDate, toDate *time.Time) ([]models.Purchase, error) {
					return samplePurchases, nil
				}
			},
			expectedLen: 1,
			expectErr:   false,
		},
		{
			name: "Repo error",
			mockSetup: func(m *MockPurchaseRepository) {
				m.ListPurchasesFunc = func(ctx context.Context, supplierID *int64, fromDate, toDate *time.Time) ([]models.Purchase, error) {
					return nil, errors.New("db error")
				}
			},
			expectedLen: 0,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockPurchaseRepository{}
			tt.mockSetup(repo)

			svc := service.NewPurchaseService(repo, nil)
			list, err := svc.ListPurchases(context.Background(), nil, nil, nil)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error %v, got %v", tt.expectErr, err)
			}
			if !tt.expectErr && len(list) != tt.expectedLen {
				t.Errorf("expected %d purchases, got %d", tt.expectedLen, len(list))
			}
		})
	}
}

func TestGetPurchase(t *testing.T) {
	sample := &models.Purchase{
		ID:          1,
		SupplierID:  1,
		TotalAmount: 50000,
	}
	details := []models.PurchaseDetail{
		{ID: 1, PurchaseID: 1, ProductID: 1, QuantityPurchased: 5, UnitCost: 10000},
	}

	tests := []struct {
		name      string
		id        int64
		mockSetup func(m *MockPurchaseRepository)
		expectErr error
	}{
		{
			name: "Success found with details",
			id:   1,
			mockSetup: func(m *MockPurchaseRepository) {
				m.GetPurchaseByIDFunc = func(ctx context.Context, id int64) (*models.Purchase, error) {
					return sample, nil
				}
				m.GetPurchaseDetailsFunc = func(ctx context.Context, purchaseID int64) ([]models.PurchaseDetail, error) {
					return details, nil
				}
			},
			expectErr: nil,
		},
		{
			name: "Not found",
			id:   99,
			mockSetup: func(m *MockPurchaseRepository) {
				m.GetPurchaseByIDFunc = func(ctx context.Context, id int64) (*models.Purchase, error) {
					return nil, sql.ErrNoRows
				}
			},
			expectErr: service.ErrPurchaseNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockPurchaseRepository{}
			tt.mockSetup(repo)

			svc := service.NewPurchaseService(repo, nil)
			p, err := svc.GetPurchase(context.Background(), tt.id)

			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("expected error %v, got %v", tt.expectErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(p.Details) != 1 {
					t.Errorf("expected 1 detail item, got %d", len(p.Details))
				}
			}
		})
	}
}

func TestUpdatePurchase(t *testing.T) {
	now := time.Now()
	invoice := "INV-REV-2001"
	unitID := int64(2)

	existingPurchase := &models.Purchase{
		ID:           10,
		SupplierID:   1,
		PurchaseDate: now,
		TotalAmount:  100000,
	}

	oldDetails := []models.PurchaseDetail{
		{
			ID:                1,
			PurchaseID:        10,
			ProductID:         1,
			QuantityPurchased: 2,
			UnitCost:          50000,
			ConversionFactor:  1,
		},
	}

	tests := []struct {
		name      string
		id        int64
		req       service.PurchaseRequest
		mockSetup func(m *MockPurchaseRepository, mock sqlmock.Sqlmock)
		expectErr error
	}{
		{
			name: "Success update with stock reversal and recalculation",
			id:   10,
			req: service.PurchaseRequest{
				SupplierID:    2,
				PurchaseDate:  &now,
				InvoiceNumber: &invoice,
				Items: []service.PurchaseItemRequest{
					{
						ProductID:         1,
						PurchaseUnitID:    &unitID,
						QuantityPurchased: 3,
						UnitCost:          60000,
						ConversionFactor:  1,
					},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {
				m.GetPurchaseByIDFunc = func(ctx context.Context, id int64) (*models.Purchase, error) {
					return existingPurchase, nil
				}
				m.GetPurchaseDetailsFunc = func(ctx context.Context, purchaseID int64) ([]models.PurchaseDetail, error) {
					return oldDetails, nil
				}

				mock.ExpectBegin()

				// Step 1: Revert old details
				m.GetProductStockAndCostFunc = func(ctx context.Context, tx *sql.Tx, productID int64) (float64, float64, error) {
					return 5, 50000, nil
				}
				m.UpdateProductAverageCostFunc = func(ctx context.Context, tx *sql.Tx, productID int64, newCost float64) error {
					return nil
				}
				m.DeductStockFunc = func(ctx context.Context, tx *sql.Tx, productID int64, qty float64) error {
					if qty != 2 {
						t.Errorf("expected deducted stock 2, got %f", qty)
					}
					return nil
				}

				// Step 2: Delete old details
				m.DeletePurchaseDetailsFunc = func(ctx context.Context, tx *sql.Tx, purchaseID int64) error {
					if purchaseID != 10 {
						t.Errorf("expected delete for purchase 10, got %d", purchaseID)
					}
					return nil
				}

				// Step 3: Update purchase header
				m.UpdatePurchaseFunc = func(ctx context.Context, tx *sql.Tx, p *models.Purchase) error {
					if p.TotalAmount != 180000 {
						t.Errorf("expected total amount 180000, got %f", p.TotalAmount)
					}
					return nil
				}

				// Step 4: Insert new details & apply stock/cost
				m.CreatePurchaseDetailFunc = func(ctx context.Context, tx *sql.Tx, pd *models.PurchaseDetail) (int64, error) {
					return 2, nil
				}
				m.AddStockFunc = func(ctx context.Context, tx *sql.Tx, productID int64, qty float64) error {
					if qty != 3 {
						t.Errorf("expected stock added 3, got %f", qty)
					}
					return nil
				}

				mock.ExpectCommit()
			},
			expectErr: nil,
		},
		{
			name: "Validation error invalid purchase ID",
			id:   0,
			req: service.PurchaseRequest{
				SupplierID: 1,
				Items: []service.PurchaseItemRequest{
					{ProductID: 1, QuantityPurchased: 1, UnitCost: 100},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {},
			expectErr: service.ErrPurchaseNotFound,
		},
		{
			name: "Validation error missing supplier ID",
			id:   10,
			req: service.PurchaseRequest{
				SupplierID: 0,
				Items: []service.PurchaseItemRequest{
					{ProductID: 1, QuantityPurchased: 1, UnitCost: 100},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {},
			expectErr: service.ErrSupplierIDRequired,
		},
		{
			name: "Validation error empty items",
			id:   10,
			req: service.PurchaseRequest{
				SupplierID: 1,
				Items:      []service.PurchaseItemRequest{},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {},
			expectErr: service.ErrItemsRequired,
		},
		{
			name: "Validation error invalid product ID",
			id:   10,
			req: service.PurchaseRequest{
				SupplierID: 1,
				Items: []service.PurchaseItemRequest{
					{ProductID: 0, QuantityPurchased: 1, UnitCost: 100},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {},
			expectErr: service.ErrInvalidProductID,
		},
		{
			name: "Validation error invalid quantity",
			id:   10,
			req: service.PurchaseRequest{
				SupplierID: 1,
				Items: []service.PurchaseItemRequest{
					{ProductID: 1, QuantityPurchased: 0, UnitCost: 100},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {},
			expectErr: service.ErrInvalidQuantity,
		},
		{
			name: "Validation error negative unit cost",
			id:   10,
			req: service.PurchaseRequest{
				SupplierID: 1,
				Items: []service.PurchaseItemRequest{
					{ProductID: 1, QuantityPurchased: 1, UnitCost: -10},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {},
			expectErr: service.ErrInvalidUnitCost,
		},
		{
			name: "Purchase not found in DB",
			id:   999,
			req: service.PurchaseRequest{
				SupplierID: 1,
				Items: []service.PurchaseItemRequest{
					{ProductID: 1, QuantityPurchased: 1, UnitCost: 100},
				},
			},
			mockSetup: func(m *MockPurchaseRepository, mock sqlmock.Sqlmock) {
				m.GetPurchaseByIDFunc = func(ctx context.Context, id int64) (*models.Purchase, error) {
					return nil, sql.ErrNoRows
				}
			},
			expectErr: service.ErrPurchaseNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mockSQL, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			repo := &MockPurchaseRepository{}
			tt.mockSetup(repo, mockSQL)

			svc := service.NewPurchaseService(repo, db)
			err = svc.UpdatePurchase(context.Background(), tt.id, tt.req)

			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("expected error %v, got %v", tt.expectErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if err := mockSQL.ExpectationsWereMet(); err != nil {
					t.Errorf("unfulfilled sqlmock expectations: %v", err)
				}
			}
		})
	}
}

