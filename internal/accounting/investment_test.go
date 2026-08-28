package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// § 20 InvStG staffelt die Aktienteilfreistellung nach dem Anleger, die
// Immobilienteilfreistellung nicht. Und die eine schließt die andere aus.
func TestPartialExemptionFollowsFundAndInvestor(t *testing.T) {
	cases := []struct {
		fund     FundClass
		investor domain.InvestorType
		want     int
	}{
		{FundEquity, domain.InvestorCorporate, 800},
		{FundEquity, domain.InvestorIndividualBusiness, 600},
		{FundEquity, domain.InvestorBasic, 300},
		{FundMixed, domain.InvestorCorporate, 400},
		{FundMixed, domain.InvestorIndividualBusiness, 300},
		{FundMixed, domain.InvestorBasic, 150},
		// Die Immobilienteilfreistellung hängt nicht vom Anleger ab.
		{FundRealEstate, domain.InvestorCorporate, 600},
		{FundRealEstate, domain.InvestorBasic, 600},
		{FundForeignRealEstate, domain.InvestorIndividualBusiness, 800},
		{FundOther, domain.InvestorCorporate, 0},
	}
	for _, c := range cases {
		got, err := PartialExemptionFor(c.fund, c.investor)
		if err != nil {
			t.Errorf("%s / %s: %v", c.fund, c.investor, err)
			continue
		}
		if got.Permille != c.want {
			t.Errorf("%s / %s: %d Promille — erwartet %d", c.fund, c.investor, got.Permille, c.want)
		}
		if got.Source == "" || got.Explanation == "" {
			t.Errorf("%s / %s: der Satz kommt ohne Fundstelle oder ohne Begründung", c.fund, c.investor)
		}
	}
}

// Aus der Rechtsform folgt der Satz nicht. Ist die Anlegerstellung nicht
// festgelegt, rechnet Buchfink nicht — es sagt, was zu entscheiden ist.
func TestPartialExemptionRefusesToGuessTheInvestor(t *testing.T) {
	_, err := PartialExemptionFor(FundEquity, domain.InvestorUnknown)
	if err == nil {
		t.Fatal("ohne Anlegerstellung darf keine Teilfreistellung entstehen")
	}
	if !strings.Contains(err.Error(), "Rechtsform") {
		t.Errorf("die Meldung nennt den Grund nicht: %v", err)
	}

	// Bei einer Personengesellschaft mit gemischt besteuerten Gesellschaftern
	// gibt es keinen einzigen Satz (§ 20 Abs. 3a InvStG).
	_, err = PartialExemptionFor(FundEquity, domain.InvestorMixed)
	if err == nil {
		t.Fatal("für eine gemischt besteuerte Personengesellschaft gibt es keinen einheitlichen Satz")
	}
	if !strings.Contains(err.Error(), "3a") {
		t.Errorf("die Meldung nennt die Fundstelle nicht: %v", err)
	}

	// Für einen Einzeltitel gibt es sie überhaupt nicht.
	if _, err := PartialExemptionFor(FundNone, domain.InvestorCorporate); err == nil {
		t.Error("eine Einzelaktie ist kein Investmentanteil")
	}
}

// Der Basisertrag sind 70 % des Basiszinses auf den Rücknahmepreis zu
// Jahresbeginn; die Vorabpauschale ist der Betrag, um den die Ausschüttungen
// ihn unterschreiten (§ 18 Abs. 1 InvStG).
func TestVorabpauschaleFollowsTheStatute(t *testing.T) {
	// 100.000,00 € zu Jahresbeginn, Basiszins 2,53 %, kein Ausschütter.
	// Basisertrag = 100.000 × 0,7 × 2,53 % = 1.771,00 €.
	out, err := ComputeVorabpauschale(VorabpauschaleInput{
		Year: 2025, OpeningPrice: 10_000_000, ClosingPrice: 11_000_000, BasisPoints: 253,
	})
	if err != nil {
		t.Fatalf("Vorabpauschale: %v", err)
	}
	if out.BasisReturn != 177_100 || out.Amount != 177_100 {
		t.Errorf("Basisertrag %s €, Vorabpauschale %s € — erwartet je 1.771,00 €",
			out.BasisReturn, out.Amount)
	}
	if out.AccruedOn != "2026-01-02" {
		t.Errorf("Zufluss %q — erwartet den ersten Werktag von 2026", out.AccruedOn)
	}
	if out.Capped {
		t.Error("der Wertzuwachs von 10.000,00 € begrenzt hier nichts")
	}
}

// Der Basisertrag ist auf den Wertzuwachs begrenzt (§ 18 Abs. 1 Satz 3 InvStG).
// Ein Fonds, der verloren hat, trägt keine Vorabpauschale.
func TestVorabpauschaleIsCappedByTheGrowth(t *testing.T) {
	out, err := ComputeVorabpauschale(VorabpauschaleInput{
		Year: 2025, OpeningPrice: 10_000_000, ClosingPrice: 10_050_000, BasisPoints: 253,
	})
	if err != nil {
		t.Fatalf("Vorabpauschale: %v", err)
	}
	if !out.Capped || out.Amount != 50_000 {
		t.Errorf("Vorabpauschale %s € (begrenzt: %v) — erwartet 500,00 €", out.Amount, out.Capped)
	}

	loss, err := ComputeVorabpauschale(VorabpauschaleInput{
		Year: 2025, OpeningPrice: 10_000_000, ClosingPrice: 9_000_000, BasisPoints: 253,
	})
	if err != nil {
		t.Fatalf("Vorabpauschale: %v", err)
	}
	if loss.Amount != 0 {
		t.Errorf("bei einem Wertverlust entsteht keine Vorabpauschale, hier %s €", loss.Amount)
	}
}

// Ausschüttungen mindern sie, und im Erwerbsjahr wird sie um ein Zwölftel je
// vollem Monat vor dem Erwerb gekürzt (§ 18 Abs. 1 Satz 1 und Abs. 2 InvStG).
func TestVorabpauschaleAccountsForDistributionsAndTheYearOfPurchase(t *testing.T) {
	out, err := ComputeVorabpauschale(VorabpauschaleInput{
		Year: 2025, OpeningPrice: 10_000_000, ClosingPrice: 11_000_000,
		Distributions: 100_000, BasisPoints: 253,
	})
	if err != nil {
		t.Fatalf("Vorabpauschale: %v", err)
	}
	if out.Amount != 77_100 {
		t.Errorf("Vorabpauschale %s € — erwartet 771,00 € (1.771,00 € abzüglich 1.000,00 €)", out.Amount)
	}

	// Erwerb im Oktober: vier Zwölftel.
	partial, err := ComputeVorabpauschale(VorabpauschaleInput{
		Year: 2025, OpeningPrice: 10_000_000, ClosingPrice: 11_000_000,
		BasisPoints: 253, AcquisitionMonth: 10,
	})
	if err != nil {
		t.Fatalf("Vorabpauschale: %v", err)
	}
	if partial.MonthsCounted != 3 || partial.Amount != 44_275 {
		t.Errorf("Erwerb im Oktober: %d Zwölftel über %s € — erwartet 3 und 442,75 €",
			partial.MonthsCounted, partial.Amount)
	}
}

// Der Basiszins steht nicht im Gesetz. Ohne ihn wird nicht gerechnet.
func TestVorabpauschaleNeedsTheBasisRate(t *testing.T) {
	_, err := ComputeVorabpauschale(VorabpauschaleInput{
		Year: 2026, OpeningPrice: 10_000_000, ClosingPrice: 11_000_000,
	})
	if err == nil {
		t.Fatal("ohne Basiszins darf keine Vorabpauschale entstehen")
	}
	if !strings.Contains(err.Error(), "Bundessteuerblatt") {
		t.Errorf("die Meldung nennt die Quelle nicht: %v", err)
	}
}
