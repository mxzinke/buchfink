package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// --- Kopfdatenpflicht und Altbestand ---------------------------------------

// legacyEntryOnHeaderlessReceipt stellt den Altbestand her: eine Buchung aus
// der Zeit vor Welle 6 auf einen Beleg ohne Kopfdaten, der mit ihr versiegelt
// wurde.
//
// Der Weg dorthin geht bewusst an der Kopfdatenprüfung vorbei — genau so sind
// diese Buchungen entstanden, nämlich mit einem Programmstand, der die Regel
// noch nicht kannte. Nachstellen lässt sich das nur, indem die Belegablage für
// die eine Buchung abgehängt und der Beleg über das Repository versiegelt wird
// (ReceiptService.Seal prüft die Kopfdaten und hätte damals nichts geprüft).
func legacyEntryOnHeaderlessReceipt(
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
	if receipt.HasHeader() {
		t.Fatal("der Altbeleg des Falls darf keine Kopfdaten tragen")
	}

	env.journal.SetReceiptRepo(nil)
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
	env.journal.SetReceiptRepo(env.receiptRepo)
	if err != nil {
		t.Fatalf("die Altbuchung ließ sich nicht anlegen: %v", err)
	}
	if err := env.receiptRepo.Seal(ctx, receipt.ID, entry.ID); err != nil {
		t.Fatalf("den Altbeleg versiegeln: %v", err)
	}
	return entry, receipt
}

// Eine Altbuchung auf einen versiegelten Beleg ohne Kopfdaten muss sich
// stornieren und korrigieren lassen.
//
// Die Kopfdatenpflicht gilt dem Buchen eines offenen Belegs. Träfe sie auch die
// Generalumkehr — sie kopiert den Beleg der Ursprungsbuchung —, wäre der
// gesamte Altbestand eingesperrt: stornieren ginge nicht, und die Kopfdaten
// ließen sich am versiegelten Beleg auch nicht nachtragen, weil SaveHeader den
// offenen Beleg verlangt. Genau den Weg sieht GoBD Rz. 58 für jede Korrektur
// vor.
func TestReversalAndCorrectionSurviveALegacyReceiptWithoutHeader(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	toReverse, receipt := legacyEntryOnHeaderlessReceipt(t, env, "alt-storno.pdf", "Altbuchung Storno")
	reversal, err := env.journal.Reverse(ctx, toReverse.ID, "falsches Konto")
	if err != nil {
		t.Fatalf("die Altbuchung muss sich stornieren lassen: %v", err)
	}
	if reversal.ReceiptID == nil || *reversal.ReceiptID != receipt.ID {
		t.Error("die Generalumkehr trägt den Beleg der Ursprungsbuchung")
	}

	// Und derselbe Beleg trägt auch die Neubuchung des Korrekturvorgangs: sie
	// belegt denselben Geschäftsvorfall.
	toCorrect, legacyReceipt := legacyEntryOnHeaderlessReceipt(t, env, "alt-korrektur.pdf", "Altbuchung Korrektur")
	result, err := env.journal.CorrectEntry(ctx, toCorrect.ID, "falscher Betrag", &domain.JournalEntry{
		FiscalYear: 2026, BookingDate: "2026-04-01", DocumentDate: "2026-02-01",
		Description: "Altbuchung richtig", Source: domain.EntrySourceManual,
		TaxTreatment: domain.TaxTreatmentNotTaxable,
		Currency:     "EUR", ExchangeRateMicros: 1_000_000,
		ReceiptID: &legacyReceipt.ID, ReceiptHash: legacyReceipt.ReceiptHash,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 12000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 12000},
		},
	})
	if err != nil {
		t.Fatalf("die Korrektur einer Altbuchung muss durchlaufen: %v", err)
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
// Zweierlei hängt daran: ihr Hash steht in jeder Eröffnungsbuchung — sonst
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
			t.Errorf("Vorschau: Buchung %q trägt den Beleg-Hash %q, erwartet %q",
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
			t.Errorf("Buchung %s trägt den Beleg-Hash %q, erwartet %q",
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

// --- Anlagendokumente: Aufbewahrungsfrist ----------------------------------

// Ein Anlagendokument ist eine Organisationsunterlage und wird zehn Jahre
// aufbewahrt.
//
// Der Kaufvertrag und die Rechnungskopie tragen die Bemessungsgrundlage der
// Abschreibung, und die wirkt über die ganze Nutzungsdauer fort — die verkürzte
// Belegfrist von acht Jahren passt für sie nicht. Ohne gespeicherte Klasse
// stünde in der Kartei nichts, und der Bericht über abgelaufene Objekte sähe
// die Dokumente überhaupt nicht.
func TestAssetDocumentGetsTheTenYearRetention(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)
	svc.SetDocumentStore(env.store)
	asset := env.machine(t, svc)

	updated, err := svc.AttachDocument(ctx, AttachDocumentRequest{
		AssetID: asset.ID, Kind: domain.AssetDocContract,
		Content: []byte("%PDF-1.4 Kaufvertrag\n"), FileName: "kaufvertrag.pdf",
		Title: "Kaufvertrag Fräsmaschine", DocumentDate: "2026-01-10",
	})
	if err != nil {
		t.Fatalf("das Dokument ließ sich nicht ablegen: %v", err)
	}
	if len(updated.Documents) != 1 {
		t.Fatalf("%d Dokumente am Anlagegut, erwartet 1", len(updated.Documents))
	}
	document := updated.Documents[0]
	if document.RetentionClass != domain.RetentionClassBooks {
		t.Errorf("Klasse %q, erwartet %q (Organisationsunterlage)",
			document.RetentionClass, domain.RetentionClassBooks)
	}
	if document.RetentionUntil != "2036-12-31" {
		t.Errorf("Fristende %q, erwartet 2036-12-31 (zehn Jahre ab Schluss 2026)",
			document.RetentionUntil)
	}

	// Ohne Dokumentdatum zählt das Jahr der Ablage; die Frist bleibt dieselbe
	// Klasse.
	withoutDate, err := svc.AttachDocument(ctx, AttachDocumentRequest{
		AssetID: asset.ID, Kind: domain.AssetDocPhoto,
		Content: []byte("BILD"), FileName: "maschine.png",
	})
	if err != nil {
		t.Fatalf("das zweite Dokument ließ sich nicht ablegen: %v", err)
	}
	for _, d := range withoutDate.Documents {
		if d.RetentionClass != domain.RetentionClassBooks || d.RetentionUntil == "" {
			t.Errorf("Dokument %q ohne Frist: Klasse %q, bis %q",
				d.DisplayTitle(), d.RetentionClass, d.RetentionUntil)
		}
	}
}
