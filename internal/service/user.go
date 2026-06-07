package service

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"org.banana.project/api/internal/auth"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,20}$`)

func IsValidUsername(username string) bool {
	return usernameRegex.MatchString(username)
}

type UserRequest struct {
	FullName string `json:"fullName"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role"`
	IsActive bool   `json:"isActive"`
}

type UserService struct {
	repo repository.UserRepository
	db   *sql.DB
}

func NewUserService(repo repository.UserRepository, db *sql.DB) *UserService {
	return &UserService{repo: repo, db: db}
}

func (s *UserService) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	return s.repo.GetByUsername(ctx, username)
}

func (s *UserService) ListUsers(ctx context.Context) ([]models.User, error) {
	return s.repo.List(ctx)
}

func (s *UserService) CreateUser(ctx context.Context, req UserRequest) (int64, error) {
	if req.FullName == "" {
		return 0, errors.New("full name is required")
	}

	username := strings.ToLower(strings.TrimSpace(req.Username))
	if !IsValidUsername(username) {
		return 0, errors.New("username must be between 3 and 20 characters and contain only letters, numbers, underscores, or hyphens")
	}

	if req.Password == "" {
		return 0, errors.New("password is required")
	}

	if req.Role != "ayurami-admin" && req.Role != "ayurami-salesperson" {
		return 0, errors.New("invalid role, must be 'ayurami-admin' or 'ayurami-salesperson'")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return 0, err
	}

	user := &models.User{
		FullName:     req.FullName,
		Username:     username,
		PasswordHash: hash,
		Role:         req.Role,
		IsActive:     req.IsActive,
	}

	return s.repo.Create(ctx, user)
}

func (s *UserService) UpdateUser(ctx context.Context, id int64, req UserRequest) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if req.FullName != "" {
		existing.FullName = req.FullName
	}

	if req.Username != "" {
		username := strings.ToLower(strings.TrimSpace(req.Username))
		if !IsValidUsername(username) {
			return errors.New("username must be between 3 and 20 characters and contain only letters, numbers, underscores, or hyphens")
		}
		existing.Username = username
	}

	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			return err
		}
		existing.PasswordHash = hash
	}

	if req.Role != "" {
		if req.Role != "ayurami-admin" && req.Role != "ayurami-salesperson" {
			return errors.New("invalid role, must be 'ayurami-admin' or 'ayurami-salesperson'")
		}
		existing.Role = req.Role
	}

	existing.IsActive = req.IsActive

	return s.repo.Update(ctx, existing)
}

func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
