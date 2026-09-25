package repository

import (
	"context"
	"database/sql"
	"strings"

	"org.banana.project/api/internal/models"
)

// SupplierRepository defines data access methods for suppliers.
type SupplierRepository interface {
	Create(ctx context.Context, tx *sql.Tx, s *models.Supplier) (int64, error)
	GetByID(ctx context.Context, id int64) (*models.Supplier, error)
	GetByTaxID(ctx context.Context, taxID string) (*models.Supplier, error)
	List(ctx context.Context, searchQuery string) ([]models.Supplier, error)
	Update(ctx context.Context, s *models.Supplier) error
	Delete(ctx context.Context, tx *sql.Tx, id int64) error
	HasPurchases(ctx context.Context, id int64) (bool, error)
}

// SQLSupplierRepository implements SupplierRepository using PostgreSQL.
type SQLSupplierRepository struct {
	db *sql.DB
}

// NewSQLSupplierRepository creates a new instance of SQLSupplierRepository.
func NewSQLSupplierRepository(db *sql.DB) *SQLSupplierRepository {
	return &SQLSupplierRepository{db: db}
}

// Create inserts a new supplier into the database.
func (r *SQLSupplierRepository) Create(ctx context.Context, tx *sql.Tx, s *models.Supplier) (int64, error) {
	query := `INSERT INTO suppliers (tax_id, company_name, contact_name, phone, email, description) 
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`
	var id int64
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, query, s.TaxID, s.CompanyName, s.ContactName, s.Phone, s.Email, s.Description).Scan(&id, &s.CreatedAt)
	} else {
		err = r.db.QueryRowContext(ctx, query, s.TaxID, s.CompanyName, s.ContactName, s.Phone, s.Email, s.Description).Scan(&id, &s.CreatedAt)
	}
	if err != nil {
		return 0, err
	}
	s.ID = id
	return id, nil
}

// GetByID retrieves a supplier by ID.
func (r *SQLSupplierRepository) GetByID(ctx context.Context, id int64) (*models.Supplier, error) {
	var s models.Supplier
	query := `SELECT id, tax_id, company_name, contact_name, phone, email, description, created_at FROM suppliers WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.TaxID, &s.CompanyName, &s.ContactName, &s.Phone, &s.Email, &s.Description, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetByTaxID retrieves a supplier by tax_id (NIT/Cédula).
func (r *SQLSupplierRepository) GetByTaxID(ctx context.Context, taxID string) (*models.Supplier, error) {
	var s models.Supplier
	query := `SELECT id, tax_id, company_name, contact_name, phone, email, description, created_at FROM suppliers WHERE tax_id = $1`
	err := r.db.QueryRowContext(ctx, query, taxID).Scan(&s.ID, &s.TaxID, &s.CompanyName, &s.ContactName, &s.Phone, &s.Email, &s.Description, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// List retrieves suppliers with optional search filtering by company_name, tax_id, or contact_name.
func (r *SQLSupplierRepository) List(ctx context.Context, searchQuery string) ([]models.Supplier, error) {
	var query string
	var args []any

	search := strings.TrimSpace(searchQuery)
	if search != "" {
		pattern := "%" + search + "%"
		query = `SELECT id, tax_id, company_name, contact_name, phone, email, description, created_at 
		         FROM suppliers 
		         WHERE company_name ILIKE $1 OR tax_id ILIKE $1 OR contact_name ILIKE $1 OR description ILIKE $1 
		         ORDER BY company_name ASC`
		args = append(args, pattern)
	} else {
		query = `SELECT id, tax_id, company_name, contact_name, phone, email, description, created_at 
		         FROM suppliers 
		         ORDER BY company_name ASC`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	suppliers := make([]models.Supplier, 0, 32)
	for rows.Next() {
		var s models.Supplier
		if err := rows.Scan(&s.ID, &s.TaxID, &s.CompanyName, &s.ContactName, &s.Phone, &s.Email, &s.Description, &s.CreatedAt); err != nil {
			return nil, err
		}
		suppliers = append(suppliers, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return suppliers, nil
}

// Update modifies an existing supplier.
func (r *SQLSupplierRepository) Update(ctx context.Context, s *models.Supplier) error {
	query := `UPDATE suppliers 
	          SET tax_id = $1, company_name = $2, contact_name = $3, phone = $4, email = $5, description = $6 
	          WHERE id = $7`
	res, err := r.db.ExecContext(ctx, query, s.TaxID, s.CompanyName, s.ContactName, s.Phone, s.Email, s.Description, s.ID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Delete removes a supplier by ID.
func (r *SQLSupplierRepository) Delete(ctx context.Context, tx *sql.Tx, id int64) error {
	query := `DELETE FROM suppliers WHERE id = $1`
	var res sql.Result
	var err error
	if tx != nil {
		res, err = tx.ExecContext(ctx, query, id)
	} else {
		res, err = r.db.ExecContext(ctx, query, id)
	}
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// HasPurchases checks whether a supplier has linked purchases in the purchases table.
func (r *SQLSupplierRepository) HasPurchases(ctx context.Context, id int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM purchases WHERE supplier_id = $1`, id).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
