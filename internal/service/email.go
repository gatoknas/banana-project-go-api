package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
	"org.banana.project/api/internal/sheets"
)

// EmailReceiptSyncRequest is the payload for a manual receipt sync (optional dates as YYYY-MM-DD).
type EmailReceiptSyncRequest struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

// SyncResult summarises the outcome of a receipt sync run.
type SyncResult struct {
	Fetched  int `json:"fetched"`
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Errors   int `json:"errors"`
}

type EmailReceiptService struct {
	repo          repository.EmailReceiptRepository
	sheetsReader  sheets.Reader
	spreadsheetID string
	sheetName     string
	chunkSize     int
	logger        *zap.Logger
}

func NewEmailReceiptService(
	repo repository.EmailReceiptRepository,
	sheetsReader sheets.Reader,
	spreadsheetID string,
	sheetName string,
	chunkSize int,
	logger *zap.Logger,
) *EmailReceiptService {
	if sheetName == "" {
		sheetName = "Datos_Ventas"
	}
	if chunkSize <= 0 {
		chunkSize = sheets.DefaultChunkSize
	}
	return &EmailReceiptService{
		repo:          repo,
		sheetsReader:  sheetsReader,
		spreadsheetID: spreadsheetID,
		sheetName:     sheetName,
		chunkSize:     chunkSize,
		logger:        logger,
	}
}

// Sync fetches and ingests receipts from Google Sheets in paginated chunks.
func (s *EmailReceiptService) Sync(ctx context.Context, req EmailReceiptSyncRequest) (SyncResult, error) {
	if s.sheetsReader == nil {
		return SyncResult{}, fmt.Errorf("google sheets integration is not configured")
	}

	var from, to *time.Time
	if strings.TrimSpace(req.From) != "" || strings.TrimSpace(req.To) != "" {
		f, t, err := parseDateRange(req.From, req.To)
		if err != nil {
			return SyncResult{}, err
		}
		from = &f
		to = &t
	}

	startRow := 2 // Row 1 is header
	result := SyncResult{}

	for {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		rows, err := s.sheetsReader.FetchChunk(ctx, s.spreadsheetID, s.sheetName, startRow, s.chunkSize)
		if err != nil {
			s.logger.Error("failed to fetch sheet chunk",
				zap.Int("startRow", startRow),
				zap.Error(err),
			)
			result.Errors++
			return result, fmt.Errorf("failed to fetch chunk starting at row %d: %w", startRow, err)
		}

		if len(rows) == 0 {
			break
		}

		result.Fetched += len(rows)
		for idx, row := range rows {
			receipt, parseErr := sheets.ParseRow(row)
			if parseErr != nil {
				s.logger.Warn("sheet row parsed with errors",
					zap.Int("row", startRow+idx),
					zap.Error(parseErr),
				)
				result.Errors++
				continue
			}

			// If explicit date range filter was requested, verify timestamp is within range
			if from != nil && to != nil {
				d := receipt.ReceivedAt
				if receipt.TransactionDate != nil {
					d = *receipt.TransactionDate
				}
				if d.Before(*from) || !d.Before(*to) {
					result.Skipped++
					continue
				}
			}

			inserted, err := s.repo.UpsertByMessageID(ctx, receipt)
			if err != nil {
				s.logger.Error("failed to persist sheet receipt",
					zap.String("messageId", receipt.MessageID),
					zap.Error(err),
				)
				result.Errors++
				continue
			}

			if inserted {
				result.Imported++
			} else {
				result.Skipped++
			}
		}

		startRow += len(rows)
		if len(rows) < s.chunkSize {
			break
		}
	}

	return result, nil
}

// SyncLastDay ingests receipts from Google Sheets.
func (s *EmailReceiptService) SyncLastDay(ctx context.Context) (SyncResult, error) {
	return s.Sync(ctx, EmailReceiptSyncRequest{})
}

// List returns stored receipts, optionally filtered by an inclusive date range.
func (s *EmailReceiptService) List(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error) {
	return s.repo.List(ctx, from, to)
}

// GetRevenueSummary returns aggregated revenue metrics and daily timeline buckets.
// If from or to are nil, it defaults to the current month in Colombia timezone.
func (s *EmailReceiptService) GetRevenueSummary(ctx context.Context, from, to *time.Time) (*models.RevenueSummary, error) {
	now := time.Now().In(sheets.Colombia)

	var start, end time.Time
	if from != nil {
		start = *from
	} else {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, sheets.Colombia)
	}

	if to != nil {
		end = *to
	} else {
		end = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, sheets.Colombia)
	}

	if !start.Before(end) {
		return nil, fmt.Errorf("invalid date range: 'from' must be strictly before 'to'")
	}

	return s.repo.GetRevenueSummary(ctx, start, end)
}

// parseDateRange converts two inclusive YYYY-MM-DD dates into a half-open
// [from, to) range in Colombia time (so the "to" day is fully included).
func parseDateRange(fromStr, toStr string) (time.Time, time.Time, error) {
	from, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(fromStr), sheets.Colombia)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' date, expected YYYY-MM-DD")
	}

	to, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(toStr), sheets.Colombia)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' date, expected YYYY-MM-DD")
	}

	if to.Before(from) {
		return time.Time{}, time.Time{}, fmt.Errorf("'to' date must be on or after 'from' date")
	}

	return from, to.AddDate(0, 0, 1), nil
}
