package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
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

func TestEmailReceiptService(t *testing.T) {
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
			svc := service.NewEmailReceiptService(tt.mockRepo, nil, logger)
			res, err := svc.List(context.Background(), tt.from, tt.to)

			if (err != nil) != tt.wantErr {
				t.Fatalf("List() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(res) != tt.wantCount {
				t.Errorf("got %d receipts, want %d", len(res), tt.wantCount)
			}
		})
	}

	t.Run("Sync without client returns error", func(t *testing.T) {
		svc := service.NewEmailReceiptService(&MockEmailReceiptRepo{}, nil, logger)
		_, err := svc.Sync(context.Background(), service.EmailReceiptSyncRequest{From: "2026-01-01", To: "2026-01-02"})
		if err == nil {
			t.Errorf("expected error when client is nil, got nil")
		}
	})
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
			svc := service.NewEmailReceiptService(tt.mockRepo, nil, logger)
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

