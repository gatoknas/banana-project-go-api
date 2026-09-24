package service

import (
	"context"

	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
)

// UnitOfMeasureService handles unit of measure business logic.
type UnitOfMeasureService struct {
	repo repository.UnitOfMeasureRepository
}

// NewUnitOfMeasureService creates a new UnitOfMeasureService.
func NewUnitOfMeasureService(repo repository.UnitOfMeasureRepository) *UnitOfMeasureService {
	return &UnitOfMeasureService{repo: repo}
}

// ListUnits retrieves all units of measure.
func (s *UnitOfMeasureService) ListUnits(ctx context.Context) ([]models.UnitOfMeasure, error) {
	return s.repo.List(ctx)
}

// GetUnitByID retrieves a single unit of measure by ID.
func (s *UnitOfMeasureService) GetUnitByID(ctx context.Context, id int64) (*models.UnitOfMeasure, error) {
	return s.repo.GetByID(ctx, id)
}
