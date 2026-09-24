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
	"org.banana.project/api/internal/handlers"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
	"org.banana.project/api/internal/sheets"
)

type MockEmailReceiptRepo struct {
	UpsertByMessageIDFunc func(ctx context.Context, r *models.EmailReceipt) (bool, error)
	ListFunc              func(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error)
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
	return [][]any{}, nil
}

func newEmailHandler(repo *MockEmailReceiptRepo, reader sheets.Reader) *handlers.EmailReceiptHandler {
	return handlers.NewEmailReceiptHandler(service.NewEmailReceiptService(repo, reader, "sheet-id", "Datos_Ventas", 50, zap.NewNop()), zap.NewNop())
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
			body:       `{invalid`,
			withClient: false,
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "google sheets integration disabled",
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
		{
			name:       "successful sync with date range",
			body:       `{"from":"2026-01-01","to":"2026-01-02"}`,
			withClient: true,
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusOK,
		},
		{
			name:       "successful sync with empty body",
			body:       ``,
			withClient: true,
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reader sheets.Reader
			if tt.withClient {
				reader = &MockSheetsReader{}
			}

			h := newEmailHandler(tt.repo, reader)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/email-receipts/sync", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.Sync(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Sync() status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestEmailReceiptHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		repo       *MockEmailReceiptRepo
		wantStatus int
		wantLen    int
	}{
		{
			name:  "list success",
			query: "",
			repo: &MockEmailReceiptRepo{
				ListFunc: func(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error) {
					return []models.EmailReceipt{
						{ID: 1, MessageID: "msg-1"},
					}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:  "list empty",
			query: "",
			repo: &MockEmailReceiptRepo{
				ListFunc: func(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error) {
					return nil, nil
				},
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "invalid from",
			query:      "?from=not-a-date",
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid to",
			query:      "?to=not-a-date",
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "repo error",
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
				t.Errorf("List() status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK {
				var got []models.EmailReceipt
				if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(got) != tt.wantLen {
					t.Errorf("got %d receipts, want %d", len(got), tt.wantLen)
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
			name:  "summary success",
			query: "?from=2026-09-01&to=2026-09-24",
			repo: &MockEmailReceiptRepo{
				GetRevenueSummaryFunc: func(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error) {
					return &models.RevenueSummary{
						TotalRevenue:     250000.0,
						TransactionCount: 10,
						Currency:         "COP",
					}, nil
				},
			},
			wantStatus:  http.StatusOK,
			wantRevenue: 250000.0,
		},
		{
			name:       "invalid from date",
			query:      "?from=not-a-date",
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid to date",
			query:      "?to=not-a-date",
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid date range",
			query:      "?from=2026-09-24&to=2026-09-01",
			repo:       &MockEmailReceiptRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "internal error",
			query: "?from=2026-09-01&to=2026-09-24",
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
				t.Errorf("Summary() status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK {
				var got models.RevenueSummary
				if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if got.TotalRevenue != tt.wantRevenue {
					t.Errorf("got TotalRevenue = %v, want %v", got.TotalRevenue, tt.wantRevenue)
				}
			}
		})
	}
}
