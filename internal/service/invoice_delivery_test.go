package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Versandvermerk hält fest, wann und wie die Rechnung hinausgegangen ist.
func TestMarkInvoiceSent(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	if _, err := svc.MarkSent(ctx, inv.ID, "2026-03-02", "Brieftaube", ""); err == nil {
		t.Error("ein unbekannter Versandweg muss abgewiesen werden")
	}

	sent, err := svc.MarkSent(ctx, inv.ID, "2026-03-02", domain.InvoiceSentViaEmail, "an buchhaltung@kunde.example")
	if err != nil {
		t.Fatalf("Versand vermerken: %v", err)
	}
	if sent.SentAt != "2026-03-02" || sent.SentVia != domain.InvoiceSentViaEmail {
		t.Errorf("Versandvermerk = %q / %q", sent.SentAt, sent.SentVia)
	}
	stored, err := env.invoiceRepoOf(t).FindByID(ctx, inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.SentNote != "an buchhaltung@kunde.example" {
		t.Errorf("die Notiz zum Versand fehlt: %q", stored.SentNote)
	}
}
