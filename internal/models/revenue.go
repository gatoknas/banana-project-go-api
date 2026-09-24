package models

// DailyRevenueBucket represents the aggregated revenue and transaction count for a single day.
type DailyRevenueBucket struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

// RevenueSummary represents financial revenue metrics for a period, including comparison against the prior period.
type RevenueSummary struct {
	TotalRevenue          float64              `json:"totalRevenue"`
	TransactionCount      int                  `json:"transactionCount"`
	AverageTicket         float64              `json:"averageTicket"`
	PreviousPeriodRevenue float64              `json:"previousPeriodRevenue"`
	GrowthPercentage      float64              `json:"growthPercentage"`
	Currency              string               `json:"currency"`
	Timeline              []DailyRevenueBucket `json:"timeline"`
}
