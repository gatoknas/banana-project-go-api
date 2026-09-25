package repository

import (
	"context"
	"database/sql"
	"time"

	"org.banana.project/api/internal/models"
)

type DashboardRepository interface {
	GetKPIs(ctx context.Context, now time.Time) (models.DashboardKPIs, error)
	GetSalesTimeline(ctx context.Context, from, to time.Time) ([]models.SalesTimelineEntry, error)
	GetPaymentMethodBreakdown(ctx context.Context, from, to time.Time) ([]models.PaymentMethodEntry, error)
	GetTopProducts(ctx context.Context, from, to time.Time, limit int) ([]models.TopProductEntry, error)
	GetCategoryBreakdown(ctx context.Context, from, to time.Time) ([]models.CategoryEntry, error)
	GetPurchasesVsSales(ctx context.Context, fromMonth string) ([]models.PurchasesVsSalesEntry, error)
	GetInventoryAlerts(ctx context.Context, limit int) ([]models.InventoryAlertEntry, error)
}

type SQLDashboardRepository struct {
	db *sql.DB
}

func NewSQLDashboardRepository(db *sql.DB) *SQLDashboardRepository {
	return &SQLDashboardRepository{db: db}
}

func (r *SQLDashboardRepository) GetKPIs(ctx context.Context, now time.Time) (models.DashboardKPIs, error) {
	var kpis models.DashboardKPIs
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// 1. Today's Revenue and Transactions
	salesQuery := `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM sales
		WHERE sale_date >= $1 AND sale_date < $2`
	err := r.db.QueryRowContext(ctx, salesQuery, startOfDay, endOfDay).Scan(&kpis.TodayRevenue, &kpis.TodayTransactions)
	if err != nil && err != sql.ErrNoRows {
		return kpis, err
	}

	if kpis.TodayTransactions > 0 {
		kpis.AverageTicket = kpis.TodayRevenue / float64(kpis.TodayTransactions)
	}

	// 2. Low Stock Products Count
	lowStockQuery := `
		SELECT COUNT(*)
		FROM inventories i
		JOIN products p ON i.product_id = p.id
		WHERE p.is_for_sale = true AND i.current_stock <= i.minimum_stock`
	err = r.db.QueryRowContext(ctx, lowStockQuery).Scan(&kpis.LowStockProducts)
	if err != nil && err != sql.ErrNoRows {
		return kpis, err
	}

	// 3. Month Purchases Total
	purchasesQuery := `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM purchases
		WHERE purchase_date >= $1`
	err = r.db.QueryRowContext(ctx, purchasesQuery, startOfMonth).Scan(&kpis.MonthPurchasesTotal)
	if err != nil && err != sql.ErrNoRows {
		return kpis, err
	}

	// 4. Active Users Count
	usersQuery := `SELECT COUNT(*) FROM users WHERE is_active = true`
	err = r.db.QueryRowContext(ctx, usersQuery).Scan(&kpis.ActiveUsers)
	if err != nil && err != sql.ErrNoRows {
		return kpis, err
	}

	return kpis, nil
}

func (r *SQLDashboardRepository) GetSalesTimeline(ctx context.Context, from, to time.Time) ([]models.SalesTimelineEntry, error) {
	query := `
		SELECT TO_CHAR(sale_date, 'YYYY-MM-DD') AS day,
		       COALESCE(SUM(total_amount), 0) AS revenue,
		       COUNT(*) AS count
		FROM sales
		WHERE sale_date >= $1 AND sale_date <= $2
		GROUP BY TO_CHAR(sale_date, 'YYYY-MM-DD')
		ORDER BY day ASC`

	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]models.SalesTimelineEntry, 0)
	for rows.Next() {
		var entry models.SalesTimelineEntry
		if err := rows.Scan(&entry.Date, &entry.Revenue, &entry.Count); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *SQLDashboardRepository) GetPaymentMethodBreakdown(ctx context.Context, from, to time.Time) ([]models.PaymentMethodEntry, error) {
	query := `
		SELECT COALESCE(payment_method, 'Other') AS method,
		       COUNT(*) AS count,
		       COALESCE(SUM(total_amount), 0) AS amount
		FROM sales
		WHERE sale_date >= $1 AND sale_date <= $2
		GROUP BY payment_method
		ORDER BY amount DESC`

	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]models.PaymentMethodEntry, 0)
	for rows.Next() {
		var entry models.PaymentMethodEntry
		if err := rows.Scan(&entry.Method, &entry.Count, &entry.Amount); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *SQLDashboardRepository) GetTopProducts(ctx context.Context, from, to time.Time, limit int) ([]models.TopProductEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `
		SELECT p.id,
		       p.name,
		       COALESCE(SUM(sd.quantity), 0) AS total_quantity,
		       COALESCE(SUM(sd.subtotal), 0) AS total_revenue
		FROM sale_details sd
		JOIN sales s ON sd.sale_id = s.id
		JOIN products p ON sd.product_id = p.id
		WHERE s.sale_date >= $1 AND s.sale_date <= $2
		GROUP BY p.id, p.name
		ORDER BY total_quantity DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]models.TopProductEntry, 0)
	for rows.Next() {
		var entry models.TopProductEntry
		if err := rows.Scan(&entry.ProductID, &entry.ProductName, &entry.TotalQuantity, &entry.TotalRevenue); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *SQLDashboardRepository) GetCategoryBreakdown(ctx context.Context, from, to time.Time) ([]models.CategoryEntry, error) {
	query := `
		SELECT c.id,
		       c.name,
		       COALESCE(SUM(sd.subtotal), 0) AS total_revenue,
		       COUNT(DISTINCT s.id) AS count
		FROM sale_details sd
		JOIN sales s ON sd.sale_id = s.id
		JOIN products p ON sd.product_id = p.id
		JOIN categories c ON p.category_id = c.id
		WHERE s.sale_date >= $1 AND s.sale_date <= $2
		GROUP BY c.id, c.name
		ORDER BY total_revenue DESC`

	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]models.CategoryEntry, 0)
	for rows.Next() {
		var entry models.CategoryEntry
		if err := rows.Scan(&entry.CategoryID, &entry.CategoryName, &entry.TotalRevenue, &entry.Count); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *SQLDashboardRepository) GetPurchasesVsSales(ctx context.Context, fromMonth string) ([]models.PurchasesVsSalesEntry, error) {
	// Query monthly sales
	salesQuery := `
		SELECT TO_CHAR(sale_date, 'YYYY-MM') AS month,
		       COALESCE(SUM(total_amount), 0) AS sales
		FROM sales
		WHERE TO_CHAR(sale_date, 'YYYY-MM') >= $1
		GROUP BY TO_CHAR(sale_date, 'YYYY-MM')
		ORDER BY month ASC`

	salesRows, err := r.db.QueryContext(ctx, salesQuery, fromMonth)
	if err != nil {
		return nil, err
	}
	defer salesRows.Close()

	salesMap := make(map[string]float64)
	for salesRows.Next() {
		var m string
		var amt float64
		if err := salesRows.Scan(&m, &amt); err != nil {
			return nil, err
		}
		salesMap[m] = amt
	}

	// Query monthly purchases
	purchasesQuery := `
		SELECT TO_CHAR(purchase_date, 'YYYY-MM') AS month,
		       COALESCE(SUM(total_amount), 0) AS purchases
		FROM purchases
		WHERE TO_CHAR(purchase_date, 'YYYY-MM') >= $1
		GROUP BY TO_CHAR(purchase_date, 'YYYY-MM')
		ORDER BY month ASC`

	purchasesRows, err := r.db.QueryContext(ctx, purchasesQuery, fromMonth)
	if err != nil {
		return nil, err
	}
	defer purchasesRows.Close()

	purchasesMap := make(map[string]float64)
	for purchasesRows.Next() {
		var m string
		var amt float64
		if err := purchasesRows.Scan(&m, &amt); err != nil {
			return nil, err
		}
		purchasesMap[m] = amt
	}

	// Combine all unique months in sorted order
	monthsSet := make(map[string]struct{})
	for m := range salesMap {
		monthsSet[m] = struct{}{}
	}
	for m := range purchasesMap {
		monthsSet[m] = struct{}{}
	}

	entries := make([]models.PurchasesVsSalesEntry, 0, len(monthsSet))
	// Generate months slice sorted
	var sortedMonths []string
	for m := range monthsSet {
		sortedMonths = append(sortedMonths, m)
	}
	// Sort ascending
	for i := 0; i < len(sortedMonths); i++ {
		for j := i + 1; j < len(sortedMonths); j++ {
			if sortedMonths[i] > sortedMonths[j] {
				sortedMonths[i], sortedMonths[j] = sortedMonths[j], sortedMonths[i]
			}
		}
	}

	for _, m := range sortedMonths {
		entries = append(entries, models.PurchasesVsSalesEntry{
			Month:     m,
			Purchases: purchasesMap[m],
			Sales:     salesMap[m],
		})
	}

	return entries, nil
}

func (r *SQLDashboardRepository) GetInventoryAlerts(ctx context.Context, limit int) ([]models.InventoryAlertEntry, error) {
	if limit <= 0 {
		limit = 15
	}
	query := `
		SELECT p.id,
		       p.name,
		       i.current_stock,
		       i.minimum_stock,
		       COALESCE(u.abbreviation, 'und') AS unit
		FROM inventories i
		JOIN products p ON i.product_id = p.id
		LEFT JOIN units_of_measure u ON p.unit_of_measure_id = u.id
		WHERE i.current_stock <= i.minimum_stock
		ORDER BY (i.minimum_stock - i.current_stock) DESC
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]models.InventoryAlertEntry, 0)
	for rows.Next() {
		var entry models.InventoryAlertEntry
		if err := rows.Scan(&entry.ProductID, &entry.ProductName, &entry.CurrentStock, &entry.MinimumStock, &entry.Unit); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
