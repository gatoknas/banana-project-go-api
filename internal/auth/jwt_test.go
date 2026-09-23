package auth

import (
	"os"
	"testing"
)

func TestJWT(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-12345")

	tests := []struct {
		name      string
		userID    int64
		username  string
		role      string
		isRefresh bool
		wantErr   bool
	}{
		{
			name:      "valid access token generation and validation",
			userID:    1,
			username:  "testuser",
			role:      "admin",
			isRefresh: false,
			wantErr:   false,
		},
		{
			name:      "valid refresh token generation and validation",
			userID:    2,
			username:  "anotheruser",
			role:      "user",
			isRefresh: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var token string
			var err error
			if tt.isRefresh {
				token, err = GenerateRefreshToken(tt.userID, tt.username, tt.role)
			} else {
				token, err = GenerateToken(tt.userID, tt.username, tt.role)
			}

			if (err != nil) != tt.wantErr {
				t.Fatalf("generation error = %v, wantErr %v", err, tt.wantErr)
			}

			claims, err := ValidateToken(token)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validation error = %v, wantErr %v", err, tt.wantErr)
			}

			if claims != nil {
				if claims.UserID != tt.userID {
					t.Errorf("expected userID %d, got %d", tt.userID, claims.UserID)
				}
				if claims.Username != tt.username {
					t.Errorf("expected username %s, got %s", tt.username, claims.Username)
				}
				if claims.Role != tt.role {
					t.Errorf("expected role %s, got %s", tt.role, claims.Role)
				}
				expectedType := "access"
				if tt.isRefresh {
					expectedType = "refresh"
				}
				if claims.TokenType != expectedType {
					t.Errorf("expected tokenType %s, got %s", expectedType, claims.TokenType)
				}
			}
		})
	}

	t.Run("invalid token validation", func(t *testing.T) {
		_, err := ValidateToken("invalid.token.string")
		if err == nil {
			t.Errorf("expected error for invalid token, got nil")
		}
	})
}
