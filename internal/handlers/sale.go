package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"org.banana.project/api/internal/service"
)

type SaleHandler struct {
	service *service.SaleService
	logger  *zap.Logger
}

func NewSaleHandler(s *service.SaleService, logger *zap.Logger) *SaleHandler {
	return &SaleHandler{
		service: s,
		logger:  logger,
	}
}

// Create handles POST /api/v1/sales
// @Summary      Create a new sale
// @Description  Registers a new sale, deducts product or recipe ingredient inventory, and records details.
// @Tags         sales
// @Accept       json
// @Produce      json
// @Param        sale     body      service.SaleRequest  true  "Sale Creation Payload"
// @Success      201      {object}  map[string]interface{}
// @Failure      400      {string}  string "Bad request: invalid JSON payload or sale must have at least one item"
// @Failure      500      {string}  string "Internal server error"
// @Router       /sales [post]
func (h *SaleHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req service.SaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode sale request", zap.Error(err))
		http.Error(w, "Bad request: invalid JSON payload", http.StatusBadRequest)
		return
	}

	saleID, err := h.service.CreateSale(ctx, req)
	if err != nil {
		if err.Error() == "sale must have at least one item" || strings.Contains(err.Error(), "not found") {
			h.logger.Warn("failed to create sale due to validation", zap.Error(err))
			http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
			return
		}
		h.logger.Error("failed to process sale", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to process sale: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Sale processed successfully",
		"saleId":  saleID,
	}); err != nil {
		h.logger.Error("failed to encode sale response", zap.Error(err))
	}
}
