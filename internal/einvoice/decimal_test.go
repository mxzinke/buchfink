package einvoice

import "testing"

// Nachkommastellen zählt die Schreibweise, nicht den Wert. Genau das braucht
// die BR-DEC-Familie: 1000.000 ist derselbe Wert wie 1000.00 und trotzdem ein
// Verstoß.
func TestDecimalsCountsWhatWasWritten(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 0},
		{"1000", 0},
		{"1000.0", 1},
		{"1000.00", 2},
		{"1000.000", 3},
		{"-1.5", 1},
		{"0.123456", 6},
	}
	for _, c := range cases {
		if got := NewAmount(c.raw).Decimals(); got != c.want {
			t.Errorf("Decimals(%q) = %d, erwartet %d", c.raw, got, c.want)
		}
	}
}

// Cents rundet nicht still weg, was zu genau ist — sonst verschwände der Fehler,
// den BR-DEC melden soll.
func TestCentsRefusesMoreThanTwoDecimals(t *testing.T) {
	if _, err := NewAmount("10.005").Cents(); err == nil {
		t.Error("drei Nachkommastellen hätten abgewiesen werden müssen")
	}
	got, err := NewAmount("10.50").Cents()
	if err != nil || got != 1050 {
		t.Errorf("10.50 = %v (%v), erwartet 1050", got, err)
	}
	if got, err := NewAmount("-0.01").Cents(); err != nil || got != -1 {
		t.Errorf("-0.01 = %v (%v), erwartet -1", got, err)
	}
	if got, err := NewAmount("7").Cents(); err != nil || got != 700 {
		t.Errorf("7 = %v (%v), erwartet 700", got, err)
	}
}

// Ein fehlender Betrag ist etwas anderes als null. BR-CO-13 behandelt eine
// fehlende Nachlasssumme als null, einen fehlenden Gesamtbetrag als Verstoß.
func TestAbsentIsNotZero(t *testing.T) {
	absent := NewAmount("  ")
	if absent.Present() {
		t.Error("Leerraum ist kein angegebener Betrag")
	}
	if absent.IsZero() {
		t.Error("ein fehlender Betrag ist nicht null")
	}
	if !NewAmount("0.00").IsZero() {
		t.Error("0.00 ist null")
	}
	if got, err := absent.CentsOr(0); err != nil || got != 0 {
		t.Errorf("CentsOr = %v (%v), erwartet 0", got, err)
	}
	if _, err := NewAmount("keine Zahl").CentsOr(0); err == nil {
		t.Error("ein unlesbarer Betrag darf nicht auf den Ersatzwert fallen")
	}
}

// Der Vergleich läuft über den Wert, nicht über die Schreibweise — sonst würde
// eine Position mit "19" nicht zur Steuergruppe mit "19.00" gezählt.
func TestEqualComparesValues(t *testing.T) {
	if !NewAmount("19").Equal(NewAmount("19.00")) {
		t.Error("19 und 19.00 sind derselbe Satz")
	}
	if NewAmount("19").Equal(NewAmount("7")) {
		t.Error("19 und 7 sind es nicht")
	}
}

// Steuersätze sind nicht immer ganzzahlig. Ein in Hundertstel geparster Satz
// würde 8,375 % vor der Multiplikation auf 8,38 % runden — je Position ein bis
// zwei Cent daneben.
func TestMulPercentKeepsFractionalRates(t *testing.T) {
	got, ok := MulPercent(100000, NewAmount("8.375"))
	if !ok {
		t.Fatal("8.375 % ist ein lesbarer Satz")
	}
	if got != 8375 {
		t.Errorf("8,375 %% von 1.000,00 = %s, erwartet 83,75", got)
	}

	if got, _ := MulPercent(10000, NewAmount("19.00")); got != 1900 {
		t.Errorf("19 %% von 100,00 = %s, erwartet 19,00", got)
	}
	if _, ok := MulPercent(100, NewAmount("")); ok {
		t.Error("ein fehlender Satz ergibt kein Ergebnis")
	}
}

// Kaufmännisch gerundet wird vom Betrag weg, in beide Richtungen gleich.
func TestRoundingIsCommercial(t *testing.T) {
	cases := []struct {
		base    Cents
		percent string
		want    Cents
	}{
		{1000, "0.5", 5},  // 5,0 → 5
		{101, "0.5", 1},   // 0,505 → 1
		{-101, "0.5", -1}, // -0,505 → -1
		{100, "0.5", 1},   // genau 0,5 → 1, nicht 0
		{-100, "0.5", -1}, // genau -0,5 → -1
		{300, "0.5", 2},   // 1,5 → 2
	}
	for _, c := range cases {
		got, ok := MulPercent(c.base, NewAmount(c.percent))
		if !ok || got != c.want {
			t.Errorf("%s × %s %% = %s, erwartet %s", c.base, c.percent, got, c.want)
		}
	}
}

// Was hineingeht, muss so wieder herauskommen — sonst weicht die erzeugte
// Rechnung von der gebuchten ab.
func TestAmountFromCentsRoundTrips(t *testing.T) {
	for _, c := range []Cents{0, 1, -1, 99, 100, -12345, 123456789} {
		got, err := AmountFromCents(c).Cents()
		if err != nil || got != c {
			t.Errorf("%d → %q → %v (%v)", c, AmountFromCents(c), got, err)
		}
	}
	if got := AmountFromCents(-5).String(); got != "-0.05" {
		t.Errorf("-5 Cent = %q, erwartet \"-0.05\"", got)
	}
}

// Die deutsche Darstellung ist für die Oberfläche, nicht für das XML.
func TestCentsFormatsGerman(t *testing.T) {
	cases := map[Cents]string{
		0: "0,00", 5: "0,05", 100: "1,00",
		123456: "1.234,56", -123456: "-1.234,56", 100000000: "1.000.000,00",
	}
	for in, want := range cases {
		if got := in.String(); got != want {
			t.Errorf("%d = %q, erwartet %q", int64(in), got, want)
		}
	}
}
