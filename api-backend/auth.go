package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func createJWT(userID int, username string, secret string) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payload := map[string]interface{}{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	message := header + "." + payloadB64
	signature := sha256.Sum256([]byte(message + secret))
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature[:])

	return message + "." + signatureB64, nil
}

func verifyJWT(token string, secret string) (int, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, "", errors.New("invalid token format")
	}

	message := parts[0] + "." + parts[1]
	expectedSig := sha256.Sum256([]byte(message + secret))
	expectedSigB64 := base64.RawURLEncoding.EncodeToString(expectedSig[:])

	if parts[2] != expectedSigB64 {
		return 0, "", errors.New("invalid signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, "", err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return 0, "", err
	}

	exp, ok := payload["exp"].(float64)
	if !ok || time.Now().Unix() > int64(exp) {
		return 0, "", errors.New("token expired")
	}

	userID, ok := payload["user_id"].(float64)
	if !ok {
		return 0, "", errors.New("invalid user_id")
	}

	username, ok := payload["username"].(string)
	if !ok {
		return 0, "", errors.New("invalid username")
	}

	return int(userID), username, nil
}

func (api *API) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Log available headers for debugging (excluding sensitive data)
			headers := make(map[string]string)
			for k := range r.Header {
				if k != "Authorization" && k != "Cookie" {
					headers[k] = "(present)"
				}
			}
			api.logger.Warn("Missing authorization header", map[string]interface{}{
				"path":    r.URL.Path,
				"method":  r.Method,
				"headers": headers,
			})
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			api.logger.Warn("Invalid authorization format", map[string]interface{}{
				"path":        r.URL.Path,
				"method":      r.Method,
			})
			http.Error(w, "invalid authorization format", http.StatusUnauthorized)
			return
		}

		userID, username, err := verifyJWT(tokenParts[1], api.jwtSecret)
		if err != nil {
			api.logger.Warn("JWT verification failed", map[string]interface{}{
				"path":        r.URL.Path,
				"method":      r.Method,
				"error":       err.Error(),
			})
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Store user info in context
		ctx := r.Context()
		ctx = withUserID(ctx, userID)
		ctx = withUsername(ctx, username)

		next(w, r.WithContext(ctx))
	}
}

func (api *API) registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, nil)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	user, err := api.userRepo.Create(r.Context(), req.Username, req.Password)
	if err != nil {
		if _, ok := err.(*DuplicateUsernameError); ok {
			http.Error(w, "username already exists", http.StatusConflict)
			return
		}
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to create user", err, map[string]interface{}{
			"username": req.Username,
		})
		return
	}

	token, err := createJWT(user.ID, user.Username, api.jwtSecret)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to create token", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AuthResponse{Token: token, User: *user})
}

func (api *API) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, r, api.logger, http.StatusBadRequest, "invalid request body", err, nil)
		return
	}

	user, err := api.userRepo.FindByUsername(r.Context(), req.Username)
	if err != nil {
		// Log internal error but show generic message for security
		HTTPError(w, r, api.logger, http.StatusUnauthorized, "invalid credentials", err, map[string]interface{}{
			"username": req.Username,
			"reason":   "user_not_found",
		})
		return
	}

	if !api.userRepo.VerifyPassword(user, req.Password) {
		// Log failed login attempt
		api.logger.Warn("Failed login attempt - invalid password", map[string]interface{}{
			"username":    req.Username,
			"user_id":     user.ID,
		})
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := createJWT(user.ID, user.Username, api.jwtSecret)
	if err != nil {
		HTTPError(w, r, api.logger, http.StatusInternalServerError, "failed to create token", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return
	}

	// Log successful login
	api.logger.Info("User logged in successfully", map[string]interface{}{
		"user_id":     user.ID,
		"username":    user.Username,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{Token: token, User: *user})
}

const (
	userIDKey   string = "user_id"
	usernameKey string = "username"
)

func withUserID(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func withUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey, username)
}

func getUserID(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userIDKey).(int)
	return userID, ok
}
