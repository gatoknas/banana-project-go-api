package email

import (
	"strings"
	"testing"
	"time"
)

const sampleHTML = `
<html><body>
<div class="txtTitulo">Venta exitosa por $ 2.000</div>
<h3>Detalle de la venta</h3>
<table class="detalleVenta">
<tr><th>Monto:</th><td>$ 2.000</td></tr>
<tr><th>Estado:</th><td>Aprobada</td></tr>
<tr><th>Fecha:</th><td>21/09/2026 07:57:23</td></tr>
<tr><th>Pagador:</th><td>ANDRES FELIPE PANIAGUA</td></tr>
<tr><th>Banco:</th><td>Bancolombia</td></tr>
<tr><th>Referencia:</th><td>M02132923</td></tr>
<tr><th>N&uacute;mero de transacci&oacute;n:</th><td>4ae6a43d41404b4bb00033e10ce40b42f23</td></tr>
<tr><th>M&eacute;todo de pago:</th><td>QR Negocios Bre-B</td></tr>
</table>
</body></html>`

func TestParseReceipt(t *testing.T) {
	receipt, err := ParseReceipt(sampleHTML)
	if err != nil {
		t.Fatalf("ParseReceipt returned error: %v", err)
	}

	if receipt.Amount != 2000 {
		t.Errorf("Amount = %v, want 2000", receipt.Amount)
	}
	if receipt.Currency != "COP" {
		t.Errorf("Currency = %q, want COP", receipt.Currency)
	}
	if receipt.Status != "Aprobada" {
		t.Errorf("Status = %q, want Aprobada", receipt.Status)
	}
	if receipt.Payer != "ANDRES FELIPE PANIAGUA" {
		t.Errorf("Payer = %q", receipt.Payer)
	}
	if receipt.Bank != "Bancolombia" {
		t.Errorf("Bank = %q", receipt.Bank)
	}
	if receipt.Reference != "M02132923" {
		t.Errorf("Reference = %q", receipt.Reference)
	}
	if receipt.TransactionNumber != "4ae6a43d41404b4bb00033e10ce40b42f23" {
		t.Errorf("TransactionNumber = %q", receipt.TransactionNumber)
	}
	if receipt.PaymentMethod != "QR Negocios Bre-B" {
		t.Errorf("PaymentMethod = %q", receipt.PaymentMethod)
	}

	wantDate := time.Date(2026, 9, 21, 7, 57, 23, 0, Colombia)
	if !receipt.TransactionDate.Equal(wantDate) {
		t.Errorf("TransactionDate = %v, want %v", receipt.TransactionDate, wantDate)
	}
}

func TestParseReceiptNoFields(t *testing.T) {
	if _, err := ParseReceipt("<html><body>nothing here</body></html>"); err == nil {
		t.Fatal("expected error for body without receipt fields")
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"$ 2.000", 2000},
		{"$ 12.345.678", 12345678},
		{"$ 2.000,50", 2000.50},
		{"2000.50", 2000.50},
		{"$ 500", 500},
	}
	for _, tt := range tests {
		got, err := parseAmount(tt.in)
		if err != nil {
			t.Errorf("parseAmount(%q) error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseAmount(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseDate(t *testing.T) {
	got, err := parseDate("21/09/2026 07:57:23")
	if err != nil {
		t.Fatalf("parseDate error: %v", err)
	}
	want := time.Date(2026, 9, 21, 7, 57, 23, 0, Colombia)
	if !got.Equal(want) {
		t.Errorf("parseDate = %v, want %v", got, want)
	}
}

func TestParseMessageQuotedPrintable(t *testing.T) {
	raw := strings.Join([]string{
		"From: notificaciones@nequi.com.co",
		"To: martashenao@gmail.com",
		"Subject: Detalle de tu venta por Bre-B",
		"Date: Mon, 21 Sep 2026 12:57:24 +0000",
		"MIME-Version: 1.0",
		`Content-Type: multipart/alternative; boundary="BOUND"`,
		"",
		"--BOUND",
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: quoted-printable",
		"",
		"<html><body><th>M=C3=A9todo de pago:</th><td>QR Negocios Bre-B</td>" +
			"<th>Monto:</th><td>$ 2.000</td></body></html>",
		"--BOUND--",
		"",
	}, "\r\n")

	msg, err := ParseMessage([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMessage error: %v", err)
	}

	if msg.Sender != "notificaciones@nequi.com.co" {
		t.Errorf("Sender = %q", msg.Sender)
	}
	if msg.Subject != "Detalle de tu venta por Bre-B" {
		t.Errorf("Subject = %q", msg.Subject)
	}
	if !strings.Contains(msg.Body, "Método de pago:") {
		t.Errorf("quoted-printable body was not decoded: %q", msg.Body)
	}
	if msg.Date.IsZero() {
		t.Error("Date was not parsed")
	}

	receipt, err := ParseReceipt(msg.Body)
	if err != nil {
		t.Fatalf("ParseReceipt on decoded body error: %v", err)
	}
	if receipt.Amount != 2000 || receipt.PaymentMethod != "QR Negocios Bre-B" {
		t.Errorf("unexpected receipt: %+v", receipt)
	}
}
