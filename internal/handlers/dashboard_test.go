package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type mockDashboardRepoForHandler struct {
	fail bool
}

func (m *mockDashboardRepoForHandler) GetKPIs(ctx context.Context, now time.Time) (models.DashboardKPIs, error) {
	if m.fail {
		return models.DashboardKPIs{}, errors.New("db error")
	}
	return models.DashboardKPIs{TodayRevenue: 150000, TodayTransactions: 10, AverageTicket: 15000}, nil
}

func (m *mockDashboardRepoForHandler) GetSalesTimeline(ctx context.Context, from, to time.Time) ([]models.SalesTimelineEntry, error) {
	if m.fail {
		return nil, errors.New("db error")
	}
	return []models.SalesTimelineEntry{{Date: "2026-09-25", Revenue: 150000, Count: 10}}, nil
}

func (m *mockDashboardRepoForHandler) GetPaymentMethodBreakdown(ctx context.Context, from, to time.Time) ([]models.PaymentMethodEntry, error) {
	if m.fail {
		return nil, errors.New("db error")
	}
	return []models.PaymentMethodEntry{{Method: "Cash", Count: 8, Amount: 120000}}, nil
}

func (m *mockDashboardRepoForHandler) GetTopProducts(ctx context.Context, from, to time.Time, limit int) ([]models.TopProductEntry, error) {
	if m.fail {
		return nil, errors.New("db error")
	}
	return []models.TopProductEntry{{ProductID: 1, ProductName: "Batido", TotalQuantity: 15, TotalRevenue: 150000}}, nil
}

func (m *mockDashboardRepoForHandler) GetCategoryBreakdown(ctx context.Context, from, to time.Time) ([]models.CategoryEntry, error) {
	if m.fail {
		return nil, errors.New("db error")
	}
	return []models.CategoryEntry{{CategoryID: 1, CategoryName: "Bebidas", TotalRevenue: 150000, Count: 10}}, nil
}

func (m *mockDashboardRepoForHandler) GetPurchasesVsSales(ctx context.Context, fromMonth string) ([]models.PurchasesVsSalesEntry, error) {
	if m.fail {
		return nil, errors.New("db error")
	}
	return []models.PurchasesVsSalesEntry{{Month: "2026-09", Purchases: 50000, Sales: 150000}}, nil
}

func (m *mockDashboardRepoForHandler) GetInventoryAlerts(ctx context.Context, limit int) ([]models.InventoryAlertEntry, error) {
	if m.fail {
		return nil, errors.New("db error")
	}
	return []models.InventoryAlertEntry{{ProductID: 2, ProductName: "Leche", CurrentStock: 1, MinimumStock: 3, Unit: "lt"}}, nil
}

func TestDashboardHandler_GetStats(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		failRepo       bool
		queryString    string
		expectedStatus int
	}{
		{
			name:           "success with default params",
			failRepo:       false,
			queryString:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success with valid date filters",
			failRepo:       false,
			queryString:    "?from=2026-09-01&to=2026-09-25",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error invalid date format",
			failRepo:       false,
			queryString:    "?from=invalid-format",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "error from after to",
			failRepo:       false,
			queryString:    "?from=2026-09-30&to=2026-09-01",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "error internal server error on db failure",
			failRepo:       true,
			queryString:    "",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDashboardRepoForHandler{fail: tt.failRepo}
			svc := service.NewDashboardService(repo)
			handler := NewDashboardHandler(svc, logger)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats"+tt.queryString, nil)
			w := httptest.NewRecorder()

			handler.GetStats(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var resp models.DashboardStats
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.KPIs.TodayTransactions != 10 {
					t.Errorf("expected 10 transactions, got %d", resp.KPIs.TodayTransactions)
				}
			}
		})
	}
}
