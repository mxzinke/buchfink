package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// --- Kopfdaten sind Pflicht beim Buchen ------------------------------------

// Die Ausgangsrechnung legt ihren Beleg mit Kopfdaten ab: Rechnungsdatum, eigene
// Firma, Bruttobetrag. Sie sind an der Rechnung bekannt, und ohne sie bliebe die
// Belegliste für Ausgangsrechnungen ohne Datum und Betrag.
func TestOutgoingInvoiceReceiptCarriesHeaderData(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung des Dokuments ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()
	env.withSellerContact(t)

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := env.invoicesWiredWithDocuments(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	if inv.ReceiptID == nil {
		t.Fatal("die ausgestellte Rechnung hat keinen Beleg")
	}
	receipt, err := env.receipts.Get(ctx, *inv.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}

	if receipt.DocumentDate != inv.Date {
		t.Errorf("Belegdatum %q, erwartet das Rechnungsdatum %q", receipt.DocumentDate, inv.Date)
	}
	if receipt.IssuerName != "Pfennig Ventures GmbH" {
		t.Errorf("Aussteller %q, erwartet die eigene Firma", receipt.IssuerName)
	}
	if receipt.GrossAmount != inv.GrossAmount {
		t.Errorf("Bruttobetrag %s, erwartet %s", receipt.GrossAmount, inv.GrossAmount)
	}
	if !strings.Contains(receipt.Subject, inv.InvoiceNumber) {
		t.Errorf("Betreff %q nennt die Rechnungsnummer nicht", receipt.Subject)
	}
	if receipt.DocumentDate == "" {
		t.Error("der Beleg hat kein Belegdatum")
	}
	if err := receipt.ValidateHeader(); err != nil {
		t.Errorf("der gebuchte Beleg muss die Kopfdatenprüfung bestehen: %v", err)
	}
}

// Der Eigenbeleg einer Abschlussbuchung hat Belegdatum, Betreff und Betrag.
func TestClosingSelfIssuedVoucherCarriesHeaderData(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt, err := selfIssuedVoucher(ctx, env.receipts, 2026, closingVoucher{
		Kind: "rueckstellung", FiscalYear: 2026, Date: "2026-12-31",
		Description: "Rückstellung für Abschlusskosten",
		Explanation: "Aufwand des Jahres, Rechnung folgt im Folgejahr",
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6825", Amount: 150000},
			{Position: 2, Side: domain.SideCredit, Account: "3070", Amount: 150000},
		},
	})
	if err != nil {
		t.Fatalf("Eigenbeleg: %v", err)
	}

	if receipt.DocumentDate != "2026-12-31" {
		t.Errorf("Belegdatum %q, erwartet das Buchungsdatum", receipt.DocumentDate)
	}
	if receipt.Subject != "Rückstellung für Abschlusskosten" {
		t.Errorf("Betreff %q, erwartet die Bezeichnung", receipt.Subject)
	}
	if receipt.GrossAmount != 150000 {
		t.Errorf("Betrag %s, erwartet 1500,00 € aus der Sollseite", receipt.GrossAmount)
	}
	if err := receipt.ValidateHeader(); err != nil {
		t.Errorf("der Eigenbeleg muss die Kopfdatenprüfung bestehen: %v", err)
	}
}

// Der Journaldienst ist der einzige Schreibweg: er weist eine Buchung ab, deren
// Beleg keine Kopfdaten hat — auf jedem Weg und nicht nur im Dialog.
func TestPostRejectsAnEntryWhoseReceiptHasNoHeader(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.journal.SetReceiptRepo(env.receiptRepo)

	// Ein Beleg ohne Kopfdaten — der Papierscan, den jemand abgelegt, aber noch
	// nicht erfasst hat.
	receipt := filePDFReceipt(t, env, FileReceiptRequest{})
	if receipt.DocumentDate != "" {
		t.Fatal("der Beleg des Falls darf keine Kopfdaten haben")
	}

	entry := &domain.JournalEntry{
		FiscalYear: 2026, BookingDate: "2026-03-01", DocumentDate: "2026-03-01",
		Description: "Wareneingang", Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
		Currency: "EUR", ExchangeRateMicros: 1_000_000,
		ReceiptID: &receipt.ID, ReceiptHash: receipt.ReceiptHash,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}
	if _, err := env.journal.Post(ctx, entry); err == nil {
		t.Fatal("eine Buchung auf einen Beleg ohne Kopfdaten muss abgewiesen werden")
	} else if !strings.Contains(err.Error(), "Belegdatum") {
		t.Errorf("die Meldung muss sagen, was fehlt: %v", err)
	}

	// Mit erfassten Kopfdaten geht dieselbe Buchung durch.
	if _, err := env.receipts.SaveHeader(ctx, receipt.ID, domain.ReceiptHeader{
		DocumentDate: "2026-03-01", IssuerName: "Lieferant GmbH",
		GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	}); err != nil {
		t.Fatalf("Kopfdaten erfassen: %v", err)
	}
	filled, _ := env.receipts.Get(ctx, receipt.ID)
	entry.ReceiptHash = filled.ReceiptHash
	if _, err := env.journal.Post(ctx, entry); err != nil {
		t.Fatalf("mit Kopfdaten muss die Buchung durchgehen: %v", err)
	}
}

func TestOriginalFileCannotBeRemoved(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := filePDFReceipt(t, env, FileReceiptRequest{DocumentDate: "2026-03-01"})
	withAttachment, err := env.receipts.AddFile(ctx, receipt.ID, NewFile{
		Role: domain.ReceiptRoleAttachment, FileName: "zahlungsnachweis.pdf",
		Content: []byte("%PDF-1.4 Nachweis\n"),
	})
	if err != nil {
		t.Fatalf("Anhang konnte nicht ergänzt werden: %v", err)
	}

	var originalID, attachmentID uint
	for _, f := range withAttachment.Files {
		switch f.Role {
		case domain.ReceiptRoleOriginal:
			originalID = f.ID
		case domain.ReceiptRoleAttachment:
			attachmentID = f.ID
		}
	}

	// GoBD Rz. 131: eingehende Dokumente sind in der empfangenen Form
	// aufzubewahren. Ein Beleg, dessen Original sich entfernen lässt, hält
	// nichts fest.
	if _, err := env.receipts.RemoveFile(ctx, withAttachment.ID, originalID); err == nil {
		t.Fatal("die Originaldatei darf sich nicht entfernen lassen")
	} else if !strings.Contains(err.Error(), "empfangenen Form") {
		t.Errorf("die Meldung muss den Grund nennen, bekommen: %v", err)
	}

	// Ein Anhang darf gehen — und Name und Prüfsumme stehen danach im
	// Protokoll.
	removed, err := env.receipts.RemoveFile(ctx, withAttachment.ID, attachmentID)
	if err != nil {
		t.Fatalf("der Anhang muss sich entfernen lassen: %v", err)
	}
	if len(removed.Files) != 1 {
		t.Errorf("nach dem Entfernen bleibt eine Datei, bekommen %d", len(removed.Files))
	}

	entries, _ := repository.NewAuditRepository(env.db).FindFiltered(ctx, 0,
		domain.AuditFilter{EntityType: "RECEIPT"})
	var found bool
	for _, entry := range entries {
		if strings.Contains(entry.Details, "zahlungsnachweis.pdf") && strings.Contains(entry.Details, "SHA256") {
			found = true
		}
	}
	if !found {
		t.Error("das Entfernen einer Datei muss Name und Prüfsumme protokollieren")
	}
}

func TestSaveHeaderRecomputesTheReceiptHash(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Ein Papierscan kommt ohne Kopfdaten herein: die Ablage darf daran nicht
	// scheitern (GoBD Rz. 131), das Buchen schon.
	receipt := filePDFReceipt(t, env, FileReceiptRequest{})
	if receipt.DocumentDate != "" {
		t.Fatal("ein Beleg ohne Kopfdaten darf keine haben")
	}
	if err := receipt.ValidateBookable(); err == nil {
		t.Error("ohne Kopfdaten darf nicht gebucht werden")
	}
	hashBefore := receipt.ReceiptHash

	updated, err := env.receipts.SaveHeader(ctx, receipt.ID, domain.ReceiptHeader{
		DocumentDate: "2026-03-01", IssuerName: "Lieferant GmbH",
		GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	})
	if err != nil {
		t.Fatalf("die Kopfdaten konnten nicht nachgetragen werden: %v", err)
	}
	if updated.ReceiptHash == hashBefore {
		t.Error("die Kopfdaten stehen im Beleg-Hash — er muss sich mit ihnen ändern")
	}
	if err := updated.ValidateBookable(); err != nil {
		t.Errorf("mit vollständigen Kopfdaten ist der Beleg buchbar: %v", err)
	}
	if updated.RetentionUntil == "" {
		t.Error("mit dem Belegdatum steht auch die Aufbewahrungsfrist fest")
	}
}
