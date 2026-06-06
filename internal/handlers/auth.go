package handlers

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
	"org.banana.project/api/internal/auth"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type AuthHandler struct {
	logger *zap.Logger
}

func NewAuthHandler(logger *zap.Logger) *AuthHandler {
	return &AuthHandler{logger: logger}
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
// @Failure      500          {string}  string "Internal server error"
// @Router       /login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode login request", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// For demonstration purposes, we accept any credentials
	// In production, validate against a database!
	if req.Username == "" {
		h.logger.Warn("login attempt with empty username")
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	token, err := auth.GenerateToken(req.Username)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err), zap.String("username", req.Username))
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(LoginResponse{Token: token}); err != nil {
		h.logger.Error("failed to encode login response", zap.Error(err))
	}
}
