package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := NewLogger("todo-planner-api")
	logger.Info("Starting todo-planner API server")

	appMode := getEnvironment("APP_MODE", "cloud")
	if appMode != "cloud" && appMode != "local" {
		logger.Fatal("APP_MODE must be either cloud or local", nil, map[string]interface{}{
			"app_mode": appMode,
		})
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3648"
	}

	dataDirectory := getEnvironment("EMBEDDED_PG_DATA_DIR", "~/.todoplanner")
	var embeddedDatabase *EmbeddedDatabase
	var connectionString string
	var err error
	if appMode == "local" {
		embeddedPort, parseError := strconv.ParseUint(
			getEnvironment("EMBEDDED_PG_PORT", "54321"),
			10,
			32,
		)
		if parseError != nil {
			logger.Fatal("Invalid EMBEDDED_PG_PORT", parseError)
		}
		embeddedDatabase, err = NewEmbeddedDatabase(uint32(embeddedPort), dataDirectory)
		if err != nil {
			logger.Fatal("Failed to initialize embedded database", err)
		}
		connectionString = embeddedDatabase.ConnectionURL()
	} else {
		connectionString = os.Getenv("DATABASE_URL")
		if connectionString == "" {
			logger.Fatal("DATABASE_URL environment variable not set", nil)
		}
	}

	jwtSecret, err := jwtSecretForMode(appMode, dataDirectory)
	if err != nil {
		logger.Fatal("Failed to initialize JWT secret", err)
	}

	corsOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
	var corsOrigins []string
	if corsOriginsStr == "" {
		if appMode == "local" {
			corsOrigins = []string{"http://127.0.0.1", "http://localhost"}
		} else {
			corsOrigins = []string{"*"}
		}
	} else {
		for _, s := range strings.Split(corsOriginsStr, ",") {
			corsOrigins = append(corsOrigins, strings.TrimSpace(s))
		}
	}

	ctx := context.Background()
	db, err := initDatabase(ctx, connectionString)
	if err != nil {
		logger.Fatal("Failed to initialize database connection", err, map[string]interface{}{
			"connection_string_length": len(connectionString),
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

	mux := http.NewServeMux()

	// Public routes (logging + CORS)
	mux.HandleFunc("/health", loggingMiddleware(corsMiddleware(api.HealthHandler)))
	mux.HandleFunc("/auth/register", loggingMiddleware(corsMiddleware(api.registerHandler)))
	mux.HandleFunc("/auth/login", loggingMiddleware(corsMiddleware(api.loginHandler)))

	// Protected routes (logging + CORS + auth)
	mux.HandleFunc("/account", loggingMiddleware(corsMiddleware(api.protectedHandler(api.accountRouter))))
	mux.HandleFunc("/settings", loggingMiddleware(corsMiddleware(api.protectedHandler(api.settingsRouter))))
	mux.HandleFunc("/devices", loggingMiddleware(corsMiddleware(api.protectedHandler(api.devicesRouter))))
	mux.HandleFunc("/categories", loggingMiddleware(corsMiddleware(api.protectedHandler(api.categoriesRouter))))
	mux.HandleFunc("/categories/", loggingMiddleware(corsMiddleware(api.protectedHandler(api.categoriesRouter))))
	mux.HandleFunc("/schedule", loggingMiddleware(corsMiddleware(api.protectedHandler(api.scheduleRouter))))
	mux.HandleFunc("/schedule/", loggingMiddleware(corsMiddleware(api.protectedHandler(api.scheduleRouter))))
	mux.HandleFunc("/template-groups", loggingMiddleware(corsMiddleware(api.protectedHandler(api.templateGroupsRouter))))
	mux.HandleFunc("/template-groups/", loggingMiddleware(corsMiddleware(api.protectedHandler(api.templateGroupsRouter))))
	mux.HandleFunc("/templates", loggingMiddleware(corsMiddleware(api.protectedHandler(api.templatesRouter))))
	mux.HandleFunc("/templates/", loggingMiddleware(corsMiddleware(api.protectedHandler(api.templatesRouter))))
	mux.HandleFunc("/days", loggingMiddleware(corsMiddleware(api.protectedHandler(api.daysRouter))))
	mux.HandleFunc("/days/", loggingMiddleware(corsMiddleware(api.protectedHandler(api.daysRouter))))
	mux.HandleFunc("/init", loggingMiddleware(corsMiddleware(api.protectedHandler(api.initRouter))))
	mux.HandleFunc("/backup", loggingMiddleware(corsMiddleware(api.protectedHandler(api.backupRouter))))

	if getEnvironment("SERVE_FRONTEND", "false") == "true" {
		frontendHandler, frontendError := newFrontendHandler()
		if frontendError != nil {
			logger.Fatal("Failed to initialize embedded frontend", frontendError)
		}
		mux.Handle("/todo-planner/", frontendHandler)
		mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
			if request.URL.Path == "/" {
				http.Redirect(response, request, "/todo-planner/", http.StatusTemporaryRedirect)
				return
			}
			http.NotFound(response, request)
		})
	}

	// Start server
	logger.Info("Server listening for requests", map[string]interface{}{
		"port":         port,
		"cors_origins": corsOrigins,
		"max_db_conns": 5,
	})

	listenAddress := ":" + port
	if appMode == "local" {
		listenAddress = "127.0.0.1:" + port
	}
	server := &http.Server{
		Addr:              listenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if listenError := server.ListenAndServe(); listenError != nil && listenError != http.ErrServerClosed {
			logger.Fatal("Server failed to start", listenError, map[string]interface{}{
				"port": port,
			})
		}
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	if appMode == "local" {
		go monitorParentProcess()
	}
	<-shutdownSignal

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if shutdownError := server.Shutdown(shutdownContext); shutdownError != nil {
		logger.Error("Server shutdown failed", shutdownError)
	}
	if embeddedDatabase != nil {
		if shutdownError := embeddedDatabase.Stop(); shutdownError != nil {
			logger.Error("Embedded database shutdown failed", shutdownError)
		}
	}
}

func monitorParentProcess() {
	parentProcessID, parseError := strconv.Atoi(os.Getenv("TODO_PLANNER_PARENT_PID"))
	if parseError != nil || parentProcessID <= 1 {
		return
	}

	for {
		time.Sleep(500 * time.Millisecond)
		if error := syscall.Kill(parentProcessID, 0); error != nil {
			_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
			return
		}
	}
}

func getEnvironment(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

func jwtSecretForMode(appMode string, dataDirectory string) (string, error) {
	if configuredSecret := os.Getenv("JWT_SECRET"); configuredSecret != "" {
		return configuredSecret, nil
	}
	if appMode != "local" {
		return "", fmt.Errorf("JWT_SECRET environment variable not set")
	}

	resolvedDirectory, err := resolveDataDirectory(dataDirectory)
	if err != nil {
		return "", err
	}
	secretPath := filepath.Join(resolvedDirectory, "jwt-secret")
	if secret, readError := os.ReadFile(secretPath); readError == nil && len(secret) > 0 {
		return string(secret), nil
	}

	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", fmt.Errorf("generate local JWT secret: %w", err)
	}
	if err := os.WriteFile(secretPath, secretBytes, 0o600); err != nil {
		return "", fmt.Errorf("persist local JWT secret: %w", err)
	}
	return string(secretBytes), nil
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
