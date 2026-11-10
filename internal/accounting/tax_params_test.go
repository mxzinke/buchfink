package accounting

import "testing"

// Die steuerlichen Werte sind nach Gültigkeitszeitraum geschlüsselt: ein
// nachbearbeitetes Altjahr muss mit den Werten rechnen, die damals galten.
func TestTaxParametersAreDated(t *testing.T) {
	cases := []struct {
		date      string
		wantLimit int64
	}{
		{"2016-12-31", 15000},
		{"2017-01-01", 25000},
		{"2026-08-22", 25000},
	}
	for _, tc := range cases {
		params, err := TaxParametersFor(tc.date)
		if err != nil {
			t.Fatalf("%s: %v", tc.date, err)
		}
		if int64(params.SmallAmountInvoiceLimit) != tc.wantLimit {
			t.Errorf("%s: Kleinbetragsgrenze = %s, erwartet %d Cent",
				tc.date, params.SmallAmountInvoiceLimit, tc.wantLimit)
		}
		if params.EntertainmentDeductiblePermille != 700 {
			t.Errorf("%s: Bewirtungsquote = %d ‰, erwartet 700 ‰",
				tc.date, params.EntertainmentDeductiblePermille)
		}
	}

	if _, err := TaxParametersFor("1990-01-01"); err == nil {
		t.Error("für ein Datum vor dem ersten Parametersatz darf nicht geraten werden")
	}
	if _, err := TaxParametersFor(""); err == nil {
		t.Error("ohne Datum darf kein Parametersatz geliefert werden")
	}
}

// § 23 Abs. 1 KStG senkt den Satz ab dem Veranlagungszeitraum 2028 jährlich um
// einen Prozentpunkt; § 52 Abs. 18b EStG lässt den Verlustvortrag über eine
// Million Euro für 2024 bis 2027 zu 70 % zu, davor und danach zu 60 %;
// § 16 Abs. 4 Satz 2 GewStG hebt den Mindesthebesatz ab 2027 auf 280 %.
func TestCorporateTaxRateAndLossRulesFollowTheAssessmentYear(t *testing.T) {
	cases := []struct {
		date                  string
		corporate, loss, rate int64
	}{
		{"2023-12-31", 150, 600, 200},
		{"2024-12-31", 150, 700, 200},
		{"2026-12-31", 150, 700, 200},
		{"2027-12-31", 150, 700, 280},
		{"2028-06-30", 140, 600, 280},
		{"2029-12-31", 130, 600, 280},
		{"2030-12-31", 120, 600, 280},
		{"2031-12-31", 110, 600, 280},
		{"2032-12-31", 100, 600, 280},
		{"2040-12-31", 100, 600, 280},
	}
	for _, tc := range cases {
		params, err := TaxParametersFor(tc.date)
		if err != nil {
			t.Fatalf("%s: %v", tc.date, err)
		}
		if params.CorporateTaxPermille != tc.corporate || params.LossDeductionExcessPermille != tc.loss ||
			params.TradeTaxMinRatePercent != tc.rate {
			t.Errorf("%s: KSt %d ‰, Verlustabzug %d ‰, Mindesthebesatz %d %% — erwartet %d, %d, %d",
				tc.date, params.CorporateTaxPermille, params.LossDeductionExcessPermille,
				params.TradeTaxMinRatePercent, tc.corporate, tc.loss, tc.rate)
		}
		if params.LossDeductionBase != 100_000_000 || params.TradeLossExcessPermille != 600 ||
			params.ExemptIncomeNonDeductiblePermille != 50 {
			t.Errorf("%s: unveränderliche Werte fehlen: %+v", tc.date, params)
		}
	}
	for i := 1; i < len(taxParameterSets); i++ {
		if taxParameterSets[i-1].ValidFrom >= taxParameterSets[i].ValidFrom {
			t.Errorf("die Sätze sind nicht aufsteigend geordnet: %s vor %s",
				taxParameterSets[i-1].ValidFrom, taxParameterSets[i].ValidFrom)
		}
	}
}

// § 27 Abs. 38 UStG kennt drei Fälle, und der mittlere richtet sich nach dem
// Vorjahresumsatz des Ausstellers — den Buchfink nicht kennt. Deshalb ein
// Status, kein Ja/Nein.
func TestEInvoiceTransitionFollowsTheDeadlines(t *testing.T) {
	cases := map[string]EInvoiceTransition{
		"2025-06-30": EInvoiceTransitionAllowed,
		"2026-12-31": EInvoiceTransitionAllowed,
		"2027-01-01": EInvoiceTransitionConditional,
		"2027-12-31": EInvoiceTransitionConditional,
		"2028-01-01": EInvoiceTransitionExpired,
	}
	for date, want := range cases {
		if got := EInvoiceTransitionFor(date); got != want {
			t.Errorf("%s: %q, erwartet %q", date, got, want)
		}
	}
}

// § 23 Abs. 1 KStG: 15 % bis 2027, dann ein Punkt weniger je
// Veranlagungszeitraum bis 10 % ab 2032. Gewählt wird nach dem Kalenderjahr
// des Datums, für den Abschluss also nach dem Ende des Wirtschaftsjahres.
func TestCorporateTaxRateFollowsTheAssessmentYear(t *testing.T) {
	cases := map[string]int64{
		"2007-12-31": 250,
		"2008-12-31": 150,
		"2027-12-31": 150,
		"2028-01-01": 140,
		"2028-12-31": 140,
		"2029-06-30": 130,
		"2030-12-31": 120,
		"2031-12-31": 110,
		"2032-12-31": 100,
		"2040-12-31": 100,
	}
	for date, want := range cases {
		params, err := TaxParametersFor(date)
		if err != nil {
			t.Fatalf("%s: %v", date, err)
		}
		if params.CorporateTaxPermille != want {
			t.Errorf("%s: Körperschaftsteuersatz %d ‰, erwartet %d ‰", date, params.CorporateTaxPermille, want)
		}
	}
	if _, err := CorporateTaxPermilleFor(2006); err == nil {
		t.Error("vor 2007 ist kein Satz hinterlegt; geraten wird nicht")
	}
}
