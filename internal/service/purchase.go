package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
)

var (
	ErrSupplierIDRequired = errors.New("supplier_id is required")
	ErrItemsRequired      = errors.New("purchase must contain at least one item")
	ErrInvalidProductID   = errors.New("invalid product_id in purchase item")
	ErrInvalidQuantity    = errors.New("quantity_purchased must be greater than zero")
	ErrInvalidUnitCost    = errors.New("unit_cost cannot be negative")
	ErrPurchaseNotFound   = errors.New("purchase not found")
)

// PurchaseItemRequest represents a line-item payload for creating a purchase.
type PurchaseItemRequest struct {
	ProductID         int64   `json:"productId"`
	PurchaseUnitID    *int64  `json:"purchaseUnitId"`
	QuantityPurchased float64 `json:"quantityPurchased"`
	UnitCost          float64 `json:"unitCost"`
	ConversionFactor  float64 `json:"conversionFactor"`
}

// PurchaseRequest represents the payload for registering a restock purchase.
type PurchaseRequest struct {
	SupplierID    int64                 `json:"supplierId"`
	PurchaseDate  *time.Time            `json:"purchaseDate"`
	InvoiceNumber *string               `json:"invoiceNumber"`
	Notes         *string               `json:"notes"`
	Items         []PurchaseItemRequest `json:"items"`
}

type PurchaseService struct {
	repo repository.PurchaseRepository
	db   *sql.DB
}

func NewPurchaseService(repo repository.PurchaseRepository, db *sql.DB) *PurchaseService {
	return &PurchaseService{repo: repo, db: db}
}

// CreatePurchase registers a restock invoice, inserts detail lines, increments stock, and updates weighted average cost.
func (s *PurchaseService) CreatePurchase(ctx context.Context, req PurchaseRequest) (int64, error) {
	if req.SupplierID <= 0 {
		return 0, ErrSupplierIDRequired
	}
	if len(req.Items) == 0 {
		return 0, ErrItemsRequired
	}

	for _, item := range req.Items {
		if item.ProductID <= 0 {
			return 0, ErrInvalidProductID
		}
		if item.QuantityPurchased <= 0 {
			return 0, ErrInvalidQuantity
		}
		if item.UnitCost < 0 {
			return 0, ErrInvalidUnitCost
		}
	}

	var totalAmount float64
	for _, item := range req.Items {
		totalAmount += item.QuantityPurchased * item.UnitCost
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	purchaseDate := time.Now()
	if req.PurchaseDate != nil {
		purchaseDate = *req.PurchaseDate
	}

	p := &models.Purchase{
		SupplierID:    req.SupplierID,
		PurchaseDate:  purchaseDate,
		InvoiceNumber: req.InvoiceNumber,
		TotalAmount:   totalAmount,
		Notes:         req.Notes,
	}

	purchaseID, err := s.repo.CreatePurchase(ctx, tx, p)
	if err != nil {
		return 0, fmt.Errorf("failed to insert purchase header: %w", err)
	}

	for _, item := range req.Items {
		convFactor := item.ConversionFactor
		if convFactor <= 0 {
			convFactor = 1.0
		}

		pd := &models.PurchaseDetail{
			PurchaseID:        purchaseID,
			ProductID:         item.ProductID,
			PurchaseUnitID:    item.PurchaseUnitID,
			QuantityPurchased: item.QuantityPurchased,
			UnitCost:          item.UnitCost,
			ConversionFactor:  convFactor,
		}

		if _, err := s.repo.CreatePurchaseDetail(ctx, tx, pd); err != nil {
			return 0, fmt.Errorf("failed to insert purchase detail for product %d: %w", item.ProductID, err)
		}

		currentStock, currentAvgCost, err := s.repo.GetProductStockAndCost(ctx, tx, item.ProductID)
		if err != nil {
			return 0, fmt.Errorf("failed to retrieve product stock and cost for product %d: %w", item.ProductID, err)
		}

		baseQty := item.QuantityPurchased * convFactor
		baseUnitCost := item.UnitCost / convFactor

		var newAvgCost float64
		if currentStock <= 0 {
			newAvgCost = baseUnitCost
		} else {
			totalPrevValue := currentStock * currentAvgCost
			newPurchaseValue := baseQty * baseUnitCost
			newTotalStock := currentStock + baseQty
			if newTotalStock > 0 {
				newAvgCost = (totalPrevValue + newPurchaseValue) / newTotalStock
			} else {
				newAvgCost = baseUnitCost
			}
		}

		if err := s.repo.UpdateProductAverageCost(ctx, tx, item.ProductID, newAvgCost); err != nil {
			return 0, fmt.Errorf("failed to update average cost for product %d: %w", item.ProductID, err)
		}

		if err := s.repo.AddStock(ctx, tx, item.ProductID, baseQty); err != nil {
			return 0, fmt.Errorf("failed to increment stock for product %d: %w", item.ProductID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit purchase transaction: %w", err)
	}

	return purchaseID, nil
}

// ListPurchases retrieves purchases with optional supplier and date filtering.
func (s *PurchaseService) ListPurchases(ctx context.Context, supplierID *int64, fromDate, toDate *time.Time) ([]models.Purchase, error) {
	return s.repo.ListPurchases(ctx, supplierID, fromDate, toDate)
}

// GetPurchase retrieves a single purchase with its line items.
func (s *PurchaseService) GetPurchase(ctx context.Context, id int64) (*models.Purchase, error) {
	purchase, err := s.repo.GetPurchaseByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPurchaseNotFound
		}
		return nil, err
	}

	details, err := s.repo.GetPurchaseDetails(ctx, id)
	if err != nil {
		return nil, err
	}
	purchase.Details = details

	return purchase, nil
}
