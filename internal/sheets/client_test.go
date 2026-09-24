package sheets

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewClient_Config(t *testing.T) {
	tests := []struct {
		name          string
		cfg           Config
		wantErr       bool
		expectedErr   string
		wantChunkSize int
	}{
		{
			name: "Missing credentials",
			cfg: Config{
				SpreadsheetID: "dummy-id",
			},
			wantErr:     true,
			expectedErr: "no google service account credentials provided",
		},
		{
			name: "Non-existent credentials file",
			cfg: Config{
				CredentialsFile: "non-existent-file.json",
			},
			wantErr:     true,
			expectedErr: "credentials file not found",
		},
		{
			name: "Invalid raw JSON",
			cfg: Config{
				CredentialsJSON: "{invalid-json",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(context.Background(), tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.expectedErr != "" && err != nil && !containsSubstring(err.Error(), tt.expectedErr) {
					t.Errorf("expected error containing %q, got %v", tt.expectedErr, err)
				}
				return
			}
			if client == nil {
				t.Fatalf("expected client, got nil")
			}
		})
	}
}

func TestBuildAuthOptions(t *testing.T) {
	tempDir := t.TempDir()
	credFile := filepath.Join(tempDir, "service_account.json")
	if err := os.WriteFile(credFile, []byte(`{"type": "service_account"}`), 0600); err != nil {
		t.Fatalf("failed to write temp cred file: %v", err)
	}

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "Valid file path",
			cfg: Config{
				CredentialsFile: credFile,
			},
			wantErr: false,
		},
		{
			name: "Missing both JSON and file",
			cfg: Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := buildAuthOptions(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildAuthOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(opts) == 0 {
				t.Errorf("expected options, got none")
			}
		})
	}
}

func TestClient_FetchChunk_Validation(t *testing.T) {
	client := &Client{
		svc: nil,
		cfg: Config{
			ChunkSize: 50,
			Timeout:   5 * time.Second,
		},
	}

	tests := []struct {
		name          string
		spreadsheetID string
		startRow      int
		limit         int
		wantErr       bool
		expectedErr   string
	}{
		{
			name:          "Uninitialized service",
			spreadsheetID: "test-id",
			startRow:      2,
			limit:         50,
			wantErr:       true,
			expectedErr:   "google sheets service is not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.FetchChunk(context.Background(), tt.spreadsheetID, "Datos_Ventas", tt.startRow, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchChunk() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErr != "" && !containsSubstring(err.Error(), tt.expectedErr) {
				t.Errorf("expected error containing %q, got %v", tt.expectedErr, err)
			}
		})
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && len(s) > 0 && (s != "" && filepath.Base(s) != "" && stringContains(s, sub))))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
