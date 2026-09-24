package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/service"
)

type PurchaseHandler struct {
	service *service.PurchaseService
	logger  *zap.Logger
}

func NewPurchaseHandler(s *service.PurchaseService, logger *zap.Logger) *PurchaseHandler {
	return &PurchaseHandler{
		service: s,
		logger:  logger,
	}
}

// Create handles POST /api/v1/purchases
// @Summary      Create a new purchase order / restock
// @Description  Registers a restock invoice, inserts purchase details, atomically increments physical inventory stock, and recalculates product weighted average cost (PMP). Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         purchases
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        purchase  body      service.PurchaseRequest  true  "Purchase Payload"
// @Success      201       {object}  handlers.MessageResponse
// @Failure      400       {string}  string "Bad request: invalid payload or validation error"
// @Failure      500       {string}  string "Internal server error"
// @Router       /api/v1/purchases [post]
func (h *PurchaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req service.PurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode purchase request", zap.Error(err))
		http.Error(w, "Bad request: invalid JSON payload", http.StatusBadRequest)
		return
	}

	id, err := h.service.CreatePurchase(ctx, req)
	if err != nil {
		if errors.Is(err, service.ErrSupplierIDRequired) ||
			errors.Is(err, service.ErrItemsRequired) ||
			errors.Is(err, service.ErrInvalidProductID) ||
			errors.Is(err, service.ErrInvalidQuantity) ||
			errors.Is(err, service.ErrInvalidUnitCost) {
			h.logger.Warn("purchase validation failed", zap.Error(err))
			http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
			return
		}
		h.logger.Error("failed to create purchase", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to create purchase: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(MessageResponse{
		Status:  "success",
		Message: "Purchase registered successfully",
		ID:      id,
	}); err != nil {
		h.logger.Error("failed to encode purchase response", zap.Error(err))
	}
}

// List handles GET /api/v1/purchases
// @Summary      List purchase orders
// @Description  Retrieves purchase orders ordered by date descending with optional supplier and date range filtering. Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         purchases
// @Produce      json
// @Security     BearerAuth
// @Param        supplierId  query     int     false  "Filter by supplier ID"
// @Param        fromDate    query     string  false  "Start date filter (YYYY-MM-DD or RFC3339)"
// @Param        toDate      query     string  false  "End date filter (YYYY-MM-DD or RFC3339)"
// @Success      200         {array}   models.Purchase
// @Failure      400         {string}  string "Bad request: invalid filter parameters"
// @Failure      500         {string}  string "Internal server error"
// @Router       /api/v1/purchases [get]
func (h *PurchaseHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	var supplierID *int64
	if sIDStr := query.Get("supplierId"); sIDStr != "" {
		sID, err := strconv.ParseInt(sIDStr, 10, 64)
		if err != nil || sID <= 0 {
			http.Error(w, "Bad request: invalid supplierId", http.StatusBadRequest)
			return
		}
		supplierID = &sID
	}

	var fromDate *time.Time
	if fromStr := query.Get("fromDate"); fromStr != "" {
		parsed, err := parseDateQuery(fromStr)
		if err != nil {
			http.Error(w, "Bad request: invalid fromDate format", http.StatusBadRequest)
			return
		}
		fromDate = &parsed
	}

	var toDate *time.Time
	if toStr := query.Get("toDate"); toStr != "" {
		parsed, err := parseDateQuery(toStr)
		if err != nil {
			http.Error(w, "Bad request: invalid toDate format", http.StatusBadRequest)
			return
		}
		toDate = &parsed
	}

	purchases, err := h.service.ListPurchases(ctx, supplierID, fromDate, toDate)
	if err != nil {
		h.logger.Error("failed to query purchases", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to query purchases: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(purchases); err != nil {
		h.logger.Error("failed to encode purchases list response", zap.Error(err))
	}
}

// Get handles GET /api/v1/purchases/{id}
// @Summary      Get purchase order details
// @Description  Retrieves a single purchase header and its associated item line details. Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         purchases
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Purchase ID"
// @Success      200  {object}  models.Purchase
// @Failure      400  {string}  string "Invalid purchase ID"
// @Failure      404  {string}  string "Purchase not found"
// @Failure      500  {string}  string "Internal server error"
// @Router       /api/v1/purchases/{id} [get]
func (h *PurchaseHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Bad request: invalid purchase ID", http.StatusBadRequest)
		return
	}

	purchase, err := h.service.GetPurchase(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrPurchaseNotFound) {
			http.Error(w, "Not found: purchase not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to retrieve purchase", zap.Int64("id", id), zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to retrieve purchase: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(purchase); err != nil {
		h.logger.Error("failed to encode purchase response", zap.Error(err))
	}
}

func parseDateQuery(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}
