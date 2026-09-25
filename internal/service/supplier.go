package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
)

var (
	ErrCompanyNameRequired  = errors.New("company_name is required")
	ErrPhoneRequired        = errors.New("phone is required")
	ErrTaxIDRequired        = errors.New("tax_id is required")
	ErrTaxIDAlreadyExists   = errors.New("tax_id already registered")
	ErrSupplierNotFound     = errors.New("supplier not found")
	ErrSupplierHasPurchases = errors.New("cannot delete supplier with associated purchases")
)

// SupplierRequest defines payload for creating or updating a supplier.
type SupplierRequest struct {
	TaxID       *string `json:"taxId"`
	CompanyName string  `json:"companyName"`
	ContactName *string `json:"contactName"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
}

type SupplierService struct {
	repo repository.SupplierRepository
	db   *sql.DB
}

func NewSupplierService(repo repository.SupplierRepository, db *sql.DB) *SupplierService {
	return &SupplierService{repo: repo, db: db}
}

func (s *SupplierService) CreateSupplier(ctx context.Context, req SupplierRequest) (int64, error) {
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	if req.CompanyName == "" {
		return 0, ErrCompanyNameRequired
	}

	if req.Phone == nil || strings.TrimSpace(*req.Phone) == "" {
		return 0, ErrPhoneRequired
	}
	cleanedPhone := strings.TrimSpace(*req.Phone)
	req.Phone = &cleanedPhone

	var taxID *string
	if req.TaxID != nil {
		trimmed := strings.TrimSpace(*req.TaxID)
		if trimmed != "" {
			taxID = &trimmed
		}
	}

	if taxID != nil {
		// Check tax_id uniqueness
		existing, err := s.repo.GetByTaxID(ctx, *taxID)
		if err == nil && existing != nil {
			return 0, ErrTaxIDAlreadyExists
		}
	}

	supplier := &models.Supplier{
		TaxID:       taxID,
		CompanyName: req.CompanyName,
		ContactName: req.ContactName,
		Phone:       req.Phone,
		Email:       req.Email,
	}

	return s.repo.Create(ctx, nil, supplier)
}

func (s *SupplierService) ListSuppliers(ctx context.Context, search string) ([]models.Supplier, error) {
	return s.repo.List(ctx, search)
}

func (s *SupplierService) GetSupplier(ctx context.Context, id int64) (*models.Supplier, error) {
	supplier, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSupplierNotFound
		}
		return nil, err
	}
	return supplier, nil
}

func (s *SupplierService) UpdateSupplier(ctx context.Context, id int64, req SupplierRequest) error {
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	if req.CompanyName == "" {
		return ErrCompanyNameRequired
	}

	if req.Phone == nil || strings.TrimSpace(*req.Phone) == "" {
		return ErrPhoneRequired
	}
	cleanedPhone := strings.TrimSpace(*req.Phone)
	req.Phone = &cleanedPhone

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSupplierNotFound
		}
		return err
	}

	var taxID *string
	if req.TaxID != nil {
		trimmed := strings.TrimSpace(*req.TaxID)
		if trimmed != "" {
			taxID = &trimmed
		}
	}

	// If tax_id changed and is non-nil, verify it doesn't conflict with another supplier
	if taxID != nil {
		if existing.TaxID == nil || *existing.TaxID != *taxID {
			other, err := s.repo.GetByTaxID(ctx, *taxID)
			if err == nil && other != nil && other.ID != id {
				return ErrTaxIDAlreadyExists
			}
		}
	}

	supplier := &models.Supplier{
		ID:          id,
		TaxID:       taxID,
		CompanyName: req.CompanyName,
		ContactName: req.ContactName,
		Phone:       req.Phone,
		Email:       req.Email,
	}

	return s.repo.Update(ctx, supplier)
}

func (s *SupplierService) DeleteSupplier(ctx context.Context, id int64) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSupplierNotFound
		}
		return err
	}

	hasPurchases, err := s.repo.HasPurchases(ctx, id)
	if err != nil {
		return err
	}
	if hasPurchases {
		return ErrSupplierHasPurchases
	}

	return s.repo.Delete(ctx, nil, id)
}
