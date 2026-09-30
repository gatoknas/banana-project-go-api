package main

import (
	"testing"
)

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		wantErr bool
	}{
		{
			name:    "Production environment logger",
			env:     "production",
			wantErr: false,
		},
		{
			name:    "Development environment logger",
			env:     "development",
			wantErr: false,
		},
		{
			name:    "Custom environment logger",
			env:     "staging",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := initLogger(tt.env)
			if (err != nil) != tt.wantErr {
				t.Fatalf("initLogger() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && logger == nil {
				t.Errorf("initLogger() returned nil logger")
			}
		})
	}
}
