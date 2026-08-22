// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package domain

import "testing"

func TestParseCents(t *testing.T) {
	cases := []struct {
		in   string
		want Cents
	}{
		{"1234,56", 123456},
		{"1.234,56", 123456},
		{"1234.56", 123456},
		{"-42", -4200},
		{"0,01", 1},
		{"1.234.567,89", 123456789},
		{"  19,00 € ", 1900},
		{"+7,5", 750},
		{",5", 50},
	}
	for _, c := range cases {
		got, err := ParseCents(c.in)
		if err != nil {
			t.Errorf("ParseCents(%q) meldete einen Fehler: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseCents(%q) = %d, erwartet %d", c.in, got, c.want)
		}
	}
}

// A third decimal place is rejected rather than truncated: silently turning
// 10,555 € into 10,55 € would book an amount the user never entered.
func TestParseCentsRejectsSubCentPrecision(t *testing.T) {
	for _, in := range []string{"10,555", "1.2.3", "", "abc", "12,3x"} {
		if _, err := ParseCents(in); err == nil {
			t.Errorf("ParseCents(%q) hätte einen Fehler liefern müssen", in)
		}
	}
}

func TestCentsFormatting(t *testing.T) {
	cases := []struct {
		in            Cents
		german, plain string
	}{
		{123456, "1.234,56", "1234.56"},
		{-4200, "-42,00", "-42.00"},
		{5, "0,05", "0.05"},
		{0, "0,00", "0.00"},
		{123456789, "1.234.567,89", "1234567.89"},
	}
	for _, c := range cases {
		if got := c.in.String(); got != c.german {
			t.Errorf("%d.String() = %q, erwartet %q", c.in, got, c.german)
		}
		if got := c.in.Decimal(); got != c.plain {
			t.Errorf("%d.Decimal() = %q, erwartet %q", c.in, got, c.plain)
		}
	}
}

// The classic float trap: 0,10 € + 0,20 € must be exactly 0,30 €.
func TestSumIsExact(t *testing.T) {
	if got := SumCents(10, 20); got != 30 {
		t.Errorf("0,10 + 0,20 = %s, erwartet 0,30", got)
	}

	var total Cents
	for i := 0; i < 10; i++ {
		total += 10 // zehnmal 0,10 €
	}
	if total != 100 {
		t.Errorf("zehnmal 0,10 € = %s, erwartet 1,00", total)
	}
}

func TestMulRoundIsCommercial(t *testing.T) {
	cases := []struct {
		base     Cents
		num, den int64
		want     Cents
		what     string
	}{
		{10000, 1900, 10000, 1900, "19 % von 100,00 €"},
		{1999, 1900, 10000, 380, "19 % von 19,99 € rundet 3,7981 auf 3,80"},
		{1050, 1900, 10000, 200, "19 % von 10,50 € rundet 1,995 auf 2,00 (halber Cent aufwärts)"},
		{-1050, 1900, 10000, -200, "negative Beträge runden vom Betrag weg"},
		{333, 10000, 30000, 111, "ein Drittel von 3,33 €"},
		{10000, 0, 10000, 0, "Satz null"},
	}
	for _, c := range cases {
		if got := MulRound(c.base, c.num, c.den); got != c.want {
			t.Errorf("%s: erhalten %s, erwartet %s", c.what, got, c.want)
		}
	}
}

func TestTaxRateRoundTrip(t *testing.T) {
	// 19 % auf 100,00 € netto = 19,00 € Steuer, 119,00 € brutto.
	net := Cents(10000)
	tax := TaxRateStandard.Tax(net)
	if tax != 1900 {
		t.Fatalf("Steuer = %s, erwartet 19,00", tax)
	}
	if back := TaxRateStandard.NetFromGross(net + tax); back != net {
		t.Errorf("Nettorückrechnung aus 119,00 € = %s, erwartet %s", back, net)
	}

	if got := TaxRateReduced.Tax(10000); got != 700 {
		t.Errorf("7 %% von 100,00 € = %s, erwartet 7,00", got)
	}
	if got := TaxRateNone.Tax(10000); got != 0 {
		t.Errorf("0 %% von 100,00 € = %s, erwartet 0,00", got)
	}
}

func TestTaxRateLabel(t *testing.T) {
	if got := TaxRateStandard.Label(); got != "19 %" {
		t.Errorf("Label = %q, erwartet \"19 %%\"", got)
	}
	if got := TaxRateReduced.Label(); got != "7 %" {
		t.Errorf("Label = %q, erwartet \"7 %%\"", got)
	}
}
