package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"go.uber.org/zap"
	"org.banana.project/api/internal/service"
)

type UserHandler struct {
	service *service.UserService
	logger  *zap.Logger
}

func NewUserHandler(s *service.UserService, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		service: s,
		logger:  logger,
	}
}

// Create handles POST /api/v1/users
// @Summary      Create a new user
// @Description  Creates a new system user (admin or salesperson) with a secure hashed password.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user  body      service.UserRequest  true  "User Creation Payload"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {string}  string "Bad request"
// @Failure      500   {string}  string "Internal server error"
// @Router       /users [post]
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req service.UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode user creation request", zap.Error(err))
		http.Error(w, "Bad request: invalid JSON payload", http.StatusBadRequest)
		return
	}

	id, err := h.service.CreateUser(ctx, req)
	if err != nil {
		h.logger.Warn("user creation failed", zap.Error(err))
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "User created successfully",
		"id":      id,
	})
}

// List handles GET /api/v1/users
// @Summary      List users
// @Description  Retrieves all registered users ordered alphabetically by name.
// @Tags         users
// @Produce      json
// @Success      200   {array}   models.User
// @Failure      500   {string}  string "Internal server error"
// @Router       /users [get]
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, err := h.service.ListUsers(ctx)
	if err != nil {
		h.logger.Error("failed to list users", zap.Error(err))
		http.Error(w, fmt.Sprintf("Internal error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(users)
}

// Get handles GET /api/v1/users/{id}
// @Summary      Get user by ID
// @Description  Retrieves detailed user profile information.
// @Tags         users
// @Produce      json
// @Param        id    path      int  true  "User ID"
// @Success      200   {object}  models.User
// @Failure      400   {string}  string "Bad request"
// @Failure      404   {string}  string "User not found"
// @Failure      500   {string}  string "Internal server error"
// @Router       /users/{id} [get]
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Warn("invalid user ID in get request", zap.String("id_str", idStr), zap.Error(err))
		http.Error(w, "Bad request: invalid user ID", http.StatusBadRequest)
		return
	}

	u, err := h.service.GetUserByID(ctx, id)
	if err == sql.ErrNoRows {
		h.logger.Info("user not found", zap.Int64("id", id))
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		h.logger.Error("failed to query user", zap.Int64("id", id), zap.Error(err))
		http.Error(w, fmt.Sprintf("Internal error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
}

// Update handles PUT /api/v1/users/{id}
// @Summary      Update user
// @Description  Updates a user's role, password, fullName, or status.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id    path      int                  true  "User ID"
// @Param        user  body      service.UserRequest  true  "User Update Payload"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {string}  string "Bad request"
// @Failure      404   {string}  string "User not found"
// @Failure      500   {string}  string "Internal server error"
// @Router       /users/{id} [put]
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Warn("invalid user ID in update request", zap.String("id_str", idStr), zap.Error(err))
		http.Error(w, "Bad request: invalid user ID", http.StatusBadRequest)
		return
	}

	var req service.UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode user update request", zap.Error(err))
		http.Error(w, "Bad request: invalid JSON payload", http.StatusBadRequest)
		return
	}

	err = h.service.UpdateUser(ctx, id, req)
	if err == sql.ErrNoRows {
		h.logger.Info("user not found for update", zap.Int64("id", id))
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		h.logger.Warn("user update failed", zap.Int64("id", id), zap.Error(err))
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "User updated successfully",
	})
}

// Delete handles DELETE /api/v1/users/{id}
// @Summary      Delete user
// @Description  Deletes a user from the system.
// @Tags         users
// @Produce      json
// @Param        id    path      int  true  "User ID"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {string}  string "Bad request"
// @Failure      404   {string}  string "User not found"
// @Failure      500   {string}  string "Internal server error"
// @Router       /users/{id} [delete]
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Warn("invalid user ID in delete request", zap.String("id_str", idStr), zap.Error(err))
		http.Error(w, "Bad request: invalid user ID", http.StatusBadRequest)
		return
	}

	err = h.service.DeleteUser(ctx, id)
	if err == sql.ErrNoRows {
		h.logger.Info("user not found for deletion", zap.Int64("id", id))
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		h.logger.Error("failed to delete user", zap.Int64("id", id), zap.Error(err))
		http.Error(w, fmt.Sprintf("Internal error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "User deleted successfully",
	})
}
