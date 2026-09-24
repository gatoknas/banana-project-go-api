package sheets

import (
	"testing"
)

func TestParseRow(t *testing.T) {
	tests := []struct {
		name          string
		row           []any
		wantErr       bool
		wantMessageID string
		wantAmount    float64
		wantStatus    string
		wantPayer     string
	}{
		{
			name: "Valid row with all 10 columns",
			row: []any{
				"2026-09-24 14:00:00",
				"$ 25.000,50",
				"imported",
				"2026-09-24 13:58:00",
				"Daniel Gomez",
				"Nequi",
				"REF12345",
				"TRX9988",
				"QR Negocios Bre-B",
				"MSG-001",
			},
			wantErr:       false,
			wantMessageID: "MSG-001",
			wantAmount:    25000.50,
			wantStatus:    "imported",
			wantPayer:     "Daniel Gomez",
		},
		{
			name: "Valid row with numeric float64 amount",
			row: []any{
				"2026-09-24",
				150000.0,
				"EXITOSO",
				"2026-09-24",
				"Empresa XYZ",
				"Bancolombia",
				"REF-A1",
				"TRX-001",
				"Transferencia",
				"MSG-002",
			},
			wantErr:       false,
			wantMessageID: "MSG-002",
			wantAmount:    150000.0,
			wantStatus:    "exitoso",
			wantPayer:     "Empresa XYZ",
		},
		{
			name: "Valid row with integer amount and Colombian thousand separator",
			row: []any{
				"24/09/2026 10:30:00",
				"35.000",
				"",
				"24/09/2026 10:28:00",
				"Maria Perez",
				"Daviplata",
				"REF-DP",
				"TRX-DP-1",
				"Daviplata",
				"MSG-003",
			},
			wantErr:       false,
			wantMessageID: "MSG-003",
			wantAmount:    35000.0,
			wantStatus:    "imported",
			wantPayer:     "Maria Perez",
		},
		{
			name: "Missing message ID in column J",
			row: []any{
				"2026-09-24",
				"10000",
				"imported",
				"2026-09-24",
				"Test User",
				"Bank",
				"Ref",
				"Trx",
				"Method",
				"", // empty message id
			},
			wantErr: true,
		},
		{
			name:    "Empty row slice",
			row:     []any{},
			wantErr: true,
		},
		{
			name: "Row too short (less than 10 columns)",
			row: []any{
				"2026-09-24",
				"10000",
			},
			wantErr: true,
		},
		{
			name: "Invalid amount string",
			row: []any{
				"2026-09-24",
				"not-a-number",
				"imported",
				"2026-09-24",
				"User",
				"Bank",
				"Ref",
				"Trx",
				"Method",
				"MSG-ERR",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRow(tt.row)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseRow() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if got.MessageID != tt.wantMessageID {
				t.Errorf("got.MessageID = %v, want %v", got.MessageID, tt.wantMessageID)
			}
			if got.Amount == nil || *got.Amount != tt.wantAmount {
				t.Errorf("got.Amount = %v, want %v", got.Amount, tt.wantAmount)
			}
			if got.Status != tt.wantStatus {
				t.Errorf("got.Status = %v, want %v", got.Status, tt.wantStatus)
			}
			if got.Payer == nil || *got.Payer != tt.wantPayer {
				t.Errorf("got.Payer = %v, want %v", got.Payer, tt.wantPayer)
			}
		})
	}
}

func TestParseCellAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    float64
		wantErr bool
	}{
		{name: "Float input", input: 123.45, want: 123.45, wantErr: false},
		{name: "Integer input", input: 5000, want: 5000, wantErr: false},
		{name: "Int64 input", input: int64(99000), want: 99000, wantErr: false},
		{name: "Formatted string with COP", input: "$ 15.000 COP", want: 15000, wantErr: false},
		{name: "Comma decimal", input: "12,50", want: 12.5, wantErr: false},
		{name: "Nil value", input: nil, want: 0, wantErr: false},
		{name: "Empty string", input: "", want: 0, wantErr: false},
		{name: "Invalid string", input: "abc", want: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCellAmount(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCellAmount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseCellAmount() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "RFC3339", input: "2026-09-24T14:00:00-05:00", wantErr: false},
		{name: "Standard datetime", input: "2026-09-24 14:00:00", wantErr: false},
		{name: "Date only", input: "2026-09-24", wantErr: false},
		{name: "Colombian format slash", input: "24/09/2026 14:00:00", wantErr: false},
		{name: "Colombian date only", input: "24/09/2026", wantErr: false},
		{name: "Invalid date", input: "invalid-date", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got.IsZero() {
				t.Errorf("parseDate(%q) returned zero time", tt.input)
			}
		})
	}
}
