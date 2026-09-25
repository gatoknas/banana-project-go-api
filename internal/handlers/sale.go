package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/models"
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
// @Description  Registers a new sale, deducts product or recipe ingredient inventory, and records details. Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         sales
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        sale     body      service.SaleRequest  true  "Sale Creation Payload"
// @Success      201      {object}  handlers.SaleResponse
// @Failure      400      {string}  string "Bad request: invalid JSON payload or sale must have at least one item"
// @Failure      500      {string}  string "Internal server error"
// @Router       /api/v1/sales [post]
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
	if err := json.NewEncoder(w).Encode(SaleResponse{
		Status:  "success",
		Message: "Sale processed successfully",
		SaleID:  saleID,
	}); err != nil {
		h.logger.Error("failed to encode sale response", zap.Error(err))
	}
}

// List handles GET /api/v1/sales
// @Summary      List sales transactions
// @Description  Retrieves a paginated list of sales with optional filtering by date range, payment method, and cashier. Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Param        page          query     int     false  "Page number (default 1)"
// @Param        pageSize      query     int     false  "Items per page (default 10, max 100)"
// @Param        fromDate      query     string  false  "Start date filter (YYYY-MM-DD or RFC3339)"
// @Param        toDate        query     string  false  "End date filter (YYYY-MM-DD or RFC3339)"
// @Param        paymentMethod query     string  false  "Filter by payment method (e.g. Cash, Card)"
// @Param        userId        query     int     false  "Filter by cashier user ID"
// @Success      200           {object}  models.PaginatedSalesResponse
// @Failure      400           {string}  string "Bad request: invalid filter parameters"
// @Failure      500           {string}  string "Internal server error"
// @Router       /api/v1/sales [get]
func (h *SaleHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	var filter models.SaleFilter

	if pageStr := query.Get("page"); pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			http.Error(w, "Bad request: invalid page parameter", http.StatusBadRequest)
			return
		}
		filter.Page = p
	}

	if pageSizeStr := query.Get("pageSize"); pageSizeStr != "" {
		ps, err := strconv.Atoi(pageSizeStr)
		if err != nil || ps < 1 {
			http.Error(w, "Bad request: invalid pageSize parameter", http.StatusBadRequest)
			return
		}
		filter.PageSize = ps
	}

	if fromStr := query.Get("fromDate"); fromStr != "" {
		from, err := parseSaleDateQuery(fromStr, false)
		if err != nil {
			http.Error(w, "Bad request: invalid fromDate format (use YYYY-MM-DD or RFC3339)", http.StatusBadRequest)
			return
		}
		filter.FromDate = &from
	}

	if toStr := query.Get("toDate"); toStr != "" {
		to, err := parseSaleDateQuery(toStr, true)
		if err != nil {
			http.Error(w, "Bad request: invalid toDate format (use YYYY-MM-DD or RFC3339)", http.StatusBadRequest)
			return
		}
		filter.ToDate = &to
	}

	if pm := query.Get("paymentMethod"); pm != "" {
		filter.PaymentMethod = pm
	}

	if uidStr := query.Get("userId"); uidStr != "" {
		uid, err := strconv.ParseInt(uidStr, 10, 64)
		if err != nil || uid <= 0 {
			http.Error(w, "Bad request: invalid userId parameter", http.StatusBadRequest)
			return
		}
		filter.UserID = &uid
	}

	resp, err := h.service.ListSales(ctx, filter)
	if err != nil {
		h.logger.Error("failed to list sales", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to list sales: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode sales list response", zap.Error(err))
	}
}

// Get handles GET /api/v1/sales/{id}
// @Summary      Get sale details by ID
// @Description  Retrieves full sale ticket details including itemized products and prices. Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         sales
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Sale ID"
// @Success      200  {object}  models.Sale
// @Failure      400  {string}  string "Bad request: invalid sale ID"
// @Failure      404  {string}  string "Sale not found"
// @Failure      500  {string}  string "Internal server error"
// @Router       /api/v1/sales/{id} [get]
func (h *SaleHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Bad request: invalid sale ID", http.StatusBadRequest)
		return
	}

	sale, err := h.service.GetSaleByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			http.Error(w, "Sale not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get sale by ID", zap.Int64("id", id), zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to get sale: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sale); err != nil {
		h.logger.Error("failed to encode sale detail response", zap.Error(err))
	}
}

func parseSaleDateQuery(s string, isEnd bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	if isEnd {
		t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second + 999999999*time.Nanosecond)
	}
	return t, nil
}

