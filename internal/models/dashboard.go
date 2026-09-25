package models

import "time"

// DashboardFilter represents the date range filter for dashboard analytics.
type DashboardFilter struct {
	FromDate *time.Time
	ToDate   *time.Time
}

// DashboardKPIs holds executive KPI counters for the business.
type DashboardKPIs struct {
	TodayRevenue        float64 `json:"todayRevenue"`
	TodayTransactions   int     `json:"todayTransactions"`
	AverageTicket       float64 `json:"averageTicket"`
	LowStockProducts    int     `json:"lowStockProducts"`
	MonthPurchasesTotal float64 `json:"monthPurchasesTotal"`
	ActiveUsers         int     `json:"activeUsers"`
}

// SalesTimelineEntry represents daily revenue and transaction volume.
type SalesTimelineEntry struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue"`
	Count   int     `json:"count"`
}

// PaymentMethodEntry represents breakdown of sales by payment method.
type PaymentMethodEntry struct {
	Method string  `json:"method"`
	Count  int     `json:"count"`
	Amount float64 `json:"amount"`
}

// TopProductEntry represents a best-selling product.
type TopProductEntry struct {
	ProductID     int64   `json:"productId"`
	ProductName   string  `json:"productName"`
	TotalQuantity float64 `json:"totalQuantity"`
	TotalRevenue  float64 `json:"totalRevenue"`
}

// CategoryEntry represents sales distribution across product categories.
type CategoryEntry struct {
	CategoryID   int64   `json:"categoryId"`
	CategoryName string  `json:"categoryName"`
	TotalRevenue float64 `json:"totalRevenue"`
	Count        int     `json:"count"`
}

// PurchasesVsSalesEntry represents comparative monthly spend vs revenue.
type PurchasesVsSalesEntry struct {
	Month     string  `json:"month"`
	Purchases float64 `json:"purchases"`
	Sales     float64 `json:"sales"`
}

// InventoryAlertEntry represents an inventory item requiring restocking attention.
type InventoryAlertEntry struct {
	ProductID    int64   `json:"productId"`
	ProductName  string  `json:"productName"`
	CurrentStock float64 `json:"currentStock"`
	MinimumStock float64 `json:"minimumStock"`
	Unit         string  `json:"unit"`
}

// DashboardStats is the root aggregate payload for the executive dashboard.
type DashboardStats struct {
	KPIs                   DashboardKPIs           `json:"kpis"`
	SalesTimeline          []SalesTimelineEntry    `json:"salesTimeline"`
	PaymentMethodBreakdown []PaymentMethodEntry    `json:"paymentMethodBreakdown"`
	TopProducts            []TopProductEntry       `json:"topProducts"`
	CategoryBreakdown      []CategoryEntry         `json:"categoryBreakdown"`
	PurchasesVsSales       []PurchasesVsSalesEntry `json:"purchasesVsSales"`
	InventoryAlerts        []InventoryAlertEntry   `json:"inventoryAlerts"`
}
