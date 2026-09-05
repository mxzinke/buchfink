package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Abzinsung des § 253 Abs. 2 Satz 1 HGB: 10.000 € in drei Jahren, mit
// 1,5 % abgezinst, sind heute 10.000 / 1,015³ = 9.563,17 €.
func TestPresentValueDiscountsWithTheTableRate(t *testing.T) {
	got := PresentValue(1_000_000, 3, 15_000)
	if got != 956_317 {
		t.Errorf("Barwert %s € — erwartet 9.563,17 €", got)
	}
}

// Ohne Satz wird nicht abgezinst. Das ist keine Nachlässigkeit, sondern der
// einzige ehrliche Umgang mit einer fehlenden Eingangsgröße: ein geschätzter
// Zins sähe aus wie ein echter.
func TestPresentValueWithoutRateKeepsTheAmount(t *testing.T) {
	if got := PresentValue(1_000_000, 3, 0); got != 1_000_000 {
		t.Errorf("ohne Zinssatz %s € — erwartet den unveränderten Erfüllungsbetrag", got)
	}
}

// Abgezinst wird erst ab einer Restlaufzeit von mehr als einem Jahr.
func TestNeedsDiscountingStartsAfterOneYear(t *testing.T) {
	cases := map[string]bool{
		"2027-06-30": false, // ein halbes Jahr
		"2027-12-31": false, // genau ein Jahr
		"2028-01-02": true,  // mehr als ein Jahr
	}
	for due, want := range cases {
		if got := NeedsDiscounting("2026-12-31", due); got != want {
			t.Errorf("Fälligkeit %s: abzinsen = %v, erwartet %v", due, got, want)
		}
	}
}

// Die Restlaufzeit wird auf volle Jahre aufgerundet: die Tabelle der Bundesbank
// kennt keinen Satz für 2,4 Jahre, und die längere Laufzeit trägt den
// vorsichtigeren Satz.
func TestRemainingYearsRoundsUp(t *testing.T) {
	cases := map[string]int{
		"2026-12-31": 0,
		"2027-06-30": 1,
		"2027-12-31": 1,
		"2028-06-30": 2,
		// Genau drei Jahre — über das Schaltjahr 2028 hinweg 1096 Tage. In
		// 365-Tage-Schritten gerechnet wären das vier Jahre und damit der Satz
		// einer Laufzeit, die es nicht gibt.
		"2029-12-31": 3,
		"2030-12-31": 4,
		// Ein einziger Tag darüber rundet auf das nächste volle Jahr auf.
		"2030-01-01": 4,
	}
	for due, want := range cases {
		got, err := RemainingYears("2026-12-31", due)
		if err != nil {
			t.Fatalf("%s: %v", due, err)
		}
		if got != want {
			t.Errorf("Restlaufzeit bis %s = %d Jahre, erwartet %d", due, got, want)
		}
	}

	// Der Schalttag selbst darf die Rechnung nicht verschieben.
	if got, err := RemainingYears("2028-02-29", "2031-02-28"); err != nil || got != 3 {
		t.Errorf("Restlaufzeit vom Schalttag bis zum 28.02.2031 = %d Jahre (%v), erwartet 3", got, err)
	}
}

// Fehlt der Satz für eine Restlaufzeit, meldet die Suche das und rät nicht.
func TestDiscountRateForReportsAMissingRate(t *testing.T) {
	rates := []domain.DiscountRate{{Month: "2026-12", Years: 3, RateMicros: 15_000, Average: 7}}
	if _, ok := DiscountRateFor(rates, 3, 7); !ok {
		t.Error("der hinterlegte Satz für drei Jahre wurde nicht gefunden")
	}
	if _, ok := DiscountRateFor(rates, 5, 7); ok {
		t.Error("für fünf Jahre gibt es keinen Satz; gefunden werden darf keiner")
	}
	if _, ok := DiscountRateFor(rates, 3, 10); ok {
		t.Error("der Siebenjahresdurchschnitt darf nicht als Zehnjahresdurchschnitt gelten")
	}
}

// Jede Art des § 249 HGB bekommt ihr eigenes Bilanzkonto — sonst ließen sich
// die Arten im Rückstellungsspiegel nicht mehr trennen.
func TestProvisionAccountsAreDistinctPerKind(t *testing.T) {
	seen := map[string]domain.ProvisionKind{}
	for _, kind := range domain.AllProvisionKinds() {
		balance, expense := ProvisionAccounts(kind)
		if balance == "" || expense == "" {
			t.Fatalf("%s: Konten fehlen", kind)
		}
		if other, ok := seen[balance]; ok {
			t.Errorf("%s und %s teilen sich das Bilanzkonto %s", kind, other, balance)
		}
		seen[balance] = kind
	}
}

// Der Rückstellungsspiegel geht auf: Endbestand = Anfangsbestand + Zuführung
// + Aufzinsung − Verbrauch − Auflösung.
func TestProvisionMirrorAddsUp(t *testing.T) {
	provisions := []domain.Provision{{
		ID: 1, FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Abschluss 2026",
		Movements: []domain.ProvisionMovement{
			{Kind: domain.ProvisionFormation, FiscalYear: 2026, Amount: 300_000},
			{Kind: domain.ProvisionIncrease, FiscalYear: 2027, Amount: 50_000},
			{Kind: domain.ProvisionConsumption, FiscalYear: 2027, Amount: 320_000},
			{Kind: domain.ProvisionRelease, FiscalYear: 2027, Amount: 30_000},
		},
	}}

	mirror := BuildProvisionMirror(provisions, 2027)
	if len(mirror.Rows) != 1 {
		t.Fatalf("erwartet eine Zeile, bekommen %d", len(mirror.Rows))
	}
	row := mirror.Rows[0]
	if row.Opening != 300_000 {
		t.Errorf("Anfangsbestand %s € — erwartet 3.000,00 €", row.Opening)
	}
	if row.Additions != 50_000 || row.Used != 320_000 || row.Released != 30_000 {
		t.Errorf("Bewegungen: Zuführung %s €, Verbrauch %s €, Auflösung %s €",
			row.Additions, row.Used, row.Released)
	}
	want := row.Opening + row.Additions + row.Unwinding - row.Used - row.Released
	if row.Closing != want {
		t.Errorf("Endbestand %s € — die Zeile geht nicht auf, erwartet %s €", row.Closing, want)
	}
	if row.Closing != 0 {
		t.Errorf("Endbestand %s € — die Rückstellung ist verbraucht und aufgelöst, erwartet null", row.Closing)
	}
	if mirror.Total.Closing != row.Closing {
		t.Errorf("Summenzeile %s € weicht von der einzigen Zeile %s € ab",
			mirror.Total.Closing, row.Closing)
	}
}
