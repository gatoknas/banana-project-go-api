package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"org.banana.project/api/internal/models"
)

type SaleRepository interface {
	CreateSale(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error)
	CreateSaleDetail(ctx context.Context, tx *sql.Tx, d *models.SaleDetail) (int64, error)
	GetProductDetails(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error)
	GetRecipeIngredients(ctx context.Context, tx *sql.Tx, parentProductID int64) ([]models.ProductRecipe, error)
	DeductStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error
	ListSales(ctx context.Context, filter models.SaleFilter) ([]models.Sale, models.SalePagination, models.SaleSummaryStats, error)
	GetSaleByID(ctx context.Context, id int64) (*models.Sale, error)
}

type SQLSaleRepository struct {
	db *sql.DB
}

func NewSQLSaleRepository(db *sql.DB) *SQLSaleRepository {
	return &SQLSaleRepository{db: db}
}

func (r *SQLSaleRepository) CreateSale(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error) {
	var saleID int64
	query := `INSERT INTO sales (user_id, sale_date, total_amount, payment_method) 
			  VALUES ($1, $2, $3, $4) RETURNING id`
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, query, s.UserID, s.SaleDate, s.TotalAmount, s.PaymentMethod).Scan(&saleID)
	} else {
		err = r.db.QueryRowContext(ctx, query, s.UserID, s.SaleDate, s.TotalAmount, s.PaymentMethod).Scan(&saleID)
	}
	if err != nil {
		return 0, err
	}
	return saleID, nil
}

func (r *SQLSaleRepository) CreateSaleDetail(ctx context.Context, tx *sql.Tx, d *models.SaleDetail) (int64, error) {
	var detailID int64
	query := `INSERT INTO sale_details (sale_id, product_id, quantity, historical_unit_price, subtotal) 
			  VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, query, d.SaleID, d.ProductID, d.Quantity, d.HistoricalUnitPrice, d.Subtotal).Scan(&detailID)
	} else {
		err = r.db.QueryRowContext(ctx, query, d.SaleID, d.ProductID, d.Quantity, d.HistoricalUnitPrice, d.Subtotal).Scan(&detailID)
	}
	if err != nil {
		return 0, err
	}
	return detailID, nil
}

func (r *SQLSaleRepository) GetProductDetails(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error) {
	var sellPrice float64
	var requiresRecipe bool
	query := `SELECT sell_price, requires_recipe FROM products WHERE id = $1`
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, query, productID).Scan(&sellPrice, &requiresRecipe)
	} else {
		err = r.db.QueryRowContext(ctx, query, productID).Scan(&sellPrice, &requiresRecipe)
	}
	if err != nil {
		return 0, false, err
	}
	return sellPrice, requiresRecipe, nil
}

func (r *SQLSaleRepository) GetRecipeIngredients(ctx context.Context, tx *sql.Tx, parentProductID int64) ([]models.ProductRecipe, error) {
	query := `SELECT child_product_id, quantity FROM product_recipes WHERE parent_product_id = $1`
	var rows *sql.Rows
	var err error
	if tx != nil {
		rows, err = tx.QueryContext(ctx, query, parentProductID)
	} else {
		rows, err = r.db.QueryContext(ctx, query, parentProductID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []models.ProductRecipe
	for rows.Next() {
		var pr models.ProductRecipe
		pr.ParentProductID = parentProductID
		if err := rows.Scan(&pr.ChildProductID, &pr.Quantity); err != nil {
			return nil, err
		}
		recipes = append(recipes, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return recipes, nil
}

func (r *SQLSaleRepository) DeductStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error {
	query := `UPDATE inventories SET current_stock = current_stock - $1 WHERE product_id = $2`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, quantity, productID)
	} else {
		_, err = r.db.ExecContext(ctx, query, quantity, productID)
	}
	return err
}

// ListSales retrieves paginated sales with optional filters and computes pagination metadata and summary stats.
func (r *SQLSaleRepository) ListSales(ctx context.Context, filter models.SaleFilter) ([]models.Sale, models.SalePagination, models.SaleSummaryStats, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 10
	} else if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	baseWhere := " WHERE 1=1"
	var args []any
	argIdx := 1

	if filter.FromDate != nil {
		baseWhere += fmt.Sprintf(" AND s.sale_date >= $%d", argIdx)
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		baseWhere += fmt.Sprintf(" AND s.sale_date <= $%d", argIdx)
		args = append(args, *filter.ToDate)
		argIdx++
	}
	if filter.PaymentMethod != "" {
		baseWhere += fmt.Sprintf(" AND LOWER(s.payment_method) = LOWER($%d)", argIdx)
		args = append(args, filter.PaymentMethod)
		argIdx++
	}
	if filter.UserID != nil && *filter.UserID > 0 {
		baseWhere += fmt.Sprintf(" AND s.user_id = $%d", argIdx)
		args = append(args, *filter.UserID)
		argIdx++
	}

	countQuery := "SELECT COUNT(*), COALESCE(SUM(s.total_amount), 0) FROM sales s" + baseWhere
	var totalItems int64
	var totalAmount float64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalItems, &totalAmount); err != nil {
		return nil, models.SalePagination{}, models.SaleSummaryStats{}, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	pagination := models.SalePagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
	summary := models.SaleSummaryStats{
		TotalAmount: totalAmount,
		TotalCount:  totalItems,
	}

	dataQuery := fmt.Sprintf(`SELECT 
		s.id, s.user_id, u.full_name, s.sale_date, s.total_amount, s.payment_method, s.created_at,
		(SELECT COUNT(*) FROM sale_details sd WHERE sd.sale_id = s.id) AS items_count
	FROM sales s
	LEFT JOIN users u ON s.user_id = u.id
	%s
	ORDER BY s.sale_date DESC, s.id DESC
	LIMIT $%d OFFSET $%d`, baseWhere, argIdx, argIdx+1)

	queryArgs := append(args, pageSize, offset)
	rows, err := r.db.QueryContext(ctx, dataQuery, queryArgs...)
	if err != nil {
		return nil, pagination, summary, err
	}
	defer rows.Close()

	sales := make([]models.Sale, 0, pageSize)
	for rows.Next() {
		var s models.Sale
		var userName sql.NullString
		if err := rows.Scan(
			&s.ID, &s.UserID, &userName, &s.SaleDate, &s.TotalAmount, &s.PaymentMethod, &s.CreatedAt, &s.ItemsCount,
		); err != nil {
			return nil, pagination, summary, err
		}
		if userName.Valid {
			s.UserName = &userName.String
		}
		sales = append(sales, s)
	}

	if err := rows.Err(); err != nil {
		return nil, pagination, summary, err
	}

	return sales, pagination, summary, nil
}

// GetSaleByID retrieves a single sale header and its line items joined with products.
func (r *SQLSaleRepository) GetSaleByID(ctx context.Context, id int64) (*models.Sale, error) {
	headerQuery := `SELECT 
		s.id, s.user_id, u.full_name, s.sale_date, s.total_amount, s.payment_method, s.created_at
	FROM sales s
	LEFT JOIN users u ON s.user_id = u.id
	WHERE s.id = $1`

	var s models.Sale
	var userName sql.NullString
	err := r.db.QueryRowContext(ctx, headerQuery, id).Scan(
		&s.ID, &s.UserID, &userName, &s.SaleDate, &s.TotalAmount, &s.PaymentMethod, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if userName.Valid {
		s.UserName = &userName.String
	}

	detailsQuery := `SELECT 
		sd.id, sd.sale_id, sd.product_id, p.name, sd.quantity, sd.historical_unit_price, sd.subtotal
	FROM sale_details sd
	LEFT JOIN products p ON sd.product_id = p.id
	WHERE sd.sale_id = $1
	ORDER BY sd.id ASC`

	rows, err := r.db.QueryContext(ctx, detailsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var details []models.SaleDetail
	for rows.Next() {
		var d models.SaleDetail
		var prodName sql.NullString
		if err := rows.Scan(
			&d.ID, &d.SaleID, &d.ProductID, &prodName, &d.Quantity, &d.HistoricalUnitPrice, &d.Subtotal,
		); err != nil {
			return nil, err
		}
		if prodName.Valid {
			d.ProductName = &prodName.String
		}
		details = append(details, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	s.Details = details
	s.ItemsCount = len(details)
	return &s, nil
}

