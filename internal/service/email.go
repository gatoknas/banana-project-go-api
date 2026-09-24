package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"org.banana.project/api/internal/email"
	"org.banana.project/api/internal/models"
	"org.banana.project/api/internal/repository"
)

// EmailReceiptSyncRequest is the payload for a manual receipt sync (dates as YYYY-MM-DD).
type EmailReceiptSyncRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// SyncResult summarises the outcome of a receipt sync run.
type SyncResult struct {
	Fetched  int `json:"fetched"`
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Errors   int `json:"errors"`
}

type EmailReceiptService struct {
	repo   repository.EmailReceiptRepository
	client *email.Client
	logger *zap.Logger
}

func NewEmailReceiptService(repo repository.EmailReceiptRepository, client *email.Client, logger *zap.Logger) *EmailReceiptService {
	return &EmailReceiptService{repo: repo, client: client, logger: logger}
}

// Sync fetches and ingests receipts for an explicit date range.
func (s *EmailReceiptService) Sync(ctx context.Context, req EmailReceiptSyncRequest) (SyncResult, error) {
	if s.client == nil {
		return SyncResult{}, fmt.Errorf("gmail integration is not configured")
	}

	from, to, err := parseDateRange(req.From, req.To)
	if err != nil {
		return SyncResult{}, err
	}

	return s.syncRange(ctx, from, to)
}

// SyncLastDay ingests receipts from the last 24 hours; used by the scheduler.
func (s *EmailReceiptService) SyncLastDay(ctx context.Context) (SyncResult, error) {
	if s.client == nil {
		return SyncResult{}, fmt.Errorf("gmail integration is not configured")
	}

	now := time.Now()
	return s.syncRange(ctx, now.Add(-24*time.Hour), now)
}

// List returns stored receipts, optionally filtered by an inclusive date range.
func (s *EmailReceiptService) List(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error) {
	return s.repo.List(ctx, from, to)
}

// GetRevenueSummary returns aggregated revenue metrics and daily timeline buckets.
// If from or to are nil, it defaults to the current month in Colombia timezone.
func (s *EmailReceiptService) GetRevenueSummary(ctx context.Context, from, to *time.Time) (*models.RevenueSummary, error) {
	now := time.Now().In(email.Colombia)

	var start, end time.Time
	if from != nil {
		start = *from
	} else {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, email.Colombia)
	}

	if to != nil {
		end = *to
	} else {
		end = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, email.Colombia)
	}

	if !start.Before(end) {
		return nil, fmt.Errorf("invalid date range: 'from' must be strictly before 'to'")
	}

	return s.repo.GetRevenueSummary(ctx, start, end)
}

func (s *EmailReceiptService) syncRange(ctx context.Context, from, to time.Time) (SyncResult, error) {
	messages, err := s.client.FetchMessages(ctx, from, to)
	if err != nil {
		return SyncResult{}, fmt.Errorf("failed to fetch messages: %w", err)
	}

	result := SyncResult{Fetched: len(messages)}
	for _, raw := range messages {
		receipt, parseErr := s.buildReceipt(raw)

		inserted, err := s.repo.UpsertByMessageID(ctx, receipt)
		if err != nil {
			s.logger.Error("failed to persist email receipt",
				zap.String("messageId", raw.ID), zap.Error(err))
			result.Errors++
			continue
		}

		if parseErr != nil {
			s.logger.Warn("email receipt parsed with errors",
				zap.String("messageId", raw.ID), zap.Error(parseErr))
			result.Errors++
			continue
		}

		if inserted {
			result.Imported++
		} else {
			result.Skipped++
		}
	}

	return result, nil
}

// buildReceipt maps a raw Gmail message into a persistable EmailReceipt. When
// parsing fails, it still returns a receipt marked as "error" so the message is
// recorded (and deduplicated) rather than retried forever.
func (s *EmailReceiptService) buildReceipt(raw email.RawMessage) (*models.EmailReceipt, error) {
	receipt := &models.EmailReceipt{
		MessageID: raw.ID,
		Currency:  "COP",
		Status:    "imported",
	}

	msg, err := email.ParseMessage(raw.Raw)
	if err != nil {
		receipt.Status = "error"
		receipt.ReceivedAt = internalTimestamp(raw.InternalTS)
		parseErrMsg := err.Error()
		receipt.ParseError = &parseErrMsg
		return receipt, fmt.Errorf("failed to parse message body: %w", err)
	}

	receipt.Sender = msg.Sender
	receipt.Subject = msg.Subject
	receipt.RawBody = &msg.Body
	if !msg.Date.IsZero() {
		receipt.ReceivedAt = msg.Date
	} else {
		receipt.ReceivedAt = internalTimestamp(raw.InternalTS)
	}

	parsed, err := email.ParseReceipt(msg.Body)
	if err != nil {
		receipt.Status = "error"
		parseErr := err.Error()
		receipt.ParseError = &parseErr
		return receipt, err
	}

	receipt.Amount = &parsed.Amount
	receipt.Currency = parsed.Currency
	receipt.Payer = strPtr(parsed.Payer)
	receipt.Bank = strPtr(parsed.Bank)
	receipt.Reference = strPtr(parsed.Reference)
	receipt.TransactionNumber = strPtr(parsed.TransactionNumber)
	receipt.PaymentMethod = strPtr(parsed.PaymentMethod)
	if !parsed.TransactionDate.IsZero() {
		receipt.TransactionDate = &parsed.TransactionDate
	}

	return receipt, nil
}

// parseDateRange converts two inclusive YYYY-MM-DD dates into a half-open
// [from, to) range in Colombia time (so the "to" day is fully included).
func parseDateRange(fromStr, toStr string) (time.Time, time.Time, error) {
	from, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(fromStr), email.Colombia)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' date, expected YYYY-MM-DD")
	}

	to, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(toStr), email.Colombia)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' date, expected YYYY-MM-DD")
	}

	if to.Before(from) {
		return time.Time{}, time.Time{}, fmt.Errorf("'to' date must be on or after 'from' date")
	}

	return from, to.AddDate(0, 0, 1), nil
}

func internalTimestamp(millis int64) time.Time {
	if millis <= 0 {
		return time.Now().UTC()
	}
	return time.UnixMilli(millis).UTC()
}

func strPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
