package einvoice

import "testing"

// CII schreibt Datumsangaben als JJJJMMTT mit Formatschlüssel, UBL als ISO.
// Das Modell führt beide auf dieselbe Form zurück, sonst ließen sich Belege aus
// den zwei Syntaxen nicht vergleichen.
func TestBothSyntaxesLandOnTheSameDate(t *testing.T) {
	fromCII := NewDateFromFormat("20260315", "102")
	fromUBL := NewDate("2026-03-15")

	if fromCII.ISO() != "2026-03-15" || fromUBL.ISO() != "2026-03-15" {
		t.Fatalf("CII %q, UBL %q", fromCII.ISO(), fromUBL.ISO())
	}
	if fromCII.CII() != "20260315" {
		t.Errorf("Rückweg nach CII = %q", fromCII.CII())
	}
	if fromCII.German() != "15.03.2026" {
		t.Errorf("deutsche Darstellung = %q", fromCII.German())
	}
}

// Ein unlesbares Datum verschwindet nicht — es wird als das gemeldet, was
// ankam. Sonst stünde in der Meldung ein leeres Feld statt der Ursache.
func TestUnreadableDateKeepsWhatArrived(t *testing.T) {
	cases := []Date{
		NewDate("15.03.2026"),                  // ISO erwartet, deutsch geliefert
		NewDateFromFormat("2026-03-15", "102"), // Formatschlüssel 102 verlangt JJJJMMTT
		NewDateFromFormat("202603", "610"),     // Monatsformat, in EN 16931 nicht zulässig
		NewDate("20260231"),                    // 31. Februar
	}
	for _, d := range cases {
		if !d.Present() {
			t.Errorf("%q gilt als angegeben", d.Raw())
		}
		if d.Valid() {
			t.Errorf("%q ist kein gültiges Datum", d.Raw())
		}
		if d.String() == "" {
			t.Errorf("%q darf nicht spurlos verschwinden", d.Raw())
		}
	}
}

// BR-29 und BR-30 vergleichen Anfang und Ende eines Zeitraums.
func TestOrderingNeedsTwoReadableDates(t *testing.T) {
	start := NewDate("2026-01-01")
	end := NewDate("2026-12-31")

	if !start.Before(end) || !end.After(start) {
		t.Error("Januar liegt vor Dezember")
	}
	if start.Before(start) {
		t.Error("gleich ist nicht davor")
	}
	// Ein unlesbares Datum ordnet sich nicht ein: es hat eine eigene Regel.
	if start.Before(NewDate("kaputt")) || NewDate("kaputt").Before(start) {
		t.Error("ein unlesbares Datum ordnet sich nicht ein")
	}
}

func TestGermanInputIsAccepted(t *testing.T) {
	d, err := DateFromGerman("15.03.2026")
	if err != nil || d.ISO() != "2026-03-15" {
		t.Fatalf("= %q (%v)", d.ISO(), err)
	}
	if _, err := DateFromGerman("2026-03-15"); err == nil {
		t.Error("ISO ist hier nicht das erwartete Format")
	}
	if d, err := DateFromGerman(""); err != nil || d.Present() {
		t.Error("leer bleibt leer, ohne Fehler")
	}
}
