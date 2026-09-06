package service

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/export"
)

// Die Tabelle „pruefpfad" des Prüferpakets (RECH-08, GoBD Rz. 36).
//
// Sie ist die einzige Tabelle der Überlassung, die den Weg vom Beleg über die
// Buchung und die Zahlung bis zum Bankumsatz in einer Zeile führt. Geprüft wird
// deshalb nicht, dass die Datei entsteht, sondern was in ihr steht: eine Tabelle
// mit lauter leeren Zahlungsspalten sähe von außen genauso vollständig aus.
func TestExportAuditTrailTableLinksReceiptBookingPaymentAndBank(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	vendor := env.vendor(t, "Kabelwerk GmbH", "DE", "DE111111111")
	invoice := env.openPayable(t, vendor.ID, 100_000, domain.TaxRateStandard)

	booked, err := env.receiptRepo.FindByJournalEntry(ctx, invoice.ID)
	if err != nil || booked == nil {
		t.Fatalf("Beleg zur Buchung: %v", err)
	}
	// Bestellbezug und Leistungsnachweis stehen in derselben Zeile — sie sind
	// der Grund, aus dem der Prüfer den Weg überhaupt geht.
	if _, err := env.receipts.SaveOrderReference(ctx, booked.ID, "4711"); err != nil {
		t.Fatalf("Bestellbezug: %v", err)
	}
	if _, err := env.receipts.SaveServiceProof(
		ctx, booked.ID, "geprüft gegen Bestellung 4711 vom 28.02.2026", "2026-03-02"); err != nil {
		t.Fatalf("Leistungsnachweis: %v", err)
	}

	txID := env.bankLine(t, "2026-03-20", -119_000, "Kabelwerk GmbH", "Rechnung")
	payment, err := env.payments(t).Settle(ctx, PaymentRequest{
		BankTxID:       &txID,
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-20",
		Description:    "Überweisung Kabelwerk",
		Allocations: []AllocationRequest{
			{OpenItemEntryID: invoice.ID, SettledAmount: 119_000},
		},
	})
	if err != nil {
		t.Fatalf("Zahlung buchen: %v", err)
	}

	// Ein zweiter Beleg, der nur abgelegt ist: seine Zeile muss trotzdem
	// entstehen — ein Beleg, der im Prüfpfad fehlt, ist der Beleg, nach dem
	// gefragt wird.
	unbooked := env.fileIncoming(t, "ungebucht.pdf")

	dir := filepath.Join(t.TempDir(), "z3")
	result, err := env.exports(t).ExportZ3(ctx, env.fiscalYear, dir)
	if err != nil {
		t.Fatalf("Z3-Export: %v", err)
	}
	var file string
	for _, table := range result.Tables {
		if table.Name == "pruefpfad" {
			file = table.File
		}
	}
	if file == "" {
		t.Fatal("die Tabelle pruefpfad fehlt im Prüferpaket")
	}

	rows := readSemicolonCSV(t, filepath.Join(dir, file))
	if len(rows) < 2 {
		t.Fatalf("pruefpfad.csv hat %d Zeilen — erwartet die Kopfzeile und je Beleg eine", len(rows))
	}
	column := map[string]int{}
	for i, name := range rows[0] {
		column[name] = i
	}
	for _, name := range []string{
		"Beleg_ID", "Bestellbezug", "Leistungsnachweis", "Buchung_ID",
		"Zahlung_Buchung_ID", "Ausgleichsbetrag", "Bankumsatz_ID",
	} {
		if _, ok := column[name]; !ok {
			t.Fatalf("die Spalte %s fehlt: %v", name, rows[0])
		}
	}
	field := func(row []string, name string) string { return row[column[name]] }

	var bookedRow, unbookedRow []string
	for _, row := range rows[1:] {
		switch field(row, "Beleg_ID") {
		case export.Uint(booked.ID):
			bookedRow = row
		case export.Uint(unbooked.ID):
			unbookedRow = row
		}
	}
	if bookedRow == nil {
		t.Fatalf("der bezahlte Beleg %d steht nicht im Prüfpfad", booked.ID)
	}
	if got := field(bookedRow, "Buchung_ID"); got != export.Uint(invoice.ID) {
		t.Errorf("Buchung_ID = %q, erwartet %q", got, export.Uint(invoice.ID))
	}
	if got := field(bookedRow, "Zahlung_Buchung_ID"); got != export.Uint(payment.ID) {
		t.Errorf("Zahlung_Buchung_ID = %q, erwartet %q — ohne sie endet der Pfad vor der Zahlung",
			got, export.Uint(payment.ID))
	}
	if got := field(bookedRow, "Ausgleichsbetrag"); got != "1190.00" {
		t.Errorf("Ausgleichsbetrag = %q, erwartet 1190.00", got)
	}
	if got := field(bookedRow, "Bankumsatz_ID"); got != export.Uint(txID) {
		t.Errorf("Bankumsatz_ID = %q, erwartet %q — der Umsatz ist das Ende des Pfades", got, export.Uint(txID))
	}
	if got := field(bookedRow, "Bestellbezug"); got != "4711" {
		t.Errorf("Bestellbezug = %q, erwartet 4711", got)
	}
	if got := field(bookedRow, "Leistungsnachweis"); got == "" {
		t.Error("der Leistungsnachweis fehlt in der Zeile")
	}

	if unbookedRow == nil {
		t.Fatalf("der abgelegte Beleg %d fehlt im Prüfpfad", unbooked.ID)
	}
	if got := field(unbookedRow, "Zahlung_Buchung_ID"); got != "" {
		t.Errorf("Zahlung_Buchung_ID = %q — ein ungebuchter Beleg hat keine Zahlung", got)
	}
	if got := field(unbookedRow, "Bankumsatz_ID"); got != "" {
		t.Errorf("Bankumsatz_ID = %q — ein ungebuchter Beleg hat keinen Bankumsatz", got)
	}
}

// readSemicolonCSV liest eine Datei der Überlassung ein.
func readSemicolonCSV(t *testing.T, path string) [][]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("%s lesen: %v", filepath.Base(path), err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = ';'
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("%s zerlegen: %v", filepath.Base(path), err)
	}
	return rows
}
