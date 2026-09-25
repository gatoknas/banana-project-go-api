package service

import (
	"context"
	"errors"
	"time"

	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
)

type DashboardService struct {
	repo repository.DashboardRepository
}

func NewDashboardService(repo repository.DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

// GetDashboardStats aggregates all business metrics and chart datasets for a date range.
func (s *DashboardService) GetDashboardStats(ctx context.Context, fromStr, toStr string) (*models.DashboardStats, error) {
	now := time.Now()

	var to time.Time
	if toStr != "" {
		parsedTo, err := parseDate(toStr, true)
		if err != nil {
			return nil, errors.New("invalid 'to' date format; expected YYYY-MM-DD")
		}
		to = parsedTo
	} else {
		to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	}

	var from time.Time
	if fromStr != "" {
		parsedFrom, err := parseDate(fromStr, false)
		if err != nil {
			return nil, errors.New("invalid 'from' date format; expected YYYY-MM-DD")
		}
		from = parsedFrom
	} else {
		// Default to 30 days prior
		from = to.AddDate(0, 0, -30)
		from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	}

	if from.After(to) {
		return nil, errors.New("'from' date cannot be after 'to' date")
	}

	// 1. KPIs
	kpis, err := s.repo.GetKPIs(ctx, now)
	if err != nil {
		return nil, err
	}

	// 2. Sales Timeline
	salesTimeline, err := s.repo.GetSalesTimeline(ctx, from, to)
	if err != nil {
		return nil, err
	}

	// 3. Payment Method Breakdown
	paymentMethods, err := s.repo.GetPaymentMethodBreakdown(ctx, from, to)
	if err != nil {
		return nil, err
	}

	// 4. Top Products
	topProducts, err := s.repo.GetTopProducts(ctx, from, to, 10)
	if err != nil {
		return nil, err
	}

	// 5. Category Breakdown
	categories, err := s.repo.GetCategoryBreakdown(ctx, from, to)
	if err != nil {
		return nil, err
	}

	// 6. Purchases vs Sales (past 6 months)
	sixMonthsAgo := now.AddDate(0, -5, 0).Format("2006-01")
	purchasesVsSales, err := s.repo.GetPurchasesVsSales(ctx, sixMonthsAgo)
	if err != nil {
		return nil, err
	}

	// 7. Inventory Alerts (low stock)
	inventoryAlerts, err := s.repo.GetInventoryAlerts(ctx, 15)
	if err != nil {
		return nil, err
	}

	return &models.DashboardStats{
		KPIs:                   kpis,
		SalesTimeline:          salesTimeline,
		PaymentMethodBreakdown: paymentMethods,
		TopProducts:            topProducts,
		CategoryBreakdown:      categories,
		PurchasesVsSales:       purchasesVsSales,
		InventoryAlerts:        inventoryAlerts,
	}, nil
}

func parseDate(str string, endOfDay bool) (time.Time, error) {
	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, str); err == nil {
			if layout == "2006-01-02" {
				if endOfDay {
					return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC), nil
				}
				return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
			}
			return t, nil
		}
	}

	return time.Time{}, errors.New("unsupported date format")
}
