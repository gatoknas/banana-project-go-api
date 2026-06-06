package repository

import (
	"context"
	"database/sql"

	"org.banana.project/api/internal/models"
)

type ProductRepository interface {
	GetByID(ctx context.Context, id int64) (*models.Product, error)
	List(ctx context.Context) ([]models.Product, error)
	Create(ctx context.Context, tx *sql.Tx, p *models.Product) (int64, error)
	Update(ctx context.Context, p *models.Product) error
	Delete(ctx context.Context, tx *sql.Tx, id int64) error
	InitializeInventory(ctx context.Context, tx *sql.Tx, productID int64) error
	DeleteAllowedAdditions(ctx context.Context, tx *sql.Tx, productID int64) error
	DeleteProductRecipes(ctx context.Context, tx *sql.Tx, productID int64) error
	DeleteInventory(ctx context.Context, tx *sql.Tx, productID int64) error
}

type SQLProductRepository struct {
	db *sql.DB
}

func NewSQLProductRepository(db *sql.DB) *SQLProductRepository {
	return &SQLProductRepository{db: db}
}

func (r *SQLProductRepository) GetByID(ctx context.Context, id int64) (*models.Product, error) {
	var p models.Product
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, category_id, unit_of_measure_id, sell_price, average_cost, is_for_sale, requires_recipe, created_at, updated_at 
		 FROM products WHERE id = $1`,
		id,
	).Scan(
		&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.UnitOfMeasureID,
		&p.SellPrice, &p.AverageCost, &p.IsForSale, &p.RequiresRecipe,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *SQLProductRepository) List(ctx context.Context) ([]models.Product, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, description, category_id, unit_of_measure_id, sell_price, average_cost, is_for_sale, requires_recipe, created_at, updated_at FROM products ORDER BY name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]models.Product, 0, 32)
	for rows.Next() {
		var p models.Product
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.UnitOfMeasureID,
			&p.SellPrice, &p.AverageCost, &p.IsForSale, &p.RequiresRecipe,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *SQLProductRepository) Create(ctx context.Context, tx *sql.Tx, p *models.Product) (int64, error) {
	var productID int64
	query := `INSERT INTO products (name, description, category_id, unit_of_measure_id, sell_price, average_cost, is_for_sale, requires_recipe) 
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, query, p.Name, p.Description, p.CategoryID, p.UnitOfMeasureID, p.SellPrice, p.AverageCost, p.IsForSale, p.RequiresRecipe).Scan(&productID)
	} else {
		err = r.db.QueryRowContext(ctx, query, p.Name, p.Description, p.CategoryID, p.UnitOfMeasureID, p.SellPrice, p.AverageCost, p.IsForSale, p.RequiresRecipe).Scan(&productID)
	}
	if err != nil {
		return 0, err
	}
	return productID, nil
}

func (r *SQLProductRepository) InitializeInventory(ctx context.Context, tx *sql.Tx, productID int64) error {
	query := `INSERT INTO inventories (product_id, current_stock, minimum_stock, maximum_stock) VALUES ($1, 0, 0, 0)`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, productID)
	} else {
		_, err = r.db.ExecContext(ctx, query, productID)
	}
	return err
}

func (r *SQLProductRepository) Update(ctx context.Context, p *models.Product) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE products 
		 SET name = $1, description = $2, category_id = $3, unit_of_measure_id = $4, sell_price = $5, average_cost = $6, is_for_sale = $7, requires_recipe = $8
		 WHERE id = $9`,
		p.Name, p.Description, p.CategoryID, p.UnitOfMeasureID, p.SellPrice, p.AverageCost, p.IsForSale, p.RequiresRecipe, p.ID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLProductRepository) Delete(ctx context.Context, tx *sql.Tx, id int64) error {
	query := "DELETE FROM products WHERE id = $1"
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
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLProductRepository) DeleteAllowedAdditions(ctx context.Context, tx *sql.Tx, productID int64) error {
	query := "DELETE FROM allowed_additions WHERE main_product_id = $1 OR addition_product_id = $1"
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, productID)
	} else {
		_, err = r.db.ExecContext(ctx, query, productID)
	}
	return err
}

func (r *SQLProductRepository) DeleteProductRecipes(ctx context.Context, tx *sql.Tx, productID int64) error {
	query := "DELETE FROM product_recipes WHERE parent_product_id = $1 OR child_product_id = $1"
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, productID)
	} else {
		_, err = r.db.ExecContext(ctx, query, productID)
	}
	return err
}

func (r *SQLProductRepository) DeleteInventory(ctx context.Context, tx *sql.Tx, productID int64) error {
	query := "DELETE FROM inventories WHERE product_id = $1"
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, productID)
	} else {
		_, err = r.db.ExecContext(ctx, query, productID)
	}
	return err
}
