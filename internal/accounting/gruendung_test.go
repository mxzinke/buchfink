package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Jede Rechtsform des Katalogs hat ihren Gründungsweg, und jeder Gründungsweg
// gehört zu einer Rechtsform des Katalogs.
func TestFoundationCatalogMatchesLegalFormCatalog(t *testing.T) {
	for _, form := range domain.LegalFormCatalog() {
		if _, ok := FoundationRulesFor(form.Name); !ok {
			t.Errorf("%s sollte im Gründungskatalog stehen", form.Name)
		}
	}
	for _, r := range FoundationLegalForms() {
		if !domain.IsSupportedLegalForm(r.LegalForm) {
			t.Errorf("Rechtsform %q steht nicht in domain.LegalFormCatalog", r.LegalForm)
		}
	}
}

func TestRequiredPaidInGmbH(t *testing.T) {
	rules, _ := FoundationRulesFor("GmbH")

	tests := []struct {
		name         string
		shareCapital domain.Cents
		shares       []domain.Shareholder
		want         domain.Cents
	}{
		{
			// Regelfall: 25.000 €, ein Viertel je Anteil wären 6.250 €, die
			// Untergrenze des § 7 Abs. 2 Satz 2 GmbHG greift mit 12.500 €.
			name:         "Mindeststammkapital, Untergrenze greift",
			shareCapital: 2_500_000,
			shares: []domain.Shareholder{
				{ShareCapital: 1_500_000, Kind: domain.ContributionCash},
				{ShareCapital: 1_000_000, Kind: domain.ContributionCash},
			},
			want: 1_250_000,
		},
		{
			// 100.000 €: ein Viertel je Anteil sind 25.000 € und damit mehr als
			// die feste Untergrenze. Die Hälfte des *tatsächlichen* Kapitals
			// (50.000 €) wäre falsch.
			name:         "hohes Stammkapital, Viertelregel greift",
			shareCapital: 10_000_000,
			shares: []domain.Shareholder{
				{ShareCapital: 10_000_000, Kind: domain.ContributionCash},
			},
			want: 2_500_000,
		},
		{
			// Sacheinlagen zählen mit ihrem vollen Nennbetrag, § 7 Abs. 2 Satz 2
			// und Abs. 3 GmbHG.
			name:         "Sacheinlage zählt voll",
			shareCapital: 2_500_000,
			shares: []domain.Shareholder{
				{ShareCapital: 2_000_000, Kind: domain.ContributionInKind},
				{ShareCapital: 500_000, Kind: domain.ContributionCash},
			},
			want: 2_125_000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &domain.Foundation{ShareCapital: tc.shareCapital, Shareholders: tc.shares}
			if got := rules.RequiredPaidIn(f); got != tc.want {
				t.Errorf("RequiredPaidIn = %s, erwartet %s", got, tc.want)
			}
		})
	}
}

func TestRequiredPaidInUGIsFullCapital(t *testing.T) {
	rules, _ := FoundationRulesFor("UG (haftungsbeschränkt)")
	f := &domain.Foundation{
		ShareCapital: 100_000,
		Shareholders: []domain.Shareholder{
			{ShareCapital: 60_000, Kind: domain.ContributionCash},
			{ShareCapital: 40_000, Kind: domain.ContributionCash},
		},
	}
	if got := rules.RequiredPaidIn(f); got != 100_000 {
		t.Errorf("RequiredPaidIn = %s, erwartet volles Stammkapital 1.000,00", got)
	}
	if !rules.CashOnly {
		t.Error("die UG schließt Sacheinlagen aus (§ 5a Abs. 2 Satz 2 GmbHG)")
	}
	if !rules.LegalReserve {
		t.Error("die UG bildet eine gesetzliche Rücklage (§ 5a Abs. 3 GmbHG)")
	}
}

func TestRequiredPerShareRoundsUp(t *testing.T) {
	rules, _ := FoundationRulesFor("GmbH")
	// 1 Cent geteilt durch vier ist ein Viertel Cent. Eine Untergrenze wird
	// aufgerundet, sonst wäre sie unterschritten.
	got := rules.RequiredPerShare(domain.Shareholder{ShareCapital: 1, Kind: domain.ContributionCash})
	if got != 1 {
		t.Errorf("RequiredPerShare = %d, erwartet 1 (aufgerundet)", got)
	}
	got = rules.RequiredPerShare(domain.Shareholder{ShareCapital: 100_001, Kind: domain.ContributionCash})
	if got != 25_001 {
		t.Errorf("RequiredPerShare = %d, erwartet 25001 (aufgerundet)", got)
	}
}

// § 18 Abs. 2 Satz 4 UStG verlangt im Gründungsjahr und im Folgejahr die
// monatliche Voranmeldung; Satz 6 ersetzt das für 2021 bis 2026 durch die
// Grenze von 9.000 € auf die voraussichtliche bzw. umgerechnete Steuer.
func TestRecommendedVatPeriodFollowsTheFoundingRules(t *testing.T) {
	tax := func(c domain.Cents) *domain.Cents { return &c }
	tests := []struct {
		name          string
		founded, year int
		annualTax     *domain.Cents
		want          domain.VatPeriodType
		applies       bool
	}{
		{"vor der Aussetzung", 2020, 2020, nil, domain.VatPeriodMonth, true},
		{"Gründung 2026 ohne Schätzung", 2026, 2026, nil, "", true},
		{"Gründung 2026 über der Grenze", 2026, 2026, tax(900_001), domain.VatPeriodMonth, true},
		{"Gründung 2026 genau an der Grenze", 2026, 2026, tax(900_000), domain.VatPeriodQuarter, true},
		{"Folgejahr 2026, umgerechnet über der Grenze", 2025, 2026, tax(1_200_000), domain.VatPeriodMonth, true},
		{"Folgejahr 2026, unter der Grenze", 2025, 2026, tax(100_000), domain.VatPeriodQuarter, true},
		{"Folgejahr 2027 einer Gründung 2026", 2026, 2027, tax(100_000), domain.VatPeriodMonth, true},
		{"Gründung 2027", 2027, 2027, nil, domain.VatPeriodMonth, true},
		{"zweites Jahr nach der Gründung", 2025, 2027, nil, "", false},
	}
	for _, tc := range tests {
		got, ok := RecommendedVatPeriod(tc.founded, tc.year, tc.annualTax)
		if ok != tc.applies || got.Period != tc.want {
			t.Errorf("%s: %q (gilt %v), erwartet %q (gilt %v)", tc.name, got.Period, ok, tc.want, tc.applies)
		}
		if ok && got.Reason == "" {
			t.Errorf("%s: ohne Begründung", tc.name)
		}
	}
}

func TestAddMonthsClampsToEndOfMonth(t *testing.T) {
	tests := []struct {
		iso    string
		months int
		want   string
	}{
		{"2026-01-31", 1, "2026-02-28"},
		{"2026-09-15", 1, "2026-10-15"},
		{"2026-09-15", 6, "2027-03-15"},
		{"2026-12-31", 12, "2027-12-31"},
		{"2024-01-31", 1, "2024-02-29"},
	}
	for _, tc := range tests {
		if got := addMonths(tc.iso, tc.months); got != tc.want {
			t.Errorf("addMonths(%q, %d) = %q, erwartet %q", tc.iso, tc.months, got, tc.want)
		}
	}
	if got := addMonths("", 1); got != "" {
		t.Errorf("addMonths auf leerem Datum = %q, erwartet leer", got)
	}
}
