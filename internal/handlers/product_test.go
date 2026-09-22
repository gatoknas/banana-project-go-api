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

type MockProductRepo struct {
	GetByIDFunc                func(ctx context.Context, id int64) (*models.Product, error)
	ListFunc                   func(ctx context.Context) ([]models.Product, error)
	CreateFunc                 func(ctx context.Context, tx *sql.Tx, p *models.Product) (int64, error)
	UpdateFunc                 func(ctx context.Context, p *models.Product) error
	DeleteFunc                 func(ctx context.Context, tx *sql.Tx, id int64) error
	InitializeInventoryFunc    func(ctx context.Context, tx *sql.Tx, productID int64) error
	DeleteAllowedAdditionsFunc func(ctx context.Context, tx *sql.Tx, productID int64) error
	DeleteProductRecipesFunc   func(ctx context.Context, tx *sql.Tx, productID int64) error
	DeleteInventoryFunc        func(ctx context.Context, tx *sql.Tx, productID int64) error
}

func (m *MockProductRepo) GetByID(ctx context.Context, id int64) (*models.Product, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *MockProductRepo) List(ctx context.Context) ([]models.Product, error) {
	return m.ListFunc(ctx)
}

func (m *MockProductRepo) Create(ctx context.Context, tx *sql.Tx, p *models.Product) (int64, error) {
	return m.CreateFunc(ctx, tx, p)
}

func (m *MockProductRepo) Update(ctx context.Context, p *models.Product) error {
	return m.UpdateFunc(ctx, p)
}

func (m *MockProductRepo) Delete(ctx context.Context, tx *sql.Tx, id int64) error {
	return m.DeleteFunc(ctx, tx, id)
}

func (m *MockProductRepo) InitializeInventory(ctx context.Context, tx *sql.Tx, productID int64) error {
	return m.InitializeInventoryFunc(ctx, tx, productID)
}

func (m *MockProductRepo) DeleteAllowedAdditions(ctx context.Context, tx *sql.Tx, productID int64) error {
	return m.DeleteAllowedAdditionsFunc(ctx, tx, productID)
}

func (m *MockProductRepo) DeleteProductRecipes(ctx context.Context, tx *sql.Tx, productID int64) error {
	return m.DeleteProductRecipesFunc(ctx, tx, productID)
}

func (m *MockProductRepo) DeleteInventory(ctx context.Context, tx *sql.Tx, productID int64) error {
	return m.DeleteInventoryFunc(ctx, tx, productID)
}

func newProductHandler(repo *MockProductRepo, db *sql.DB) *handlers.ProductHandler {
	return handlers.NewProductHandler(service.NewProductService(repo, db), zap.NewNop())
}

func newSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	return db, mock
}

func TestProductHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		repo       *MockProductRepo
		expect     func(mock sqlmock.Sqlmock)
		wantStatus int
	}{
		{
			name: "success",
			body: `{"name":"Banana Shake"}`,
			repo: &MockProductRepo{
				CreateFunc:              func(ctx context.Context, tx *sql.Tx, p *models.Product) (int64, error) { return 100, nil },
				InitializeInventoryFunc: func(ctx context.Context, tx *sql.Tx, productID int64) error { return nil },
			},
			expect:     func(mock sqlmock.Sqlmock) { mock.ExpectBegin(); mock.ExpectCommit() },
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json",
			body:       `{`,
			repo:       &MockProductRepo{},
			expect:     func(mock sqlmock.Sqlmock) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing name",
			body:       `{"name":""}`,
			repo:       &MockProductRepo{},
			expect:     func(mock sqlmock.Sqlmock) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "create fails",
			body: `{"name":"Banana Bread"}`,
			repo: &MockProductRepo{
				CreateFunc: func(ctx context.Context, tx *sql.Tx, p *models.Product) (int64, error) {
					return 0, errors.New("db error")
				},
			},
			expect:     func(mock sqlmock.Sqlmock) { mock.ExpectBegin(); mock.ExpectRollback() },
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "inventory init fails",
			body: `{"name":"Banana Bread"}`,
			repo: &MockProductRepo{
				CreateFunc: func(ctx context.Context, tx *sql.Tx, p *models.Product) (int64, error) { return 7, nil },
				InitializeInventoryFunc: func(ctx context.Context, tx *sql.Tx, productID int64) error {
					return errors.New("db error")
				},
			},
			expect:     func(mock sqlmock.Sqlmock) { mock.ExpectBegin(); mock.ExpectRollback() },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			defer db.Close()
			tt.expect(mock)

			h := newProductHandler(tt.repo, db)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/products", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.Create(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantStatus == http.StatusCreated {
				var resp handlers.MessageResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.ID != 100 {
					t.Errorf("expected id 100, got %d", resp.ID)
				}
			}
		})
	}
}

func TestProductHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		repo       *MockProductRepo
		wantStatus int
		wantCount  int
	}{
		{
			name: "success",
			repo: &MockProductRepo{
				ListFunc: func(ctx context.Context) ([]models.Product, error) {
					return []models.Product{{ID: 1, Name: "A"}}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name: "repository error",
			repo: &MockProductRepo{
				ListFunc: func(ctx context.Context) ([]models.Product, error) {
					return nil, errors.New("db error")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newProductHandler(tt.repo, nil)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
			w := httptest.NewRecorder()

			h.List(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantStatus == http.StatusOK {
				var resp []models.Product
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(resp) != tt.wantCount {
					t.Errorf("expected %d products, got %d", tt.wantCount, len(resp))
				}
			}
		})
	}
}

func TestProductHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		repo       *MockProductRepo
		wantStatus int
	}{
		{
			name: "success",
			id:   "1",
			repo: &MockProductRepo{
				GetByIDFunc: func(ctx context.Context, id int64) (*models.Product, error) {
					return &models.Product{ID: id, Name: "A"}, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			id:         "abc",
			repo:       &MockProductRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			id:   "9",
			repo: &MockProductRepo{
				GetByIDFunc: func(ctx context.Context, id int64) (*models.Product, error) {
					return nil, sql.ErrNoRows
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "repository error",
			id:   "9",
			repo: &MockProductRepo{
				GetByIDFunc: func(ctx context.Context, id int64) (*models.Product, error) {
					return nil, errors.New("db error")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newProductHandler(tt.repo, nil)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+tt.id, nil)
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			h.Get(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestProductHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		repo       *MockProductRepo
		wantStatus int
	}{
		{
			name: "success",
			id:   "1",
			body: `{"name":"Updated"}`,
			repo: &MockProductRepo{
				UpdateFunc: func(ctx context.Context, p *models.Product) error { return nil },
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			id:         "abc",
			body:       `{"name":"X"}`,
			repo:       &MockProductRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			id:         "1",
			body:       `{`,
			repo:       &MockProductRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing name",
			id:         "1",
			body:       `{"name":""}`,
			repo:       &MockProductRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			id:   "9",
			body: `{"name":"X"}`,
			repo: &MockProductRepo{
				UpdateFunc: func(ctx context.Context, p *models.Product) error { return sql.ErrNoRows },
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "repository error",
			id:   "9",
			body: `{"name":"X"}`,
			repo: &MockProductRepo{
				UpdateFunc: func(ctx context.Context, p *models.Product) error { return errors.New("db error") },
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newProductHandler(tt.repo, nil)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/products/"+tt.id, strings.NewReader(tt.body))
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			h.Update(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestProductHandler_Delete(t *testing.T) {
	okRepo := func(deleteErr error) *MockProductRepo {
		return &MockProductRepo{
			DeleteAllowedAdditionsFunc: func(ctx context.Context, tx *sql.Tx, productID int64) error { return nil },
			DeleteProductRecipesFunc:   func(ctx context.Context, tx *sql.Tx, productID int64) error { return nil },
			DeleteInventoryFunc:        func(ctx context.Context, tx *sql.Tx, productID int64) error { return nil },
			DeleteFunc:                 func(ctx context.Context, tx *sql.Tx, id int64) error { return deleteErr },
		}
	}

	tests := []struct {
		name       string
		id         string
		repo       *MockProductRepo
		expect     func(mock sqlmock.Sqlmock)
		wantStatus int
	}{
		{
			name:       "success",
			id:         "1",
			repo:       okRepo(nil),
			expect:     func(mock sqlmock.Sqlmock) { mock.ExpectBegin(); mock.ExpectCommit() },
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			id:         "abc",
			repo:       &MockProductRepo{},
			expect:     func(mock sqlmock.Sqlmock) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			id:         "9",
			repo:       okRepo(sql.ErrNoRows),
			expect:     func(mock sqlmock.Sqlmock) { mock.ExpectBegin(); mock.ExpectRollback() },
			wantStatus: http.StatusNotFound,
		},
		{
			name: "repository error",
			id:   "9",
			repo: &MockProductRepo{
				DeleteAllowedAdditionsFunc: func(ctx context.Context, tx *sql.Tx, productID int64) error {
					return errors.New("db error")
				},
			},
			expect:     func(mock sqlmock.Sqlmock) { mock.ExpectBegin(); mock.ExpectRollback() },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			defer db.Close()
			tt.expect(mock)

			h := newProductHandler(tt.repo, db)
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/"+tt.id, nil)
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			h.Delete(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
