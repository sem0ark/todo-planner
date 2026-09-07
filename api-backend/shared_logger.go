package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

// Logger provides structured logging for the application
type Logger struct {
	serviceName string
}

// NewLogger creates a new logger instance
func NewLogger(serviceName string) *Logger {
	return &Logger{
		serviceName: serviceName,
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	Service     string                 `json:"service"`
	Message     string                 `json:"message"`
	Error       string                 `json:"error,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	RequestInfo *RequestInfo           `json:"request,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// RequestInfo contains information about an HTTP request
type RequestInfo struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	UserID     int    `json:"user_id,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Duration   string `json:"duration,omitempty"`
}

// log writes a structured log entry to stdout
func (l *Logger) log(level LogLevel, message string, err error, requestInfo *RequestInfo, extra map[string]interface{}) {
	entry := LogEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Level:       level,
		Service:     l.serviceName,
		Message:     message,
		RequestInfo: requestInfo,
		Extra:       extra,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// Add stack trace for errors and fatals
	if level == LogLevelError || level == LogLevelFatal {
		entry.StackTrace = string(debug.Stack())
	}

	// Marshal to JSON and write to stdout
	jsonBytes, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		// Fallback to plain text if JSON marshaling fails
		fmt.Fprintf(os.Stdout, "[%s] %s: %s (marshal error: %v)\n",
			entry.Timestamp, level, message, marshalErr)
		return
	}

	fmt.Fprintln(os.Stdout, string(jsonBytes))
}

// Info logs an informational message
func (l *Logger) Info(message string, extra ...map[string]interface{}) {
	var extraMap map[string]interface{}
	if len(extra) > 0 {
		extraMap = extra[0]
	}
	l.log(LogLevelInfo, message, nil, nil, extraMap)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, extra ...map[string]interface{}) {
	var extraMap map[string]interface{}
	if len(extra) > 0 {
		extraMap = extra[0]
	}
	l.log(LogLevelWarn, message, nil, nil, extraMap)
}

// Error logs an error message with optional error object
func (l *Logger) Error(message string, err error, extra ...map[string]interface{}) {
	var extraMap map[string]interface{}
	if len(extra) > 0 {
		extraMap = extra[0]
	}
	l.log(LogLevelError, message, err, nil, extraMap)
}

// Fatal logs a fatal error and exits the program
func (l *Logger) Fatal(message string, err error, extra ...map[string]interface{}) {
	var extraMap map[string]interface{}
	if len(extra) > 0 {
		extraMap = extra[0]
	}
	l.log(LogLevelFatal, message, err, nil, extraMap)
	os.Exit(1)
}

// LogRequest logs an HTTP request with response information
func (l *Logger) LogRequest(r *http.Request, statusCode int, duration time.Duration, userID int) {
	requestInfo := &RequestInfo{
		Method:     r.Method,
		Path:       r.URL.Path,
		UserID:     userID,
		StatusCode: statusCode,
		Duration:   duration.String(),
	}

	message := fmt.Sprintf("%s %s - %d (%s)", r.Method, r.URL.Path, statusCode, duration)
	l.log(LogLevelInfo, message, nil, requestInfo, nil)
}

// LogError logs an HTTP request that resulted in an error
func (l *Logger) LogError(r *http.Request, statusCode int, err error, userID int) {
	requestInfo := &RequestInfo{
		Method:     r.Method,
		Path:       r.URL.Path,
		UserID:     userID,
		StatusCode: statusCode,
	}

	message := fmt.Sprintf("%s %s - %d ERROR", r.Method, r.URL.Path, statusCode)
	l.log(LogLevelError, message, err, requestInfo, nil)
}

// HTTPError logs detailed error information and sends a generic error to the client
// This prevents leaking internal error details while ensuring full logging for debugging
func HTTPError(w http.ResponseWriter, r *http.Request, logger *Logger, statusCode int, userMessage string, internalErr error, extra map[string]interface{}) {
	// Get user ID from context if available
	userID := 0
	if uid, ok := getUserID(r.Context()); ok {
		userID = uid
	}

	// Build request info for logging
	requestInfo := &RequestInfo{
		Method:     r.Method,
		Path:       r.URL.Path,
		UserID:     userID,
		StatusCode: statusCode,
	}

	// Log the detailed error internally
	logMessage := fmt.Sprintf("%s %s - %d: %s", r.Method, r.URL.Path, statusCode, userMessage)

	// Merge extra fields with request info for comprehensive logging
	if extra == nil {
		extra = make(map[string]interface{})
	}
	extra["user_message"] = userMessage

	logger.log(LogLevelError, logMessage, internalErr, requestInfo, extra)

	// Send generic error message to user
	http.Error(w, userMessage, statusCode)
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// LoggingMiddleware logs all incoming HTTP requests
func LoggingMiddleware(logger *Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the response writer to capture status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Recover from panics
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error(
						fmt.Sprintf("PANIC: %s %s", r.Method, r.URL.Path),
						fmt.Errorf("panic recovered: %v", rec),
						map[string]interface{}{
							"method": r.Method,
							"path":   r.URL.Path,
						},
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()

			// Call the next handler
			next(rw, r)

			// Log the request
			duration := time.Since(start)
			userID := 0
			if uid, ok := getUserID(r.Context()); ok {
				userID = uid
			}

			logger.LogRequest(r, rw.statusCode, duration, userID)
		}
	}
}
