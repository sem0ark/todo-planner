package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Initialize database connection
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		log.Fatal("DATABASE_URL not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
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
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := ApplyMigrations(ctx, db, GetMigrations()); err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	// Initialize API handlers with CORS middleware
	corsMiddleware := NewCORSMiddleware(corsOrigins)
	api := NewAPI(db)

	// Register routes
	http.HandleFunc("/health", api.HealthHandler)
	http.HandleFunc("/todos", corsMiddleware(api.TodosHandler))
	http.HandleFunc("/todos/", corsMiddleware(api.TodoHandler))

	// Start server
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
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
