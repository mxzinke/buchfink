package service

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// --- Kopfdaten vor dem Schreiben, nicht erst beim Versiegeln ---------------

// Die Inventurliste bekommt die Buchung erst hinter dem Journal-Commit
// (Seal). Fehlen ihre Kopfdaten, muss der Vorgang vorher scheitern — sonst
// stünde die Bestandsveränderung im Journal, während die Liste unverbunden im
// Belegspeicher liegt und der Prüflauf sie dauerhaft als ungebucht meldet.
func TestInventorySheetWithoutHeaderLeavesNoBooking(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	stock := &domain.JournalEntry{
		BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
		ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
		Description: "Wareneinkauf auf Bestand", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "1140", Amount: 500_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 500_000},
		},
	}
	if _, err := env.journal.Post(ctx, stock); err != nil {
		t.Fatalf("Bestandsbuchung: %v", err)
	}
	before, _ := env.journalRepo.FindAll(ctx, 2026)

	// Eine Inventurliste ohne Betreff: Belegdatum ja, sonst nichts. Aussteller
	// und Betrag hat sie nicht — sie ist keine Rechnung.
	sheet := filePDFReceipt(t, env, FileReceiptRequest{
		Kind: domain.ReceiptKindOther, DocumentDate: "2026-12-31",
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "inventurliste.pdf",
			Content: []byte("%PDF-1.4 Inventurliste\n"),
		}},
	})

	_, err := m.bookings.BookInventory(ctx, InventoryRequest{
		FiscalYear: 2026, Account: "1140", Amount: 400_000,
		CountedOn: "2026-12-31", Method: "Stichtagsinventur", ReceiptID: sheet.ID,
	})
	if err == nil {
		t.Fatal("eine Inventurliste ohne Kopfdaten darf die Buchung nicht tragen")
	}
	if !strings.Contains(err.Error(), "Inventurliste") {
		t.Errorf("die Meldung muss sagen, welches Dokument fehlt: %v", err)
	}

	after, _ := env.journalRepo.FindAll(ctx, 2026)
	if len(after) != len(before) {
		t.Errorf("der abgewiesene Vorgang hat %d Buchung(en) hinterlassen",
			len(after)-len(before))
	}

	// Mit Betreff geht derselbe Vorgang durch, und die Liste ist danach
	// versiegelt.
	sheet, err = env.receipts.SaveHeader(ctx, sheet.ID, domain.ReceiptHeader{
		Kind: domain.ReceiptKindOther, DocumentDate: "2026-12-31",
		Subject: "Inventurliste Warenbestand zum 31.12.2026",
	})
	if err != nil {
		t.Fatalf("Kopfdaten nachtragen: %v", err)
	}
	count, err := m.bookings.BookInventory(ctx, InventoryRequest{
		FiscalYear: 2026, Account: "1140", Amount: 400_000,
		CountedOn: "2026-12-31", Method: "Stichtagsinventur", ReceiptID: sheet.ID,
	})
	if err != nil {
		t.Fatalf("mit Belegdatum und Betreff muss die Inventur buchbar sein: %v", err)
	}
	if count.JournalEntryID == nil {
		t.Fatal("die Bestandsveränderung wurde nicht gebucht")
	}
	sealed, err := env.receipts.Get(ctx, sheet.ID)
	if err != nil {
		t.Fatalf("Inventurliste lesen: %v", err)
	}
	if sealed.Status != domain.ReceiptStatusSealed {
		t.Errorf("die Inventurliste steht nach der Buchung im Status %q", sealed.Status)
	}
}

// Dasselbe für das Beschlussdokument der Ergebnisverwendung.
func TestAppropriationDecisionWithoutHeaderLeavesNoBooking(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	opening := &domain.JournalEntry{
		BookingDate: "2027-01-01", DocumentDate: "2027-01-01",
		ServiceDateFrom: "2027-01-01", ServiceDateTo: "2027-01-01",
		Description: "Saldenvortrag", Source: domain.EntrySourceOpening,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 400_000},
			{Side: domain.SideCredit, Account: domain.AccountGewinnvortrag, Amount: 400_000},
		},
	}
	if _, err := env.journal.Post(ctx, opening); err != nil {
		t.Fatalf("Vortrag buchen: %v", err)
	}
	before, _ := env.journalRepo.FindAll(ctx, 2027)

	decision := filePDFReceipt(t, env, FileReceiptRequest{
		FiscalYear: 2027, Kind: domain.ReceiptKindOther, DocumentDate: "2027-05-20",
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "beschluss.pdf",
			Content: []byte("%PDF-1.4 Beschluss\n"),
		}},
	})

	_, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", Distribution: 100_000, Text: "Beschluss",
		ReceiptID: decision.ID,
	})
	if err == nil {
		t.Fatal("ein Beschlussdokument ohne Kopfdaten darf die Buchung nicht tragen")
	}
	if !strings.Contains(err.Error(), "Beschlussdokument") {
		t.Errorf("die Meldung muss sagen, welches Dokument fehlt: %v", err)
	}

	after, _ := env.journalRepo.FindAll(ctx, 2027)
	if len(after) != len(before) {
		t.Errorf("der abgewiesene Beschluss hat %d Buchung(en) hinterlassen",
			len(after)-len(before))
	}
}

// --- Kopfdaten aus der E-Rechnung auf dem Weg, den die Anwendung geht ------

// Geprüft wird ExtractStructuredPart und nicht prefillHeader: das ist der Weg,
// den ein eingehender Beleg in der Anwendung nimmt. Ein Test auf der privaten
// Hilfsfunktion beweist nur, dass die Hilfsfunktion tut, was sie tut — nicht,
// dass sie auf diesem Weg überhaupt aufgerufen wird.
func TestExtractStructuredPartFillsTheReceiptHeader(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt, err := env.receipts.File(ctx, FileReceiptRequest{
		Direction: domain.DirectionIncoming, FiscalYear: env.fiscalYear,
		ReceivedAt: "2026-05-05", ReceivedVia: domain.ReceivedViaEmail,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: "rechnung.xml", Content: []byte("<Invoice/>")},
		},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}
	if receipt.HasHeader() {
		t.Fatal("der Beleg soll ohne Kopfdaten hereinkommen — sonst prüft der Test nichts")
	}

	updated, err := env.einvoicesWith(fakeReader{invoice: receivedInvoice()}).
		ExtractStructuredPart(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("strukturierten Teil holen: %v", err)
	}

	if updated.DocumentDate != "2026-05-04" {
		t.Errorf("Belegdatum %q, erwartet das Rechnungsdatum aus dem Datensatz", updated.DocumentDate)
	}
	if updated.IssuerName != "Netzwerk GmbH" {
		t.Errorf("Aussteller %q, erwartet den Lieferanten aus dem Datensatz", updated.IssuerName)
	}
	if updated.GrossAmount != 119000 || updated.TaxAmount != 19000 {
		t.Errorf("Beträge %s / %s, erwartet 1.190,00 / 190,00", updated.GrossAmount, updated.TaxAmount)
	}
	if updated.Subject != "LR-2026-0815" {
		t.Errorf("Betreff %q, erwartet die Rechnungsnummer", updated.Subject)
	}
	if err := updated.ValidateHeader(); err != nil {
		t.Errorf("nach dem Lesen ist der Beleg buchbar: %v", err)
	}
}

// --- Verfahrensdokumentation als PDF --------------------------------------

// Entscheidung 7 verlangt beide Formen: Markdown und PDF über Typst. Geprüft
// wird mit dem echten Setzer und nicht mit einer Attrappe — eine Attrappe
// bewiese nur, dass eine Zeichenkette weitergereicht wird, während die Frage
// ist, ob sich der übersetzte Text überhaupt setzen lässt.
func TestProcedureDocumentationIsAlsoTypesetAsPDF(t *testing.T) {
	env := newTestEnv(t)
	svc := newProcDocService(t, env)
	svc.SetRenderer(sharedRenderer())
	ctx := context.Background()

	result, err := svc.Generate(ctx, time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("die Verfahrensdokumentation ließ sich nicht erzeugen: %v", err)
	}
	if result.PDFNote != "" {
		t.Fatalf("der Satz ist gescheitert: %s", result.PDFNote)
	}
	doc := result.Document
	if doc.PDFStoredPath == "" || doc.PDFSHA256 == "" || doc.PDFSize == 0 {
		t.Fatalf("das PDF ist nicht abgelegt: %+v", doc)
	}
	if !strings.HasSuffix(doc.PDFFileName, ".pdf") ||
		strings.TrimSuffix(doc.FileName, ".md") != strings.TrimSuffix(doc.PDFFileName, ".pdf") {
		t.Errorf("die beiden Fassungen heißen verschieden: %q / %q", doc.FileName, doc.PDFFileName)
	}

	// Beide Formen liegen nebeneinander im Belegspeicher und sind über ihre
	// Prüfsumme wiederzufinden.
	content, err := os.ReadFile(filepath.Join(env.dataDir, doc.PDFStoredPath))
	if err != nil {
		t.Fatalf("das abgelegte PDF ließ sich nicht lesen: %v", err)
	}
	if !bytes.HasPrefix(content, []byte("%PDF")) {
		t.Errorf("die abgelegte Datei ist kein PDF: % x", content[:8])
	}
	if int64(len(content)) != doc.PDFSize {
		t.Errorf("Größe %d, festgehalten %d", len(content), doc.PDFSize)
	}

	stored, err := svc.Documentations(ctx)
	if err != nil || len(stored) != 1 {
		t.Fatalf("die Fassung wurde nicht festgehalten: %v (%d)", err, len(stored))
	}
	if stored[0].PDFStoredPath != doc.PDFStoredPath {
		t.Errorf("der Verweis auf das PDF fehlt in der Historie: %+v", stored[0])
	}
}

// Ohne Setzer bleibt es beim Markdown — die Fassung gilt trotzdem, und der
// Grund steht im Ergebnis. Eine Verfahrensdokumentation an einer Formfrage
// scheitern zu lassen wäre die schlechtere Antwort.
func TestProcedureDocumentationSurvivesAFailedTypesetting(t *testing.T) {
	env := newTestEnv(t)
	svc := newProcDocService(t, env)
	svc.SetRenderer(failingRenderer{})
	ctx := context.Background()

	result, err := svc.Generate(ctx, time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ein Satzfehler darf die Fassung nicht verhindern: %v", err)
	}
	if result.Document.SHA256 == "" {
		t.Error("das Markdown muss trotzdem abgelegt sein")
	}
	if result.PDFNote == "" {
		t.Error("der Grund für das fehlende PDF muss im Ergebnis stehen")
	}
	if result.Document.PDFStoredPath != "" {
		t.Error("ohne Satz gibt es keinen PDF-Pfad")
	}
}

type failingRenderer struct{}

func (failingRenderer) RenderDocumentPDF(ctx context.Context, template, ident string) ([]byte, error) {
	return nil, fmt.Errorf("der Setzer steht nicht zur Verfügung")
}

// --- Eröffnungsbilanz: Fälligkeit und Einmaligkeit ------------------------

// openingRequest ist die Eröffnungsbilanz eines Umsteigers mit einer Forderung,
// deren Fälligkeit im Altsystem vereinbart wurde.
func openingRequest(t *testing.T, env *testEnv, customerID uint, receiptID uint) OpeningBalanceRequest {
	t.Helper()
	return OpeningBalanceRequest{
		FiscalYear: 2026, Date: "2026-01-01", ReceiptID: &receiptID,
		LegacySystem: "Vorgängerprogramm",
		Accounts: []OpeningBalanceLine{
			{Account: "1800", Side: domain.SideDebit, Amount: 500000, LegacyRef: "ALT-1800"},
			{Account: "2000", Side: domain.SideCredit, Amount: 500000, LegacyRef: "ALT-2000"},
		},
		Receivables: []OpenOpeningItem{{
			ContactID: customerID, Amount: 119000,
			DocumentNumber: "RE-2025-0099", DocumentDate: "2025-12-01",
			DueDate: "2026-06-30", LegacyRef: "ALT-RE-99",
		}},
	}
}

// Die im Altsystem vereinbarte Fälligkeit wird übernommen. Ohne sie liefe der
// Posten mit dem Zahlungsziel des Kontakts — vierzehn Tage ab Belegdatum — und
// stünde in der Altersstruktur als überfällig, obwohl er es nicht ist.
func TestOpeningOpenItemKeepsItsAgreedDueDate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	closing := fileClosingStatement(t, env)

	if _, err := env.journal.BookOpeningBalance(ctx, openingRequest(t, env, customer.ID, closing.ID)); err != nil {
		t.Fatalf("die Eröffnungsbilanz ließ sich nicht buchen: %v", err)
	}

	items, err := env.payments(t).OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten lesen: %v", err)
	}
	var found bool
	for _, item := range items {
		if item.DocumentNumber != "RE-2025-0099" {
			continue
		}
		found = true
		if item.DueDate != "2026-06-30" {
			t.Errorf("Fälligkeit %q, erwartet die übernommene 2026-06-30", item.DueDate)
		}
	}
	if !found {
		t.Fatalf("der übernommene Posten steht nicht in der Offene-Posten-Liste: %+v", items)
	}

	aging := accounting.AgeOpenItems(items, "2026-03-31")
	for _, bucket := range aging.Sides[0].Buckets {
		if bucket.Key == domain.AgingNotDue && bucket.Items != 1 {
			t.Errorf("der Posten steht nicht als „nicht fällig“ in der Altersstruktur: %+v", aging.Sides[0].Buckets)
		}
		if bucket.Key == domain.AgingWithoutDate && bucket.Items != 0 {
			t.Errorf("der Posten steht als „ohne Fälligkeit“ in der Altersstruktur")
		}
	}
}

// Zweimal erfasst hieße doppelte Bestände. Die Sperre gehört ins Backend: eine
// Regel, die an der Sichtbarkeit eines Knopfes hängt, ist keine.
func TestOpeningBalanceIsBookedOnlyOnceAndOnlyInTheFirstYear(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	closing := fileClosingStatement(t, env)
	req := openingRequest(t, env, customer.ID, closing.ID)

	if _, err := env.journal.BookOpeningBalance(ctx, req); err != nil {
		t.Fatalf("die erste Eröffnungsbilanz ließ sich nicht buchen: %v", err)
	}
	before, _ := env.journalRepo.FindAll(ctx, 2026)

	_, err := env.journal.BookOpeningBalance(ctx, req)
	if err == nil {
		t.Fatal("eine zweite Eröffnungsbilanz verdoppelte die Bestände und muss abgewiesen werden")
	}
	if !strings.Contains(err.Error(), "Eröffnungsbuchungen") {
		t.Errorf("die Meldung muss den Grund nennen: %v", err)
	}
	after, _ := env.journalRepo.FindAll(ctx, 2026)
	if len(after) != len(before) {
		t.Errorf("der abgewiesene zweite Lauf hat %d Buchungen hinterlassen", len(after)-len(before))
	}
}

func TestOpeningBalanceIsRefusedWhenThePriorYearCarriesBookings(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Eine Buchung im Vorjahr: dann ist 2026 nicht das erste Jahr, und die
	// Bestände kommen aus dem Saldenvortrag.
	prior := simpleEntry("6815", domain.AccountBank, 10000)
	prior.BookingDate, prior.DocumentDate = "2025-03-01", "2025-03-01"
	prior.ServiceDateFrom, prior.ServiceDateTo = "2025-03-01", "2025-03-01"
	prior.FiscalYear = 2025
	if _, err := env.journal.Post(ctx, prior); err != nil {
		t.Fatalf("Vorjahresbuchung: %v", err)
	}

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	closing := fileClosingStatement(t, env)
	_, err := env.journal.BookOpeningBalance(ctx, openingRequest(t, env, customer.ID, closing.ID))
	if err == nil {
		t.Fatal("im zweiten Jahr gibt es keine Eröffnungsbilanz mehr")
	}
	if !strings.Contains(err.Error(), "Saldenvortrag") {
		t.Errorf("die Meldung muss auf den Saldenvortrag verweisen: %v", err)
	}
}

// fileClosingStatement legt die Schlussbilanz des Altsystems als Beleg ab.
func fileClosingStatement(t *testing.T, env *testEnv) *domain.Receipt {
	t.Helper()
	return filePDFReceipt(t, env, FileReceiptRequest{
		Kind: domain.ReceiptKindOther, DocumentDate: "2025-12-31",
		Subject: "Schlussbilanz zum 31.12.2025 aus dem Vorgängerprogramm",
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "schlussbilanz.pdf",
			Content: []byte("%PDF-1.4 Schlussbilanz\n"),
		}},
	})
}

// --- Gesperrter Geschäftspartner an den Schreibwegen -----------------------

// Die Sperre wirkt nicht nur in den Auswahllisten: sie hängt sonst daran, dass
// jede Maske die gefilterte Liste verwendet, und ein Weg daneben — ein
// Eröffnungsposten, ein Belegbuchen mit bekannter Kennung — führte die Sperre
// vor.
func TestBlockedContactIsRefusedOnTheWritePaths(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	if _, _, err := env.contacts.BlockContact(ctx, customer.ID, "Löschverlangen nach Art. 17 DSGVO"); err != nil {
		t.Fatalf("Kontakt sperren: %v", err)
	}

	// Neue Ausgangsrechnung.
	inv := &domain.Invoice{
		FiscalYear: 2026, ContactID: customer.ID, Date: "2026-03-01",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01",
		Items: []domain.InvoiceItem{{
			Position: 1, Description: "Beratung", QuantityMilli: 1000, Unit: "Std",
			UnitPrice: 100000, TaxRate: domain.TaxRateStandard,
		}},
	}
	err := env.invoices(t).Issue(ctx, inv)
	if err == nil {
		t.Fatal("eine neue Rechnung an einen gesperrten Kontakt muss abgewiesen werden")
	}
	if !strings.Contains(err.Error(), "gesperrt") {
		t.Errorf("die Meldung muss die Sperre nennen: %v", err)
	}

	// Eröffnungsposten des Umsteigers.
	closing := fileClosingStatement(t, env)
	_, err = env.journal.PreviewOpeningBalance(ctx, openingRequest(t, env, customer.ID, closing.ID))
	if err == nil {
		t.Fatal("ein Eröffnungsposten auf einen gesperrten Kontakt muss abgewiesen werden")
	}
	if !strings.Contains(err.Error(), "gesperrt") {
		t.Errorf("die Meldung muss die Sperre nennen: %v", err)
	}
}
