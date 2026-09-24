package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"go.uber.org/zap"
	"org.banana.project/api/internal/service"
)

type SupplierHandler struct {
	service *service.SupplierService
	logger  *zap.Logger
}

func NewSupplierHandler(s *service.SupplierService, logger *zap.Logger) *SupplierHandler {
	return &SupplierHandler{
		service: s,
		logger:  logger,
	}
}

// Create handles POST /api/v1/suppliers
// @Summary      Create a new supplier
// @Description  Registers a new goods or ingredients provider. Requires the ayurami-admin role.
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        supplier  body      service.SupplierRequest  true  "Supplier Creation Payload"
// @Success      201       {object}  handlers.MessageResponse
// @Failure      400       {string}  string "Bad request: invalid payload or missing fields"
// @Failure      409       {string}  string "Conflict: tax ID already registered"
// @Failure      500       {string}  string "Internal server error"
// @Router       /api/v1/suppliers [post]
func (h *SupplierHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req service.SupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode supplier request", zap.Error(err))
		http.Error(w, "Bad request: invalid JSON payload", http.StatusBadRequest)
		return
	}

	id, err := h.service.CreateSupplier(ctx, req)
	if err != nil {
		if errors.Is(err, service.ErrCompanyNameRequired) || errors.Is(err, service.ErrTaxIDRequired) {
			h.logger.Warn("supplier creation validation failed", zap.Error(err))
			http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrTaxIDAlreadyExists) {
			h.logger.Warn("supplier creation conflict: tax_id exists", zap.String("tax_id", req.TaxID))
			http.Error(w, "Conflict: tax_id already registered", http.StatusConflict)
			return
		}
		h.logger.Error("failed to create supplier", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to create supplier: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(MessageResponse{
		Status:  "success",
		Message: "Supplier created successfully",
		ID:      id,
	}); err != nil {
		h.logger.Error("failed to encode supplier creation response", zap.Error(err))
	}
}

// List handles GET /api/v1/suppliers
// @Summary      List suppliers
// @Description  Retrieves all registered suppliers with optional search filtering by company name, tax ID, or contact. Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         suppliers
// @Produce      json
// @Security     BearerAuth
// @Param        search   query     string  false  "Search filter query"
// @Success      200      {array}   models.Supplier
// @Failure      500      {string}  string "Internal server error"
// @Router       /api/v1/suppliers [get]
func (h *SupplierHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	search := r.URL.Query().Get("search")

	suppliers, err := h.service.ListSuppliers(ctx, search)
	if err != nil {
		h.logger.Error("failed to list suppliers", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to query suppliers: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(suppliers); err != nil {
		h.logger.Error("failed to encode suppliers list response", zap.Error(err))
	}
}

// Get handles GET /api/v1/suppliers/{id}
// @Summary      Get a supplier by ID
// @Description  Retrieves detailed information of a single supplier. Requires the ayurami-admin or ayurami-salesperson role.
// @Tags         suppliers
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int  true  "Supplier ID"
// @Success      200      {object}  models.Supplier
// @Failure      400      {string}  string "Invalid supplier ID"
// @Failure      404      {string}  string "Supplier not found"
// @Failure      500      {string}  string "Internal server error"
// @Router       /api/v1/suppliers/{id} [get]
func (h *SupplierHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Bad request: invalid supplier ID", http.StatusBadRequest)
		return
	}

	supplier, err := h.service.GetSupplier(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrSupplierNotFound) {
			http.Error(w, "Not found: supplier not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get supplier", zap.Int64("id", id), zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to retrieve supplier: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(supplier); err != nil {
		h.logger.Error("failed to encode supplier response", zap.Error(err))
	}
}

// Update handles PUT /api/v1/suppliers/{id}
// @Summary      Update a supplier
// @Description  Modifies an existing supplier's details. Requires the ayurami-admin role.
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id        path      int                      true  "Supplier ID"
// @Param        supplier  body      service.SupplierRequest  true  "Supplier Update Payload"
// @Success      200       {object}  handlers.MessageResponse
// @Failure      400       {string}  string "Bad request: invalid payload or missing fields"
// @Failure      404       {string}  string "Supplier not found"
// @Failure      409       {string}  string "Conflict: tax ID already registered"
// @Failure      500       {string}  string "Internal server error"
// @Router       /api/v1/suppliers/{id} [put]
func (h *SupplierHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Bad request: invalid supplier ID", http.StatusBadRequest)
		return
	}

	var req service.SupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode supplier update request", zap.Error(err))
		http.Error(w, "Bad request: invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateSupplier(ctx, id, req); err != nil {
		if errors.Is(err, service.ErrCompanyNameRequired) || errors.Is(err, service.ErrTaxIDRequired) {
			http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrSupplierNotFound) {
			http.Error(w, "Not found: supplier not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrTaxIDAlreadyExists) {
			http.Error(w, "Conflict: tax_id already registered", http.StatusConflict)
			return
		}
		h.logger.Error("failed to update supplier", zap.Int64("id", id), zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to update supplier: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(MessageResponse{
		Status:  "success",
		Message: "Supplier updated successfully",
		ID:      id,
	}); err != nil {
		h.logger.Error("failed to encode supplier update response", zap.Error(err))
	}
}

// Delete handles DELETE /api/v1/suppliers/{id}
// @Summary      Delete a supplier
// @Description  Deletes a supplier by ID if they have no associated purchases. Requires the ayurami-admin role.
// @Tags         suppliers
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int  true  "Supplier ID"
// @Success      200      {object}  handlers.MessageResponse
// @Failure      400      {string}  string "Invalid supplier ID"
// @Failure      404      {string}  string "Supplier not found"
// @Failure      409      {string}  string "Conflict: cannot delete supplier with associated purchases"
// @Failure      500      {string}  string "Internal server error"
// @Router       /api/v1/suppliers/{id} [delete]
func (h *SupplierHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Bad request: invalid supplier ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteSupplier(ctx, id); err != nil {
		if errors.Is(err, service.ErrSupplierNotFound) {
			http.Error(w, "Not found: supplier not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrSupplierHasPurchases) {
			http.Error(w, "Conflict: cannot delete supplier with associated purchases", http.StatusConflict)
			return
		}
		h.logger.Error("failed to delete supplier", zap.Int64("id", id), zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to delete supplier: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(MessageResponse{
		Status:  "success",
		Message: "Supplier deleted successfully",
		ID:      id,
	}); err != nil {
		h.logger.Error("failed to encode supplier deletion response", zap.Error(err))
	}
}
