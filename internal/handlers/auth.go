package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"org.banana.project/api/internal/auth"
	"org.banana.project/api/internal/service"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type AuthHandler struct {
	userService *service.UserService
	logger      *zap.Logger
}

func NewAuthHandler(us *service.UserService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		userService: us,
		logger:      logger,
	}
}

// Login handles POST /login
// @Summary      User Login
// @Description  Authenticates a user and returns a JWT token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials  body      LoginRequest  true  "User Credentials"
// @Success      200          {object}  LoginResponse
// @Failure      400          {string}  string "Bad request: invalid JSON payload or missing username"
// @Failure      401          {string}  string "Unauthorized: invalid credentials"
// @Failure      403          {string}  string "Forbidden: account is inactive"
// @Failure      500          {string}  string "Internal server error"
// @Router       /login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode login request", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	username := strings.ToLower(strings.TrimSpace(req.Username))
	if username == "" {
		h.logger.Warn("login attempt with empty username")
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	user, err := h.userService.GetUserByUsername(ctx, username)
	if err == sql.ErrNoRows {
		h.logger.Warn("login failed: user not found", zap.String("username", username))
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		h.logger.Error("failed to query user on login", zap.String("username", username), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !user.IsActive {
		h.logger.Warn("login failed: user is inactive", zap.String("username", username))
		http.Error(w, "Account is disabled", http.StatusForbidden)
		return
	}

	valid, err := auth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil {
		h.logger.Error("failed to verify password hash", zap.String("username", username), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !valid {
		h.logger.Warn("login failed: invalid password", zap.String("username", username))
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err), zap.String("username", username))
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(LoginResponse{Token: token}); err != nil {
		h.logger.Error("failed to encode login response", zap.Error(err))
	}
}
