package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// --- Beleg-Hash ------------------------------------------------------------

func sampleReceipt() *domain.Receipt {
	return &domain.Receipt{
		ReceiptNumber: "ER-2025-0001",
		FiscalYear:    2025,
		Direction:     domain.DirectionIncoming,
		Kind:          domain.ReceiptKindInvoice,
		Status:        domain.ReceiptStatusFiled,
		ReceivedAt:    "2025-03-01",
		ReceivedVia:   domain.ReceivedViaEmail,
		DocumentDate:  "2025-03-01",
		IssuerName:    "Lieferant GmbH",
		GrossAmount:   11900,
		TaxAmount:     1900,
		Currency:      "EUR",
		Subject:       "Bürobedarf",
		Files: []domain.ReceiptFile{{
			Position:   1,
			Role:       domain.ReceiptRoleOriginal,
			FileName:   "a.pdf",
			MimeType:   "application/pdf",
			SHA256:     strings.Repeat("aa", 32),
			Size:       3,
			StoredPath: "x/aa",
		}},
	}
}

func TestReceiptHashCoversHeader(t *testing.T) {
	hashed := ReceiptHash(sampleReceipt())

	changes := map[string]func(r *domain.Receipt){
		"Belegdatum": func(r *domain.Receipt) { r.DocumentDate = "" },
		"Betrag":     func(r *domain.Receipt) { r.GrossAmount = 11901 },
		"Steuer":     func(r *domain.Receipt) { r.TaxAmount = 1901 },
		"Aussteller": func(r *domain.Receipt) { r.IssuerName = "Jemand anderes GmbH" },
		"Betreff":    func(r *domain.Receipt) { r.Subject = "etwas ganz anderes" },
		"Belegart":   func(r *domain.Receipt) { r.Kind = domain.ReceiptKindLetter },
		"Datei":      func(r *domain.Receipt) { r.Files[0].SHA256 = strings.Repeat("bb", 32) },
	}
	for name, change := range changes {
		receipt := sampleReceipt()
		change(receipt)
		if ReceiptHash(receipt) == hashed {
			t.Errorf("%s muss im Beleg-Hash gedeckt sein", name)
		}
	}
}
