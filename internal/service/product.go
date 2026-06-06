package service

import (
	"context"
	"database/sql"
	"errors"

	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
)

// ProductRequest represents the payload for creating or updating a product.
type ProductRequest struct {
	Name            string   `json:"name"`
	Description     *string  `json:"description"`
	CategoryID      int64    `json:"categoryId"`
	UnitOfMeasureID int64    `json:"unitOfMeasureId"`
	SellPrice       float64  `json:"sellPrice"`
	AverageCost     float64  `json:"averageCost"`
	IsForSale       bool     `json:"isForSale"`
	RequiresRecipe  bool     `json:"requiresRecipe"`
}

type ProductService struct {
	repo repository.ProductRepository
	db   *sql.DB
}

func NewProductService(repo repository.ProductRepository, db *sql.DB) *ProductService {
	return &ProductService{repo: repo, db: db}
}

func (s *ProductService) CreateProduct(ctx context.Context, req ProductRequest) (int64, error) {
	if req.Name == "" {
		return 0, errors.New("name is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	p := &models.Product{
		Name:            req.Name,
		Description:     req.Description,
		CategoryID:      req.CategoryID,
		UnitOfMeasureID: req.UnitOfMeasureID,
		SellPrice:       req.SellPrice,
		AverageCost:     req.AverageCost,
		IsForSale:       req.IsForSale,
		RequiresRecipe:  req.RequiresRecipe,
	}

	id, err := s.repo.Create(ctx, tx, p)
	if err != nil {
		return 0, err
	}

	err = s.repo.InitializeInventory(ctx, tx, id)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

func (s *ProductService) ListProducts(ctx context.Context) ([]models.Product, error) {
	return s.repo.List(ctx)
}

func (s *ProductService) GetProduct(ctx context.Context, id int64) (*models.Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProductService) UpdateProduct(ctx context.Context, id int64, req ProductRequest) error {
	if req.Name == "" {
		return errors.New("name is required")
	}

	p := &models.Product{
		ID:              id,
		Name:            req.Name,
		Description:     req.Description,
		CategoryID:      req.CategoryID,
		UnitOfMeasureID: req.UnitOfMeasureID,
		SellPrice:       req.SellPrice,
		AverageCost:     req.AverageCost,
		IsForSale:       req.IsForSale,
		RequiresRecipe:  req.RequiresRecipe,
	}

	return s.repo.Update(ctx, p)
}

func (s *ProductService) DeleteProduct(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.repo.DeleteAllowedAdditions(ctx, tx, id); err != nil {
		return err
	}

	if err := s.repo.DeleteProductRecipes(ctx, tx, id); err != nil {
		return err
	}

	if err := s.repo.DeleteInventory(ctx, tx, id); err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, tx, id); err != nil {
		return err
	}

	return tx.Commit()
}
