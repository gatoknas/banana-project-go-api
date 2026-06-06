package database

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

// Init initializes the database connection pool using the provided connection string.
func Init(connStr string) error {
	var err error
	DB, err = sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Verify the connection is actually working
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// Close closes the database connection pool.
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
