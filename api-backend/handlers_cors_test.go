package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddlewareAllowsLocalhostOriginsOnAnyPort(t *testing.T) {
	testCases := []struct {
		name          string
		origin        string
		allowedOrigin string
		expectAllowed bool
	}{
		{
			name:          "localhost arbitrary port",
			origin:        "http://localhost:5173",
			expectAllowed: true,
		},
		{
			name:          "loopback HTTPS arbitrary port",
			origin:        "https://127.0.0.1:8443",
			expectAllowed: true,
		},
		{
			name:          "localhost without port",
			origin:        "http://localhost",
			expectAllowed: true,
		},
		{
			name:          "non-loopback origin",
			origin:        "https://example.com",
			expectAllowed: false,
		},
		{
			name:          "explicit origin",
			origin:        "https://app.example.com",
			allowedOrigin: "https://app.example.com",
			expectAllowed: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := NewCORSMiddleware([]string{testCase.allowedOrigin})(
				func(responseWriter http.ResponseWriter, request *http.Request) {
					responseWriter.WriteHeader(http.StatusNoContent)
				},
			)
			request := httptest.NewRequest(http.MethodGet, "http://api.example.com/health", nil)
			request.Header.Set("Origin", testCase.origin)
			response := httptest.NewRecorder()

			handler(response, request)

			if response.Code != http.StatusNoContent {
				t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
			}
			if testCase.expectAllowed {
				if got := response.Header().Get("Access-Control-Allow-Origin"); got != testCase.origin {
					t.Fatalf("expected reflected origin %q, got %q", testCase.origin, got)
				}
				return
			}
			if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Fatalf("expected no CORS origin header, got %q", got)
			}
		})
	}
}
