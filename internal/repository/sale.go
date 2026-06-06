package repository

import (
	"context"
	"database/sql"

	"org.banana.project/api/internal/models"
)

type SaleRepository interface {
	CreateSale(ctx context.Context, tx *sql.Tx, s *models.Sale) (int64, error)
	CreateSaleDetail(ctx context.Context, tx *sql.Tx, d *models.SaleDetail) (int64, error)
	GetProductDetails(ctx context.Context, tx *sql.Tx, productID int64) (float64, bool, error)
	GetRecipeIngredients(ctx context.Context, tx *sql.Tx, parentProductID int64) ([]models.ProductRecipe, error)
	DeductStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error
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
