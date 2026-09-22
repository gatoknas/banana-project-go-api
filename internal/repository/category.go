package repository

import (
	"context"
	"database/sql"

	"org.banana.project/api/internal/models"
)

// CategoryRepository defines data access methods for categories.
type CategoryRepository interface {
	List(ctx context.Context) ([]models.Category, error)
	GetByID(ctx context.Context, id int64) (*models.Category, error)
}

// SQLCategoryRepository implements CategoryRepository with PostgreSQL.
type SQLCategoryRepository struct {
	db *sql.DB
}

// NewSQLCategoryRepository creates a new instance of SQLCategoryRepository.
func NewSQLCategoryRepository(db *sql.DB) *SQLCategoryRepository {
	return &SQLCategoryRepository{db: db}
}

// List returns all categories ordered by ID ascending.
func (r *SQLCategoryRepository) List(ctx context.Context) ([]models.Category, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, created_at FROM categories ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]models.Category, 0, 16)
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

// GetByID returns a single category by its primary key ID.
func (r *SQLCategoryRepository) GetByID(ctx context.Context, id int64) (*models.Category, error) {
	var c models.Category
	err := r.db.QueryRowContext(ctx, `SELECT id, name, created_at FROM categories WHERE id = $1`, id).
		Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
