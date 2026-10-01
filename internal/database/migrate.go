package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.uber.org/zap"
)

// RunMigrations reads and executes all SQL migration files in migrationsDir against db in alphabetical order.
func RunMigrations(db *sql.DB, migrationsDir string, logger *zap.Logger) error {
	pattern := filepath.Join(migrationsDir, "*.sql")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to search migration files in %s: %w", migrationsDir, err)
	}

	if len(matches) == 0 {
		if logger != nil {
			logger.Info("No migration files found to apply", zap.String("dir", migrationsDir))
		}
		return nil
	}

	sort.Strings(matches)

	for _, file := range matches {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
		if logger != nil {
			logger.Info("Successfully applied database migration", zap.String("file", file))
		}
	}

	return nil
}
