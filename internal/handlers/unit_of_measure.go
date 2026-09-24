package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
	"org.banana.project/api/internal/service"
)

// UnitOfMeasureHandler handles HTTP requests for units of measure.
type UnitOfMeasureHandler struct {
	service *service.UnitOfMeasureService
	logger  *zap.Logger
}

// NewUnitOfMeasureHandler creates a new UnitOfMeasureHandler instance.
func NewUnitOfMeasureHandler(s *service.UnitOfMeasureService, logger *zap.Logger) *UnitOfMeasureHandler {
	return &UnitOfMeasureHandler{
		service: s,
		logger:  logger,
	}
}

// List handles GET /api/v1/units-of-measure
// @Summary      List units of measure
// @Description  Retrieves all available units of measure ordered by name. Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         units-of-measure
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   models.UnitOfMeasure
// @Failure      500  {string}  string "Internal server error"
// @Router       /api/v1/units-of-measure [get]
func (h *UnitOfMeasureHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	units, err := h.service.ListUnits(ctx)
	if err != nil {
		h.logger.Error("failed to list units of measure", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to query units of measure: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(units); err != nil {
		h.logger.Error("failed to encode units of measure response", zap.Error(err))
	}
}
