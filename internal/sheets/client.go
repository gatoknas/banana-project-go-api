package sheets

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	sheetsv4 "google.golang.org/api/sheets/v4"
)

// DefaultChunkSize is the fallback number of rows fetched per request.
const DefaultChunkSize = 500

// DefaultTimeout is the per-chunk HTTP request timeout.
const DefaultTimeout = 15 * time.Second

// MaxRetries is the maximum number of retry attempts when rate limited.
const MaxRetries = 5

// Config holds Google Sheets authentication and target document settings.
type Config struct {
	// CredentialsJSON holds the raw service account JSON string, or a Base64-encoded representation.
	CredentialsJSON string
	// CredentialsFile holds the file path to the service account JSON file.
	CredentialsFile string
	// SpreadsheetID is the Google Spreadsheet ID (e.g. 1W_qTHc3RxTwwjCItOwCrRbnI34-63nLuLqG5it5LY7E).
	SpreadsheetID string
	// SheetName is the tab/sheet name (e.g. Datos_Ventas).
	SheetName string
	// ChunkSize is the number of rows per chunk (defaults to 500).
	ChunkSize int
	// Timeout is the maximum duration for a single chunk fetch (defaults to 15s).
	Timeout time.Duration
}

// Reader abstracts the fetching of sheet rows in paginated chunks.
type Reader interface {
	FetchChunk(ctx context.Context, spreadsheetID, sheetName string, startRow, limit int) ([][]any, error)
}

// Client wraps the official Google Sheets API v4 service.
type Client struct {
	svc *sheetsv4.Service
	cfg Config
}

// NewClient initializes a Google Sheets client authenticated via Google Cloud Service Account.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = DefaultChunkSize
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}

	opts, err := buildAuthOptions(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to configure google sheets auth: %w", err)
	}

	svc, err := sheetsv4.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create google sheets service: %w", err)
	}

	return &Client{
		svc: svc,
		cfg: cfg,
	}, nil
}

// FetchChunk reads a bounded row range from Google Sheets starting at startRow up to startRow+limit-1.
// Returns an empty slice if the requested range has no rows (EOF).
// Automatically retries with exponential backoff when a 429 Rate Limit error is returned by Google Sheets API.
func (c *Client) FetchChunk(ctx context.Context, spreadsheetID, sheetName string, startRow, limit int) ([][]any, error) {
	if c == nil || c.svc == nil {
		return nil, fmt.Errorf("google sheets service is not initialized")
	}
	if spreadsheetID == "" {
		spreadsheetID = c.cfg.SpreadsheetID
	}
	if spreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheet ID cannot be empty")
	}
	if sheetName == "" {
		sheetName = c.cfg.SheetName
	}
	if sheetName == "" {
		sheetName = "Datos_Ventas"
	}
	if limit <= 0 {
		limit = c.cfg.ChunkSize
	}
	if limit <= 0 {
		limit = DefaultChunkSize
	}
	if startRow < 1 {
		startRow = 1
	}

	endRow := startRow + limit - 1
	rangeSpec := fmt.Sprintf("%s!A%d:J%d", sheetName, startRow, endRow)

	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		fetchCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
		resp, err := c.svc.Spreadsheets.Values.Get(spreadsheetID, rangeSpec).Context(fetchCtx).Do()
		cancel()

		if err == nil {
			if resp == nil || len(resp.Values) == 0 {
				return [][]any{}, nil
			}
			return resp.Values, nil
		}

		lastErr = err
		if isRateLimitError(err) && attempt < MaxRetries {
			// Exponential backoff: 3s, 6s, 12s, 24s, 48s
			backoff := time.Duration(1<<attempt) * 3 * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}

		break
	}

	return nil, fmt.Errorf("failed to fetch sheet range %q: %w", rangeSpec, lastErr)
}

// IsRateLimitError checks if an error returned by Google API is a quota or rate-limit violation (HTTP 429).
func IsRateLimitError(err error) bool {
	return isRateLimitError(err)
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		if gErr.Code == http.StatusTooManyRequests || gErr.Code == 429 {
			return true
		}
		for _, item := range gErr.Errors {
			if item.Reason == "rateLimitExceeded" || item.Reason == "userRateLimitExceeded" || item.Reason == "quotaExceeded" {
				return true
			}
		}
	}
	msg := err.Error()
	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "RATE_LIMIT_EXCEEDED") ||
		strings.Contains(msg, "rateLimitExceeded") ||
		strings.Contains(msg, "userRateLimitExceeded") ||
		strings.Contains(msg, "Quota exceeded")
}

// buildAuthOptions creates Google client options from JSON string, base64 string, or file path.
func buildAuthOptions(cfg Config) ([]option.ClientOption, error) {
	rawJSON := strings.TrimSpace(cfg.CredentialsJSON)
	if rawJSON != "" {
		// If base64-encoded, decode it first
		if !strings.HasPrefix(rawJSON, "{") {
			decoded, err := base64.StdEncoding.DecodeString(rawJSON)
			if err == nil && strings.HasPrefix(strings.TrimSpace(string(decoded)), "{") {
				rawJSON = string(decoded)
			}
		}
		return []option.ClientOption{
			option.WithCredentialsJSON([]byte(rawJSON)),
			option.WithScopes(sheetsv4.SpreadsheetsReadonlyScope),
		}, nil
	}

	filePath := strings.TrimSpace(cfg.CredentialsFile)
	if filePath != "" {
		if _, err := os.Stat(filePath); err != nil {
			return nil, fmt.Errorf("credentials file not found at %q: %w", filePath, err)
		}
		return []option.ClientOption{
			option.WithCredentialsFile(filePath),
			option.WithScopes(sheetsv4.SpreadsheetsReadonlyScope),
		}, nil
	}

	// Fallback to Application Default Credentials if GOOGLE_APPLICATION_CREDENTIALS exists in environment
	if envPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); envPath != "" {
		return []option.ClientOption{
			option.WithScopes(sheetsv4.SpreadsheetsReadonlyScope),
		}, nil
	}

	return nil, fmt.Errorf("no google service account credentials provided (set GOOGLE_SHEETS_CREDENTIALS_JSON or GOOGLE_APPLICATION_CREDENTIALS)")
}
