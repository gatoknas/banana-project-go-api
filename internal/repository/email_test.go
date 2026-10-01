package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
)

func newStatusRejectingMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
		normalized := strings.ToLower(actualSQL)
		if strings.Contains(normalized, "status") {
			return fmt.Errorf("query must not filter by status: %s", actualSQL)
		}
		if !strings.Contains(normalized, "from email_receipts") {
			return fmt.Errorf("unexpected query: %s", actualSQL)
		}
		return nil
	})))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	return db, mock
}

func TestSQLEmailReceiptRepository_GetRevenueSummary(t *testing.T) {
	now := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	from := now.AddDate(0, 0, -7)
	to := now

	tests := []struct {
		name          string
		mockExpect    func(mock sqlmock.Sqlmock)
		from          time.Time
		to            time.Time
		interval      string
		wantRevenue   float64
		wantCount     int
		wantGrowth    float64
		wantAvgTicket float64
		wantMinSlots  int
		wantErr       bool
	}{
		{
			name:     "success with revenue and growth default interval",
			from:     from,
			to:       to,
			interval: "day",
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
			name:     "success with hourly interval covers 06 to 19",
			from:     from,
			to:       to,
			interval: "hour",
			mockExpect: func(mock sqlmock.Sqlmock) {
				summaryRows := sqlmock.NewRows([]string{"cur_rev", "cur_count", "prev_rev"}).
					AddRow(50000.0, 2, 0.0)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to, from.Add(-to.Sub(from))).
					WillReturnRows(summaryRows)

				timelineRows := sqlmock.NewRows([]string{"time_bucket", "amount", "count"}).
					AddRow("08:00", 20000.0, 1).
					AddRow("14:00", 30000.0, 1)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to).
					WillReturnRows(timelineRows)
			},
			wantRevenue:   50000.0,
			wantCount:     2,
			wantGrowth:    100.0,
			wantAvgTicket: 25000.0,
			wantMinSlots:  14, // 06:00 to 19:00 inclusive = 14 slots
			wantErr:       false,
		},
		{
			name:     "success with monthly interval",
			from:     from,
			to:       to,
			interval: "month",
			mockExpect: func(mock sqlmock.Sqlmock) {
				summaryRows := sqlmock.NewRows([]string{"cur_rev", "cur_count", "prev_rev"}).
					AddRow(75000.0, 3, 50000.0)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to, from.Add(-to.Sub(from))).
					WillReturnRows(summaryRows)

				timelineRows := sqlmock.NewRows([]string{"time_bucket", "amount", "count"}).
					AddRow("2026-08", 35000.0, 1).
					AddRow("2026-09", 40000.0, 2)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to).
					WillReturnRows(timelineRows)
			},
			wantRevenue:   75000.0,
			wantCount:     3,
			wantGrowth:    50.0,
			wantAvgTicket: 25000.0,
			wantErr:       false,
		},
		{
			name:     "zero previous revenue gives 100 percent growth",
			from:     from,
			to:       to,
			interval: "day",
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
			name:     "error querying summary",
			from:     from,
			to:       to,
			interval: "day",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
					WithArgs(from, to, from.Add(-to.Sub(from))).
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name:     "error querying timeline",
			from:     from,
			to:       to,
			interval: "day",
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
			res, err := repo.GetRevenueSummary(context.Background(), tt.from, tt.to, tt.interval)

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
				if tt.wantMinSlots > 0 && len(res.Timeline) < tt.wantMinSlots {
					t.Errorf("got len(Timeline) %d, want at least %d", len(res.Timeline), tt.wantMinSlots)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestSQLEmailReceiptRepository_GetRevenueSummary_NoStatusFilter(t *testing.T) {
	now := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	from := now.AddDate(0, 0, -7)
	to := now

	tests := []struct {
		name          string
		mockExpect    func(mock sqlmock.Sqlmock)
		wantRevenue   float64
		wantCount     int
		wantGrowth    float64
		wantAvgTicket float64
		wantTimeline  int
		wantErr       bool
	}{
		{
			name: "success returns totals and timeline without status filter",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(".*").
					WillReturnRows(sqlmock.NewRows([]string{"cur_rev", "cur_count", "prev_rev"}).
						AddRow(120000.0, 4, 100000.0))

				mock.ExpectQuery(".*").
					WillReturnRows(sqlmock.NewRows([]string{"time_bucket", "amount", "count"}).
						AddRow("2026-09-18", 20000.0, 1).
						AddRow("2026-09-19", 100000.0, 3))
			},
			wantRevenue:   120000.0,
			wantCount:     4,
			wantGrowth:    20.0,
			wantAvgTicket: 30000.0,
			wantTimeline:  2,
			wantErr:       false,
		},
		{
			name: "zero revenue and empty timeline still status free",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(".*").
					WillReturnRows(sqlmock.NewRows([]string{"cur_rev", "cur_count", "prev_rev"}).
						AddRow(0.0, 0, 0.0))

				mock.ExpectQuery(".*").
					WillReturnRows(sqlmock.NewRows([]string{"time_bucket", "amount", "count"}))
			},
			wantRevenue:   0,
			wantCount:     0,
			wantGrowth:    0,
			wantAvgTicket: 0,
			wantTimeline:  0,
			wantErr:       false,
		},
		{
			name: "error querying summary uses status free query",
			mockExpect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(".*").WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newStatusRejectingMock(t)
			defer db.Close()

			tt.mockExpect(mock)

			repo := repository.NewSQLEmailReceiptRepository(db)
			res, err := repo.GetRevenueSummary(context.Background(), from, to, "day")

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
				if len(res.Timeline) != tt.wantTimeline {
					t.Errorf("got len(Timeline) %d, want %d", len(res.Timeline), tt.wantTimeline)
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
