package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/handlers"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/service"
)

type MockUserRepo struct {
	GetByIDFunc       func(ctx context.Context, id int64) (*models.User, error)
	GetByUsernameFunc func(ctx context.Context, username string) (*models.User, error)
	ListFunc          func(ctx context.Context) ([]models.User, error)
	CreateFunc        func(ctx context.Context, u *models.User) (int64, error)
	UpdateFunc        func(ctx context.Context, u *models.User) error
	DeleteFunc        func(ctx context.Context, id int64) error
}

func (m *MockUserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *MockUserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return m.GetByUsernameFunc(ctx, username)
}

func (m *MockUserRepo) List(ctx context.Context) ([]models.User, error) {
	return m.ListFunc(ctx)
}

func (m *MockUserRepo) Create(ctx context.Context, u *models.User) (int64, error) {
	return m.CreateFunc(ctx, u)
}

func (m *MockUserRepo) Update(ctx context.Context, u *models.User) error {
	return m.UpdateFunc(ctx, u)
}

func (m *MockUserRepo) Delete(ctx context.Context, id int64) error {
	return m.DeleteFunc(ctx, id)
}

func newUserHandler(repo *MockUserRepo) *handlers.UserHandler {
	return handlers.NewUserHandler(service.NewUserService(repo, nil), zap.NewNop())
}

func TestUserHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		repo       *MockUserRepo
		wantStatus int
	}{
		{
			name: "success",
			body: `{"fullName":"Jane Doe","username":"jdoe","password":"secret123","role":"ayurami-admin","isActive":true}`,
			repo: &MockUserRepo{
				CreateFunc: func(ctx context.Context, u *models.User) (int64, error) { return 42, nil },
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json",
			body:       `{`,
			repo:       &MockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid role",
			body:       `{"fullName":"Jane","username":"jdoe","password":"secret123","role":"nope"}`,
			repo:       &MockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newUserHandler(tt.repo)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.Create(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantStatus == http.StatusCreated {
				var resp handlers.MessageResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.ID != 42 || resp.Status != "success" {
					t.Errorf("unexpected response: %+v", resp)
				}
			}
		})
	}
}

func TestUserHandler_List(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		repo       *MockUserRepo
		wantStatus int
		wantCount  int
	}{
		{
			name: "success",
			repo: &MockUserRepo{
				ListFunc: func(ctx context.Context) ([]models.User, error) {
					return []models.User{{ID: 1, FullName: "A", CreatedAt: now}}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name: "repository error",
			repo: &MockUserRepo{
				ListFunc: func(ctx context.Context) ([]models.User, error) {
					return nil, errors.New("db error")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newUserHandler(tt.repo)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
			w := httptest.NewRecorder()

			h.List(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantStatus == http.StatusOK {
				var resp []models.User
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(resp) != tt.wantCount {
					t.Errorf("expected %d users, got %d", tt.wantCount, len(resp))
				}
			}
		})
	}
}

func TestUserHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		repo       *MockUserRepo
		wantStatus int
	}{
		{
			name: "success",
			id:   "1",
			repo: &MockUserRepo{
				GetByIDFunc: func(ctx context.Context, id int64) (*models.User, error) {
					return &models.User{ID: id, FullName: "Jane"}, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			id:         "abc",
			repo:       &MockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			id:   "9",
			repo: &MockUserRepo{
				GetByIDFunc: func(ctx context.Context, id int64) (*models.User, error) {
					return nil, sql.ErrNoRows
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "repository error",
			id:   "9",
			repo: &MockUserRepo{
				GetByIDFunc: func(ctx context.Context, id int64) (*models.User, error) {
					return nil, errors.New("db error")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newUserHandler(tt.repo)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+tt.id, nil)
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			h.Get(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestUserHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		repo       *MockUserRepo
		wantStatus int
	}{
		{
			name: "success",
			id:   "1",
			body: `{"fullName":"New Name","isActive":true}`,
			repo: &MockUserRepo{
				GetByIDFunc: func(ctx context.Context, id int64) (*models.User, error) {
					return &models.User{ID: id, Username: "jdoe"}, nil
				},
				UpdateFunc: func(ctx context.Context, u *models.User) error { return nil },
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			id:         "abc",
			body:       `{}`,
			repo:       &MockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			id:         "1",
			body:       `{`,
			repo:       &MockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			id:   "9",
			body: `{}`,
			repo: &MockUserRepo{
				GetByIDFunc: func(ctx context.Context, id int64) (*models.User, error) {
					return nil, sql.ErrNoRows
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "repository error",
			id:   "9",
			body: `{}`,
			repo: &MockUserRepo{
				GetByIDFunc: func(ctx context.Context, id int64) (*models.User, error) {
					return nil, errors.New("db error")
				},
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newUserHandler(tt.repo)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+tt.id, strings.NewReader(tt.body))
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			h.Update(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestUserHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		repo       *MockUserRepo
		wantStatus int
	}{
		{
			name: "success",
			id:   "1",
			repo: &MockUserRepo{
				DeleteFunc: func(ctx context.Context, id int64) error { return nil },
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			id:         "abc",
			repo:       &MockUserRepo{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			id:   "9",
			repo: &MockUserRepo{
				DeleteFunc: func(ctx context.Context, id int64) error { return sql.ErrNoRows },
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "repository error",
			id:   "9",
			repo: &MockUserRepo{
				DeleteFunc: func(ctx context.Context, id int64) error { return errors.New("db error") },
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newUserHandler(tt.repo)
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+tt.id, nil)
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			h.Delete(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
