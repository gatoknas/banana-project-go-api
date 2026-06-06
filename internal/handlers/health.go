package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

type HealthHandler struct {
	logger *zap.Logger
	db     *sql.DB
}

func NewHealthHandler(logger *zap.Logger, db *sql.DB) *HealthHandler {
	return &HealthHandler{
		logger: logger,
		db:     db,
	}
}

// Health handles GET /status
// @Summary      System Health Status
// @Description  Checks if the API is running and verifies connection to the database.
// @Tags         system
// @Produce      json
// @Success      200      {object}  map[string]string "Status OK"
// @Failure      503      {object}  map[string]string "Service Unavailable"
// @Router       /status [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dbStatus := "connected"
	if h.db == nil {
		dbStatus = "uninitialized"
		h.logger.Error("database connection is uninitialized")
	} else if err := h.db.PingContext(r.Context()); err != nil {
		dbStatus = fmt.Sprintf("error: %v", err)
		h.logger.Error("database ping failure", zap.Error(err))
	}

	responseCode := http.StatusOK
	if dbStatus != "connected" {
		responseCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(responseCode)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":   "alive",
		"database": dbStatus,
	}); err != nil {
		h.logger.Error("failed to encode health check response", zap.Error(err))
	}
}

