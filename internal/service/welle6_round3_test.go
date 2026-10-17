package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// --- Kopfdatenpflicht, Storno und Korrektur -------------------------------

// entryOnSealedReceiptWithoutHeader stellt eine Buchung auf einen versiegelten
// Beleg her, dessen Kopfdaten die heutige Prüfung nicht bestehen.
//
// So sieht ein Beleg aus, der unter einer Regel gebucht wurde, die eine spätere
// Programmfassung verschärft hat. Nachstellen lässt sich das, indem der Beleg
// über das Repository auf eine Vorbuchung versiegelt wird: die Kopfdatenpflicht
// gilt nur dem offenen Beleg.
func entryOnSealedReceiptWithoutHeader(
	t *testing.T, env *testEnv, fileName, description string,
) (*domain.JournalEntry, *domain.Receipt) {
	t.Helper()
	ctx := context.Background()

	receipt := filePDFReceipt(t, env, FileReceiptRequest{
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: fileName,
			Content: []byte("%PDF-1.4 " + fileName + "\n"),
		}},
	})
	if receipt.DocumentDate != "" {
		t.Fatal("der Beleg des Falls darf keine Kopfdaten haben")
	}

	before, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 100))
	if err != nil {
		t.Fatalf("Vorbuchung: %v", err)
	}
	if err := env.receiptRepo.Seal(ctx, receipt.ID, before.ID); err != nil {
		t.Fatalf("den Beleg versiegeln: %v", err)
	}
	entry, err := env.journal.Post(ctx, &domain.JournalEntry{
		FiscalYear: 2026, BookingDate: "2026-02-01", DocumentDate: "2026-02-01",
		Description: description, Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
		Currency: "EUR", ExchangeRateMicros: 1_000_000,
		ReceiptID: &receipt.ID, ReceiptHash: receipt.ReceiptHash,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	})
	if err != nil {
		t.Fatalf("die Buchung ließ sich nicht anlegen: %v", err)
	}
	return entry, receipt
}

// Eine Buchung auf einen versiegelten Beleg muss sich stornieren und
// korrigieren lassen, auch wenn seine Kopfdaten die heutige Prüfung nicht
// bestehen.
//
// Die Kopfdatenpflicht gilt dem Buchen eines offenen Belegs. Träfe sie auch die
// Generalumkehr — sie kopiert den Beleg der Ursprungsbuchung —, ließe sich die
// Buchung nicht zurücknehmen, und am versiegelten Beleg lassen sich die
// Kopfdaten nicht nachtragen. Genau den Weg sieht GoBD Rz. 58 für jede
// Korrektur vor.
func TestReversalAndCorrectionSurviveASealedReceiptWithoutHeader(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	toReverse, receipt := entryOnSealedReceiptWithoutHeader(t, env, "storno.pdf", "Buchung Storno")
	reversal, err := env.journal.Reverse(ctx, toReverse.ID, "falsches Konto")
	if err != nil {
		t.Fatalf("die Buchung muss sich stornieren lassen: %v", err)
	}
	if reversal.ReceiptID == nil || *reversal.ReceiptID != receipt.ID {
		t.Error("die Generalumkehr übernimmt den Beleg der Ursprungsbuchung")
	}

	// Und derselbe Beleg gilt auch für die Neubuchung des Korrekturvorgangs: sie
	// belegt denselben Geschäftsvorfall.
	toCorrect, sealed := entryOnSealedReceiptWithoutHeader(t, env, "korrektur.pdf", "Buchung Korrektur")
	result, err := env.journal.CorrectEntry(ctx, toCorrect.ID, "falscher Betrag", &domain.JournalEntry{
		FiscalYear: 2026, BookingDate: "2026-04-01", DocumentDate: "2026-02-01",
		Description: "Buchung richtig", Source: domain.EntrySourceManual,
		TaxTreatment: domain.TaxTreatmentNotTaxable,
		Currency:     "EUR", ExchangeRateMicros: 1_000_000,
		ReceiptID: &sealed.ID, ReceiptHash: sealed.ReceiptHash,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 12000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 12000},
		},
	})
	if err != nil {
		t.Fatalf("die Korrektur muss durchlaufen: %v", err)
	}
	if result.Replacement.CorrectsEntryID == nil || *result.Replacement.CorrectsEntryID != toCorrect.ID {
		t.Error("die Neubuchung muss auf die ersetzte Buchung verweisen")
	}
}

// Scheitert die Neubuchung an den Kopfdaten ihres Belegs, darf kein Storno
// entstehen.
//
// Die Reihenfolge „erst prüfen, dann stornieren, dann neu buchen" ist nur so
// viel wert wie die Prüfung: fehlte ihr eine Regel, die Post kennt, bliebe der
// Vorgang halb ausgeführt zurück — die alte Buchung zurückgenommen, die
// richtige nicht erfasst.
func TestCorrectEntryDoesNotReverseWhenTheNewReceiptHasNoHeader(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	original, err := env.journal.Post(ctx, &domain.JournalEntry{
		FiscalYear: 2026, BookingDate: "2026-02-01", DocumentDate: "2026-02-01",
		Description: "Bürobedarf", Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
		Currency: "EUR", ExchangeRateMicros: 1_000_000,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	})
	if err != nil {
		t.Fatalf("die Ursprungsbuchung: %v", err)
	}

	// Der Papierscan, den jemand abgelegt, aber noch nicht erfasst hat: offen
	// und ohne Kopfdaten.
	receipt := filePDFReceipt(t, env, FileReceiptRequest{})

	_, err = env.journal.CorrectEntry(ctx, original.ID, "falsches Konto", &domain.JournalEntry{
		FiscalYear: 2026, BookingDate: "2026-04-01", DocumentDate: "2026-02-01",
		Description: "Bürobedarf richtig", Source: domain.EntrySourceManual,
		TaxTreatment: domain.TaxTreatmentNotTaxable,
		Currency:     "EUR", ExchangeRateMicros: 1_000_000,
		ReceiptID: &receipt.ID, ReceiptHash: receipt.ReceiptHash,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6820", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	})
	if err == nil {
		t.Fatal("eine Neubuchung auf einen Beleg ohne Kopfdaten darf die Korrektur nicht auslösen")
	}
	if !strings.Contains(err.Error(), "deshalb wurde nicht storniert") {
		t.Errorf("die Meldung muss sagen, dass nichts storniert wurde: %v", err)
	}
	if !strings.Contains(err.Error(), "Belegdatum") {
		t.Errorf("die Meldung muss den Grund nennen: %v", err)
	}

	existing, err := env.journalRepo.FindReversalOf(ctx, original.ID)
	if err != nil {
		t.Fatalf("Stornos zur Ursprungsbuchung lesen: %v", err)
	}
	if existing != nil {
		t.Errorf("die Ursprungsbuchung wurde storniert (%s), obwohl die Neubuchung nicht buchbar war",
			existing.EntryNumber)
	}
}

// --- Eröffnungsbilanz: Beleg-Hash und Siegel -------------------------------

// Die Schlussbilanz des Altsystems ist nach der Eröffnungsbilanz ein gebuchter
// Beleg.
//
// Zweierlei folgt daraus: ihr Hash steht in jeder Eröffnungsbuchung — sonst
// verwiese ausgerechnet die Buchung, deren Werte aus einem fremden System
// stammen, nur über eine Nummer auf ein austauschbares Dokument —, und sie ist
// versiegelt, sodass ihre Kopfdaten feststehen und der Prüflauf sie nicht
// dauerhaft als „abgelegt, aber nicht gebucht" meldet (das wäre eine Sperre vor
// jeder Festschreibung des ersten Jahres).
func TestOpeningBalanceSealsTheClosingReceiptAndCarriesItsHash(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	closing := filePDFReceipt(t, env, FileReceiptRequest{
		Kind: domain.ReceiptKindOther, DocumentDate: "2025-12-31",
		ReceivedAt: "2026-01-02",
		Subject:    "Schlussbilanz zum 31.12.2025 aus dem Vorgängerprogramm",
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "schlussbilanz-altsystem.pdf",
			Content: []byte("%PDF-1.4 Schlussbilanz\n"),
		}},
	})

	req := OpeningBalanceRequest{
		FiscalYear: 2026, Date: "2026-01-01", ReceiptID: &closing.ID,
		LegacySystem: "Vorgängerprogramm",
		Accounts: []OpeningBalanceLine{
			{Account: "1800", Side: domain.SideDebit, Amount: 500000, LegacyRef: "ALT-1800"},
			{Account: "2900", Side: domain.SideCredit, Amount: 619000, LegacyRef: "ALT-2900"},
		},
		Receivables: []OpenOpeningItem{{
			ContactID: customer.ID, Amount: 119000,
			DocumentNumber: "RE-2025-0099", DocumentDate: "2025-12-01",
			DueDate: "2026-01-15", LegacyRef: "ALT-RE-99",
		}},
	}

	// Die Vorschau zeigt, was gebucht wird — den Beleg-Hash eingeschlossen.
	preview, err := env.journal.PreviewOpeningBalance(ctx, req)
	if err != nil {
		t.Fatalf("die Vorschau ist fehlgeschlagen: %v", err)
	}
	for _, entry := range preview.Entries {
		if entry.ReceiptHash != closing.ReceiptHash {
			t.Errorf("Vorschau: Buchung %q hat den Beleg-Hash %q, erwartet %q",
				entry.Description, entry.ReceiptHash, closing.ReceiptHash)
		}
	}

	booked, err := env.journal.BookOpeningBalance(ctx, req)
	if err != nil {
		t.Fatalf("die Eröffnungsbilanz ließ sich nicht buchen: %v", err)
	}
	if len(booked.Entries) == 0 {
		t.Fatal("die Eröffnungsbilanz hat keine Buchung erzeugt")
	}
	for _, entry := range booked.Entries {
		if entry.ReceiptHash != closing.ReceiptHash {
			t.Errorf("Buchung %s hat den Beleg-Hash %q, erwartet %q",
				entry.EntryNumber, entry.ReceiptHash, closing.ReceiptHash)
		}
	}

	sealed, err := env.receiptRepo.FindByID(ctx, closing.ID)
	if err != nil {
		t.Fatalf("den Beleg lesen: %v", err)
	}
	if sealed.Status != domain.ReceiptStatusSealed {
		t.Errorf("der Beleg der Schlussbilanz steht auf %q, erwartet %q",
			sealed.Status, domain.ReceiptStatusSealed)
	}

	// Der Prüflauf darf ihn nach der Buchung nicht mehr als ungebucht führen.
	run := runChecks(t, env.checksOn(t, "2026-04-01"), "2026-03-31")
	for _, finding := range findingsFor(run, domain.CheckRuleReceiptUnbooked) {
		if finding.ObjectName == closing.ReceiptNumber {
			t.Errorf("der gebuchte Beleg der Eröffnungsbilanz wird als ungebucht gemeldet: %s (%s)",
				finding.Message, finding.Severity)
		}
	}
}
