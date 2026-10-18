package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// -------------------------------------------------------------------------
// Fremdwährung am Belegweg (BEW-10)
// -------------------------------------------------------------------------

// currencyPosting verdrahtet Kursdienst und Belegweg gegen einen gefakten
// EZB-Endpunkt.
func (e *testEnv) currencyPosting(t *testing.T, endpoint string) *CurrencyService {
	t.Helper()
	svc := e.currency(t, endpoint)
	e.posting.SetCurrencyConverter(svc)
	return svc
}

// Ein Beleg über 1.085,00 USD vom 10.03.2026: der Aufwand folgt dem
// EZB-Tageskurs (1,0850 → 1.000,00 €), die Bemessungsgrundlage der Umsatzsteuer
// dem Durchschnittskurs des Monats (1,1000 → 986,36 €), und die Differenz ist
// Kursaufwand auf 6880. Kurs, Quelle und Kurstag stehen am Buchungskopf.
func TestForeignCurrencyReceiptBooksBothRates(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0850", "2026-03-10")
	svc := env.currencyPosting(t, server.URL)
	if _, err := svc.SaveVatRate(ctx, domain.VatExchangeRate{
		Month: "2026-03", Currency: "USD", RateMicros: 1_100_000,
	}); err != nil {
		t.Fatalf("Durchschnittskurs: %v", err)
	}
	vendor := env.vendor(t, "Supplier Inc.", "US", "")

	req := env.receipt(t, vendor.ID, "fremdleistungen", 108_500,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Currency = "USD"
	req.ForeignAmount = 108_500

	preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Conversion == nil || preview.Conversion.Rate.RateMicros != 1_085_000 {
		t.Fatalf("die Vorschau muss den verwendeten Kurs zeigen: %+v", preview.Conversion)
	}

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	if entry.Currency != "USD" {
		t.Errorf("Währung %q — erwartet USD", entry.Currency)
	}
	if entry.ExchangeRateMicros != 1_085_000 {
		t.Errorf("Kurs %d Millionstel — erwartet 1.085.000 und niemals 1.000.000 (Kurs 1,0)",
			entry.ExchangeRateMicros)
	}
	if !strings.Contains(entry.ExchangeRateSource, "EZB") || entry.ExchangeRateDate != "2026-03-10" {
		t.Errorf("Quelle %q zum %q — beide gehören an die Buchung",
			entry.ExchangeRateSource, entry.ExchangeRateDate)
	}

	// Der Aufwand: 1.085,00 USD zu 1,0850 sind 1.000,00 €.
	if got := debitOn(entry, "5906"); got != 100_000 {
		t.Errorf("Aufwand %s € — erwartet 1.000,00 € zum Tageskurs", got)
	}
	var expense *domain.JournalLine
	for i := range entry.Lines {
		if entry.Lines[i].Account == "5906" {
			expense = &entry.Lines[i]
		}
	}
	if expense == nil || expense.ForeignAmount != 108_500 {
		t.Errorf("Fremdbetrag der Aufwandszeile %+v — erwartet 1.085,00 USD", expense)
	}

	// Die Steuerzeile: Bemessungsgrundlage aus dem Umsatzsteuerkurs (986,36 €),
	// Vorsteuer 19 % davon (187,41 €).
	var tax *domain.JournalLine
	for i := range entry.Lines {
		if entry.Lines[i].TaxKey == "VST19" {
			tax = &entry.Lines[i]
		}
	}
	if tax == nil {
		t.Fatalf("keine Vorsteuerzeile: %s", accountsOf(entry))
	}
	if tax.TaxBase != 98_636 {
		t.Errorf("Bemessungsgrundlage %s € — erwartet 986,36 € zum Durchschnittskurs des Monats "+
			"(§ 16 Abs. 6 UStG)", tax.TaxBase)
	}
	if tax.Amount != 18_741 {
		t.Errorf("Vorsteuer %s € — erwartet 187,41 €", tax.Amount)
	}

	// Die Differenz zwischen Tages- und Umsatzsteuerkurs ist Kursaufwand: 19 %
	// von 1.000,00 € sind 190,00 €, gebucht werden 187,41 € — 2,59 € auf 6880.
	if got := debitOn(entry, accounting.CurrencyLossAccount); got != 259 {
		t.Errorf("Kursaufwand %s € auf %s — erwartet 2,59 €",
			got, accounting.CurrencyLossAccount)
	}

	// Und die Gegenzeile zeigt, was tatsächlich zu zahlen ist: 1.290,15 USD zum
	// Tageskurs sind 1.190,00 €.
	settlement := entry.Lines[len(entry.Lines)-1]
	if settlement.Amount != 119_000 {
		t.Errorf("Gegenzeile %s € — erwartet 1.190,00 € brutto zum Tageskurs", settlement.Amount)
	}
	if settlement.ForeignAmount != 129_115 {
		t.Errorf("Fremdbetrag der Gegenzeile %s — erwartet 1.291,15 USD", settlement.ForeignAmount)
	}
}

// Ohne Kurs wird nicht gebucht. Ein Rückfallwert von 1,0 bucht einen
// Dollarbetrag als Eurobetrag — in richtiger Größenordnung und mit falschem
// Wert, und niemandem fällt es auf.
func TestForeignCurrencyReceiptRefusedWithoutARate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	// Ein Server, der sofort geschlossen wird: es gibt keinen Kurs.
	server, _ := ecbServer(t, "1.0850", "2026-03-10")
	url := server.URL
	server.Close()
	env.currencyPosting(t, url)
	vendor := env.vendor(t, "Supplier Inc.", "US", "")

	req := env.receipt(t, vendor.ID, "fremdleistungen", 108_500,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Currency = "USD"

	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Fatal("ohne Kurs darf kein Fremdwährungsbeleg gebucht werden")
	}

	// Mit einem von Hand erfassten Kurs geht es — er speichert seine Quelle.
	svc := env.currency(t, url)
	env.posting.SetCurrencyConverter(svc)
	if _, err := svc.SaveRate(ctx, domain.ExchangeRate{
		Currency: "USD", Date: "2026-03-10", RateMicros: 1_085_000,
		Source: "EZB-Referenzkurs, abgelesen am 11.03.2026",
	}); err != nil {
		t.Fatalf("Kurs von Hand: %v", err)
	}
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("mit erfasstem Kurs muss der Beleg buchbar sein: %v", err)
	}
	if entry.ExchangeRateMicros != 1_085_000 {
		t.Errorf("Kurs %d Millionstel — erwartet den erfassten", entry.ExchangeRateMicros)
	}
}

// Die Kontrollsumme des Belegs wird gegen die Positionen gehalten. Ein
// Zahlendreher ergibt sonst eine Buchung, die in sich aufgeht und mit der
// Rechnung daneben nicht übereinstimmt.
func TestForeignCurrencyControlTotalIsChecked(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0850", "2026-03-10")
	env.currencyPosting(t, server.URL)
	vendor := env.vendor(t, "Supplier Inc.", "US", "")

	req := env.receipt(t, vendor.ID, "fremdleistungen", 108_500,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Currency = "USD"
	req.ForeignAmount = 180_500

	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Fatal("die Endsumme des Belegs muss zu seinen Positionen passen")
	}
}
