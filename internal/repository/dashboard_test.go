package repository_test

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
)

func TestSQLDashboardRepository_GetIncomeVsPurchasesChart(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		mockExpect func(mock sqlmock.Sqlmock)
		want       []models.IncomeVsPurchasesChartEntry
		wantErr    bool
	}{
		{
			name: "merges income and purchases sorted by date",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("FROM email_receipts")).
					WithArgs(from, to).
					WillReturnRows(sqlmock.NewRows([]string{"day", "income"}).
						AddRow("2026-01-01", 100.0).
						AddRow("2026-01-02", 50.0))
				mock.ExpectQuery(regexp.QuoteMeta("FROM purchases")).
					WithArgs(from, to).
					WillReturnRows(sqlmock.NewRows([]string{"day", "purchase"}).
						AddRow("2026-01-02", 30.0).
						AddRow("2026-01-03", 10.0))
			},
			want: []models.IncomeVsPurchasesChartEntry{
				{Date: "2026-01-01", IncomeAmount: 100.0, PurchaseAmount: 0.0},
				{Date: "2026-01-02", IncomeAmount: 50.0, PurchaseAmount: 30.0},
				{Date: "2026-01-03", IncomeAmount: 0.0, PurchaseAmount: 10.0},
			},
			wantErr: false,
		},
		{
			name: "empty range returns no entries",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("FROM email_receipts")).
					WithArgs(from, to).
					WillReturnRows(sqlmock.NewRows([]string{"day", "income"}))
				mock.ExpectQuery(regexp.QuoteMeta("FROM purchases")).
					WithArgs(from, to).
					WillReturnRows(sqlmock.NewRows([]string{"day", "purchase"}))
			},
			want:    []models.IncomeVsPurchasesChartEntry{},
			wantErr: false,
		},
		{
			name: "income query error propagates",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("FROM email_receipts")).
					WithArgs(from, to).
					WillReturnError(errors.New("income query failed"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "purchase query error propagates",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("FROM email_receipts")).
					WithArgs(from, to).
					WillReturnRows(sqlmock.NewRows([]string{"day", "income"}).
						AddRow("2026-01-01", 5.0))
				mock.ExpectQuery(regexp.QuoteMeta("FROM purchases")).
					WithArgs(from, to).
					WillReturnError(errors.New("purchase query failed"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock: %v", err)
			}
			defer db.Close()

			tt.mockExpect(mock)

			repo := repository.NewSQLDashboardRepository(db)
			got, err := repo.GetIncomeVsPurchasesChart(context.Background(), from, to)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d entries, got %d", len(tt.want), len(got))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("entry %d: want %+v, got %+v", i, tt.want[i], got[i])
				}
			}
		})
	}
}
