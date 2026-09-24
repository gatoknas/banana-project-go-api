package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/email"
	"org.banana.project/api/internal/handlers"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type MockEmailReceiptRepo struct {
	UpsertByMessageIDFunc func(ctx context.Context, r *models.EmailReceipt) (bool, error)
	ListFunc              func(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error)
	GetRevenueSummaryFunc func(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error)
}

func (m *MockEmailReceiptRepo) UpsertByMessageID(ctx context.Context, r *models.EmailReceipt) (bool, error) {
	return m.UpsertByMessageIDFunc(ctx, r)
}

func (m *MockEmailReceiptRepo) List(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error) {
	return m.ListFunc(ctx, from, to)
}

func (m *MockEmailReceiptRepo) GetRevenueSummary(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error) {
	if m.GetRevenueSummaryFunc != nil {
		return m.GetRevenueSummaryFunc(ctx, from, to)
	}
	return nil, nil
}

func newEmailHandler(repo *MockEmailReceiptRepo, client *email.Client) *handlers.EmailReceiptHandler {
	return handlers.NewEmailReceiptHandler(service.NewEmailReceiptService(repo, client, zap.NewNop()), zap.NewNop())
}

func mustEmailClient(t *testing.T) *email.Client {
	t.Helper()
	client, err := email.NewClient(context.Background(), email.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RefreshToken: "test-refresh",
	})
	if err != nil {
		t.Fatalf("failed to build email client: %v", err)
	}
	return client
}

func TestEmailReceiptHandler_Sync(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		withClient bool
		repo       *MockEmailReceiptRepo
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			withClient: false,
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "gmail integration disabled",
			body:       `{"from":"2026-01-01","to":"2026-01-02"}`,
			withClient: false,
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid from date",
			body:       `{"from":"bad","to":"2026-01-02"}`,
			withClient: true,
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid to date",
			body:       `{"from":"2026-01-01","to":"bad"}`,
			withClient: true,
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "to before from",
			body:       `{"from":"2026-01-05","to":"2026-01-01"}`,
			withClient: true,
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *email.Client
			if tt.withClient {
				client = mustEmailClient(t)
			}

			h := newEmailHandler(tt.repo, client)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/email-receipts/sync", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.Sync(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestEmailReceiptHandler_List(t *testing.T) {
	now := time.Now()
	one := 12.5

	tests := []struct {
		name       string
		query      string
		repo       *MockEmailReceiptRepo
		wantStatus int
		wantCount  int
	}{
		{
			name:  "success",
			query: "?from=2026-01-01&to=2026-01-31",
			repo: &MockEmailReceiptRepo{
				ListFunc: func(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error) {
					return []models.EmailReceipt{{ID: 1, Amount: &one, ReceivedAt: now}}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:  "empty list returns array",
			query: "",
			repo: &MockEmailReceiptRepo{
				ListFunc: func(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error) {
					return nil, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "invalid from",
			query:      "?from=bad",
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid to",
			query:      "?to=bad",
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "repository error",
			query: "",
			repo: &MockEmailReceiptRepo{
				ListFunc: func(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error) {
					return nil, errors.New("db error")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newEmailHandler(tt.repo, nil)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/email-receipts"+tt.query, nil)
			w := httptest.NewRecorder()

			h.List(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantStatus == http.StatusOK {
				var resp []models.EmailReceipt
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(resp) != tt.wantCount {
					t.Errorf("expected %d receipts, got %d", tt.wantCount, len(resp))
				}
			}
		})
	}
}

func TestEmailReceiptHandler_Summary(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		repo        *MockEmailReceiptRepo
		wantStatus  int
		wantRevenue float64
	}{
		{
			name:  "valid date range success",
			query: "?from=2026-09-01&to=2026-09-20",
			repo: &MockEmailReceiptRepo{
				GetRevenueSummaryFunc: func(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error) {
					return &models.RevenueSummary{
						TotalRevenue:     350000.0,
						TransactionCount: 12,
						AverageTicket:    29166.67,
						Currency:         "COP",
					}, nil
				},
			},
			wantStatus:  http.StatusOK,
			wantRevenue: 350000.0,
		},
		{
			name:  "default dates success",
			query: "",
			repo: &MockEmailReceiptRepo{
				GetRevenueSummaryFunc: func(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error) {
					return &models.RevenueSummary{
						TotalRevenue:     100000.0,
						TransactionCount: 2,
						Currency:         "COP",
					}, nil
				},
			},
			wantStatus:  http.StatusOK,
			wantRevenue: 100000.0,
		},
		{
			name:       "invalid from date",
			query:      "?from=not-a-date",
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid to date",
			query:      "?to=2026-99-99",
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "invalid range from after to",
			query: "?from=2026-09-20&to=2026-09-01",
			repo:  &MockEmailReceiptRepo{},
			// service returns "invalid date range"
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "internal service error",
			query: "?from=2026-09-01&to=2026-09-20",
			repo: &MockEmailReceiptRepo{
				GetRevenueSummaryFunc: func(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error) {
					return nil, errors.New("db error")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newEmailHandler(tt.repo, nil)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/email-receipts/summary"+tt.query, nil)
			w := httptest.NewRecorder()

			h.Summary(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				var resp models.RevenueSummary
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.TotalRevenue != tt.wantRevenue {
					t.Errorf("expected total revenue %v, got %v", tt.wantRevenue, resp.TotalRevenue)
				}
			}
		})
	}
}

