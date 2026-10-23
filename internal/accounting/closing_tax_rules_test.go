package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Mindestbesteuerung: 1,5 Mio. € Einkommen, 2 Mio. € Verlustvortrag. Für 2026
// mindert der Vortrag bis 1 Mio. € voll und darüber 70 % (§ 10d Abs. 2 EStG in
// der Fassung für 2024 bis 2027), beim Gewerbeertrag 60 % (§ 10a GewStG).
//
//	KSt:   1.000.000 + 70 % × 500.000 = 1.350.000 abziehbar → 150.000 €
//	       Körperschaftsteuer 22.500,00 €, SolZ 1.237,50 €
//	GewSt: 1.000.000 + 60 % × 500.000 = 1.300.000 abziehbar → 200.000 €
//	       Messbetrag 7.000,00 €, Gewerbesteuer bei 400 % 28.000,00 €
func TestLossCarryforwardFollowsMinimumTaxation(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax:           150_000_000,
		LossCarryforwardCorporate: 200_000_000, LossCarryforwardTrade: 200_000_000,
		TradeTaxRatePercent: 400, Date: "2026-12-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.LossDeducted != 135_000_000 || result.TaxableIncome != 15_000_000 ||
		result.LossRemaining != 65_000_000 {
		t.Errorf("KSt: abgezogen %s €, zu versteuern %s €, Vortrag %s €",
			result.LossDeducted, result.TaxableIncome, result.LossRemaining)
	}
	if result.CorporateTax != 2_250_000 || result.Solidarity != 123_750 {
		t.Errorf("KSt %s €, SolZ %s €", result.CorporateTax, result.Solidarity)
	}
	if result.TradeLossDeducted != 130_000_000 || result.TradeIncome != 20_000_000 ||
		result.TradeLossRemaining != 70_000_000 {
		t.Errorf("GewSt: abgezogen %s €, Gewerbeertrag %s €, Vortrag %s €",
			result.TradeLossDeducted, result.TradeIncome, result.TradeLossRemaining)
	}
	if result.TradeBase != 700_000 || result.TradeTax != 2_800_000 {
		t.Errorf("Messbetrag %s €, Gewerbesteuer %s €", result.TradeBase, result.TradeTax)
	}

	// Ab dem Veranlagungszeitraum 2028: 14 % und wieder 60 %.
	later, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 150_000_000, LossCarryforwardCorporate: 200_000_000,
		TradeTaxRatePercent: 400, Date: "2028-12-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if later.LossDeducted != 130_000_000 || later.TaxableIncome != 20_000_000 ||
		later.CorporateTax != 2_800_000 || later.Solidarity != 154_000 {
		t.Errorf("2028: abgezogen %s €, zu versteuern %s €, KSt %s €, SolZ %s €",
			later.LossDeducted, later.TaxableIncome, later.CorporateTax, later.Solidarity)
	}
	if !strings.Contains(later.RatesUsed, "KSt 14 %") {
		t.Errorf("Sätze: %s", later.RatesUsed)
	}
}

// Ein kleiner Vortrag wird ganz verbraucht; ein Verlust des Jahres erhöht ihn.
func TestLossCarryforwardBelowTheBaseIsUsedUp(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 5_000_000, LossCarryforwardCorporate: 2_000_000,
		LossCarryforwardTrade: 2_000_000, TradeTaxRatePercent: 400, Date: "2026-12-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TaxableIncome != 3_000_000 || result.LossRemaining != 0 || result.TradeIncome != 3_000_000 {
		t.Errorf("zu versteuern %s €, Vortrag %s €, Gewerbeertrag %s €",
			result.TaxableIncome, result.LossRemaining, result.TradeIncome)
	}
	loss, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: -1_000_000, LossCarryforwardCorporate: 2_000_000,
		TradeTaxRatePercent: 400, Date: "2026-12-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if loss.LossRemaining != 3_000_000 || loss.CorporateTax != 0 {
		t.Errorf("Verlustjahr: Vortrag %s €, KSt %s €", loss.LossRemaining, loss.CorporateTax)
	}
}

// § 8b KStG: Eine Dividende von 50.000 € bleibt außer Ansatz, 5 % davon gelten
// als nicht abziehbar. Ergebnis 100.000 € → Einkommen 52.500 €.
//
//	Körperschaftsteuer 15 %   7.875,00 €
//	SolZ 5,5 %                  433,12 € (433,125 €; Bruchteile eines Cents
//	                                     bleiben außer Ansatz, § 4 Satz 3 SolZG)
//	Gewerbeertrag 52.500 €, Messbetrag 1.837,50 €, Gewerbesteuer 7.350,00 €
func TestExemptDividendsCarryFivePercent(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 10_000_000, ExemptDividends: 5_000_000,
		TradeTaxRatePercent: 400, Date: "2026-12-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExemptAddBack != 250_000 || result.TaxableIncome != 5_250_000 {
		t.Errorf("5 %% %s €, Einkommen %s €", result.ExemptAddBack, result.TaxableIncome)
	}
	if result.CorporateTax != 787_500 || result.Solidarity != 43_312 {
		t.Errorf("KSt %s €, SolZ %s €", result.CorporateTax, result.Solidarity)
	}
	if result.TradeIncome != 5_250_000 || result.TradeTax != 735_000 {
		t.Errorf("Gewerbeertrag %s €, Gewerbesteuer %s €", result.TradeIncome, result.TradeTax)
	}
}

// Vorauszahlungen nur des Jahres und die angerechnete Kapitalertragsteuer
// stehen getrennt: KSt 15.150 € − 10.000 € − 2.500 € = 2.650 €, SolZ
// 833,25 € − 550 € − 137,50 € = 145,75 €.
func TestWithholdingTaxIsCreditedSeparately(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 10_000_000, NonDeductible: 100_000, TradeTaxRatePercent: 400,
		PrepaidCorporate: 1_000_000, PrepaidSolidarity: 55_000,
		CreditableWithholding: 250_000, CreditableSolidarity: 13_750,
		Date: "2026-12-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.CorporateDue != 265_000 || result.SolidarityDue != 14_575 || result.IncomeProvision != 279_575 {
		t.Errorf("KSt offen %s €, SolZ offen %s €, Rückstellung %s €",
			result.CorporateDue, result.SolidarityDue, result.IncomeProvision)
	}
}

// Ein Verlustjahr mit Vorauszahlungen: die Vorauszahlungen kommen zurück.
func TestPrepaymentsOfALossYearAreRefunded(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: -500_000, TradeTaxRatePercent: 400,
		PrepaidCorporate: 500_000, PrepaidSolidarity: 27_500, PrepaidTrade: 400_000,
		Date: "2026-12-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IncomeRefund != 527_500 || result.TradeRefund != 400_000 || result.IncomeProvision != 0 {
		t.Errorf("Erstattung KSt/SolZ %s €, GewSt %s €, Rückstellung %s €",
			result.IncomeRefund, result.TradeRefund, result.IncomeProvision)
	}
}

// Zuwendungen: Ergebnis 10.000 € nach 5.000 € Spenden, Einkommen vor Spenden
// 15.000 €. Höchstbetrag: 20 % = 3.000 € oder 4 ‰ von 1 Mio. € Umsätzen und
// Löhnen = 4.000 €; der höhere gilt. Abziehbar 4.000 €, vorgetragen 1.000 €.
func TestDonationsAreDeductedWithinTheHigherLimit(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 1_000_000, Donations: 500_000, TurnoverAndWages: 100_000_000,
		TradeTaxRatePercent: 400, Date: "2026-12-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IncomeBeforeDonations != 1_500_000 || result.DonationsDeducted != 400_000 ||
		result.DonationsCarried != 100_000 || result.TaxableIncome != 1_100_000 {
		t.Errorf("vor Spenden %s €, abgezogen %s €, vorgetragen %s €, zu versteuern %s €",
			result.IncomeBeforeDonations, result.DonationsDeducted, result.DonationsCarried,
			result.TaxableIncome)
	}
	if result.CorporateTax != 165_000 {
		t.Errorf("KSt %s €", result.CorporateTax)
	}
}

// Hinzurechnungen und Kürzungen treffen nur die Gewerbesteuer.
func TestTradeAdditionsAndDeductionsOnlyMoveTheTradeTax(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 10_000_000, TradeAdditions: 1_000_000, TradeDeductions: 200_000,
		TradeTaxRatePercent: 400, Date: "2026-12-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TaxableIncome != 10_000_000 || result.TradeIncome != 10_800_000 {
		t.Errorf("Einkommen %s €, Gewerbeertrag %s €", result.TaxableIncome, result.TradeIncome)
	}
}

// § 16 Abs. 4 Satz 2 GewStG: ab 2027 mindestens 280 %.
func TestTradeTaxRateBelowTheMinimumIsRefused(t *testing.T) {
	if _, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 10_000_000, TradeTaxRatePercent: 250, Date: "2026-12-31",
	}); err != nil {
		t.Errorf("2026 sind 250 %% zulässig: %v", err)
	}
	if _, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 10_000_000, TradeTaxRatePercent: 250, Date: "2027-12-31",
	}); err == nil || !strings.Contains(err.Error(), "280") {
		t.Errorf("2027 liegen 250 %% unter dem Mindesthebesatz: %v", err)
	}
	if _, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 10_000_000, TradeTaxRatePercent: 400, LossCarryforwardCorporate: -1,
		Date: "2026-12-31",
	}); err == nil {
		t.Error("ein negativer Verlustvortrag ist keine Angabe")
	}
}

func TestSplitIncomeTaxKeepsTheTotal(t *testing.T) {
	cases := []struct{ kst, soli, wantKst, wantSoli domain.Cents }{
		{265_000, 14_575, 265_000, 14_575},
		{-10_000, 5_000, -5_000, 0},
		{10_000, -2_000, 8_000, 0},
		{-2_000, 10_000, 0, 8_000},
		{-10_000, -500, -10_000, -500},
	}
	for _, c := range cases {
		kst, soli := SplitIncomeTax(c.kst, c.soli)
		if kst != c.wantKst || soli != c.wantSoli {
			t.Errorf("SplitIncomeTax(%s, %s) = %s, %s — erwartet %s, %s",
				c.kst, c.soli, kst, soli, c.wantKst, c.wantSoli)
		}
	}
}

func TestTaxAdjustmentAccountsExist(t *testing.T) {
	chart := NewChart(DefaultSKR04Accounts())
	for _, a := range TaxAdjustmentAccounts() {
		if _, ok := chart.Lookup(a.Account); !ok {
			t.Errorf("Konto %s (%s) steht nicht im SKR04-Katalog", a.Account, a.Kind)
		}
	}
	for _, number := range []string{AccountKapitalertragsteuer25, AccountSoliKapitalertragsteuer,
		AccountKoerperschaftsteuerForderung, AccountGewerbesteuerForderung} {
		if _, ok := chart.Lookup(number); !ok {
			t.Errorf("Konto %s steht nicht im SKR04-Katalog", number)
		}
	}
	wage, _ := chart.Lookup("6020")
	revenue, _ := chart.Lookup("4400")
	if !IsWageAccount(wage) || !IsRevenueAccount(revenue) || IsRevenueAccount(wage) {
		t.Error("Löhne und Umsatzerlöse werden nicht erkannt")
	}
	if !IsIncomeTaxAccount("7610") || IsIncomeTaxAccount("7650") || IsIncomeTaxAccount("76100") {
		t.Error("Ertragsteuerkonten werden nicht erkannt")
	}
}
