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

func TestSQLEmailReceiptRepository_GetRevenueSummary(t *testing.T) {
	now := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	from := now.AddDate(0, 0, -7)
	to := now

	tests := []struct {
		name          string
		mockExpect    func(mock sqlmock.Sqlmock)
		from          time.Time
		to            time.Time
		wantRevenue   float64
		wantCount     int
		wantGrowth    float64
		wantAvgTicket float64
		wantErr       bool
	}{
		{
			name: "success with revenue and growth",
			from: from,
			to:   to,
			mockExpect: func(mock sqlmock.Sqlmock) {
				summaryRows := sqlmock.NewRows([]string{"cur_rev", "cur_count", "prev_rev"}).
					AddRow(100000.0, 5, 80000.0)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to, from.Add(-to.Sub(from))).
					WillReturnRows(summaryRows)

				timelineRows := sqlmock.NewRows([]string{"day_date", "amount", "count"}).
					AddRow("2026-09-18", 40000.0, 2).
					AddRow("2026-09-19", 60000.0, 3)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to).
					WillReturnRows(timelineRows)
			},
			wantRevenue:   100000.0,
			wantCount:     5,
			wantGrowth:    25.0,
			wantAvgTicket: 20000.0,
			wantErr:       false,
		},
		{
			name: "zero previous revenue gives 100 percent growth",
			from: from,
			to:   to,
			mockExpect: func(mock sqlmock.Sqlmock) {
				summaryRows := sqlmock.NewRows([]string{"cur_rev", "cur_count", "prev_rev"}).
					AddRow(50000.0, 2, 0.0)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to, from.Add(-to.Sub(from))).
					WillReturnRows(summaryRows)

				timelineRows := sqlmock.NewRows([]string{"day_date", "amount", "count"}).
					AddRow("2026-09-20", 50000.0, 2)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to).
					WillReturnRows(timelineRows)
			},
			wantRevenue:   50000.0,
			wantCount:     2,
			wantGrowth:    100.0,
			wantAvgTicket: 25000.0,
			wantErr:       false,
		},
		{
			name: "error querying summary",
			from: from,
			to:   to,
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to, from.Add(-to.Sub(from))).
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "error querying timeline",
			from: from,
			to:   to,
			mockExpect: func(mock sqlmock.Sqlmock) {
				summaryRows := sqlmock.NewRows([]string{"cur_rev", "cur_count", "prev_rev"}).
					AddRow(50000.0, 2, 50000.0)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to, from.Add(-to.Sub(from))).
					WillReturnRows(summaryRows)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to).
					WillReturnError(errors.New("timeline db error"))
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

			tt.mockExpect(mock)

			repo := repository.NewSQLEmailReceiptRepository(db)
			res, err := repo.GetRevenueSummary(context.Background(), tt.from, tt.to)

			if (err != nil) != tt.wantErr {
				t.Fatalf("GetRevenueSummary() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if res.TotalRevenue != tt.wantRevenue {
					t.Errorf("got TotalRevenue %v, want %v", res.TotalRevenue, tt.wantRevenue)
				}
				if res.TransactionCount != tt.wantCount {
					t.Errorf("got TransactionCount %v, want %v", res.TransactionCount, tt.wantCount)
				}
				if res.GrowthPercentage != tt.wantGrowth {
					t.Errorf("got GrowthPercentage %v, want %v", res.GrowthPercentage, tt.wantGrowth)
				}
				if res.AverageTicket != tt.wantAvgTicket {
					t.Errorf("got AverageTicket %v, want %v", res.AverageTicket, tt.wantAvgTicket)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestSQLEmailReceiptRepository_UpsertAndList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := repository.NewSQLEmailReceiptRepository(db)

	t.Run("upsert success", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO email_receipts")).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(123))

		inserted, err := repo.UpsertByMessageID(context.Background(), &models.EmailReceipt{
			MessageID: "msg-123",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !inserted {
			t.Errorf("expected inserted=true")
		}
	})

	t.Run("list success", func(t *testing.T) {
		now := time.Now()
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, message_id")).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "message_id", "sender", "subject", "received_at", "transaction_date",
				"amount", "currency", "payer", "bank", "reference", "transaction_number",
				"payment_method", "status", "parse_error", "created_at",
			}).AddRow(
				1, "msg-1", "s@test.com", "Sub", now, now, 50000.0, "COP",
				"Payer", "Bank", "Ref", "Tx1", "QR", "imported", nil, now,
			))

		list, err := repo.List(context.Background(), nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("expected 1 receipt, got %d", len(list))
		}
	})
}
