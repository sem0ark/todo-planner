package main

import (
	"net/http"

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
	dayService        *DayService
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
	api.dayService = NewDayService(api.dayRecordRepo, api.categoryRepo)
	return api
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
