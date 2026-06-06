package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type MockProductRepository struct {
	GetByIDFunc                 func(ctx context.Context, id int64) (*models.Product, error)
	ListFunc                    func(ctx context.Context) ([]models.Product, error)
	CreateFunc                  func(ctx context.Context, tx *sql.Tx, p *models.Product) (int64, error)
	UpdateFunc                  func(ctx context.Context, p *models.Product) error
	DeleteFunc                  func(ctx context.Context, tx *sql.Tx, id int64) error
	InitializeInventoryFunc     func(ctx context.Context, tx *sql.Tx, productID int64) error
	DeleteAllowedAdditionsFunc  func(ctx context.Context, tx *sql.Tx, productID int64) error
	DeleteProductRecipesFunc    func(ctx context.Context, tx *sql.Tx, productID int64) error
	DeleteInventoryFunc         func(ctx context.Context, tx *sql.Tx, productID int64) error
}

func (m *MockProductRepository) GetByID(ctx context.Context, id int64) (*models.Product, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *MockProductRepository) List(ctx context.Context) ([]models.Product, error) {
	return m.ListFunc(ctx)
}

func (m *MockProductRepository) Create(ctx context.Context, tx *sql.Tx, p *models.Product) (int64, error) {
	return m.CreateFunc(ctx, tx, p)
}

func (m *MockProductRepository) Update(ctx context.Context, p *models.Product) error {
	return m.UpdateFunc(ctx, p)
}

func (m *MockProductRepository) Delete(ctx context.Context, tx *sql.Tx, id int64) error {
	return m.DeleteFunc(ctx, tx, id)
}

func (m *MockProductRepository) InitializeInventory(ctx context.Context, tx *sql.Tx, productID int64) error {
	return m.InitializeInventoryFunc(ctx, tx, productID)
}

func (m *MockProductRepository) DeleteAllowedAdditions(ctx context.Context, tx *sql.Tx, productID int64) error {
	return m.DeleteAllowedAdditionsFunc(ctx, tx, productID)
}

func (m *MockProductRepository) DeleteProductRecipes(ctx context.Context, tx *sql.Tx, productID int64) error {
	return m.DeleteProductRecipesFunc(ctx, tx, productID)
}

func (m *MockProductRepository) DeleteInventory(ctx context.Context, tx *sql.Tx, productID int64) error {
	return m.DeleteInventoryFunc(ctx, tx, productID)
}

func TestCreateProduct(t *testing.T) {
	desc := "Delicious banana product"
	tests := []struct {
		name           string
		req            service.ProductRequest
		mockSetup      func(m *MockProductRepository, mock sqlmock.Sqlmock)
		expectedID     int64
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name: "Success",
			req: service.ProductRequest{
				Name:        "Banana Milkshake",
				Description: &desc,
				CategoryID:  1,
			},
			mockSetup: func(m *MockProductRepository, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				m.CreateFunc = func(ctx context.Context, tx *sql.Tx, p *models.Product) (int64, error) {
					return 100, nil
				}
				m.InitializeInventoryFunc = func(ctx context.Context, tx *sql.Tx, productID int64) error {
					return nil
				}
				mock.ExpectCommit()
			},
			expectedID: 100,
			expectErr:  false,
		},
		{
			name: "Missing Name",
			req: service.ProductRequest{
				Name: "",
			},
			mockSetup:      func(m *MockProductRepository, mock sqlmock.Sqlmock) {},
			expectedID:     0,
			expectErr:      true,
			expectedErrMsg: "name is required",
		},
		{
			name: "Create Fails",
			req: service.ProductRequest{
				Name: "Banana Bread",
			},
			mockSetup: func(m *MockProductRepository, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				m.CreateFunc = func(ctx context.Context, tx *sql.Tx, p *models.Product) (int64, error) {
					return 0, errors.New("db error")
				}
				mock.ExpectRollback()
			},
			expectedID: 0,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %s", err)
			}
			defer db.Close()

			repo := &MockProductRepository{}
			tt.mockSetup(repo, mock)

			svc := service.NewProductService(repo, db)
			id, err := svc.CreateProduct(context.Background(), tt.req)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
					t.Errorf("expected error %q, got %q", tt.expectedErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
				if id != tt.expectedID {
					t.Errorf("expected ID %d, got %d", tt.expectedID, id)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet mock expectations: %s", err)
			}
		})
	}
}
