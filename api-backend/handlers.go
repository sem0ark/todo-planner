package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	db                *pgxpool.Pool
	jwtSecret         string
	logger            *Logger
	userRepo          *UserRepository
	settingsRepo      *UserSettingsRepository
	deviceRepo        *DeviceRepository
	categoryRepo      *CategoryRepository
	templateGroupRepo *TemplateGroupRepository
	dayTemplateRepo   *DayTemplateRepository
	scheduleRepo      *ScheduleRepository
	dayRecordRepo     *DayRecordRepository
}

func NewAPI(db *pgxpool.Pool, jwtSecret string, logger *Logger) *API {
	api := &API{
		db:                db,
		jwtSecret:         jwtSecret,
		logger:            logger,
		userRepo:          NewUserRepository(db),
		settingsRepo:      NewUserSettingsRepository(db),
		deviceRepo:        NewDeviceRepository(db),
		categoryRepo:      NewCategoryRepository(db),
		templateGroupRepo: NewTemplateGroupRepository(db),
		dayTemplateRepo:   NewDayTemplateRepository(db),
		scheduleRepo:      NewScheduleRepository(db),
		dayRecordRepo:     NewDayRecordRepository(db),
	}
	return api
}

// protectedHandler authenticates requests and guarantees that protected routes
// receive a request context containing an authenticated user ID.
func (api *API) protectedHandler(handler http.HandlerFunc) http.HandlerFunc {
	return api.authMiddleware(func(responseWriter http.ResponseWriter, request *http.Request) {
		if _, authenticated := getUserID(request.Context()); !authenticated {
			http.Error(responseWriter, "unauthorized", http.StatusUnauthorized)
			return
		}
		handler(responseWriter, request)
	})
}

func userIDFromRequest(request *http.Request) int {
	userID, ok := getUserID(request.Context())
	if !ok {
		panic("user ID not found in request context")
	}
	return userID
}

func NewCORSMiddleware(allowedOrigins []string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			isAllowed := false
			for _, allowed := range allowedOrigins {
				if allowed == "*" || origin == allowed {
					isAllowed = true
					break
				}
			}

			if isAllowed {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
				w.Header().Add("Vary", "Origin")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400") // Cache preflight for 24 hours

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}
}

func writeJSON(writer http.ResponseWriter, value interface{}) {
	encodedValue, err := json.Marshal(value)
	if err != nil {
		if wrappedWriter, ok := writer.(*responseWriter); ok && wrappedWriter.logger != nil {
			wrappedWriter.logger.Error("Failed to encode JSON response", err)
		}
		http.Error(writer, "failed to encode response", http.StatusInternalServerError)
		return
	}
	if wrappedWriter, ok := writer.(*responseWriter); ok && wrappedWriter.logger != nil {
		wrappedWriter.logger.Info("Writing JSON response", map[string]interface{}{
			"response_type":  "json",
			"content_length": len(encodedValue),
			"response_body":  string(encodedValue),
		})
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Content-Length", strconv.Itoa(len(encodedValue)))
	if writtenBytes, err := writer.Write(encodedValue); err != nil {
		if wrappedWriter, ok := writer.(*responseWriter); ok && wrappedWriter.logger != nil {
			wrappedWriter.logger.Error("Failed to write JSON response", err, map[string]interface{}{
				"bytes_requested": len(encodedValue),
				"bytes_written":   writtenBytes,
			})
		}
	}
}
