package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type MockSupplierRepository struct {
	CreateFunc       func(ctx context.Context, tx *sql.Tx, s *models.Supplier) (int64, error)
	GetByIDFunc      func(ctx context.Context, id int64) (*models.Supplier, error)
	GetByTaxIDFunc   func(ctx context.Context, taxID string) (*models.Supplier, error)
	ListFunc         func(ctx context.Context, searchQuery string) ([]models.Supplier, error)
	UpdateFunc       func(ctx context.Context, s *models.Supplier) error
	DeleteFunc       func(ctx context.Context, tx *sql.Tx, id int64) error
	HasPurchasesFunc func(ctx context.Context, id int64) (bool, error)
}

func (m *MockSupplierRepository) Create(ctx context.Context, tx *sql.Tx, s *models.Supplier) (int64, error) {
	return m.CreateFunc(ctx, tx, s)
}

func (m *MockSupplierRepository) GetByID(ctx context.Context, id int64) (*models.Supplier, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *MockSupplierRepository) GetByTaxID(ctx context.Context, taxID string) (*models.Supplier, error) {
	return m.GetByTaxIDFunc(ctx, taxID)
}

func (m *MockSupplierRepository) List(ctx context.Context, searchQuery string) ([]models.Supplier, error) {
	return m.ListFunc(ctx, searchQuery)
}

func (m *MockSupplierRepository) Update(ctx context.Context, s *models.Supplier) error {
	return m.UpdateFunc(ctx, s)
}

func (m *MockSupplierRepository) Delete(ctx context.Context, tx *sql.Tx, id int64) error {
	return m.DeleteFunc(ctx, tx, id)
}

func (m *MockSupplierRepository) HasPurchases(ctx context.Context, id int64) (bool, error) {
	return m.HasPurchasesFunc(ctx, id)
}

func TestCreateSupplier(t *testing.T) {
	tests := []struct {
		name      string
		req       service.SupplierRequest
		mockSetup func(m *MockSupplierRepository)
		expectErr error
	}{
		{
			name: "Success",
			req: service.SupplierRequest{
				TaxID:       "900123456-1",
				CompanyName: "Distribuidora Frutas SAS",
			},
			mockSetup: func(m *MockSupplierRepository) {
				m.GetByTaxIDFunc = func(ctx context.Context, taxID string) (*models.Supplier, error) {
					return nil, sql.ErrNoRows
				}
				m.CreateFunc = func(ctx context.Context, tx *sql.Tx, s *models.Supplier) (int64, error) {
					return 1, nil
				}
			},
			expectErr: nil,
		},
		{
			name: "Missing company name",
			req: service.SupplierRequest{
				TaxID:       "900123456-1",
				CompanyName: "",
			},
			mockSetup: func(m *MockSupplierRepository) {},
			expectErr: service.ErrCompanyNameRequired,
		},
		{
			name: "Missing tax ID",
			req: service.SupplierRequest{
				TaxID:       "",
				CompanyName: "Distribuidora Frutas SAS",
			},
			mockSetup: func(m *MockSupplierRepository) {},
			expectErr: service.ErrTaxIDRequired,
		},
		{
			name: "Tax ID already exists",
			req: service.SupplierRequest{
				TaxID:       "900123456-1",
				CompanyName: "Distribuidora Frutas SAS",
			},
			mockSetup: func(m *MockSupplierRepository) {
				m.GetByTaxIDFunc = func(ctx context.Context, taxID string) (*models.Supplier, error) {
					return &models.Supplier{ID: 2, TaxID: "900123456-1"}, nil
				}
			},
			expectErr: service.ErrTaxIDAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockSupplierRepository{}
			tt.mockSetup(mock)

			svc := service.NewSupplierService(mock, nil)
			id, err := svc.CreateSupplier(context.Background(), tt.req)

			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("expected error %v, got %v", tt.expectErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if id != 1 {
					t.Errorf("expected id 1, got %d", id)
				}
			}
		})
	}
}

func TestListSuppliers(t *testing.T) {
	now := time.Now()
	sampleSuppliers := []models.Supplier{
		{ID: 1, TaxID: "123", CompanyName: "Fruit Supplier", CreatedAt: now},
		{ID: 2, TaxID: "456", CompanyName: "Dairy Supplier", CreatedAt: now},
	}

	tests := []struct {
		name        string
		search      string
		mockSetup   func(m *MockSupplierRepository)
		expectedLen int
		expectErr   bool
	}{
		{
			name:   "Success list",
			search: "",
			mockSetup: func(m *MockSupplierRepository) {
				m.ListFunc = func(ctx context.Context, searchQuery string) ([]models.Supplier, error) {
					return sampleSuppliers, nil
				}
			},
			expectedLen: 2,
			expectErr:   false,
		},
		{
			name:   "Repo error",
			search: "fruit",
			mockSetup: func(m *MockSupplierRepository) {
				m.ListFunc = func(ctx context.Context, searchQuery string) ([]models.Supplier, error) {
					return nil, errors.New("db error")
				}
			},
			expectedLen: 0,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockSupplierRepository{}
			tt.mockSetup(mock)

			svc := service.NewSupplierService(mock, nil)
			list, err := svc.ListSuppliers(context.Background(), tt.search)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got %v", tt.expectErr, err)
			}
			if !tt.expectErr && len(list) != tt.expectedLen {
				t.Errorf("expected %d suppliers, got %d", tt.expectedLen, len(list))
			}
		})
	}
}

func TestGetSupplier(t *testing.T) {
	sample := &models.Supplier{ID: 1, TaxID: "123", CompanyName: "Fruit Supplier"}

	tests := []struct {
		name      string
		id        int64
		mockSetup func(m *MockSupplierRepository)
		expected  *models.Supplier
		expectErr error
	}{
		{
			name: "Success found",
			id:   1,
			mockSetup: func(m *MockSupplierRepository) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return sample, nil
				}
			},
			expected:  sample,
			expectErr: nil,
		},
		{
			name: "Not found",
			id:   99,
			mockSetup: func(m *MockSupplierRepository) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return nil, sql.ErrNoRows
				}
			},
			expected:  nil,
			expectErr: service.ErrSupplierNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockSupplierRepository{}
			tt.mockSetup(mock)

			svc := service.NewSupplierService(mock, nil)
			res, err := svc.GetSupplier(context.Background(), tt.id)

			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("expected error %v, got %v", tt.expectErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.ID != tt.expected.ID {
					t.Errorf("expected supplier %d, got %d", tt.expected.ID, res.ID)
				}
			}
		})
	}
}

func TestUpdateSupplier(t *testing.T) {
	existing := &models.Supplier{ID: 1, TaxID: "123", CompanyName: "Fruit Supplier"}

	tests := []struct {
		name      string
		id        int64
		req       service.SupplierRequest
		mockSetup func(m *MockSupplierRepository)
		expectErr error
	}{
		{
			name: "Success update",
			id:   1,
			req: service.SupplierRequest{
				TaxID:       "123",
				CompanyName: "Fruit Supplier New Name",
			},
			mockSetup: func(m *MockSupplierRepository) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return existing, nil
				}
				m.UpdateFunc = func(ctx context.Context, s *models.Supplier) error {
					return nil
				}
			},
			expectErr: nil,
		},
		{
			name: "Tax ID conflict with another supplier",
			id:   1,
			req: service.SupplierRequest{
				TaxID:       "456",
				CompanyName: "Fruit Supplier",
			},
			mockSetup: func(m *MockSupplierRepository) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return existing, nil
				}
				m.GetByTaxIDFunc = func(ctx context.Context, taxID string) (*models.Supplier, error) {
					return &models.Supplier{ID: 2, TaxID: "456"}, nil
				}
			},
			expectErr: service.ErrTaxIDAlreadyExists,
		},
		{
			name: "Supplier not found",
			id:   99,
			req: service.SupplierRequest{
				TaxID:       "123",
				CompanyName: "Name",
			},
			mockSetup: func(m *MockSupplierRepository) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return nil, sql.ErrNoRows
				}
			},
			expectErr: service.ErrSupplierNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockSupplierRepository{}
			tt.mockSetup(mock)

			svc := service.NewSupplierService(mock, nil)
			err := svc.UpdateSupplier(context.Background(), tt.id, tt.req)

			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("expected error %v, got %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDeleteSupplier(t *testing.T) {
	existing := &models.Supplier{ID: 1, TaxID: "123", CompanyName: "Fruit Supplier"}

	tests := []struct {
		name      string
		id        int64
		mockSetup func(m *MockSupplierRepository)
		expectErr error
	}{
		{
			name: "Success delete",
			id:   1,
			mockSetup: func(m *MockSupplierRepository) {
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
			expectErr: nil,
		},
		{
			name: "Has purchases error",
			id:   1,
			mockSetup: func(m *MockSupplierRepository) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return existing, nil
				}
				m.HasPurchasesFunc = func(ctx context.Context, id int64) (bool, error) {
					return true, nil
				}
			},
			expectErr: service.ErrSupplierHasPurchases,
		},
		{
			name: "Not found",
			id:   99,
			mockSetup: func(m *MockSupplierRepository) {
				m.GetByIDFunc = func(ctx context.Context, id int64) (*models.Supplier, error) {
					return nil, sql.ErrNoRows
				}
			},
			expectErr: service.ErrSupplierNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockSupplierRepository{}
			tt.mockSetup(mock)

			svc := service.NewSupplierService(mock, nil)
			err := svc.DeleteSupplier(context.Background(), tt.id)

			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("expected error %v, got %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
