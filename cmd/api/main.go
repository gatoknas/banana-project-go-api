package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
	"github.com/swaggo/swag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	_ "org.banana.project/api/docs"
	"org.banana.project/api/internal/auth"
	"org.banana.project/api/internal/database"
	"org.banana.project/api/internal/handlers"
	"org.banana.project/api/internal/middleware"
	"org.banana.project/api/internal/repository"
	"org.banana.project/api/internal/service"
)

// @title           Banana Project Go REST API
// @version         1.0
// @description     This is the API server for Banana Project.
// @host            localhost:8082
// @BasePath        /

func main() {
	// Load environment variables from .env file
	envLoadErr := godotenv.Load()

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Initialize Logger
	logger, err := initLogger(env)
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	if envLoadErr != nil {
		logger.Info("No .env file found, relying on environment variables")
	}

	// Validate required environment variables
	requiredEnvVars := []string{"DATABASE_URL", "JWT_SECRET"}
	missingVars := false
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			logger.Error("Missing required environment variable", zap.String("variable", envVar))
			missingVars = true
		}
	}
	if missingVars {
		logger.Fatal("Application failed to start due to missing environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")

	// Initialize Database
	if err := database.Init(dbURL); err != nil {
		logger.Warn("Failed to connect to database", zap.Error(err))
	} else {
		logger.Info("Successfully connected to the database")
		seedDefaultAdmin(database.DB, logger)
		defer func() {
			if err := database.Close(); err != nil {
				logger.Error("Error closing database", zap.Error(err))
			}
		}()
	}

	// Setup Router
	router := setupRouter(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	if len(port) > 0 && port[0] != ':' {
		port = ":" + port
	}

	// Print Startup Banner
	printBanner(env, port)

	logger.Info("Server is starting", zap.String("port", port), zap.String("env", env))
	corsRouter := middleware.CORS(router)
	if err := http.ListenAndServe(port, corsRouter); err != nil {
		logger.Fatal("Could not start server", zap.Error(err))
	}
}

// initLogger configures and returns a new zap.Logger based on the environment
func initLogger(env string) (*zap.Logger, error) {
	if env == "production" {
		return zap.NewProduction()
	}
	// Use a clean console development logger with colorized output for local debugging
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return config.Build()
}

// setupRouter instantiates and wires services, repositories, handlers, and middlewares
func setupRouter(logger *zap.Logger) http.Handler {
	mux := http.NewServeMux()

	// Wire user dependencies
	userRepo := repository.NewSQLUserRepository(database.DB)
	userService := service.NewUserService(userRepo, database.DB)
	userHandler := handlers.NewUserHandler(userService, logger)

	// Instantiate handlers with dependencies
	authHandler := handlers.NewAuthHandler(userService, logger)
	healthHandler := handlers.NewHealthHandler(logger, database.DB)
	helloHandler := handlers.NewHelloHandler(logger)

	// Public endpoints
	mux.HandleFunc("GET /status", healthHandler.Health)
	mux.HandleFunc("GET /hello", helloHandler.Hello)
	mux.HandleFunc("GET /hello/logo.png", helloHandler.Logo)
	mux.HandleFunc("POST /login", authHandler.Login)

	// API Documentation (Scalar UI)
	mux.HandleFunc("GET /docs/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		doc, err := swag.ReadDoc("swagger")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(doc))
	})
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!doctype html>
<html>
  <head>
    <title>API Reference - Banana Project</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <script id="api-reference" data-url="/docs/swagger.json"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`))
	})

	// Protected endpoints (under /api/v1/)
	// For Go 1.22+, we create a handler for the sub-route and wrap it in middleware
	protectedMux := http.NewServeMux()

	// Define RBAC role groups
	adminOnly := middleware.RequireRoles("ayurami-admin")
	salesAndAdmin := middleware.RequireRoles("ayurami-admin", "ayurami-salesperson")

	// User CRUD endpoints (Admin only)
	protectedMux.Handle("POST /users", adminOnly(http.HandlerFunc(userHandler.Create)))
	protectedMux.Handle("GET /users", adminOnly(http.HandlerFunc(userHandler.List)))
	protectedMux.Handle("GET /users/{id}", adminOnly(http.HandlerFunc(userHandler.Get)))
	protectedMux.Handle("PUT /users/{id}", adminOnly(http.HandlerFunc(userHandler.Update)))
	protectedMux.Handle("DELETE /users/{id}", adminOnly(http.HandlerFunc(userHandler.Delete)))

	// Wire sale dependencies
	saleRepo := repository.NewSQLSaleRepository(database.DB)
	saleService := service.NewSaleService(saleRepo, database.DB)
	saleHandler := handlers.NewSaleHandler(saleService, logger)

	protectedMux.Handle("POST /sales", salesAndAdmin(http.HandlerFunc(saleHandler.Create)))

	// Wire product dependencies
	productRepo := repository.NewSQLProductRepository(database.DB)
	productService := service.NewProductService(productRepo, database.DB)
	productHandler := handlers.NewProductHandler(productService, logger)

	protectedMux.Handle("POST /products", adminOnly(http.HandlerFunc(productHandler.Create)))
	protectedMux.Handle("GET /products", salesAndAdmin(http.HandlerFunc(productHandler.List)))
	protectedMux.Handle("GET /products/{id}", salesAndAdmin(http.HandlerFunc(productHandler.Get)))
	protectedMux.Handle("PUT /products/{id}", adminOnly(http.HandlerFunc(productHandler.Update)))
	protectedMux.Handle("DELETE /products/{id}", adminOnly(http.HandlerFunc(productHandler.Delete)))

	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", middleware.RequireAuth(protectedMux)))

	return mux
}

// printBanner displays the themed console startup banner
func printBanner(env, port string) {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFE135")). // Banana Yellow
		Padding(0, 1)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true, false, true, true).
		BorderForeground(lipgloss.Color("#FFE135")).
		Padding(1, 2).
		Margin(1, 0, 1, 2)

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89B4FA")) // Ice Blue
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))            // Green

	content := fmt.Sprintf(
		"🍌 %s\n\n%s %s\n%s %s\n%s %s\n%s %s",
		titleStyle.Render("BANANA PROJECT GO REST API"),
		labelStyle.Render("🌍 Env:   "), valueStyle.Render(env),
		labelStyle.Render("🚀 Port:  "), valueStyle.Render(port),
		labelStyle.Render("✨ Status:"), valueStyle.Render("Online"),
		labelStyle.Render("🔗 URL:   "), valueStyle.Render("http://localhost"+port),
	)

	fmt.Println(boxStyle.Render(content))
}

// seedDefaultAdmin checks if the users table is empty and inserts a default admin user
func seedDefaultAdmin(db *sql.DB, logger *zap.Logger) {
	var count int
	// Verify table exists first (if database has not run migrations yet it could error)
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		logger.Warn("Skipped seeding admin user (users table might not exist yet)", zap.Error(err))
		return
	}

	if count == 0 {
		logger.Info("No users found in database. Seeding default admin user...")
		hash, err := auth.HashPassword("adminpassword")
		if err != nil {
			logger.Error("Failed to hash default admin password", zap.Error(err))
			return
		}

		_, err = db.Exec(
			`INSERT INTO users (full_name, username, password_hash, role, is_active) 
			 VALUES ($1, $2, $3, $4, $5)`,
			"Admin Ayurami", "admin", hash, "ayurami-admin", true,
		)
		if err != nil {
			logger.Error("Failed to seed default admin user", zap.Error(err))
			return
		}
		logger.Info("Successfully seeded default admin user (username: admin, password: adminpassword)")
	}
}

