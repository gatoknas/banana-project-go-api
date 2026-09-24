package repository

import (
	"context"
	"database/sql"

	"org.banana.project/api/internal/models"
)

// UnitOfMeasureRepository defines data access methods for units of measure.
type UnitOfMeasureRepository interface {
	List(ctx context.Context) ([]models.UnitOfMeasure, error)
	GetByID(ctx context.Context, id int64) (*models.UnitOfMeasure, error)
}

// SQLUnitOfMeasureRepository implements UnitOfMeasureRepository with PostgreSQL.
type SQLUnitOfMeasureRepository struct {
	db *sql.DB
}

// NewSQLUnitOfMeasureRepository creates a new instance of SQLUnitOfMeasureRepository.
func NewSQLUnitOfMeasureRepository(db *sql.DB) *SQLUnitOfMeasureRepository {
	return &SQLUnitOfMeasureRepository{db: db}
}

// List returns all units of measure ordered by name ascending.
func (r *SQLUnitOfMeasureRepository) List(ctx context.Context) ([]models.UnitOfMeasure, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, abbreviation, created_at FROM units_of_measure ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	units := make([]models.UnitOfMeasure, 0, 16)
	for rows.Next() {
		var u models.UnitOfMeasure
		if err := rows.Scan(&u.ID, &u.Name, &u.Abbreviation, &u.CreatedAt); err != nil {
			return nil, err
		}
		units = append(units, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return units, nil
}

// GetByID returns a single unit of measure by its primary key ID.
func (r *SQLUnitOfMeasureRepository) GetByID(ctx context.Context, id int64) (*models.UnitOfMeasure, error) {
	var u models.UnitOfMeasure
	err := r.db.QueryRowContext(ctx, `SELECT id, name, abbreviation, created_at FROM units_of_measure WHERE id = $1`, id).
		Scan(&u.ID, &u.Name, &u.Abbreviation, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
