package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestFoundationRulesForOnlyCoversKapitalgesellschaften(t *testing.T) {
	for _, name := range []string{"GmbH", "UG (haftungsbeschränkt)", "AG"} {
		if _, ok := FoundationRulesFor(name); !ok {
			t.Errorf("%s sollte im Gründungskatalog stehen", name)
		}
	}
	// Bei einer Personengesellschaft gibt es keine Vorgesellschaft — der
	// Gründungsweg darf dort gar nicht erst erscheinen.
	for _, name := range []string{"GbR", "OHG", "KG", "GmbH & Co. KG", "Einzelunternehmen", "Sonstige"} {
		if _, ok := FoundationRulesFor(name); ok {
			t.Errorf("%s darf nicht im Gründungskatalog stehen", name)
		}
	}
}

func TestFoundationCatalogMatchesLegalFormCatalog(t *testing.T) {
	// Die Schreibweise muss zum Rechtsformkatalog passen, sonst findet
	// FoundationRulesFor die Rechtsform des Mandanten nie.
	for _, r := range FoundationLegalForms() {
		if _, ok := domain.LookupLegalForm(r.LegalForm); !ok {
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

func TestRecommendedVatPeriodAtTheStichjahr(t *testing.T) {
	tests := []struct {
		year int
		want string
	}{
		{2020, "month"},
		{2021, "quarter"},
		{2026, "quarter"},
		{2027, "month"},
		{2030, "month"},
	}
	for _, tc := range tests {
		if got := RecommendedVatPeriod(tc.year); got != tc.want {
			t.Errorf("RecommendedVatPeriod(%d) = %q, erwartet %q", tc.year, got, tc.want)
		}
		if VatPeriodReason(tc.year) == "" {
			t.Errorf("VatPeriodReason(%d) ist leer", tc.year)
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

func TestFoundationDutiesDependOnTheStage(t *testing.T) {
	rules, _ := FoundationRulesFor("GmbH")
	f := &domain.Foundation{NotarizedOn: "2026-09-15", ShareCapital: 2_500_000}

	open := FoundationDuties(f, rules, nil)
	keys := map[string]domain.FoundationDuty{}
	for _, d := range open {
		keys[d.Key] = d
	}
	if _, ok := keys[DutyFragebogen]; !ok {
		t.Error("der Fragebogen fehlt in der Vorgesellschaft")
	}
	if keys[DutyFragebogen].DueDate != "2026-10-15" {
		t.Errorf("Fragebogen fällig am %q, erwartet 2026-10-15", keys[DutyFragebogen].DueDate)
	}
	// Was die Eintragung voraussetzt, darf vorher nicht als Frist dastehen.
	if _, ok := keys[DutyTransparenzregister]; ok {
		t.Error("das Transparenzregister erscheint erst nach der Eintragung")
	}

	f.RegisteredOn = "2026-10-20"
	after := FoundationDuties(f, rules, map[string]string{DutyFragebogen: "2026-10-01"})
	found := false
	for _, d := range after {
		if d.Key == DutyTransparenzregister {
			found = true
		}
		if d.Key == DutyFragebogen {
			if !d.IsDone || d.DoneOn != "2026-10-01" {
				t.Error("der erledigte Fragebogen trägt sein Datum nicht")
			}
		}
	}
	if !found {
		t.Error("das Transparenzregister fehlt nach der Eintragung")
	}
}

func TestFoundationDutiesRuecklageOnlyForUG(t *testing.T) {
	f := &domain.Foundation{NotarizedOn: "2026-09-15"}

	gmbh, _ := FoundationRulesFor("GmbH")
	for _, d := range FoundationDuties(f, gmbh, nil) {
		if d.Key == DutyRuecklage {
			t.Error("die GmbH bildet keine Rücklage nach § 5a Abs. 3 GmbHG")
		}
	}

	ug, _ := FoundationRulesFor("UG (haftungsbeschränkt)")
	found := false
	for _, d := range FoundationDuties(f, ug, nil) {
		if d.Key == DutyRuecklage {
			found = true
		}
	}
	if !found {
		t.Error("der UG fehlt die gesetzliche Rücklage")
	}
}

func TestFoundationDutiesEmptyWithoutBeurkundung(t *testing.T) {
	rules, _ := FoundationRulesFor("GmbH")
	if got := FoundationDuties(&domain.Foundation{}, rules, nil); got != nil {
		t.Errorf("ohne Beurkundungsdatum gibt es keine Fristen, bekam %d", len(got))
	}
	if got := FoundationDuties(nil, rules, nil); got != nil {
		t.Error("ohne Gründung gibt es keine Fristen")
	}
}
