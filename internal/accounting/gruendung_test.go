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

// Jede Pflicht hängt an dem Ereignis, das sie auslöst. Was die Eintragung
// voraussetzt, steht schon vorher in der Liste — als wartender Posten ohne
// Datum, damit der Gründer sieht, was noch kommt, ohne dafür überfällig zu sein.
func TestFoundationDutiesDependOnTheStage(t *testing.T) {
	rules, _ := FoundationRulesFor("GmbH")
	f := &domain.Foundation{NotarizedOn: "2026-09-15", ShareCapital: 2_500_000}

	open := FoundationDuties(f, rules, nil)
	keys := map[string]domain.FoundationDuty{}
	for _, d := range open {
		keys[d.Key] = d
	}

	// Die Vorgesellschaft ist bereits Körperschaftsteuersubjekt: der Fragebogen
	// läuft ab der Beurkundung, nicht ab der Eintragung.
	fragebogen, ok := keys[DutyFragebogen]
	if !ok {
		t.Fatal("der Fragebogen fehlt in der Vorgesellschaft")
	}
	if fragebogen.Anchor != domain.AnchorBeurkundung {
		t.Errorf("Fragebogen hängt an %q, erwartet die Beurkundung", fragebogen.Anchor)
	}
	if fragebogen.DueDate != "2026-10-15" || fragebogen.IsPending {
		t.Errorf("Fragebogen fällig am %q (wartend: %v), erwartet 2026-10-15 und nicht wartend",
			fragebogen.DueDate, fragebogen.IsPending)
	}

	// Was die Eintragung voraussetzt, wartet auf sie — mit Anker, ohne Datum.
	for _, key := range []string{DutyTransparenzregister, DutyGewerbeanmeldung} {
		duty, ok := keys[key]
		if !ok {
			t.Errorf("%s fehlt in der Liste; wartende Pflichten gehören hinein", key)
			continue
		}
		if duty.Anchor != domain.AnchorEintragung {
			t.Errorf("%s hängt an %q, erwartet die Eintragung", key, duty.Anchor)
		}
		if !duty.IsPending {
			t.Errorf("%s muss vor der Eintragung warten", key)
		}
		if duty.DueDate != "" {
			t.Errorf("%s hat vor der Eintragung das Datum %q; erfunden wird keins", key, duty.DueDate)
		}
	}

	f.RegisteredOn = "2026-10-20"
	after := FoundationDuties(f, rules, map[string]string{DutyFragebogen: "2026-10-01"})
	seen := map[string]domain.FoundationDuty{}
	for _, d := range after {
		seen[d.Key] = d
	}
	if transparenz := seen[DutyTransparenzregister]; transparenz.IsPending {
		t.Error("nach der Eintragung wartet das Transparenzregister nicht mehr")
	}
	// § 20 GwG sagt „unverzüglich" — daraus wird kein Tagesdatum.
	if got := seen[DutyTransparenzregister].DueDate; got != "" {
		t.Errorf("das Transparenzregister hat das Datum %q; § 20 GwG nennt keine Tagesfrist", got)
	}
	gewerbe := seen[DutyGewerbeanmeldung]
	if gewerbe.IsPending || gewerbe.DueDate != "2026-11-20" {
		t.Errorf("Gewerbeanmeldung fällig am %q (wartend: %v), erwartet 2026-11-20 einen Monat nach der Eintragung",
			gewerbe.DueDate, gewerbe.IsPending)
	}
	if fb := seen[DutyFragebogen]; !fb.IsDone || fb.DoneOn != "2026-10-01" {
		t.Error("der erledigte Fragebogen hat sein Datum nicht")
	}
}

// Eine erledigte Pflicht wartet nicht mehr, auch wenn ihr Anker fehlt: das
// nachgewiesene Datum schlägt jede Erwartung darüber, wann etwas eintritt.
func TestFoundationDutiesDoneBeatsPending(t *testing.T) {
	rules, _ := FoundationRulesFor("GmbH")
	f := &domain.Foundation{NotarizedOn: "2026-09-15", ShareCapital: 2_500_000}

	duties := FoundationDuties(f, rules, map[string]string{DutyGewerbeanmeldung: "2026-09-20"})
	for _, d := range duties {
		if d.Key != DutyGewerbeanmeldung {
			continue
		}
		if d.IsPending {
			t.Error("eine erledigte Pflicht wartet nicht mehr")
		}
		if !d.IsDone || d.DoneOn != "2026-09-20" {
			t.Errorf("Erledigung = %v am %q, erwartet den 2026-09-20", d.IsDone, d.DoneOn)
		}
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
