package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"org.banana.project/api/internal/models"
)

type mockDashboardRepo struct {
	getKPIsFn                 func(ctx context.Context, now time.Time) (models.DashboardKPIs, error)
	getSalesTimelineFn        func(ctx context.Context, from, to time.Time) ([]models.SalesTimelineEntry, error)
	getPaymentMethodBreakdownFn func(ctx context.Context, from, to time.Time) ([]models.PaymentMethodEntry, error)
	getTopProductsFn           func(ctx context.Context, from, to time.Time, limit int) ([]models.TopProductEntry, error)
	getCategoryBreakdownFn     func(ctx context.Context, from, to time.Time) ([]models.CategoryEntry, error)
	getPurchasesVsSalesFn      func(ctx context.Context, fromMonth string) ([]models.PurchasesVsSalesEntry, error)
	getInventoryAlertsFn       func(ctx context.Context, limit int) ([]models.InventoryAlertEntry, error)
}

func (m *mockDashboardRepo) GetKPIs(ctx context.Context, now time.Time) (models.DashboardKPIs, error) {
	if m.getKPIsFn != nil {
		return m.getKPIsFn(ctx, now)
	}
	return models.DashboardKPIs{TodayRevenue: 1000, TodayTransactions: 5}, nil
}

func (m *mockDashboardRepo) GetSalesTimeline(ctx context.Context, from, to time.Time) ([]models.SalesTimelineEntry, error) {
	if m.getSalesTimelineFn != nil {
		return m.getSalesTimelineFn(ctx, from, to)
	}
	return []models.SalesTimelineEntry{{Date: "2026-09-01", Revenue: 500, Count: 2}}, nil
}

func (m *mockDashboardRepo) GetPaymentMethodBreakdown(ctx context.Context, from, to time.Time) ([]models.PaymentMethodEntry, error) {
	if m.getPaymentMethodBreakdownFn != nil {
		return m.getPaymentMethodBreakdownFn(ctx, from, to)
	}
	return []models.PaymentMethodEntry{{Method: "Cash", Count: 5, Amount: 1000}}, nil
}

func (m *mockDashboardRepo) GetTopProducts(ctx context.Context, from, to time.Time, limit int) ([]models.TopProductEntry, error) {
	if m.getTopProductsFn != nil {
		return m.getTopProductsFn(ctx, from, to, limit)
	}
	return []models.TopProductEntry{{ProductID: 1, ProductName: "Banana Smoothie", TotalQuantity: 10, TotalRevenue: 50000}}, nil
}

func (m *mockDashboardRepo) GetCategoryBreakdown(ctx context.Context, from, to time.Time) ([]models.CategoryEntry, error) {
	if m.getCategoryBreakdownFn != nil {
		return m.getCategoryBreakdownFn(ctx, from, to)
	}
	return []models.CategoryEntry{{CategoryID: 1, CategoryName: "Beverages", TotalRevenue: 50000, Count: 10}}, nil
}

func (m *mockDashboardRepo) GetPurchasesVsSales(ctx context.Context, fromMonth string) ([]models.PurchasesVsSalesEntry, error) {
	if m.getPurchasesVsSalesFn != nil {
		return m.getPurchasesVsSalesFn(ctx, fromMonth)
	}
	return []models.PurchasesVsSalesEntry{{Month: "2026-09", Purchases: 20000, Sales: 50000}}, nil
}

func (m *mockDashboardRepo) GetInventoryAlerts(ctx context.Context, limit int) ([]models.InventoryAlertEntry, error) {
	if m.getInventoryAlertsFn != nil {
		return m.getInventoryAlertsFn(ctx, limit)
	}
	return []models.InventoryAlertEntry{{ProductID: 1, ProductName: "Milk", CurrentStock: 1, MinimumStock: 5, Unit: "lt"}}, nil
}

func TestDashboardService_GetDashboardStats(t *testing.T) {
	tests := []struct {
		name      string
		repo      *mockDashboardRepo
		fromStr   string
		toStr     string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "success with empty date range (defaults to 30 days)",
			repo:    &mockDashboardRepo{},
			fromStr: "",
			toStr:   "",
			wantErr: false,
		},
		{
			name:    "success with explicit valid dates",
			repo:    &mockDashboardRepo{},
			fromStr: "2026-09-01",
			toStr:   "2026-09-25",
			wantErr: false,
		},
		{
			name:      "error invalid from date",
			repo:      &mockDashboardRepo{},
			fromStr:   "invalid-date",
			toStr:     "2026-09-25",
			wantErr:   true,
			errSubstr: "invalid 'from' date format",
		},
		{
			name:      "error invalid to date",
			repo:      &mockDashboardRepo{},
			fromStr:   "2026-09-01",
			toStr:     "not-a-date",
			wantErr:   true,
			errSubstr: "invalid 'to' date format",
		},
		{
			name:      "error from date after to date",
			repo:      &mockDashboardRepo{},
			fromStr:   "2026-09-30",
			toStr:     "2026-09-01",
			wantErr:   true,
			errSubstr: "cannot be after",
		},
		{
			name: "error from repository GetKPIs",
			repo: &mockDashboardRepo{
				getKPIsFn: func(ctx context.Context, now time.Time) (models.DashboardKPIs, error) {
					return models.DashboardKPIs{}, errors.New("db kpi error")
				},
			},
			fromStr:   "2026-09-01",
			toStr:     "2026-09-25",
			wantErr:   true,
			errSubstr: "db kpi error",
		},
		{
			name: "error from repository GetSalesTimeline",
			repo: &mockDashboardRepo{
				getSalesTimelineFn: func(ctx context.Context, from, to time.Time) ([]models.SalesTimelineEntry, error) {
					return nil, errors.New("timeline db error")
				},
			},
			fromStr:   "2026-09-01",
			toStr:     "2026-09-25",
			wantErr:   true,
			errSubstr: "timeline db error",
		},
		{
			name: "error from repository GetPaymentMethodBreakdown",
			repo: &mockDashboardRepo{
				getPaymentMethodBreakdownFn: func(ctx context.Context, from, to time.Time) ([]models.PaymentMethodEntry, error) {
					return nil, errors.New("payment breakdown error")
				},
			},
			fromStr:   "2026-09-01",
			toStr:     "2026-09-25",
			wantErr:   true,
			errSubstr: "payment breakdown error",
		},
		{
			name: "error from repository GetTopProducts",
			repo: &mockDashboardRepo{
				getTopProductsFn: func(ctx context.Context, from, to time.Time, limit int) ([]models.TopProductEntry, error) {
					return nil, errors.New("top products error")
				},
			},
			fromStr:   "2026-09-01",
			toStr:     "2026-09-25",
			wantErr:   true,
			errSubstr: "top products error",
		},
		{
			name: "error from repository GetCategoryBreakdown",
			repo: &mockDashboardRepo{
				getCategoryBreakdownFn: func(ctx context.Context, from, to time.Time) ([]models.CategoryEntry, error) {
					return nil, errors.New("category breakdown error")
				},
			},
			fromStr:   "2026-09-01",
			toStr:     "2026-09-25",
			wantErr:   true,
			errSubstr: "category breakdown error",
		},
		{
			name: "error from repository GetPurchasesVsSales",
			repo: &mockDashboardRepo{
				getPurchasesVsSalesFn: func(ctx context.Context, fromMonth string) ([]models.PurchasesVsSalesEntry, error) {
					return nil, errors.New("purchases vs sales error")
				},
			},
			fromStr:   "2026-09-01",
			toStr:     "2026-09-25",
			wantErr:   true,
			errSubstr: "purchases vs sales error",
		},
		{
			name: "error from repository GetInventoryAlerts",
			repo: &mockDashboardRepo{
				getInventoryAlertsFn: func(ctx context.Context, limit int) ([]models.InventoryAlertEntry, error) {
					return nil, errors.New("inventory alerts error")
				},
			},
			fromStr:   "2026-09-01",
			toStr:     "2026-09-25",
			wantErr:   true,
			errSubstr: "inventory alerts error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewDashboardService(tt.repo)
			stats, err := svc.GetDashboardStats(context.Background(), tt.fromStr, tt.toStr)

			if (err != nil) != tt.wantErr {
				t.Fatalf("GetDashboardStats() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if stats == nil {
					t.Fatal("expected non-nil DashboardStats")
				}
				if len(stats.SalesTimeline) == 0 {
					t.Errorf("expected sales timeline entries, got 0")
				}
			}
		})
	}
}
