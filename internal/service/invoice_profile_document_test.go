package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Das Zielformat des Empfängers und der Beleg, der zur Rechnung gehört.

// Ein unbekanntes E-Rechnungsformat wird nicht gespeichert.
//
// Vorher wurde nur das leere Feld belegt. Ein Format aus der Fachplanung, das
// Buchfink nicht erzeugt („xrechnung_ubl"), stand danach am Kontakt und wirkte
// beim Ausstellen still wie ZUGFeRD: der Empfänger bekäme ein anderes Dokument,
// als an ihm hinterlegt ist — und niemand sähe es.
func TestSaveContactRejectsUnknownEInvoiceProfile(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	contact := &domain.Contact{
		Type: domain.ContactTypeCustomer, Name: "Landeshauptstadt München",
		Street: "Marienplatz 8", PostalCode: "80331", City: "München", CountryCode: "DE",
		EInvoiceProfile: domain.EInvoiceProfile("xrechnung_ubl"),
	}
	err := env.contacts.SaveContact(ctx, contact)
	if err == nil {
		t.Fatal("ein unbekanntes E-Rechnungsformat wurde gespeichert")
	}
	if !strings.Contains(err.Error(), "xrechnung_ubl") {
		t.Errorf("die Meldung nennt das abgewiesene Format nicht: %v", err)
	}

	// Die bekannten Formate gehen weiterhin durch, und das leere wird belegt.
	contact.EInvoiceProfile = domain.EInvoiceProfileXRechnungCII
	if err := env.contacts.SaveContact(ctx, contact); err != nil {
		t.Fatalf("bekanntes Format speichern: %v", err)
	}
	plain := &domain.Contact{
		Type: domain.ContactTypeCustomer, Name: "Kunde GmbH",
		Street: "Hauptstraße 1", PostalCode: "80331", City: "München", CountryCode: "DE",
	}
	if err := env.contacts.SaveContact(ctx, plain); err != nil {
		t.Fatalf("Kontakt ohne Format speichern: %v", err)
	}
	if plain.EInvoiceProfile != domain.EInvoiceProfileZUGFeRD {
		t.Errorf("ohne Angabe gilt der Regelfall, hier steht %q", plain.EInvoiceProfile)
	}
}

// Eine Rechnung mit einem Format, das es nicht gibt, wird nicht ausgestellt —
// und kostet keine Nummer.
func TestIssueRejectsUnknownEInvoiceProfile(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	inv.EInvoiceProfile = domain.EInvoiceProfile("xrechnung_ubl")
	if err := env.invoicesWired(t).Issue(ctx, inv); err == nil {
		t.Fatal("die Rechnung wurde in einem Format ausgestellt, das Buchfink nicht erzeugt")
	}
	next, err := env.numberRepo.Peek(ctx, domain.NumberRangeInvoice, 2026)
	if err != nil {
		t.Fatalf("Nummernkreis lesen: %v", err)
	}
	if next != 1 {
		t.Errorf("der Zähler steht auf %d, erwartet 1", next)
	}
}

// Liegt der Beleg schon, entsteht kein zweiter.
//
// Der Verweis auf den Beleg wird gespeichert, sobald er abgelegt ist — vor
// allem, was danach noch scheitern kann. Bleibt die Rechnung trotzdem auf
// „Dokument fehlt" stehen, holt „Dokument erneut erzeugen" den letzten Schritt
// nach, statt einen zweiten Beleg unter derselben Rechnungsnummer anzulegen.
func TestRegenerateDocumentFinishesAnAlreadyFiledReceipt(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWiredWithDocuments(t)

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	if inv.ReceiptID == nil {
		t.Fatal("die ausgestellte Rechnung muss auf ihren Beleg zeigen")
	}
	before, err := env.receipts.List(ctx, "")
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}

	// Der Zustand nach einem Fehler hinter dem Ablegen: Beleg da, Rechnung
	// weiter auf „Dokument fehlt".
	inv.Status = domain.InvoiceStatusPendingDocument
	if err := env.invoiceRepoOf(t).Save(ctx, inv); err != nil {
		t.Fatalf("Zustand setzen: %v", err)
	}

	finished, err := svc.RegenerateDocument(ctx, inv.ID)
	if err != nil {
		t.Fatalf("Dokument nachholen: %v", err)
	}
	if finished.Status != domain.InvoiceStatusIssued {
		t.Errorf("die Rechnung steht auf %q, erwartet %q", finished.Status, domain.InvoiceStatusIssued)
	}
	if finished.ReceiptID == nil || *finished.ReceiptID != *inv.ReceiptID {
		t.Error("die Rechnung muss auf denselben Beleg zeigen wie vorher")
	}
	after, err := env.receipts.List(ctx, "")
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	if len(after) != len(before) {
		t.Errorf("es liegen jetzt %d Belege statt %d — ein zweiter trüge dieselbe Rechnungsnummer",
			len(after), len(before))
	}

	// Eine fertige Rechnung bekommt weiterhin kein zweites Dokument.
	if _, err := svc.RegenerateDocument(ctx, inv.ID); err == nil {
		t.Error("zu einer vollständigen Rechnung darf kein zweites Dokument entstehen")
	}
}
