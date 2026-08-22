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

// § 27 Abs. 38 UStG kennt drei Fälle, und der mittlere hängt am Vorjahresumsatz
// des Ausstellers — den Buchfink nicht kennt. Deshalb ein Status, kein Ja/Nein.
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
