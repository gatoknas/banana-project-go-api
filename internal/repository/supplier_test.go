package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
)

func supplierStrPtr(s string) *string {
	return &s
}

func TestSQLSupplierRepository_Create(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	phone := "+57 300 123 4567, +57 310 987 6543"
	address := "Cra 10 #20-30, Calle 5 #6-7"

	tests := []struct {
		name       string
		supplier   *models.Supplier
		mockExpect func(mock sqlmock.Sqlmock)
		wantID     int64
		wantErr    bool
	}{
		{
			name: "success inserts address and returns id",
			supplier: &models.Supplier{
				CompanyName: "Distribuidora Frutas SAS",
				Phone:       &phone,
				Address:     &address,
			},
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO suppliers")).
					WithArgs(nil, "Distribuidora Frutas SAS", nil, phone, address, nil, nil).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(7), now))
			},
			wantID:  7,
			wantErr: false,
		},
		{
			name: "nil address persists as NULL",
			supplier: &models.Supplier{
				CompanyName: "Distribuidora Frutas SAS",
				Phone:       &phone,
			},
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO suppliers")).
					WithArgs(nil, "Distribuidora Frutas SAS", nil, phone, nil, nil, nil).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(8), now))
			},
			wantID:  8,
			wantErr: false,
		},
		{
			name: "database error propagates",
			supplier: &models.Supplier{
				CompanyName: "Distribuidora Frutas SAS",
				Phone:       &phone,
			},
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO suppliers")).
					WithArgs(nil, "Distribuidora Frutas SAS", nil, phone, nil, nil, nil).
					WillReturnError(errors.New("insert failed"))
			},
			wantID:  0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			tt.mockExpect(mock)

			repo := repository.NewSQLSupplierRepository(db)
			id, err := repo.Create(context.Background(), nil, tt.supplier)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if id != tt.wantID {
					t.Errorf("expected id %d, got %d", tt.wantID, id)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func TestSQLSupplierRepository_GetByID(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	phone := "+57 300 123 4567, +57 310 987 6543"
	address := "Cra 10 #20-30, Calle 5 #6-7"

	cols := []string{"id", "tax_id", "company_name", "contact_name", "phone", "address", "email", "description", "created_at"}

	tests := []struct {
		name       string
		id         int64
		mockExpect func(mock sqlmock.Sqlmock)
		wantAddr   *string
		wantErr    bool
	}{
		{
			name: "success returns address",
			id:   1,
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("FROM suppliers WHERE id = $1")).
					WithArgs(int64(1)).
					WillReturnRows(sqlmock.NewRows(cols).AddRow(
						int64(1), "900123456-7", "Distribuidora Frutas SAS", "Carlos Mendoza",
						phone, address, "contacto@distrifrutera.com", "notas", now,
					))
			},
			wantAddr: supplierStrPtr(address),
			wantErr:  false,
		},
		{
			name: "not found returns error",
			id:   99,
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("FROM suppliers WHERE id = $1")).
					WithArgs(int64(99)).
					WillReturnError(sql.ErrNoRows)
			},
			wantAddr: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			tt.mockExpect(mock)

			repo := repository.NewSQLSupplierRepository(db)
			got, err := repo.GetByID(context.Background(), tt.id)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.Address == nil || *got.Address != *tt.wantAddr {
					t.Errorf("expected address %q, got %v", *tt.wantAddr, got.Address)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func TestSQLSupplierRepository_List(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	phone := "+57 300 123 4567"
	address := "Cra 10 #20-30"

	cols := []string{"id", "tax_id", "company_name", "contact_name", "phone", "address", "email", "description", "created_at"}

	tests := []struct {
		name       string
		search     string
		mockExpect func(mock sqlmock.Sqlmock)
		wantLen    int
		wantErr    bool
	}{
		{
			name:   "success without search includes address",
			search: "",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, tax_id, company_name, contact_name, phone, address, email, description, created_at")).
					WillReturnRows(sqlmock.NewRows(cols).AddRow(
						int64(1), "900123456-7", "Distribuidora Frutas SAS", "Carlos Mendoza",
						phone, address, "contacto@distrifrutera.com", nil, now,
					))
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:   "success with search filters",
			search: "Frutas",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("WHERE company_name ILIKE $1")).
					WithArgs("%Frutas%").
					WillReturnRows(sqlmock.NewRows(cols))
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:   "query error propagates",
			search: "",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, tax_id, company_name, contact_name, phone, address, email, description, created_at")).
					WillReturnError(errors.New("query failed"))
			},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			tt.mockExpect(mock)

			repo := repository.NewSQLSupplierRepository(db)
			got, err := repo.List(context.Background(), tt.search)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(got) != tt.wantLen {
					t.Errorf("expected %d suppliers, got %d", tt.wantLen, len(got))
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func TestSQLSupplierRepository_Update(t *testing.T) {
	phone := "+57 300 123 4567"
	address := "Av 1 #2-3, Barrio Centro"

	tests := []struct {
		name       string
		supplier   *models.Supplier
		mockExpect func(mock sqlmock.Sqlmock)
		wantErr    error
	}{
		{
			name: "success updates address",
			supplier: &models.Supplier{
				ID:          1,
				CompanyName: "Distribuidora Frutas SAS",
				Phone:       &phone,
				Address:     &address,
			},
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("UPDATE suppliers")).
					WithArgs(nil, "Distribuidora Frutas SAS", nil, phone, address, nil, nil, int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name: "no rows affected returns ErrNoRows",
			supplier: &models.Supplier{
				ID:          99,
				CompanyName: "Distribuidora Frutas SAS",
				Phone:       &phone,
			},
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("UPDATE suppliers")).
					WithArgs(nil, "Distribuidora Frutas SAS", nil, phone, nil, nil, nil, int64(99)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			tt.mockExpect(mock)

			repo := repository.NewSQLSupplierRepository(db)
			err = repo.Update(context.Background(), tt.supplier)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}
