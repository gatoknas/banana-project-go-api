package sheets

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"org.banana.project/api/internal/models"
)

// Colombia is the timezone used for receipt timestamps (UTC-5, no DST).
var Colombia = time.FixedZone("COT", -5*60*60)

// ParseRow transforms a raw Google Sheet row into a models.EmailReceipt.
// Expected row format (10 columns):
// Index 0: Fecha Procesado
// Index 1: Monto
// Index 2: Estado
// Index 3: Fecha Transacción
// Index 4: Pagador
// Index 5: Banco
// Index 6: Referencia
// Index 7: Número de Transacción
// Index 8: Método de Pago
// Index 9: ID Mensaje (Unique Identifier)
func ParseRow(row []any) (*models.EmailReceipt, error) {
	if len(row) == 0 {
		return nil, fmt.Errorf("empty row")
	}

	// Column 9: ID Mensaje (Required)
	messageID := ""
	if len(row) > 9 {
		messageID = strings.TrimSpace(getCellString(row[9]))
	}
	if messageID == "" {
		return nil, fmt.Errorf("missing unique message ID in column J")
	}

	receipt := &models.EmailReceipt{
		MessageID: messageID,
		Currency:  "COP",
		Status:    "imported",
	}

	// Column 0: Fecha Procesado
	if len(row) > 0 {
		if procStr := strings.TrimSpace(getCellString(row[0])); procStr != "" {
			if t, err := parseDate(procStr); err == nil {
				receipt.ReceivedAt = t
			}
		}
	}
	if receipt.ReceivedAt.IsZero() {
		receipt.ReceivedAt = time.Now().In(Colombia)
	}

	// Column 1: Monto
	if len(row) > 1 {
		amt, err := parseCellAmount(row[1])
		if err != nil {
			return nil, fmt.Errorf("invalid amount %v: %w", row[1], err)
		}
		receipt.Amount = &amt
	}

	// Column 2: Estado
	if len(row) > 2 {
		if status := strings.TrimSpace(getCellString(row[2])); status != "" {
			receipt.Status = strings.ToLower(status)
		}
	}

	// Column 3: Fecha Transacción
	if len(row) > 3 {
		if txDateStr := strings.TrimSpace(getCellString(row[3])); txDateStr != "" {
			if t, err := parseDate(txDateStr); err == nil {
				receipt.TransactionDate = &t
			}
		}
	}

	// Column 4: Pagador
	if len(row) > 4 {
		receipt.Payer = strPtr(getCellString(row[4]))
	}

	// Column 5: Banco
	if len(row) > 5 {
		receipt.Bank = strPtr(getCellString(row[5]))
	}

	// Column 6: Referencia
	if len(row) > 6 {
		receipt.Reference = strPtr(getCellString(row[6]))
	}

	// Column 7: Número de Transacción
	if len(row) > 7 {
		receipt.TransactionNumber = strPtr(getCellString(row[7]))
	}

	// Column 8: Método de Pago
	if len(row) > 8 {
		receipt.PaymentMethod = strPtr(getCellString(row[8]))
	}

	return receipt, nil
}

func getCellString(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func parseCellAmount(val any) (float64, error) {
	if val == nil {
		return 0, nil
	}
	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return parseAmountString(v)
	default:
		return parseAmountString(fmt.Sprintf("%v", v))
	}
}

func parseAmountString(s string) (float64, error) {
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "COP", "")
	s = strings.ReplaceAll(s, "cop", "")
	s = strings.ReplaceAll(s, "\u00a0", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	if strings.Contains(s, ",") {
		// Colombian format: dots are thousands, comma is decimal
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else if idx := strings.LastIndex(s, "."); idx >= 0 && len(s)-idx-1 == 3 {
		// A lone dot with exactly 3 trailing digits is a thousands separator in COP
		s = strings.ReplaceAll(s, ".", "")
	}

	return strconv.ParseFloat(s, 64)
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"02/01/2006",
		"2/1/2006 15:04:05",
		"2/1/2006 15:04",
		"2/1/2006",
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, Colombia); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %q", s)
}

func strPtr(s string) *string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
