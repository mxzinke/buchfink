package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func paramsFor(t *testing.T, date string) TaxParameters {
	t.Helper()
	p, err := TaxParametersFor(date)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Der Verlustabzug über dem Sockel ist ein Höchstbetrag („bis zu"): 333 Cent
// über 1 Mio. € bei 60 % ergeben 199 Cent, nicht 200.
func TestLossDeductionTruncatesTheExcessShare(t *testing.T) {
	base := domain.Cents(100_000_000)
	if got := LossDeduction(base+333, 200_000_000, base, 600); got != base+199 {
		t.Errorf("Verlustabzug %s, erwartet %s", got, base+199)
	}
	if got := LossDeduction(50_000, 20_000, base, 600); got != 20_000 {
		t.Errorf("Vortrag kleiner als Einkommen: %s", got)
	}
	if got := LossDeduction(-50_000, 20_000, base, 600); got != 0 {
		t.Errorf("Verlust im Jahr: %s", got)
	}
}

// 15 % von 100.009,99 € sind 15.001,4985 €; die Steuer wird zugunsten auf
// volle Euro gerundet (§ 31 Abs. 1 Satz 2 KStG), der Solidaritätszuschlag
// schneidet Bruchteile eines Cents ab (§ 4 Satz 3 SolZG).
func TestCorporateTaxAndSolidarityRounding(t *testing.T) {
	p := paramsFor(t, "2026-12-31")
	if got := CorporateTaxOn(10_000_999, p); got != 1_500_100 {
		t.Errorf("KSt %s, erwartet 15.001,00", got)
	}
	if got := SolidarityOn(1_500_100, p); got != 82_505 {
		t.Errorf("SolZ %s, erwartet 825,05", got)
	}
	if got := CorporateTaxOn(-100, p); got != 0 {
		t.Errorf("KSt auf Verlust: %s", got)
	}
}

// Die anrechenbare Kapitalertragsteuer wird auf volle Euro aufgerundet
// (§ 36 Abs. 3 EStG).
func TestCreditableWithholdingRoundsUp(t *testing.T) {
	for in, want := range map[domain.Cents]domain.Cents{250_040: 250_100, 250_000: 250_000, 1: 100, 0: 0} {
		if got := CreditableWithholding(in); got != want {
			t.Errorf("%s → %s, erwartet %s", in, got, want)
		}
	}
}

// Höchstbetrag der Zuwendungen: 20 % von 100.000,03 € = 20.000,006 €,
// abgeschnitten 20.000,00 €; 4 ‰ von 1.500.000 € = 6.000 €.
func TestDonationCapTakesTheHigherTruncatedLimit(t *testing.T) {
	p := paramsFor(t, "2026-12-31")
	limit, byIncome, byTurnover := DonationCap(10_000_003, 150_000_000, p)
	if limit != 2_000_000 || byIncome != 2_000_000 || byTurnover != 600_000 {
		t.Errorf("Höchstbetrag %s (%s / %s)", limit, byIncome, byTurnover)
	}
	if limit, _, _ := DonationCap(-500_000, 150_000_000, p); limit != 600_000 {
		t.Errorf("bei Verlust gilt der Umsatzhöchstbetrag: %s", limit)
	}
}

// Gewerbeertrag 197.549,99 € wird auf 197.500 € abgerundet, Messbetrag 3,5 %
// = 6.912,50 €, bei 410 % Gewerbesteuer 28.341,25 €.
func TestTradeTaxOn(t *testing.T) {
	p := paramsFor(t, "2026-12-31")
	got := TradeTaxOn(19_754_999, 410, p)
	if got.Rounded != 19_750_000 || got.Base != 691_250 || got.Tax != 2_834_125 {
		t.Errorf("Gewerbesteuer %+v", got)
	}
	if got := TradeTaxOn(-1, 410, p); got != (TradeTaxResult{}) {
		t.Errorf("Gewerbeverlust: %+v", got)
	}
}
