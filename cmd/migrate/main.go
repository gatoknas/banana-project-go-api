// Command migrate runs SQL migrations against the database.
//
// Usage:
//
//	go run ./cmd/migrate
//	go run ./cmd/migrate -file db/migrations/0002_add_cafeteria_category.sql
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dbURL := flag.String("database-url", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	targetFile := flag.String("file", "", "specific SQL migration file to apply")
	flag.Parse()

	if *dbURL == "" {
		log.Fatal("DATABASE_URL is required (set in .env or pass -database-url)")
	}

	db, err := sql.Open("pgx", *dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	fmt.Println("Connected to database successfully.")

	// Determine files to apply
	var filesToApply []string
	if *targetFile != "" {
		filesToApply = []string{*targetFile}
	} else {
		matches, err := filepath.Glob("db/migrations/*.sql")
		if err != nil {
			log.Fatalf("failed to list migrations: %v", err)
		}
		sort.Strings(matches)
		filesToApply = matches
	}

	if len(filesToApply) == 0 {
		fmt.Println("No migration files found to apply.")
		return
	}

	for _, file := range filesToApply {
		fmt.Printf("Applying migration: %s ...\n", file)
		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("failed to read file %s: %v", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			log.Fatalf("failed to apply migration %s: %v", file, err)
		}
		fmt.Printf("Successfully applied %s\n", file)
	}

	// Verify Cafeteria category in categories table
	var id int64
	var name string
	err = db.QueryRow("SELECT id, name FROM categories WHERE name = 'Cafeteria'").Scan(&id, &name)
	if err != nil {
		fmt.Printf("Note: Check categories query: %v\n", err)
	} else {
		fmt.Printf("Verified category in database: ID=%d, Name=%s\n", id, name)
	}
}
