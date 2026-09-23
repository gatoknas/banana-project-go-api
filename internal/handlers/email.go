package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/email"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type EmailReceiptHandler struct {
	service *service.EmailReceiptService
	logger  *zap.Logger
}

func NewEmailReceiptHandler(s *service.EmailReceiptService, logger *zap.Logger) *EmailReceiptHandler {
	return &EmailReceiptHandler{
		service: s,
		logger:  logger,
	}
}

// Sync handles POST /api/v1/email-receipts/sync
// @Summary      Sync bank receipts from the email inbox
// @Description  Reads receipt emails from the configured Gmail inbox for a date range and stores the sale details. Requires the ayurami-admin role.
// @Tags         email-receipts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        range  body      service.EmailReceiptSyncRequest  true  "Date range (from/to as YYYY-MM-DD)"
// @Success      200    {object}  handlers.SyncResponse
// @Failure      400    {string}  string "Bad request: invalid JSON payload or date range"
// @Failure      500    {string}  string "Internal server error"
// @Router       /api/v1/email-receipts/sync [post]
func (h *EmailReceiptHandler) Sync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req service.EmailReceiptSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode email receipt sync request", zap.Error(err))
		http.Error(w, "Bad request: invalid JSON payload", http.StatusBadRequest)
		return
	}

	result, err := h.service.Sync(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "must be") {
			h.logger.Warn("email receipt sync validation failed", zap.Error(err))
			http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
			return
		}
		h.logger.Error("email receipt sync failed", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to sync email receipts: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(SyncResponse{
		Status:  "success",
		Message: "Email receipts synced successfully",
		Result:  result,
	})
}

// List handles GET /api/v1/email-receipts
// @Summary      List stored email receipts
// @Description  Retrieves stored bank receipts, optionally filtered by a received_at date range. Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         email-receipts
// @Produce      json
// @Security     BearerAuth
// @Param        from  query     string  false  "Start date (YYYY-MM-DD, inclusive)"
// @Param        to    query     string  false  "End date (YYYY-MM-DD, inclusive)"
// @Success      200   {array}   models.EmailReceipt
// @Failure      400   {string}  string "Bad request: invalid date"
// @Failure      500   {string}  string "Internal server error"
// @Router       /api/v1/email-receipts [get]
func (h *EmailReceiptHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var from, to *time.Time
	if v := strings.TrimSpace(r.URL.Query().Get("from")); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, email.Colombia)
		if err != nil {
			http.Error(w, "Bad request: invalid 'from' date, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		from = &t
	}
	if v := strings.TrimSpace(r.URL.Query().Get("to")); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, email.Colombia)
		if err != nil {
			http.Error(w, "Bad request: invalid 'to' date, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		t = t.AddDate(0, 0, 1) // make the "to" day inclusive
		to = &t
	}

	receipts, err := h.service.List(ctx, from, to)
	if err != nil {
		h.logger.Error("failed to list email receipts", zap.Error(err))
		http.Error(w, fmt.Sprintf("Internal error: %v", err), http.StatusInternalServerError)
		return
	}

	if receipts == nil {
		receipts = []models.EmailReceipt{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(receipts)
}
