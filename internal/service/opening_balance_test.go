package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Jede Eröffnungsbuchung ist für sich ausgeglichen; aufgehen muss die
// Eröffnungsbilanz als Ganzes. Fehlt eine Passivseite, darf sie nicht gebucht
// werden, auch wenn der Ausgleich auf 9000 die einzelne Buchung schließt.
func TestOpeningBalanceMustBalanceAcrossCarryForwardAccounts(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	closing := fileClosingStatement(t, env)
	req := OpeningBalanceRequest{
		FiscalYear: 2026, ReceiptID: &closing.ID, LegacySystem: "Vorgängerprogramm",
		Accounts: []OpeningBalanceLine{
			{Account: "1800", Side: domain.SideDebit, Amount: 500000},
			{Account: "2900", Side: domain.SideCredit, Amount: 500000},
		},
		Receivables: []OpenOpeningItem{{
			ContactID: customer.ID, Amount: 119000, DocumentNumber: "RE-1", DocumentDate: "2025-12-01",
		}},
	}

	preview, err := env.journal.PreviewOpeningBalance(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Balanced {
		t.Error("die Forderung von 1.190 € steht keiner Passivseite gegenüber; die Vorschau darf nicht ausgeglichen sein")
	}
	if preview.DebitTotal != 619000 || preview.CreditTotal != 500000 {
		t.Errorf("Aktiva/Passiva %s/%s, erwartet 6.190,00/5.000,00", preview.DebitTotal, preview.CreditTotal)
	}
	if _, err := env.journal.BookOpeningBalance(ctx, req); err == nil || !strings.Contains(err.Error(), "geht nicht auf") {
		t.Fatalf("eine Eröffnungsbilanz, die nicht aufgeht, darf nicht gebucht werden; Fehler: %v", err)
	}
	if entries, _ := env.journalRepo.FindAll(ctx, 2026); len(entries) != 0 {
		t.Errorf("%d Buchungen stehen nach der Abweisung im Journal", len(entries))
	}

	req.Accounts = append(req.Accounts, OpeningBalanceLine{
		Account: domain.AccountSaldenvortraegeSachkonten, Side: domain.SideCredit, Amount: 119000})
	if _, err := env.journal.PreviewOpeningBalance(ctx, req); err == nil {
		t.Error("ein Vortragskonto in den Beständen würde jede Differenz verdecken")
	}

	req.Accounts = req.Accounts[:2]
	req.Accounts[1].Amount = 619000
	booked, err := env.journal.BookOpeningBalance(ctx, req)
	if err != nil {
		t.Fatalf("die ausgeglichene Eröffnungsbilanz muss buchbar sein: %v", err)
	}
	if !booked.Balanced || carryForwardBalance(booked.Entries) != 0 {
		t.Errorf("Vortragskonten nach der Buchung: %s", carryForwardBalance(booked.Entries))
	}
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
			{Account: "2900", Side: domain.SideCredit, Amount: 619000, LegacyRef: "ALT-2900"},
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
// Regel, die sich nach der Sichtbarkeit eines Knopfes richtet, ist keine.
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

// --- Eröffnungsbilanz des Umsteigers --------------------------------------

func TestOpeningBalanceBooksAccountsAndOpenItems(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	vendor := env.vendor(t, "Lieferant GmbH", "DE", "")
	// Die Schlussbilanz des Altsystems ist keine Rechnung: sie hat keinen
	// leistenden Unternehmer und keinen einzelnen Betrag. Ihre Kopfdaten sind
	// deshalb Belegdatum und Betreff (siehe Receipt.ValidateHeader).
	closing := filePDFReceipt(t, env, FileReceiptRequest{
		Kind: domain.ReceiptKindOther, DocumentDate: "2025-12-31",
		Subject: "Schlussbilanz zum 31.12.2025 aus dem Vorgängerprogramm",
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "schlussbilanz-altsystem.pdf",
			Content: []byte("%PDF-1.4 Schlussbilanz\n"),
		}},
	})

	req := OpeningBalanceRequest{
		FiscalYear:   2026,
		Date:         "2026-01-01",
		ReceiptID:    &closing.ID,
		LegacySystem: "Vorgängerprogramm",
		// Zwei Bestandskonten der Aktivseite. Die Sammelkonten der Bilanz —
		// 1200 Forderungen, 3300 Verbindlichkeiten — stehen bewusst nicht hier:
		// sie werden nicht direkt bebucht, ihre Werte kommen aus den
		// Personenkonten weiter unten.
		Accounts: []OpeningBalanceLine{
			{Account: "1800", Side: domain.SideDebit, Amount: 500000, LegacyRef: "ALT-1800"},
			{Account: "1600", Side: domain.SideDebit, Amount: 20000, LegacyRef: "ALT-1600"},
			{Account: "2900", Side: domain.SideCredit, Amount: 579500, LegacyRef: "ALT-2900"},
		},
		Receivables: []OpenOpeningItem{{
			ContactID: customer.ID, Amount: 119000,
			DocumentNumber: "RE-2025-0099", DocumentDate: "2025-12-01",
			DueDate: "2026-01-15", LegacyRef: "ALT-RE-99",
		}},
		Payables: []OpenOpeningItem{{
			ContactID: vendor.ID, Amount: 59500,
			DocumentNumber: "ER-2025-0042", DocumentDate: "2025-12-10",
			LegacyRef: "ALT-ER-42",
		}},
	}

	preview, err := env.journal.PreviewOpeningBalance(ctx, req)
	if err != nil {
		t.Fatalf("die Vorschau ist fehlgeschlagen: %v", err)
	}
	if len(preview.Entries) != 3 {
		t.Errorf("%d Buchungen in der Vorschau, erwartet 3 (Sachkonten, Forderung, Verbindlichkeit)",
			len(preview.Entries))
	}
	if !preview.Balanced {
		t.Errorf("die Eröffnungsbilanz muss ausgeglichen sein: Soll %s, Haben %s",
			preview.DebitTotal, preview.CreditTotal)
	}

	booked, err := env.journal.BookOpeningBalance(ctx, req)
	if err != nil {
		t.Fatalf("die Eröffnungsbilanz ließ sich nicht buchen: %v", err)
	}

	var receivable *domain.JournalEntry
	for _, entry := range booked.Entries {
		if entry.Source != domain.EntrySourceOpening {
			t.Errorf("Buchung %s hat die Quelle %q, erwartet %q",
				entry.EntryNumber, entry.Source, domain.EntrySourceOpening)
		}
		if entry.LegacyRef == "" {
			t.Errorf("Buchung %s hat keine Herkunftskennung aus dem Altsystem", entry.EntryNumber)
		}
		if entry.ContactID != nil && *entry.ContactID == customer.ID {
			receivable = entry
		}
	}
	if receivable == nil {
		t.Fatal("die Forderung des Umsteigers wurde nicht als eigene Buchung erfasst")
	}
	if receivable.LegacyRef != "ALT-RE-99" {
		t.Errorf("Herkunftskennung %q, erwartet ALT-RE-99", receivable.LegacyRef)
	}

	// Je offener Posten eine Buchung auf dem Personenkonto gegen 9008/9009 —
	// eine Sammelbuchung ließe sich später nicht ausgleichen.
	var onLedger, onCarryforward bool
	for _, line := range receivable.Lines {
		if line.Account == customer.LedgerAccount && line.Side == domain.SideDebit {
			onLedger = true
		}
		if line.Account == domain.AccountSaldenvortraegeDebitoren {
			onCarryforward = true
		}
	}
	if !onLedger || !onCarryforward {
		t.Errorf("die Forderung muss auf dem Personenkonto gegen %s stehen: %+v",
			domain.AccountSaldenvortraegeDebitoren, receivable.Lines)
	}

	check, err := env.journal.VerifyIntegrity(ctx)
	if err != nil || !check.IsValid {
		t.Errorf("die Eröffnungsbilanz darf die Kette nicht brechen: %v / %s", err, check.Message)
	}
}

func TestOpeningBalanceNeedsTheClosingStatementAsReceipt(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.journal.PreviewOpeningBalance(context.Background(), OpeningBalanceRequest{
		FiscalYear: 2026,
		Accounts: []OpeningBalanceLine{
			{Account: "1800", Side: domain.SideDebit, Amount: 100000},
		},
	})
	if err == nil {
		t.Fatal("eine Eröffnungsbilanz ohne Beleg wäre eine Behauptung über fremde Zahlen (§ 146 Abs. 1 AO)")
	}
	if !strings.Contains(err.Error(), "Schlussbilanz") {
		t.Errorf("die Meldung muss sagen, was fehlt: %v", err)
	}
}
