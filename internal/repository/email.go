package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"org.banana.project/api/internal/models"
)

// EmailReceiptRepository persists bank/PSP payment receipts ingested from email.
type EmailReceiptRepository interface {
	// UpsertByMessageID inserts a receipt, returning false when it already exists
	// (deduplicated on message_id).
	UpsertByMessageID(ctx context.Context, r *models.EmailReceipt) (bool, error)
	// List returns receipts optionally filtered by a received_at range.
	List(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error)
	// GetRevenueSummary returns aggregated revenue metrics and daily timeline buckets for the given date range.
	GetRevenueSummary(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error)
}

type SQLEmailReceiptRepository struct {
	db *sql.DB
}

func NewSQLEmailReceiptRepository(db *sql.DB) *SQLEmailReceiptRepository {
	return &SQLEmailReceiptRepository{db: db}
}

func (r *SQLEmailReceiptRepository) UpsertByMessageID(ctx context.Context, e *models.EmailReceipt) (bool, error) {
	query := `INSERT INTO email_receipts
		(message_id, sender, subject, received_at, transaction_date, amount, currency,
		 payer, bank, reference, transaction_number, payment_method, status, parse_error, raw_body)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (message_id) DO NOTHING
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		e.MessageID,
		e.Sender,
		e.Subject,
		e.ReceivedAt,
		e.TransactionDate,
		e.Amount,
		e.Currency,
		e.Payer,
		e.Bank,
		e.Reference,
		e.TransactionNumber,
		e.PaymentMethod,
		e.Status,
		e.ParseError,
		e.RawBody,
	).Scan(&id)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	e.ID = id
	return true, nil
}

func (r *SQLEmailReceiptRepository) List(ctx context.Context, from, to *time.Time) ([]models.EmailReceipt, error) {
	query := `SELECT id, message_id, sender, subject, received_at, transaction_date, amount, currency,
		payer, bank, reference, transaction_number, payment_method, status, parse_error, created_at
		FROM email_receipts WHERE 1 = 1`
	args := []any{}

	if from != nil {
		args = append(args, *from)
		query += fmt.Sprintf(" AND received_at >= $%d", len(args))
	}
	if to != nil {
		args = append(args, *to)
		query += fmt.Sprintf(" AND received_at < $%d", len(args))
	}
	query += " ORDER BY received_at DESC, id DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var receipts []models.EmailReceipt
	for rows.Next() {
		var (
			e                 models.EmailReceipt
			transactionDate   sql.NullTime
			amount            sql.NullFloat64
			payer             sql.NullString
			bank              sql.NullString
			reference         sql.NullString
			transactionNumber sql.NullString
			paymentMethod     sql.NullString
			parseError        sql.NullString
		)

		if err := rows.Scan(
			&e.ID,
			&e.MessageID,
			&e.Sender,
			&e.Subject,
			&e.ReceivedAt,
			&transactionDate,
			&amount,
			&e.Currency,
			&payer,
			&bank,
			&reference,
			&transactionNumber,
			&paymentMethod,
			&e.Status,
			&parseError,
			&e.CreatedAt,
		); err != nil {
			return nil, err
		}

		if transactionDate.Valid {
			e.TransactionDate = &transactionDate.Time
		}
		if amount.Valid {
			e.Amount = &amount.Float64
		}
		if payer.Valid {
			e.Payer = &payer.String
		}
		if bank.Valid {
			e.Bank = &bank.String
		}
		if reference.Valid {
			e.Reference = &reference.String
		}
		if transactionNumber.Valid {
			e.TransactionNumber = &transactionNumber.String
		}
		if paymentMethod.Valid {
			e.PaymentMethod = &paymentMethod.String
		}
		if parseError.Valid {
			e.ParseError = &parseError.String
		}

		receipts = append(receipts, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return receipts, nil
}

func (r *SQLEmailReceiptRepository) GetRevenueSummary(ctx context.Context, from, to time.Time) (*models.RevenueSummary, error) {
	duration := to.Sub(from)
	prevFrom := from.Add(-duration)

	summaryQuery := `SELECT 
		COALESCE(SUM(CASE WHEN COALESCE(transaction_date, received_at) >= $1 AND COALESCE(transaction_date, received_at) < $2 THEN amount ELSE 0 END), 0),
		COUNT(CASE WHEN COALESCE(transaction_date, received_at) >= $1 AND COALESCE(transaction_date, received_at) < $2 THEN 1 END),
		COALESCE(SUM(CASE WHEN COALESCE(transaction_date, received_at) >= $3 AND COALESCE(transaction_date, received_at) < $1 THEN amount ELSE 0 END), 0)
		FROM email_receipts
		WHERE status = 'imported'
		  AND COALESCE(transaction_date, received_at) >= $3
		  AND COALESCE(transaction_date, received_at) < $2`

	var (
		totalRevenue          float64
		transactionCount      int
		previousPeriodRevenue float64
	)

	err := r.db.QueryRowContext(ctx, summaryQuery, from, to, prevFrom).Scan(
		&totalRevenue,
		&transactionCount,
		&previousPeriodRevenue,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue summary: %w", err)
	}

	var averageTicket float64
	if transactionCount > 0 {
		averageTicket = math.Round((totalRevenue/float64(transactionCount))*100) / 100
	}

	var growthPercentage float64
	if previousPeriodRevenue > 0 {
		growthPercentage = math.Round(((totalRevenue-previousPeriodRevenue)/previousPeriodRevenue)*10000) / 100
	} else if totalRevenue > 0 {
		growthPercentage = 100.0
	}

	timelineQuery := `SELECT 
		TO_CHAR(COALESCE(transaction_date, received_at) AT TIME ZONE 'America/Bogota', 'YYYY-MM-DD') AS day_date,
		COALESCE(SUM(amount), 0),
		COUNT(id)
		FROM email_receipts
		WHERE status = 'imported'
		  AND COALESCE(transaction_date, received_at) >= $1
		  AND COALESCE(transaction_date, received_at) < $2
		GROUP BY day_date
		ORDER BY day_date ASC`

	rows, err := r.db.QueryContext(ctx, timelineQuery, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue timeline: %w", err)
	}
	defer rows.Close()

	timeline := make([]models.DailyRevenueBucket, 0)
	for rows.Next() {
		var b models.DailyRevenueBucket
		if err := rows.Scan(&b.Date, &b.Amount, &b.Count); err != nil {
			return nil, fmt.Errorf("failed to scan timeline row: %w", err)
		}
		timeline = append(timeline, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("timeline rows error: %w", err)
	}

	return &models.RevenueSummary{
		TotalRevenue:          totalRevenue,
		TransactionCount:      transactionCount,
		AverageTicket:         averageTicket,
		PreviousPeriodRevenue: previousPeriodRevenue,
		GrowthPercentage:      growthPercentage,
		Currency:              "COP",
		Timeline:              timeline,
	}, nil
}

