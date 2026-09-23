package email

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestDecodeRawMessage(t *testing.T) {
	payload := []byte("from: nequi\r\nsubject: hi\r\n\r\nbody")

	tests := []struct {
		name string
		in   string
	}{
		{"url-safe no padding", base64.RawURLEncoding.EncodeToString(payload)},
		{"url-safe with padding", base64.URLEncoding.EncodeToString(payload)},
		{"standard with padding", base64.StdEncoding.EncodeToString(payload)},
		{"standard no padding", base64.RawStdEncoding.EncodeToString(payload)},
		{"whitespace wrapped", "  " + base64.RawURLEncoding.EncodeToString(payload) + "  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeRawMessage(tt.in)
			if err != nil {
				t.Fatalf("decodeRawMessage error: %v", err)
			}
			if !bytes.Equal(got, payload) {
				t.Errorf("decodeRawMessage = %q, want %q", got, payload)
			}
		})
	}
}
