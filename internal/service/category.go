package service

import (
	"context"

	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
)

// CategoryService handles category business logic.
type CategoryService struct {
	repo repository.CategoryRepository
}

// NewCategoryService creates a new CategoryService.
func NewCategoryService(repo repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

// ListCategories retrieves all product categories.
func (s *CategoryService) ListCategories(ctx context.Context) ([]models.Category, error) {
	return s.repo.List(ctx)
}

// GetCategoryByID retrieves a single product category by ID.
func (s *CategoryService) GetCategoryByID(ctx context.Context, id int64) (*models.Category, error) {
	return s.repo.GetByID(ctx, id)
}
