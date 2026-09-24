package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
	"org.banana.project/api/internal/sheets"
)

type MockEmailReceiptRepo struct {
	UpsertByMessageIDFunc func(ctx context.Context, receipt *models.EmailReceipt) (bool, error)
	ListFunc              func(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error)
	GetByIDFunc           func(ctx context.Context, id int64) (*models.EmailReceipt, error)
	GetByMessageIDFunc    func(ctx context.Context, messageID string) (*models.EmailReceipt, error)
	GetRevenueSummaryFunc func(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error)
}

func (m *MockEmailReceiptRepo) UpsertByMessageID(ctx context.Context, r *models.EmailReceipt) (bool, error) {
	if m.UpsertByMessageIDFunc != nil {
		return m.UpsertByMessageIDFunc(ctx, r)
	}
	return false, nil
}

func (m *MockEmailReceiptRepo) List(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, from, to)
	}
	return nil, nil
}

func (m *MockEmailReceiptRepo) GetByID(ctx context.Context, id int64) (*models.EmailReceipt, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockEmailReceiptRepo) GetByMessageID(ctx context.Context, messageID string) (*models.EmailReceipt, error) {
	if m.GetByMessageIDFunc != nil {
		return m.GetByMessageIDFunc(ctx, messageID)
	}
	return nil, nil
}

func (m *MockEmailReceiptRepo) GetRevenueSummary(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error) {
	if m.GetRevenueSummaryFunc != nil {
		return m.GetRevenueSummaryFunc(ctx, from, to)
	}
	return nil, nil
}

type MockSheetsReader struct {
	FetchChunkFunc func(ctx context.Context, spreadsheetID, sheetName string, startRow, limit int) ([][]any, error)
}

func (m *MockSheetsReader) FetchChunk(ctx context.Context, spreadsheetID, sheetName string, startRow, limit int) ([][]any, error) {
	if m.FetchChunkFunc != nil {
		return m.FetchChunkFunc(ctx, spreadsheetID, sheetName, startRow, limit)
	}
	return nil, nil
}

func TestEmailReceiptService_Sync(t *testing.T) {
	logger := zap.NewNop()

	sampleRow1 := []any{"2026-09-24 10:00:00", "$ 15.000", "imported", "2026-09-24", "User 1", "Nequi", "REF1", "TRX1", "QR", "MSG-1"}
	sampleRow2 := []any{"2026-09-24 11:00:00", "$ 25.000", "imported", "2026-09-24", "User 2", "Nequi", "REF2", "TRX2", "QR", "MSG-2"}
	invalidRow := []any{"2026-09-24", "$ 10.000", "imported", "2026-09-24", "User 3", "Nequi", "REF3", "TRX3", "QR", ""} // missing MSG-ID

	tests := []struct {
		name         string
		mockRepo     *MockEmailReceiptRepo
		mockSheets   *MockSheetsReader
		req          service.EmailReceiptSyncRequest
		wantFetched  int
		wantImported int
		wantSkipped  int
		wantErrors   int
		wantErr      bool
	}{
		{
			name: "single chunk success with new and duplicate rows",
			mockRepo: &MockEmailReceiptRepo{
				UpsertByMessageIDFunc: func(ctx context.Context, receipt *models.EmailReceipt) (bool, error) {
					if receipt.MessageID == "MSG-1" {
						return true, nil // inserted
					}
					return false, nil // skipped/already exists
				},
			},
			mockSheets: &MockSheetsReader{
				FetchChunkFunc: func(ctx context.Context, spreadsheetID, sheetName string, startRow, limit int) ([][]any, error) {
					if startRow == 2 {
						return [][]any{sampleRow1, sampleRow2}, nil
					}
					return [][]any{}, nil
				},
			},
			req:          service.EmailReceiptSyncRequest{},
			wantFetched:  2,
			wantImported: 1,
			wantSkipped:  1,
			wantErrors:   0,
			wantErr:      false,
		},
		{
			name: "multi-chunk pagination until EOF",
			mockRepo: &MockEmailReceiptRepo{
				UpsertByMessageIDFunc: func(ctx context.Context, receipt *models.EmailReceipt) (bool, error) {
					return true, nil
				},
			},
			mockSheets: &MockSheetsReader{
				FetchChunkFunc: func(ctx context.Context, spreadsheetID, sheetName string, startRow, limit int) ([][]any, error) {
					if startRow == 2 {
						return [][]any{sampleRow1, sampleRow2}, nil // full chunk (limit=2)
					}
					return [][]any{}, nil // EOF
				},
			},
			req:          service.EmailReceiptSyncRequest{},
			wantFetched:  2,
			wantImported: 2,
			wantSkipped:  0,
			wantErrors:   0,
			wantErr:      false,
		},
		{
			name: "sheet with invalid row increments error count and continues",
			mockRepo: &MockEmailReceiptRepo{
				UpsertByMessageIDFunc: func(ctx context.Context, receipt *models.EmailReceipt) (bool, error) {
					return true, nil
				},
			},
			mockSheets: &MockSheetsReader{
				FetchChunkFunc: func(ctx context.Context, spreadsheetID, sheetName string, startRow, limit int) ([][]any, error) {
					if startRow == 2 {
						return [][]any{invalidRow, sampleRow1}, nil
					}
					return [][]any{}, nil
				},
			},
			req:          service.EmailReceiptSyncRequest{},
			wantFetched:  2,
			wantImported: 1,
			wantSkipped:  0,
			wantErrors:   1,
			wantErr:      false,
		},
		{
			name: "database error on upsert increments error count",
			mockRepo: &MockEmailReceiptRepo{
				UpsertByMessageIDFunc: func(ctx context.Context, receipt *models.EmailReceipt) (bool, error) {
					return false, errors.New("db connection failure")
				},
			},
			mockSheets: &MockSheetsReader{
				FetchChunkFunc: func(ctx context.Context, spreadsheetID, sheetName string, startRow, limit int) ([][]any, error) {
					if startRow == 2 {
						return [][]any{sampleRow1}, nil
					}
					return [][]any{}, nil
				},
			},
			req:          service.EmailReceiptSyncRequest{},
			wantFetched:  1,
			wantImported: 0,
			wantSkipped:  0,
			wantErrors:   1,
			wantErr:      false,
		},
		{
			name:       "nil sheets reader returns error",
			mockRepo:   &MockEmailReceiptRepo{},
			mockSheets: nil,
			req:        service.EmailReceiptSyncRequest{},
			wantErr:    true,
		},
		{
			name:       "invalid date range returns error",
			mockRepo:   &MockEmailReceiptRepo{},
			mockSheets: &MockSheetsReader{},
			req: service.EmailReceiptSyncRequest{
				From: "invalid-date",
				To:   "2026-09-24",
			},
			wantErr: true,
		},
		{
			name: "sheets fetch failure returns error",
			mockRepo: &MockEmailReceiptRepo{},
			mockSheets: &MockSheetsReader{
				FetchChunkFunc: func(ctx context.Context, spreadsheetID, sheetName string, startRow, limit int) ([][]any, error) {
					return nil, fmt.Errorf("network timeout")
				},
			},
			req:     service.EmailReceiptSyncRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reader sheets.Reader
			if tt.mockSheets != nil {
				reader = tt.mockSheets
			}
			svc := service.NewEmailReceiptService(tt.mockRepo, reader, "sheet-id", "Datos_Ventas", 2, logger)
			got, err := svc.Sync(context.Background(), tt.req)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Sync() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if got.Fetched != tt.wantFetched {
				t.Errorf("got.Fetched = %d, want %d", got.Fetched, tt.wantFetched)
			}
			if got.Imported != tt.wantImported {
				t.Errorf("got.Imported = %d, want %d", got.Imported, tt.wantImported)
			}
			if got.Skipped != tt.wantSkipped {
				t.Errorf("got.Skipped = %d, want %d", got.Skipped, tt.wantSkipped)
			}
			if got.Errors != tt.wantErrors {
				t.Errorf("got.Errors = %d, want %d", got.Errors, tt.wantErrors)
			}
		})
	}
}

func TestEmailReceiptService_List(t *testing.T) {
	logger := zap.NewNop()
	now := time.Now()

	tests := []struct {
		name      string
		mockRepo  *MockEmailReceiptRepo
		from      *time.Time
		to        *time.Time
		wantCount int
		wantErr   bool
	}{
		{
			name: "list receipts successfully",
			mockRepo: &MockEmailReceiptRepo{
				ListFunc: func(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error) {
					return []models.EmailReceipt{
						{ID: 1, MessageID: "msg-1", Subject: "Receipt 1", ReceivedAt: now},
						{ID: 2, MessageID: "msg-2", Subject: "Receipt 2", ReceivedAt: now},
					}, nil
				},
			},
			from:      &now,
			to:        &now,
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewEmailReceiptService(tt.mockRepo, nil, "sheet-id", "Datos_Ventas", 50, logger)
			res, err := svc.List(context.Background(), tt.from, tt.to)

			if (err != nil) != tt.wantErr {
				t.Fatalf("List() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(res) != tt.wantCount {
				t.Errorf("got %d receipts, want %d", len(res), tt.wantCount)
			}
		})
	}
}

func TestEmailReceiptService_GetRevenueSummary(t *testing.T) {
	logger := zap.NewNop()
	now := time.Now()
	validFrom := now.Add(-7 * 24 * time.Hour)
	validTo := now
	invalidFrom := now
	invalidTo := now.Add(-7 * 24 * time.Hour)

	tests := []struct {
		name        string
		mockRepo    *MockEmailReceiptRepo
		from        *time.Time
		to          *time.Time
		wantRevenue float64
		wantErr     bool
	}{
		{
			name: "custom valid range success",
			mockRepo: &MockEmailReceiptRepo{
				GetRevenueSummaryFunc: func(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error) {
					return &models.RevenueSummary{
						TotalRevenue:     125000.0,
						TransactionCount: 4,
						Currency:         "COP",
					}, nil
				},
			},
			from:        &validFrom,
			to:          &validTo,
			wantRevenue: 125000.0,
			wantErr:     false,
		},
		{
			name: "default nil range success",
			mockRepo: &MockEmailReceiptRepo{
				GetRevenueSummaryFunc: func(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error) {
					return &models.RevenueSummary{
						TotalRevenue:     50000.0,
						TransactionCount: 1,
						Currency:         "COP",
					}, nil
				},
			},
			from:        nil,
			to:          nil,
			wantRevenue: 50000.0,
			wantErr:     false,
		},
		{
			name:     "invalid range from after to",
			mockRepo: &MockEmailReceiptRepo{},
			from:     &invalidFrom,
			to:       &invalidTo,
			wantErr:  true,
		},
		{
			name: "repo error returns error",
			mockRepo: &MockEmailReceiptRepo{
				GetRevenueSummaryFunc: func(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error) {
					return nil, errors.New("db query failed")
				},
			},
			from:    &validFrom,
			to:      &validTo,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewEmailReceiptService(tt.mockRepo, nil, "sheet-id", "Datos_Ventas", 50, logger)
			res, err := svc.GetRevenueSummary(context.Background(), tt.from, tt.to)

			if (err != nil) != tt.wantErr {
				t.Fatalf("GetRevenueSummary() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if res == nil || res.TotalRevenue != tt.wantRevenue {
					t.Errorf("got TotalRevenue %v, want %v", res.TotalRevenue, tt.wantRevenue)
				}
			}
		})
	}
}

func TestEmailReceiptService_SyncLastDay(t *testing.T) {
	logger := zap.NewNop()
	called := false
	mockSheets := &MockSheetsReader{
		FetchChunkFunc: func(ctx context.Context, spreadsheetID, sheetName string, startRow, limit int) ([][]any, error) {
			called = true
			return [][]any{}, nil
		},
	}
	svc := service.NewEmailReceiptService(&MockEmailReceiptRepo{}, mockSheets, "sheet-id", "Datos_Ventas", 50, logger)
	res, err := svc.SyncLastDay(context.Background())
	if err != nil {
		t.Fatalf("SyncLastDay() error = %v", err)
	}
	if !called {
		t.Errorf("expected FetchChunk to be called")
	}
	if res.Fetched != 0 {
		t.Errorf("expected 0 fetched, got %d", res.Fetched)
	}
}
