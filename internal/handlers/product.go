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

type ProductHandler struct {
	service *service.ProductService
	logger  *zap.Logger
}

func NewProductHandler(s *service.ProductService, logger *zap.Logger) *ProductHandler {
	return &ProductHandler{
		service: s,
		logger:  logger,
	}
}

// Create handles POST /api/v1/products
// @Summary      Create a new product
// @Description  Creates a new product in the catalog and initializes its inventory to zero.
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        product  body      service.ProductRequest  true  "Product Creation Payload"
// @Success      201      {object}  map[string]interface{}
// @Failure      400      {string}  string "Bad request: invalid JSON payload or missing name"
// @Failure      500      {string}  string "Internal server error"
// @Router       /products [post]
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req service.ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode product request", zap.Error(err))
		http.Error(w, "Bad request: invalid JSON payload", http.StatusBadRequest)
		return
	}

	id, err := h.service.CreateProduct(ctx, req)
	if err != nil {
		if err.Error() == "name is required" {
			h.logger.Warn("product creation failed: name is required")
			http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
			return
		}
		h.logger.Error("failed to create product", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to create product: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Product created successfully",
		"id":      id,
	}); err != nil {
		h.logger.Error("failed to encode product creation response", zap.Error(err))
	}
}

// List handles GET /api/v1/products
// @Summary      List products
// @Description  Retrieves all products ordered alphabetically by name.
// @Tags         products
// @Produce      json
// @Success      200      {array}   models.Product
// @Failure      500      {string}  string "Internal server error"
// @Router       /products [get]
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	products, err := h.service.ListProducts(ctx)
	if err != nil {
		h.logger.Error("failed to list products", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to query products: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(products); err != nil {
		h.logger.Error("failed to encode product list response", zap.Error(err))
	}
}

// Get handles GET /api/v1/products/{id}
// @Summary      Get a product by ID
// @Description  Retrieves detailed information of a single product.
// @Tags         products
// @Produce      json
// @Param        id       path      int  true  "Product ID"
// @Success      200      {object}  models.Product
// @Failure      400      {string}  string "Bad request: invalid product ID"
// @Failure      404      {string}  string "Product not found"
// @Failure      500      {string}  string "Internal server error"
// @Router       /products/{id} [get]
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Warn("invalid product ID in get request", zap.String("id_str", idStr), zap.Error(err))
		http.Error(w, "Bad request: invalid product ID", http.StatusBadRequest)
		return
	}

	p, err := h.service.GetProduct(ctx, id)
	if err == sql.ErrNoRows {
		h.logger.Info("product not found", zap.Int64("id", id))
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	} else if err != nil {
		h.logger.Error("failed to query product", zap.Int64("id", id), zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to query product: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(p); err != nil {
		h.logger.Error("failed to encode product response", zap.Error(err))
	}
}

// Update handles PUT /api/v1/products/{id}
// @Summary      Update a product
// @Description  Updates an existing product's fields.
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id       path      int             true  "Product ID"
// @Param        product  body      service.ProductRequest  true  "Product Update Payload"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {string}  string "Bad request: invalid ID or invalid JSON payload or missing name"
// @Failure      404      {string}  string "Product not found"
// @Failure      500      {string}  string "Internal server error"
// @Router       /products/{id} [put]
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Warn("invalid product ID in update request", zap.String("id_str", idStr), zap.Error(err))
		http.Error(w, "Bad request: invalid product ID", http.StatusBadRequest)
		return
	}

	var req service.ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode product update request", zap.Int64("id", id), zap.Error(err))
		http.Error(w, "Bad request: invalid JSON payload", http.StatusBadRequest)
		return
	}

	err = h.service.UpdateProduct(ctx, id, req)
	if err == sql.ErrNoRows {
		h.logger.Info("product to update not found", zap.Int64("id", id))
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	} else if err != nil {
		if err.Error() == "name is required" {
			h.logger.Warn("product update failed: name is required", zap.Int64("id", id))
			http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
			return
		}
		h.logger.Error("failed to update product", zap.Int64("id", id), zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to update product: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Product updated successfully",
	}); err != nil {
		h.logger.Error("failed to encode product update response", zap.Error(err))
	}
}

// Delete handles DELETE /api/v1/products/{id}
// @Summary      Delete a product
// @Description  Deletes a product along with all associated inventory, recipe entries, and allowed additions.
// @Tags         products
// @Produce      json
// @Param        id       path      int  true  "Product ID"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {string}  string "Bad request: invalid ID"
// @Failure      404      {string}  string "Product not found"
// @Failure      500      {string}  string "Internal server error"
// @Router       /products/{id} [delete]
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Warn("invalid product ID in delete request", zap.String("id_str", idStr), zap.Error(err))
		http.Error(w, "Bad request: invalid product ID", http.StatusBadRequest)
		return
	}

	err = h.service.DeleteProduct(ctx, id)
	if err == sql.ErrNoRows {
		h.logger.Info("product to delete not found", zap.Int64("id", id))
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	} else if err != nil {
		h.logger.Error("failed to delete product", zap.Int64("id", id), zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to delete product: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Product deleted successfully",
	}); err != nil {
		h.logger.Error("failed to encode product delete response", zap.Error(err))
	}
}
