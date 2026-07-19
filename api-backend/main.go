package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Initialize structured logger
	logger := NewLogger("todo-planner-api")
	logger.Info("Starting todo-planner API server")

	// Initialize database connection
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		logger.Fatal("DATABASE_URL environment variable not set", nil)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET environment variable not set", nil)
	}

	// Parse CORS allowed origins from environment
	corsOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
	var corsOrigins []string
	if corsOriginsStr == "" {
		corsOrigins = []string{"*"}
	} else {
		for _, s := range strings.Split(corsOriginsStr, ",") {
			corsOrigins = append(corsOrigins, strings.TrimSpace(s))
		}
	}

	ctx := context.Background()
	db, err := initDatabase(ctx, connString)
	if err != nil {
		logger.Fatal("Failed to initialize database connection", err, map[string]interface{}{
			"connection_string_length": len(connString),
		})
	}
	defer db.Close()
	logger.Info("Database connection established")

	// Run migrations
	if err := ApplyMigrations(ctx, db, GetMigrations()); err != nil {
		logger.Fatal("Failed to apply database migrations", err)
	}
	logger.Info("Database migrations applied successfully")

	// Initialize API handlers with middlewares
	loggingMiddleware := LoggingMiddleware(logger)
	corsMiddleware := NewCORSMiddleware(corsOrigins)
	api := NewAPI(db, jwtSecret, logger)

	// Register routes with centralized middleware chain:
	// loggingMiddleware -> corsMiddleware -> authMiddleware (if needed) -> handler

	// Public routes (logging + CORS)
	http.HandleFunc("/health", loggingMiddleware(corsMiddleware(api.HealthHandler)))
	http.HandleFunc("/auth/register", loggingMiddleware(corsMiddleware(api.registerHandler)))
	http.HandleFunc("/auth/login", loggingMiddleware(corsMiddleware(api.loginHandler)))

	// Protected routes (logging + CORS + auth)
	http.HandleFunc("/account", loggingMiddleware(corsMiddleware(api.authMiddleware(api.deleteAccountHandler))))
	http.HandleFunc("/settings", loggingMiddleware(corsMiddleware(api.authMiddleware(api.settingsHandler))))
	http.HandleFunc("/devices", loggingMiddleware(corsMiddleware(api.authMiddleware(api.registerDeviceHandler))))
	http.HandleFunc("/sync", loggingMiddleware(corsMiddleware(api.authMiddleware(api.syncHandler))))
	http.HandleFunc("/categories", loggingMiddleware(corsMiddleware(api.authMiddleware(api.categoriesHandler))))
	http.HandleFunc("/categories/", loggingMiddleware(corsMiddleware(api.authMiddleware(api.categoriesHandler))))
	http.HandleFunc("/schedule", loggingMiddleware(corsMiddleware(api.authMiddleware(api.scheduleHandler))))
	http.HandleFunc("/schedule/", loggingMiddleware(corsMiddleware(api.authMiddleware(api.scheduleHandler))))
	http.HandleFunc("/template-groups", loggingMiddleware(corsMiddleware(api.authMiddleware(api.templateGroupsHandler))))
	http.HandleFunc("/template-groups/", loggingMiddleware(corsMiddleware(api.authMiddleware(api.templateGroupsHandler))))
	http.HandleFunc("/templates", loggingMiddleware(corsMiddleware(api.authMiddleware(api.dayTemplatesHandler))))
	http.HandleFunc("/templates/", loggingMiddleware(corsMiddleware(api.authMiddleware(api.dayTemplatesHandler))))
	http.HandleFunc("/day-records", loggingMiddleware(corsMiddleware(api.authMiddleware(api.dayRecordsHandler))))
	http.HandleFunc("/day-records/", loggingMiddleware(corsMiddleware(api.authMiddleware(api.dayRecordsHandler))))

	// Start server
	logger.Info("Server listening for requests", map[string]interface{}{
		"port":         port,
		"cors_origins": corsOrigins,
		"max_db_conns": 5,
	})

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Fatal("Server failed to start", err, map[string]interface{}{
			"port": port,
		})
	}
}

func initDatabase(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 5
	config.MinConns = 0
	config.MaxConnIdleTime = 2 * time.Minute
	config.MaxConnLifetime = 15 * time.Minute

	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return db, nil
}
