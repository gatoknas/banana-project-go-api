package email

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Colombia is the timezone used for receipt transaction timestamps (UTC-5, no DST).
var Colombia = time.FixedZone("COT", -5*60*60)

// ParsedReceipt holds the sale information extracted from a bank/PSP receipt email.
type ParsedReceipt struct {
	Amount            float64
	Currency          string
	Status            string
	TransactionDate   time.Time
	Payer             string
	Bank              string
	Reference         string
	TransactionNumber string
	PaymentMethod     string
}

var (
	// Matches the receipt detail table rows: <th>Label:</th><td>Value</td>
	cellRe = regexp.MustCompile(`(?is)<th[^>]*>\s*(.*?)\s*</th>\s*<td[^>]*>\s*(.*?)\s*</td>`)
	// Matches the headline amount: "Venta exitosa por $ 2.000"
	titleAmountRe = regexp.MustCompile(`(?is)Venta exitosa por\s*([^<]+)`)
	tagRe         = regexp.MustCompile(`(?s)<[^>]*>`)
	spaceRe       = regexp.MustCompile(`\s+`)

	accentReplacer = strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u", "Ü", "u", "Ñ", "n",
	)
)

// ParseReceipt extracts sale fields from a decoded receipt email body (HTML or text).
// It is tuned to the Nequi "Detalle de tu venta" template but relies on generic
// label/value matching, so it is resilient to minor markup changes.
func ParseReceipt(body string) (*ParsedReceipt, error) {
	fields := map[string]string{}
	for _, match := range cellRe.FindAllStringSubmatch(body, -1) {
		key := normalizeLabel(match[1])
		value := cleanValue(match[2])
		if key != "" && value != "" {
			fields[key] = value
		}
	}

	receipt := &ParsedReceipt{Currency: "COP"}

	amountStr := fields["monto"]
	if amountStr == "" {
		if match := titleAmountRe.FindStringSubmatch(body); len(match) == 2 {
			amountStr = cleanValue(match[1])
		}
	}
	if amountStr != "" {
		amount, err := parseAmount(amountStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse amount %q: %w", amountStr, err)
		}
		receipt.Amount = amount
	}

	receipt.Status = fields["estado"]
	if dateStr := fields["fecha"]; dateStr != "" {
		t, err := parseDate(dateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse transaction date %q: %w", dateStr, err)
		}
		receipt.TransactionDate = t
	}
	receipt.Payer = fields["pagador"]
	receipt.Bank = fields["banco"]
	receipt.Reference = fields["referencia"]
	receipt.TransactionNumber = fields["numero de transaccion"]
	receipt.PaymentMethod = fields["metodo de pago"]

	if receipt.Amount == 0 && receipt.Reference == "" && receipt.TransactionNumber == "" {
		return nil, fmt.Errorf("no recognizable receipt fields found")
	}

	return receipt, nil
}

// normalizeLabel strips markup/entities, removes a trailing colon, lowercases and
// de-accentuates a table label so it can be used as a stable map key.
func normalizeLabel(s string) string {
	s = tagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), ":"))
	s = strings.ToLower(s)
	s = accentReplacer.Replace(s)
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// cleanValue strips markup/entities and collapses whitespace in a table value.
func cleanValue(s string) string {
	s = tagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// parseAmount parses a Colombian-formatted amount (e.g. "$ 2.000" -> 2000).
func parseAmount(s string) (float64, error) {
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "\u00a0", "")
	s = strings.ReplaceAll(s, " ", "")
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}

	if strings.Contains(s, ",") {
		// Colombian format: dots are thousands separators, comma is the decimal mark.
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else if idx := strings.LastIndex(s, "."); idx >= 0 && len(s)-idx-1 == 3 {
		// A lone dot with exactly 3 trailing digits is a thousands separator (COP).
		s = strings.ReplaceAll(s, ".", "")
	}

	return strconv.ParseFloat(s, 64)
}

// parseDate parses the "Fecha" value (DD/MM/YYYY HH:MM:SS) in Colombia time.
func parseDate(s string) (time.Time, error) {
	layouts := []string{
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, Colombia); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format")
}
