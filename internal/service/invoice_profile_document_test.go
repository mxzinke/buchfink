package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Das Zielformat des Empfängers und der Beleg, der zur Rechnung gehört.

// Buchfink speichert ein unbekanntes E-Rechnungsformat nicht.
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

// Zu einer ausgestellten Rechnung entsteht kein zweites Dokument: es hätte
// dieselbe Nummer und wäre ein zweiter Beleg zu einem Vorgang.
func TestRegenerateDocumentRefusesASecondDocument(t *testing.T) {
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
	before, err := env.receipts.List(ctx, "")
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	if _, err := svc.RegenerateDocument(ctx, inv.ID); err == nil {
		t.Error("zu einer vollständigen Rechnung darf kein zweites Dokument entstehen")
	}
	after, err := env.receipts.List(ctx, "")
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	if len(after) != len(before) {
		t.Errorf("es liegen jetzt %d Belege statt %d", len(after), len(before))
	}
}
