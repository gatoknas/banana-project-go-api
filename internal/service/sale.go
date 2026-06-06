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

// SaleItemRequest represents an item in the sale creation payload.
type SaleItemRequest struct {
	ProductID int64   `json:"productId"`
	Quantity  float64 `json:"quantity"`
}

// SaleRequest represents the sales creation payload.
type SaleRequest struct {
	UserID        int64             `json:"userId"`
	TotalAmount   float64           `json:"totalAmount"`
	PaymentMethod string            `json:"paymentMethod"`
	Items         []SaleItemRequest `json:"items"`
}

type SaleService struct {
	repo repository.SaleRepository
	db   *sql.DB
}

func NewSaleService(repo repository.SaleRepository, db *sql.DB) *SaleService {
	return &SaleService{repo: repo, db: db}
}

func (s *SaleService) CreateSale(ctx context.Context, req SaleRequest) (int64, error) {
	if len(req.Items) == 0 {
		return 0, errors.New("sale must have at least one item")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 1. Insert into sales table
	sale := &models.Sale{
		UserID:        req.UserID,
		SaleDate:      time.Now(),
		TotalAmount:   req.TotalAmount,
		PaymentMethod: req.PaymentMethod,
	}

	saleID, err := s.repo.CreateSale(ctx, tx, sale)
	if err != nil {
		return 0, fmt.Errorf("failed to insert sale header: %w", err)
	}

	// 2. Loop through sale items, retrieve current price, insert details, and update inventory
	for _, item := range req.Items {
		sellPrice, requiresRecipe, err := s.repo.GetProductDetails(ctx, tx, item.ProductID)
		if err != nil {
			return 0, fmt.Errorf("product ID %d not found: %w", item.ProductID, err)
		}

		subtotal := item.Quantity * sellPrice

		// Insert into sale_details table
		detail := &models.SaleDetail{
			SaleID:              saleID,
			ProductID:           item.ProductID,
			Quantity:            item.Quantity,
			HistoricalUnitPrice: sellPrice,
			Subtotal:            subtotal,
		}

		_, err = s.repo.CreateSaleDetail(ctx, tx, detail)
		if err != nil {
			return 0, fmt.Errorf("failed to insert sale detail: %w", err)
		}

		// Update inventory
		if requiresRecipe {
			recipes, err := s.repo.GetRecipeIngredients(ctx, tx, item.ProductID)
			if err != nil {
				return 0, fmt.Errorf("failed to load recipe: %w", err)
			}

			for _, r := range recipes {
				deduction := r.Quantity * item.Quantity
				err = s.repo.DeductStock(ctx, tx, r.ChildProductID, deduction)
				if err != nil {
					return 0, fmt.Errorf("failed to deduct ingredient stock %d: %w", r.ChildProductID, err)
				}
			}
		} else {
			// Deduct product stock itself
			err = s.repo.DeductStock(ctx, tx, item.ProductID, item.Quantity)
			if err != nil {
				return 0, fmt.Errorf("failed to deduct product stock %d: %w", item.ProductID, err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return saleID, nil
}
