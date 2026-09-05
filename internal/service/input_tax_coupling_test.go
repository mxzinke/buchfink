package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Kopplung des Vorsteuerabzugs an die geprüfte Rechnung (UST-07, RECH-07).

// incompleteVendor ist ein Lieferant ohne Steuernummer und ohne USt-IdNr. — der
// Fall, in dem die Rechnung eine Pflichtangabe des § 14 Abs. 4 UStG nicht
// tragen kann.
func (e *testEnv) incompleteVendor(t *testing.T, name string) *domain.Contact {
	t.Helper()
	c := &domain.Contact{
		Type: domain.ContactTypeVendor, Name: name, CountryCode: "DE",
		Street: "Lieferantenweg 3", PostalCode: "20095", City: "Hamburg",
	}
	if err := e.contacts.SaveContact(context.Background(), c); err != nil {
		t.Fatalf("Lieferant %s: %v", name, err)
	}
	return c
}

// structuredReceipt legt einen Beleg mit strukturiertem Rechnungsdatensatz ab
// und vermerkt das Ergebnis der Prüfung an ihm.
func (e *testEnv) structuredReceipt(t *testing.T, validationErrors int) *domain.Receipt {
	t.Helper()
	ctx := context.Background()
	receipt, err := e.receipts.File(ctx, FileReceiptRequest{
		Direction: domain.DirectionIncoming, FiscalYear: e.fiscalYear,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, Path: e.writeTempFile(t, "rechnung.pdf", minimalPDF)},
			{Role: domain.ReceiptRoleStructured, FileName: "rechnung.xml",
				Content: []byte("<Invoice/>"), Derived: true},
		},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}
	if err := e.receipts.SaveValidation(ctx, receipt.ID, domain.ReceiptValidation{
		Format: "cii", Profile: "EN 16931", At: "2026-03-10T10:00:00Z",
		Ruleset: "EN16931", Version: "1.3.13", Coverage: "model",
		Errors: validationErrors,
	}); err != nil {
		t.Fatalf("Prüfergebnis speichern: %v", err)
	}
	reloaded, err := e.receipts.Get(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	return reloaded
}

// Eine E-Rechnung mit Befunden aus der Prüfung nach EN 16931 hält die Buchung
// mit Vorsteuer an. Bis hierher las der Buchungsweg das eigene Prüfergebnis
// nicht — eine Rechnung mit fünf Inhaltsfehlern lief durch, und die Vorsteuer
// wurde gezogen, als sei nichts gewesen.
func TestInputTaxRefusedForAnInvalidEInvoice(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Agentur GmbH", "DE", "")
	receipt := env.structuredReceipt(t, 3)

	req := ReceiptRequest{
		ContactID: vendor.ID, ReceiptID: receipt.ID,
		BookingDate: "2026-03-10", DocumentDate: "2026-03-10",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-31",
		Description: "Beratung", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions: []ReceiptPosition{
			{PostingGroup: "fremdleistungen", Net: 100_000, TaxRate: domain.TaxRateStandard},
		},
		Settlement: SettlementOpen,
	}

	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Fatal("eine E-Rechnung mit Fehlern darf nicht ohne Weiteres mit Vorsteuer gebucht werden")
	} else if !strings.Contains(err.Error(), "§ 15 Abs. 1") {
		t.Errorf("die Meldung muss die Vorschrift nennen: %v", err)
	}

	// Die Vorschau zeigt denselben Befund, ohne anzuhalten: sie ist die Stelle,
	// an der der Anwender ihn zu sehen bekommt.
	preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("die Vorschau darf nicht anhalten: %v", err)
	}
	if len(preview.InputTaxFindings) != 1 ||
		preview.InputTaxFindings[0].Code != findingEInvoiceInvalid {
		t.Fatalf("Befunde %+v — erwartet genau den Befund zur fehlerhaften E-Rechnung",
			preview.InputTaxFindings)
	}

	// Mit Grund wird gebucht, und der Grund steht am Beleg.
	req.OverrideReason = "Lieferant hat die Rechnung berichtigt zugesagt, Vorsteuerabzug bleibt bestehen"
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("mit Grund muss die Buchung durchgehen: %v", err)
	}
	if entry.EntryNumber == "" {
		t.Error("die Buchung hat keine Nummer")
	}
	sealed, err := env.receipts.Get(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if sealed.InputTaxOverride != req.OverrideReason {
		t.Errorf("der Grund am Beleg lautet %q — erwartet %q",
			sealed.InputTaxOverride, req.OverrideReason)
	}
	if sealed.InputTaxOverrideAt == "" {
		t.Error("zum Grund gehört sein Zeitpunkt")
	}

	// Und er steht im Protokoll.
	logs, err := repository.NewAuditRepository(env.db).FindAll(ctx, 200)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	found := false
	for _, entry := range logs {
		if strings.Contains(entry.Details, "übersteuert") {
			found = true
		}
	}
	if !found {
		t.Error("die Übersteuerung gehört ins Protokoll — sie ist eine Entscheidung gegen eine Prüfung")
	}
}

// Eine sonstige Rechnung ohne Steuernummer des Ausstellers trägt eine
// Pflichtangabe nicht (§ 14 Abs. 4 Nr. 2 UStG).
func TestInputTaxRefusedWithoutTheIssuersTaxNumber(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.incompleteVendor(t, "Zettelwirtschaft e. K.")

	req := env.receipt(t, vendor.ID, "fremdleistungen", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	_, err := env.posting.PostIncomingReceipt(ctx, req)
	if err == nil {
		t.Fatal("ohne Steuernummer oder USt-IdNr. des Ausstellers fehlt eine Pflichtangabe")
	}
	if !strings.Contains(err.Error(), "§ 14 Abs. 4 Nr. 2") {
		t.Errorf("die Meldung muss die fehlende Angabe benennen: %v", err)
	}

	// Dieselbe Rechnung ohne Vorsteuer geht durch: die Pflichtangaben sind
	// Voraussetzung des *Abzugs*, nicht der Buchung.
	exempt := env.receipt(t, vendor.ID, "versicherungen", 100_000,
		domain.TaxRateNone, domain.TaxTreatmentExempt)
	if _, err := env.posting.PostIncomingReceipt(ctx, exempt); err != nil {
		t.Fatalf("ohne Vorsteuerabzug gibt es nichts zu sperren: %v", err)
	}
}

// Der Vorsteuerschlüssel der gemischten Nutzung: bei 600 ‰ wird nur der
// abziehbare Teil der Vorsteuer gezogen, der Rest gehört zum Aufwand
// (§ 15 Abs. 4 UStG, § 9b Abs. 1 EStG).
func TestInputTaxShareSplitsTheDeduction(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Autohaus GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "fahrzeugkosten", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Positions[0].InputTaxShare = 600
	req.Positions[0].InputTaxShareReason = "Kfz zu 60 % betrieblich genutzt, Fahrtenbuch 2026"

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}

	var expense, inputTax, base, settlement domain.Cents
	var share int
	for _, l := range entry.Lines {
		switch {
		case l.TaxKey == "VST19":
			inputTax += l.Amount
			base = l.TaxBase
		case l.Side == domain.SideDebit:
			expense += l.Amount
			share = l.InputTaxShare
		default:
			settlement += l.Amount
		}
	}
	// 19 % von 1.000 € sind 190 €; abziehbar sind 60 % davon, also 114 €.
	if inputTax != 11_400 {
		t.Errorf("Vorsteuer %s € — erwartet 114,00 € (60 %% von 190,00 €)", inputTax)
	}
	// Die Bemessungsgrundlage ist der abziehbare Teil: 600 €. Nur er geht in
	// die Kennziffer 66.
	if base != 60_000 {
		t.Errorf("Bemessungsgrundlage %s € — erwartet 600,00 €", base)
	}
	// Der Aufwand trägt den Nettobetrag plus die nicht abziehbare Vorsteuer.
	if expense != 107_600 {
		t.Errorf("Aufwand %s € — erwartet 1.076,00 € (1.000,00 + 76,00 nicht abziehbare Vorsteuer)",
			expense)
	}
	if share != 600 {
		t.Errorf("der Vorsteueranteil steht mit %d an der Aufwandszeile — erwartet 600", share)
	}
	// An den Lieferanten gehen die vollen 1.190 €.
	if settlement != 119_000 {
		t.Errorf("Gegenzeile %s € — erwartet 1.190,00 €", settlement)
	}
	if !entry.IsBalanced() {
		t.Error("die Buchung ist nicht ausgeglichen")
	}

	// Und die Voranmeldung meldet nur den abziehbaren Teil.
	period, err := accounting.VatPeriodOf("2026-03-10", "quarter")
	if err != nil {
		t.Fatalf("Zeitraum: %v", err)
	}
	ret := accounting.BuildVatReturn(period, accounting.VatReturnSource{
		Entries: []domain.JournalEntry{*entry},
	})
	if got := ret.Tax(accounting.VatCodeInputTax); got != 11_400 {
		t.Errorf("Kennziffer 66 meldet %s € — erwartet 114,00 €", got)
	}
}

// Ein geteilter Vorsteuerabzug ohne Maßstab ist keine sachgerechte Schätzung
// (§ 15 Abs. 4 Satz 2 UStG).
func TestInputTaxShareNeedsItsReason(t *testing.T) {
	env := newTestEnv(t)
	vendor := env.vendor(t, "Autohaus GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "fahrzeugkosten", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Positions[0].InputTaxShare = 600

	if _, err := env.posting.PostIncomingReceipt(context.Background(), req); err == nil {
		t.Fatal("ohne Maßstab darf der Vorsteuerabzug nicht geteilt werden")
	} else if !strings.Contains(err.Error(), "§ 15 Abs. 4") {
		t.Errorf("die Meldung muss die Vorschrift nennen: %v", err)
	}
}
