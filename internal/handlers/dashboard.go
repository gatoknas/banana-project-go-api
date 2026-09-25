package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"org.banana.project/api/internal/service"
)

type DashboardHandler struct {
	service *service.DashboardService
	logger  *zap.Logger
}

func NewDashboardHandler(s *service.DashboardService, logger *zap.Logger) *DashboardHandler {
	return &DashboardHandler{
		service: s,
		logger:  logger,
	}
}

// GetStats handles GET /api/v1/dashboard/stats
// @Summary      Get aggregated dashboard stats
// @Description  Returns executive KPIs, sales timeline, payment method breakdown, top products, category distribution, purchases vs sales, and inventory stock alerts. Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Param        from query string false "Start date filter (YYYY-MM-DD)"
// @Param        to   query string false "End date filter (YYYY-MM-DD)"
// @Success      200  {object} models.DashboardStats
// @Failure      400  {string} string "Bad request: invalid date format"
// @Failure      500  {string} string "Internal server error"
// @Router       /api/v1/dashboard/stats [get]
func (h *DashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	stats, err := h.service.GetDashboardStats(ctx, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "cannot be after") {
			h.logger.Warn("invalid dashboard query parameters", zap.Error(err), zap.String("from", from), zap.String("to", to))
			http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
			return
		}
		h.logger.Error("failed to retrieve dashboard stats", zap.Error(err))
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		h.logger.Error("failed to encode dashboard stats response", zap.Error(err))
	}
}
