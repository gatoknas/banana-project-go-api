package repository

import (
	"context"
	"database/sql"

	"org.banana.project/api/internal/models"
)

type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	List(ctx context.Context) ([]models.User, error)
	Create(ctx context.Context, u *models.User) (int64, error)
	Update(ctx context.Context, u *models.User) error
	Delete(ctx context.Context, id int64) error
}

type SQLUserRepository struct {
	db *sql.DB
}

func NewSQLUserRepository(db *sql.DB) *SQLUserRepository {
	return &SQLUserRepository{db: db}
}

func (r *SQLUserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	var u models.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, full_name, username, password_hash, role, is_active, created_at 
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.FullName, &u.Username, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *SQLUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, full_name, username, password_hash, role, is_active, created_at 
		 FROM users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.FullName, &u.Username, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *SQLUserRepository) List(ctx context.Context) ([]models.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, full_name, username, password_hash, role, is_active, created_at 
		 FROM users ORDER BY full_name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0, 16)
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ID, &u.FullName, &u.Username, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *SQLUserRepository) Create(ctx context.Context, u *models.User) (int64, error) {
	var userID int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (full_name, username, password_hash, role, is_active) 
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		u.FullName, u.Username, u.PasswordHash, u.Role, u.IsActive,
	).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (r *SQLUserRepository) Update(ctx context.Context, u *models.User) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users 
		 SET full_name = $1, username = $2, password_hash = $3, role = $4, is_active = $5
		 WHERE id = $6`,
		u.FullName, u.Username, u.PasswordHash, u.Role, u.IsActive, u.ID,
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

func (r *SQLUserRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
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
