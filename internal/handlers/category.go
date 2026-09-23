package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
	"org.banana.project/api/internal/service"
)

// CategoryHandler handles HTTP requests for product categories.
type CategoryHandler struct {
	service *service.CategoryService
	logger  *zap.Logger
}

// NewCategoryHandler creates a new CategoryHandler instance.
func NewCategoryHandler(s *service.CategoryService, logger *zap.Logger) *CategoryHandler {
	return &CategoryHandler{
		service: s,
		logger:  logger,
	}
}

// List handles GET /api/v1/categories
// @Summary      List product categories
// @Description  Retrieves all product categories ordered by ID. Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         categories
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   models.Category
// @Failure      500  {string}  string "Internal server error"
// @Router       /api/v1/categories [get]
func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	categories, err := h.service.ListCategories(ctx)
	if err != nil {
		h.logger.Error("failed to list categories", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to query categories: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(categories); err != nil {
		h.logger.Error("failed to encode categories response", zap.Error(err))
	}
}
