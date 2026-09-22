package models

import "time"

// EmailReceipt represents a bank/PSP payment receipt ingested from an email inbox
// (Spanish: Recibo de pago por correo).
type EmailReceipt struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// MessageID is the source email message id; unique and used for idempotent ingestion (Spanish: ID Mensaje).
	MessageID string `json:"messageId" db:"message_id"`
	// Sender is the email sender address (Spanish: Remitente).
	Sender string `json:"sender" db:"sender"`
	// Subject is the email subject line (Spanish: Asunto).
	Subject string `json:"subject" db:"subject"`
	// ReceivedAt is when the inbox received the email (Spanish: Recibido En).
	ReceivedAt time.Time `json:"receivedAt" db:"received_at"`
	// TransactionDate is the date/time of the payment as reported in the receipt body (Spanish: Fecha de Transacción).
	TransactionDate *time.Time `json:"transactionDate" db:"transaction_date"`
	// Amount is the payment amount (Spanish: Monto).
	Amount *float64 `json:"amount" db:"amount"`
	// Currency is the ISO currency code (Spanish: Moneda).
	Currency string `json:"currency" db:"currency"`
	// Payer is the person/entity that paid (Spanish: Pagador).
	Payer *string `json:"payer" db:"payer"`
	// Bank is the originating bank (Spanish: Banco).
	Bank *string `json:"bank" db:"bank"`
	// Reference is the bank reference code (Spanish: Referencia).
	Reference *string `json:"reference" db:"reference"`
	// TransactionNumber is the unique transaction id from the bank (Spanish: Número de Transacción).
	TransactionNumber *string `json:"transactionNumber" db:"transaction_number"`
	// PaymentMethod is the payment method label (e.g. QR Negocios Bre-B) (Spanish: Método de Pago).
	PaymentMethod *string `json:"paymentMethod" db:"payment_method"`
	// Status is the ingestion status: imported or error (Spanish: Estado).
	Status string `json:"status" db:"status"`
	// ParseError holds the parsing failure message when Status is error (Spanish: Error de Parseo).
	ParseError *string `json:"parseError,omitempty" db:"parse_error"`
	// RawBody is the decoded email body kept for auditing/debugging (Spanish: Cuerpo Original).
	RawBody *string `json:"-" db:"raw_body"`
	// CreatedAt is the ingestion timestamp (Spanish: Creado En).
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}
