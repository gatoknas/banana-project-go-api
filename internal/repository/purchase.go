package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"org.banana.project/api/internal/models"
)

// PurchaseRepository defines data access methods for purchases and inventory restock.
type PurchaseRepository interface {
	CreatePurchase(ctx context.Context, tx *sql.Tx, p *models.Purchase) (int64, error)
	CreatePurchaseDetail(ctx context.Context, tx *sql.Tx, pd *models.PurchaseDetail) (int64, error)
	GetProductStockAndCost(ctx context.Context, tx *sql.Tx, productID int64) (currentStock float64, averageCost float64, err error)
	UpdateProductAverageCost(ctx context.Context, tx *sql.Tx, productID int64, newAverageCost float64) error
	AddStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error
	ListPurchases(ctx context.Context, supplierID *int64, fromDate, toDate *time.Time) ([]models.Purchase, error)
	GetPurchaseByID(ctx context.Context, id int64) (*models.Purchase, error)
	GetPurchaseDetails(ctx context.Context, purchaseID int64) ([]models.PurchaseDetail, error)
	UpdatePurchase(ctx context.Context, tx *sql.Tx, p *models.Purchase) error
	DeletePurchaseDetails(ctx context.Context, tx *sql.Tx, purchaseID int64) error
	DeductStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error
}

// SQLPurchaseRepository implements PurchaseRepository using PostgreSQL.
type SQLPurchaseRepository struct {
	db *sql.DB
}

// NewSQLPurchaseRepository creates a new instance of SQLPurchaseRepository.
func NewSQLPurchaseRepository(db *sql.DB) *SQLPurchaseRepository {
	return &SQLPurchaseRepository{db: db}
}

// CreatePurchase inserts a new purchase header.
func (r *SQLPurchaseRepository) CreatePurchase(ctx context.Context, tx *sql.Tx, p *models.Purchase) (int64, error) {
	query := `INSERT INTO purchases (supplier_id, purchase_date, invoice_number, total_amount, notes) 
	          VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	var id int64
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, query, p.SupplierID, p.PurchaseDate, p.InvoiceNumber, p.TotalAmount, p.Notes).Scan(&id, &p.CreatedAt)
	} else {
		err = r.db.QueryRowContext(ctx, query, p.SupplierID, p.PurchaseDate, p.InvoiceNumber, p.TotalAmount, p.Notes).Scan(&id, &p.CreatedAt)
	}
	if err != nil {
		return 0, err
	}
	p.ID = id
	return id, nil
}

// CreatePurchaseDetail inserts a line item into purchase_details.
func (r *SQLPurchaseRepository) CreatePurchaseDetail(ctx context.Context, tx *sql.Tx, pd *models.PurchaseDetail) (int64, error) {
	query := `INSERT INTO purchase_details (purchase_id, product_id, purchase_unit_id, quantity_purchased, unit_cost, conversion_factor) 
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	var id int64
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, query, pd.PurchaseID, pd.ProductID, pd.PurchaseUnitID, pd.QuantityPurchased, pd.UnitCost, pd.ConversionFactor).Scan(&id)
	} else {
		err = r.db.QueryRowContext(ctx, query, pd.PurchaseID, pd.ProductID, pd.PurchaseUnitID, pd.QuantityPurchased, pd.UnitCost, pd.ConversionFactor).Scan(&id)
	}
	if err != nil {
		return 0, err
	}
	pd.ID = id
	return id, nil
}

// GetProductStockAndCost retrieves the product's current stock and average cost for restock recalculation.
func (r *SQLPurchaseRepository) GetProductStockAndCost(ctx context.Context, tx *sql.Tx, productID int64) (float64, float64, error) {
	query := `SELECT COALESCE(inv.current_stock, 0), p.average_cost 
	          FROM products p 
	          LEFT JOIN inventories inv ON inv.product_id = p.id 
	          WHERE p.id = $1 FOR UPDATE OF p`
	var currentStock float64
	var averageCost float64
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, query, productID).Scan(&currentStock, &averageCost)
	} else {
		err = r.db.QueryRowContext(ctx, query, productID).Scan(&currentStock, &averageCost)
	}
	if err != nil {
		return 0, 0, err
	}
	return currentStock, averageCost, nil
}

// UpdateProductAverageCost updates the product's weighted average cost.
func (r *SQLPurchaseRepository) UpdateProductAverageCost(ctx context.Context, tx *sql.Tx, productID int64, newAverageCost float64) error {
	query := `UPDATE products SET average_cost = $1 WHERE id = $2`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, newAverageCost, productID)
	} else {
		_, err = r.db.ExecContext(ctx, query, newAverageCost, productID)
	}
	return err
}

// AddStock increases the inventory stock by the given quantity (upserting if not initialized).
func (r *SQLPurchaseRepository) AddStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error {
	query := `INSERT INTO inventories (product_id, current_stock, minimum_stock, maximum_stock, updated_at) 
	          VALUES ($1, $2, 0, 0, CURRENT_TIMESTAMP) 
	          ON CONFLICT (product_id) DO UPDATE 
	          SET current_stock = inventories.current_stock + EXCLUDED.current_stock, 
	              updated_at = CURRENT_TIMESTAMP`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, productID, quantity)
	} else {
		_, err = r.db.ExecContext(ctx, query, productID, quantity)
	}
	return err
}

// ListPurchases retrieves purchases with optional supplier and date range filters.
func (r *SQLPurchaseRepository) ListPurchases(ctx context.Context, supplierID *int64, fromDate, toDate *time.Time) ([]models.Purchase, error) {
	query := `SELECT p.id, p.supplier_id, s.company_name, p.purchase_date, p.invoice_number, 
	                 p.total_amount, p.notes, p.created_at 
	          FROM purchases p 
	          JOIN suppliers s ON s.id = p.supplier_id 
	          WHERE 1=1`
	var args []any
	argIndex := 1

	if supplierID != nil && *supplierID > 0 {
		query += fmt.Sprintf(" AND p.supplier_id = $%d", argIndex)
		args = append(args, *supplierID)
		argIndex++
	}

	if fromDate != nil {
		query += fmt.Sprintf(" AND p.purchase_date >= $%d", argIndex)
		args = append(args, *fromDate)
		argIndex++
	}

	if toDate != nil {
		query += fmt.Sprintf(" AND p.purchase_date <= $%d", argIndex)
		args = append(args, *toDate)
		argIndex++
	}

	query += ` ORDER BY p.purchase_date DESC, p.id DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	purchases := make([]models.Purchase, 0, 32)
	for rows.Next() {
		var p models.Purchase
		err := rows.Scan(
			&p.ID, &p.SupplierID, &p.SupplierName, &p.PurchaseDate,
			&p.InvoiceNumber, &p.TotalAmount, &p.Notes, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		purchases = append(purchases, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return purchases, nil
}

// GetPurchaseByID retrieves a single purchase header by ID.
func (r *SQLPurchaseRepository) GetPurchaseByID(ctx context.Context, id int64) (*models.Purchase, error) {
	query := `SELECT p.id, p.supplier_id, s.company_name, p.purchase_date, p.invoice_number, 
	                 p.total_amount, p.notes, p.created_at 
	          FROM purchases p 
	          JOIN suppliers s ON s.id = p.supplier_id 
	          WHERE p.id = $1`
	var p models.Purchase
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.SupplierID, &p.SupplierName, &p.PurchaseDate,
		&p.InvoiceNumber, &p.TotalAmount, &p.Notes, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPurchaseDetails retrieves all line items for a purchase.
func (r *SQLPurchaseRepository) GetPurchaseDetails(ctx context.Context, purchaseID int64) ([]models.PurchaseDetail, error) {
	query := `SELECT pd.id, pd.purchase_id, pd.product_id, prod.name, 
	                 pd.purchase_unit_id, u.name, pd.quantity_purchased, pd.unit_cost, pd.conversion_factor 
	          FROM purchase_details pd 
	          JOIN products prod ON prod.id = pd.product_id 
	          LEFT JOIN units_of_measure u ON u.id = pd.purchase_unit_id 
	          WHERE pd.purchase_id = $1 
	          ORDER BY pd.id ASC`
	rows, err := r.db.QueryContext(ctx, query, purchaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	details := make([]models.PurchaseDetail, 0, 16)
	for rows.Next() {
		var pd models.PurchaseDetail
		err := rows.Scan(
			&pd.ID, &pd.PurchaseID, &pd.ProductID, &pd.ProductName,
			&pd.PurchaseUnitID, &pd.PurchaseUnitName,
			&pd.QuantityPurchased, &pd.UnitCost, &pd.ConversionFactor,
		)
		if err != nil {
			return nil, err
		}
		pd.Subtotal = pd.QuantityPurchased * pd.UnitCost
		pd.BaseQuantity = pd.QuantityPurchased * pd.ConversionFactor
		details = append(details, pd)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return details, nil
}

// UpdatePurchase updates the purchase header.
func (r *SQLPurchaseRepository) UpdatePurchase(ctx context.Context, tx *sql.Tx, p *models.Purchase) error {
	query := `UPDATE purchases 
	          SET supplier_id = $1, purchase_date = $2, invoice_number = $3, total_amount = $4, notes = $5 
	          WHERE id = $6`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, p.SupplierID, p.PurchaseDate, p.InvoiceNumber, p.TotalAmount, p.Notes, p.ID)
	} else {
		_, err = r.db.ExecContext(ctx, query, p.SupplierID, p.PurchaseDate, p.InvoiceNumber, p.TotalAmount, p.Notes, p.ID)
	}
	return err
}

// DeletePurchaseDetails deletes all detail line items for a given purchase ID.
func (r *SQLPurchaseRepository) DeletePurchaseDetails(ctx context.Context, tx *sql.Tx, purchaseID int64) error {
	query := `DELETE FROM purchase_details WHERE purchase_id = $1`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, purchaseID)
	} else {
		_, err = r.db.ExecContext(ctx, query, purchaseID)
	}
	return err
}

// DeductStock decreases the inventory stock by the given quantity.
func (r *SQLPurchaseRepository) DeductStock(ctx context.Context, tx *sql.Tx, productID int64, quantity float64) error {
	query := `UPDATE inventories 
	          SET current_stock = current_stock - $1, 
	              updated_at = CURRENT_TIMESTAMP 
	          WHERE product_id = $2`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, quantity, productID)
	} else {
		_, err = r.db.ExecContext(ctx, query, quantity, productID)
	}
	return err
}

