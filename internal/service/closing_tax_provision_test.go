package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// postLines bucht eine nicht steuerbare Buchung am Tag mit den Zeilen.
func postLines(t *testing.T, env *testEnv, date, text string, lines ...domain.JournalLine) *domain.JournalEntry {
	t.Helper()
	entry := &domain.JournalEntry{
		BookingDate: date, DocumentDate: date, ServiceDateFrom: date, ServiceDateTo: date,
		Description: text, Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
		Lines: lines,
	}
	created, err := env.journal.Post(context.Background(), entry)
	if err != nil {
		t.Fatalf("%s buchen: %v", text, err)
	}
	return created
}

func debit(account string, amount domain.Cents) domain.JournalLine {
	return domain.JournalLine{Side: domain.SideDebit, Account: account, Amount: amount}
}

func credit(account string, amount domain.Cents) domain.JournalLine {
	return domain.JournalLine{Side: domain.SideCredit, Account: account, Amount: amount}
}

// Nur die Vorauszahlungen für das Jahr mindern die Rückstellung. Die
// Nachzahlung für das Vorjahr (7603, 7641) gehört nicht in die Abrechnung,
// die Kapitalertragsteuer (7630, 7633) wird getrennt angerechnet.
//
// Ergebnis 100.000 € mit 1.000 € nicht abziehbarer Bewirtung:
//
//	KSt 15.150,00 − Vorauszahlung 10.000,00 − KapESt 2.500,00 = 2.650,00 €
//	SolZ   833,25 − Vorauszahlung    550,00 − SolZ     137,50 =   145,75 €
//	GewSt 14.140,00 − Vorauszahlung 12.000,00                 = 2.140,00 €
func TestTaxProvisionCountsOnlyTheYearsPrepayments(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	env.setSetting(t, domain.SettingTradeTaxRate, "400")

	postLines(t, env, "2026-06-30", "Erlöse",
		debit(domain.AccountBank, 10_000_000), debit("6644", 100_000), credit("4400", 10_100_000))
	postLines(t, env, "2026-03-10", "KSt-Vorauszahlung",
		debit(domain.AccountKoerperschaftsteuer, 1_000_000), debit(domain.AccountSolidaritaetszuschlag, 55_000),
		credit(domain.AccountBank, 1_055_000))
	postLines(t, env, "2026-05-15", "GewSt-Vorauszahlung",
		debit(domain.AccountGewerbesteuer, 1_200_000), credit(domain.AccountBank, 1_200_000))
	postLines(t, env, "2026-04-20", "Nachzahlung Vorjahr",
		debit("7603", 300_000), debit("7641", 200_000), credit(domain.AccountBank, 500_000))
	// Zinsen 10.000 € mit Kapitalertragsteuer 2.500 € und SolZ 137,50 €.
	postLines(t, env, "2026-12-30", "Zinsen",
		debit(domain.AccountBank, 736_250), debit(accounting.AccountKapitalertragsteuer25, 250_000),
		debit(accounting.AccountSoliKapitalertragsteuer, 13_750), credit("7100", 1_000_000))

	preview, err := m.bookings.PreviewTaxProvision(ctx, 2026, TaxProvisionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// Die Zinsen erhöhen das Ergebnis auf 110.000 €.
	if preview.Input.ProfitBeforeTax != 11_000_000 {
		t.Fatalf("Ergebnis vor Steuern %s € — erwartet 110.000,00 €", preview.Input.ProfitBeforeTax)
	}
	if preview.Input.PrepaidCorporate != 1_000_000 || preview.Input.PrepaidSolidarity != 55_000 ||
		preview.Input.PrepaidTrade != 1_200_000 {
		t.Errorf("Vorauszahlungen KSt %s €, SolZ %s €, GewSt %s € — die Nachzahlung für das Vorjahr "+
			"gehört nicht dazu", preview.Input.PrepaidCorporate, preview.Input.PrepaidSolidarity,
			preview.Input.PrepaidTrade)
	}
	if preview.Input.CreditableWithholding != 250_000 || preview.Input.CreditableSolidarity != 13_750 {
		t.Errorf("anrechenbar KapESt %s €, SolZ %s €",
			preview.Input.CreditableWithholding, preview.Input.CreditableSolidarity)
	}
	// 111.000 € Bemessungsgrundlage: KSt 16.650 €, SolZ 915,75 €, GewSt 15.540 €.
	if preview.CorporateTax != 1_665_000 || preview.Solidarity != 91_575 || preview.TradeTax != 1_554_000 {
		t.Fatalf("KSt %s €, SolZ %s €, GewSt %s €", preview.CorporateTax, preview.Solidarity, preview.TradeTax)
	}
	if preview.CorporateDue != 415_000 || preview.SolidarityDue != 22_825 || preview.TradeProvision != 354_000 {
		t.Errorf("offen KSt %s €, SolZ %s €, GewSt %s €",
			preview.CorporateDue, preview.SolidarityDue, preview.TradeProvision)
	}
	if preview.TradeTaxRateSource != "einstellungen" {
		t.Errorf("Herkunft des Hebesatzes %q", preview.TradeTaxRateSource)
	}
	if !containsText(preview.Notes, "Steuerbescheinigung") {
		t.Errorf("der Hinweis auf die Steuerbescheinigung fehlt: %v", preview.Notes)
	}

	without, err := m.bookings.PreviewTaxProvision(ctx, 2026, TaxProvisionOptions{SkipWithholdingCredit: true})
	if err != nil {
		t.Fatal(err)
	}
	if without.CorporateDue != 665_000 || without.SolidarityDue != 36_575 {
		t.Errorf("ohne Anrechnung KSt %s €, SolZ %s €", without.CorporateDue, without.SolidarityDue)
	}

	if _, err := m.bookings.BookTaxProvision(ctx, TaxProvisionRequest{FiscalYear: 2026}); err != nil {
		t.Fatal(err)
	}
	after := balances(t, env, 2026)
	if after[domain.AccountRueckstellungKoerperschaft] != -437_825 ||
		after[domain.AccountRueckstellungGewerbesteuer] != -354_000 {
		t.Errorf("Rückstellungen 3040 %s €, 3035 %s €",
			after[domain.AccountRueckstellungKoerperschaft], after[domain.AccountRueckstellungGewerbesteuer])
	}
	if _, err := m.bookings.BookTaxProvision(ctx, TaxProvisionRequest{FiscalYear: 2026}); err == nil {
		t.Error("ein zweiter Lauf hätte die Rückstellung ein zweites Mal gebildet")
	}
}

// Ohne Hebesatz wird nicht geschätzt; die Angabe aus dem Bescheid geht vor.
func TestTaxProvisionRequiresTheMunicipalRate(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	postLines(t, env, "2026-06-30", "Erlöse", debit(domain.AccountBank, 1_000_000), credit("4400", 1_000_000))

	if _, err := m.bookings.PreviewTaxProvision(ctx, 2026, TaxProvisionOptions{}); err == nil ||
		!strings.Contains(err.Error(), "Hebesatz") {
		t.Fatalf("ohne Hebesatz: %v", err)
	}
	env.setSetting(t, domain.SettingTradeTaxRate, "400")
	preview, err := m.bookings.PreviewTaxProvision(ctx, 2026, TaxProvisionOptions{TradeTaxRatePercent: 490})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Input.TradeTaxRatePercent != 490 || preview.TradeTaxRateSource != "eingabe" {
		t.Errorf("Hebesatz %d %% aus %q", preview.Input.TradeTaxRatePercent, preview.TradeTaxRateSource)
	}
}

// Die nicht abziehbaren Aufwendungen kommen aus dem ganzen Katalog: Geschenke
// über der Freigrenze, Bewirtung, Verspätungszuschlag, Geldbuße, Hälfte der
// Aufsichtsratsvergütung. Eine Dividende aus einer Beteiligung bleibt nach
// § 8b Abs. 1 KStG außer Ansatz, 5 % davon gelten als nicht abziehbar.
//
//	Ergebnis           100.000,00 € (Erlöse 120.000 €, Aufwand 30.000 €, Dividende 10.000 €)
//	nicht abziehbar      5.000,00 € (6620 1.000, 6644 1.000, 6437 500, 6968 1.500, 6875 1.000)
//	Dividende 7006     − 10.000,00 €, 5 % = 500,00 €
//	Bemessungsgrundlage 95.500,00 €
func TestTaxProvisionAddsBackTheNonDeductibleCatalog(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	postLines(t, env, "2026-06-30", "Erlöse", debit(domain.AccountBank, 12_000_000), credit("4400", 12_000_000))
	postLines(t, env, "2026-07-31", "Aufwand",
		debit("6620", 100_000), debit("6644", 100_000), debit("6437", 50_000), debit("6968", 150_000),
		debit("6875", 100_000), debit("6300", 2_500_000), credit(domain.AccountBank, 3_000_000))
	postLines(t, env, "2026-08-31", "Dividende Tochter", debit(domain.AccountBank, 1_000_000), credit("7006", 1_000_000))

	preview, err := m.bookings.PreviewTaxProvision(ctx, 2026, TaxProvisionOptions{TradeTaxRatePercent: 400})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Input.ProfitBeforeTax != 10_000_000 || preview.Input.NonDeductible != 500_000 ||
		preview.Input.ExemptDividends != 1_000_000 {
		t.Fatalf("Ergebnis %s €, nicht abziehbar %s €, steuerfrei %s €",
			preview.Input.ProfitBeforeTax, preview.Input.NonDeductible, preview.Input.ExemptDividends)
	}
	if preview.TaxableIncome != 9_550_000 {
		t.Errorf("Bemessungsgrundlage %s € — erwartet 95.500,00 €", preview.TaxableIncome)
	}
	if len(preview.Adjustments) != 6 {
		t.Errorf("Korrekturkonten: %+v", preview.Adjustments)
	}
	if !containsText(preview.Notes, "§ 9 Nr. 2a GewStG") {
		t.Errorf("der Hinweis auf die Gewerbesteuer fehlt: %v", preview.Notes)
	}

	// Die Angabe ersetzt die Vorbelegung: Streubesitz ist steuerpflichtig.
	zero := domain.Cents(0)
	taxable, err := m.bookings.PreviewTaxProvision(ctx, 2026, TaxProvisionOptions{
		TradeTaxRatePercent: 400, ExemptDividends: &zero,
		LossCarryforwardCorporate: 2_000_000, IncomeAdjustments: 300_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 100.000 + 5.000 + 3.000 − 20.000 Verlustvortrag = 88.000 €
	if taxable.TaxableIncome != 8_800_000 || taxable.LossDeducted != 2_000_000 {
		t.Errorf("zu versteuern %s €, Verlustabzug %s €", taxable.TaxableIncome, taxable.LossDeducted)
	}
}

// Ein Verlustjahr mit Vorauszahlungen bucht Erstattungsansprüche: die
// Überzahlung ist am Stichtag eine Forderung (1450, 1435).
func TestTaxProvisionBooksRefundClaimsOfALossYear(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	postLines(t, env, "2026-06-30", "Aufwand", debit("6300", 500_000), credit(domain.AccountBank, 500_000))
	postLines(t, env, "2026-03-10", "Vorauszahlungen",
		debit(domain.AccountKoerperschaftsteuer, 400_000), debit(domain.AccountSolidaritaetszuschlag, 22_000),
		debit(domain.AccountGewerbesteuer, 300_000), credit(domain.AccountBank, 722_000))

	provisions, err := m.bookings.BookTaxProvision(ctx, TaxProvisionRequest{
		FiscalYear: 2026, Options: TaxProvisionOptions{TradeTaxRatePercent: 400},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(provisions) != 0 {
		t.Errorf("ein Verlustjahr bildet keine Rückstellung: %+v", provisions)
	}
	after := balances(t, env, 2026)
	if after[accounting.AccountKoerperschaftsteuerForderung] != 422_000 ||
		after[accounting.AccountGewerbesteuerForderung] != 300_000 {
		t.Errorf("Erstattungsansprüche 1450 %s €, 1435 %s €",
			after[accounting.AccountKoerperschaftsteuerForderung], after[accounting.AccountGewerbesteuerForderung])
	}
	if after[domain.AccountKoerperschaftsteuer] != 0 || after[domain.AccountGewerbesteuer] != 0 {
		t.Errorf("der Steueraufwand eines Verlustjahres ist null: 7600 %s €, 7610 %s €",
			after[domain.AccountKoerperschaftsteuer], after[domain.AccountGewerbesteuer])
	}
	steps, err := m.steps.Steps(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range steps.Steps {
		if step.Key == domain.ClosingStepTaxProvision && step.State != domain.ClosingStepDone {
			t.Errorf("Schritt Steuerrückstellung: %s (%s)", step.State, step.Detail)
		}
	}
	if _, err := m.bookings.BookTaxProvision(ctx, TaxProvisionRequest{
		FiscalYear: 2026, Options: TaxProvisionOptions{TradeTaxRatePercent: 400},
	}); err == nil {
		t.Error("ein zweiter Lauf hätte die Ansprüche ein zweites Mal gebucht")
	}
}

func containsText(lines []string, needle string) bool {
	for _, line := range lines {
		if strings.Contains(line, needle) {
			return true
		}
	}
	return false
}
