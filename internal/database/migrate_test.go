package database_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"
	"org.banana.project/api/internal/database"
)

func TestRunMigrations(t *testing.T) {
	tests := []struct {
		name          string
		setupFiles    func(dir string)
		setupMock     func(mock sqlmock.Sqlmock)
		useSubDir     string
		wantErr       bool
	}{
		{
			name:       "Nonexistent directory yields no files and succeeds",
			setupFiles: func(dir string) {},
			setupMock:  func(mock sqlmock.Sqlmock) {},
			useSubDir:  "non_existent_dir_12345",
			wantErr:    false,
		},
		{
			name: "Applies migration files in alphabetical order successfully",
			setupFiles: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "0001_init.sql"), []byte("CREATE TABLE foo (id INT);"), 0644)
				_ = os.WriteFile(filepath.Join(dir, "0002_add_col.sql"), []byte("ALTER TABLE foo ADD COLUMN bar TEXT;"), 0644)
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("CREATE TABLE foo").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("ALTER TABLE foo").WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "Returns error if SQL execution fails",
			setupFiles: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "0001_fail.sql"), []byte("BAD SQL QUERY;"), 0644)
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("BAD SQL QUERY").WillReturnError(sqlmock.ErrCancelled)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			dir := t.TempDir()
			targetDir := dir
			if tt.useSubDir != "" {
				targetDir = filepath.Join(dir, tt.useSubDir)
			} else {
				tt.setupFiles(targetDir)
			}

			tt.setupMock(mock)

			logger := zap.NewNop()
			err = database.RunMigrations(db, targetDir, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunMigrations() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil && !tt.wantErr {
				t.Errorf("unmet mock expectations: %v", err)
			}
		})
	}
}
