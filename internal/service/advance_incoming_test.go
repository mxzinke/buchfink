package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Eingangsseite der Anzahlungen: geleistete Anzahlungen.
//
// Sie ist das Spiegelbild der eigenen Abschlagsrechnung. Die
// Anzahlungsrechnung des Lieferanten wird nicht mit ihrem Eingang gebucht,
// sondern mit der Zahlung: der Vorsteuerabzug setzt nach § 15 Abs. 1 Satz 1
// Nr. 1 Satz 3 UStG neben der Rechnung die Entrichtung des Entgelts voraus. Und
// die Schlussrechnung des Lieferanten setzt die Anzahlung wieder ab — sonst
// stünde sie doppelt im Vermögen und die Vorsteuer zweimal in der Voranmeldung.

// postingWithAdvances liefert den Belegweg mit angekoppelten geleisteten
// Anzahlungen.
func (e *testEnv) postingWithAdvances(t *testing.T) *PostingService {
	t.Helper()
	e.posting.SetVendorAdvances(repository.NewVendorAdvanceRepository(e.db))
	// Mit der Transaktionsklammer, wie im Betrieb: Buchung und Vermerk der
	// Anzahlung gehören zusammen.
	e.posting.SetTxRunner(repository.NewTxRunner(e.db))
	return e.posting
}

// Die geleistete Anzahlung bucht 1180 und die Vorsteuer gegen das
// Zahlungsmittel — im Zeitraum der Zahlung.
func TestIncomingAdvanceBooksAssetAndInputTaxOnPayment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Maschinenbau GmbH", "DE", "")
	posting := env.postingWithAdvances(t)

	req := env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.AdvanceTarget = accounting.AdvanceTargetInventory

	// Auf Ziel gibt es sie nicht: vor der Zahlung ist weder etwas geliefert
	// noch die Vorsteuer abziehbar.
	_, err := posting.PostIncomingReceipt(ctx, req)
	if err == nil || !strings.Contains(err.Error(), "§ 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG") {
		t.Fatalf("eine offene Anzahlungsrechnung darf nicht gebucht werden, erhalten: %v", err)
	}

	req.Settlement = SettlementPaid
	req.PaymentAccount = domain.AccountBank
	entry, err := posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Anzahlung buchen: %v", err)
	}

	if got := lineAmount(t, entry, accounting.AccountGeleisteteAnzahlungenVorraete, domain.SideDebit); got != 100000 {
		t.Errorf("1180 im Soll = %s, erwartet 1000,00", got)
	}
	if got := lineAmount(t, entry, domain.AccountVorsteuer19, domain.SideDebit); got != 19000 {
		t.Errorf("Vorsteuer im Soll = %s, erwartet 190,00", got)
	}
	if got := lineAmount(t, entry, domain.AccountBank, domain.SideCredit); got != 119000 {
		t.Errorf("Bank im Haben = %s, erwartet 1190,00", got)
	}

	// Die Vorsteuer fällt in den Zeitraum der Zahlung und nicht in den der
	// Leistung — geleistet ist noch nichts.
	var taxLine domain.JournalLine
	for _, l := range entry.Lines {
		if l.TaxKey != "" {
			taxLine = l
		}
	}
	if taxLine.TaxKey == "" {
		t.Fatal("die Vorsteuerzeile muss den Steuerschlüssel tragen")
	}
	if got := accounting.VatPeriodFor(entry, taxLine, ""); got != req.BookingDate {
		t.Errorf("die Vorsteuer fällt in den Zeitraum %q, erwartet den Zahlungstag %q", got, req.BookingDate)
	}

	open, err := posting.OpenVendorAdvances(ctx, vendor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].NetAmount != 100000 || open[0].GrossAmount != 119000 {
		t.Fatalf("erwartet eine offene geleistete Anzahlung über 1000,00 netto, erhalten %+v", open)
	}
	if open[0].Account != accounting.AccountGeleisteteAnzahlungenVorraete {
		t.Errorf("die Anzahlung steht auf %q, erwartet 1180", open[0].Account)
	}
}

// Die Schlussrechnung des Lieferanten setzt die Anzahlung ab: 1180 wird
// aufgelöst, und die Vorsteuer entsteht nur noch auf den Restbetrag.
func TestVendorFinalInvoiceDeductsTheAdvance(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Maschinenbau GmbH", "DE", "")
	posting := env.postingWithAdvances(t)

	advance := env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	advance.AdvanceTarget = accounting.AdvanceTargetInventory
	advance.Settlement = SettlementPaid
	advance.PaymentAccount = domain.AccountBank
	if _, err := posting.PostIncomingReceipt(ctx, advance); err != nil {
		t.Fatalf("Anzahlung buchen: %v", err)
	}
	open, err := posting.OpenVendorAdvances(ctx, vendor.ID)
	if err != nil || len(open) != 1 {
		t.Fatalf("die geleistete Anzahlung fehlt: %v %+v", err, open)
	}

	final := env.receipt(t, vendor.ID, "fremdleistungen", 300000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	final.SettledAdvanceIDs = []uint{open[0].ID}
	entry, err := posting.PostIncomingReceipt(ctx, final)
	if err != nil {
		t.Fatalf("Schlussrechnung buchen: %v", err)
	}

	// Der volle Aufwand …
	if got := lineAmount(t, entry, "5906", domain.SideDebit); got != 300000 {
		t.Errorf("Aufwand im Soll = %s, erwartet 3000,00", got)
	}
	// … die Auflösung der Anzahlung …
	if got := lineAmount(t, entry, accounting.AccountGeleisteteAnzahlungenVorraete, domain.SideCredit); got != 100000 {
		t.Errorf("1180 im Haben = %s, erwartet 1000,00", got)
	}
	// … die Vorsteuer nur auf den Restbetrag von 2000,00 …
	if got := lineAmount(t, entry, domain.AccountVorsteuer19, domain.SideDebit); got != 38000 {
		t.Errorf("Vorsteuer im Soll = %s, erwartet 380,00 auf den Restbetrag", got)
	}
	// … und als Verbindlichkeit bleibt der Rest.
	if got := lineAmount(t, entry, vendor.LedgerAccount, domain.SideCredit); got != 238000 {
		t.Errorf("Verbindlichkeit = %s, erwartet 2380,00", got)
	}

	// Die Anzahlung ist verrechnet und steht kein zweites Mal zur Verfügung.
	stillOpen, err := posting.OpenVendorAdvances(ctx, vendor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stillOpen) != 0 {
		t.Errorf("die verrechnete Anzahlung ist weiter offen: %+v", stillOpen)
	}
	second := env.receipt(t, vendor.ID, "fremdleistungen", 300000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	second.SettledAdvanceIDs = []uint{open[0].ID}
	if _, err := posting.PostIncomingReceipt(ctx, second); err == nil {
		t.Error("eine bereits verrechnete Anzahlung darf nicht ein zweites Mal abgesetzt werden")
	}

	// Über alle Buchungen hinweg: 1180 ist ausgeglichen, die Vorsteuer ist
	// genau einmal auf 3000,00 netto gezogen.
	if got := env.accountBalance(t, accounting.AccountGeleisteteAnzahlungenVorraete); got != 0 {
		t.Errorf("Konto 1180 trägt %s, erwartet null", got)
	}
	if got := env.accountBalance(t, domain.AccountVorsteuer19); got != 57000 {
		t.Errorf("Vorsteuer insgesamt = %s, erwartet 570,00 auf 3000,00 netto", got)
	}
}

// Die Anzahlung eines anderen Lieferanten setzt diese Schlussrechnung nicht ab.
func TestVendorAdvanceBelongsToItsSupplier(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	first := env.vendor(t, "Maschinenbau GmbH", "DE", "")
	second := env.vendor(t, "Anderer Lieferant GmbH", "DE", "")
	posting := env.postingWithAdvances(t)

	advance := env.receipt(t, first.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	advance.AdvanceTarget = accounting.AdvanceTargetTangible
	advance.Settlement = SettlementPaid
	advance.PaymentAccount = domain.AccountBank
	if _, err := posting.PostIncomingReceipt(ctx, advance); err != nil {
		t.Fatalf("Anzahlung buchen: %v", err)
	}
	open, err := posting.OpenVendorAdvances(ctx, first.ID)
	if err != nil || len(open) != 1 {
		t.Fatalf("die geleistete Anzahlung fehlt: %v %+v", err, open)
	}
	if open[0].Account != accounting.AccountGeleisteteAnzahlungenAnlagen {
		t.Errorf("die Anzahlung auf eine Sachanlage steht auf %q, erwartet 0700", open[0].Account)
	}

	wrong := env.receipt(t, second.ID, "fremdleistungen", 300000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	wrong.SettledAdvanceIDs = []uint{open[0].ID}
	_, err = posting.PostIncomingReceipt(ctx, wrong)
	if err == nil || !strings.Contains(err.Error(), "anderen Lieferanten") {
		t.Errorf("die Anzahlung eines anderen Lieferanten darf nicht abgesetzt werden, erhalten: %v", err)
	}
}
