package domain

import (
	"testing"
)

// Das Nummernformat ist einstellbar, weil ein Mandant mit vorhandener
// Buchhaltung seine Systematik fortführen muss. Was es nicht darf, ist den
// Zähler verlieren: § 14 Abs. 4 Nr. 4 UStG verlangt eine einmalige,
// fortlaufende Nummer, und ein Format ohne {NR} gäbe jeder Rechnung dieselbe.
func TestInvoiceNumberFormat(t *testing.T) {
	cases := []struct {
		format string
		seq    int64
		want   string
	}{
		{DefaultInvoiceNumberFormat, 7, "RE-2026-0007"},
		{"{JAHR}-{NR:6}", 42, "2026-000042"},
		{"R{NR}", 5, "R5"},
		{"AR/{JAHR}/{NR:3}", 128, "AR/2026/128"},
		// Ein unbrauchbares Format fällt auf die Voreinstellung zurück, statt
		// eine Nummer ohne Zähler zu erzeugen.
		{"RE-{JAHR}", 7, "RE-2026-0007"},
	}
	for _, c := range cases {
		if got := FormatInvoiceNumberWith(c.format, 2026, c.seq); got != c.want {
			t.Errorf("FormatInvoiceNumberWith(%q, 2026, %d) = %q, erwartet %q", c.format, c.seq, got, c.want)
		}
	}

	if err := ValidateInvoiceNumberFormat("RE-{JAHR}"); err == nil {
		t.Error("ein Format ohne {NR} muss abgewiesen werden — jede Rechnung trüge dieselbe Nummer")
	}
	if err := ValidateInvoiceNumberFormat(""); err == nil {
		t.Error("ein leeres Nummernformat muss abgewiesen werden")
	}
	if err := ValidateInvoiceNumberFormat("{JAHR}-{NR:4}"); err != nil {
		t.Errorf("ein gültiges Format wurde abgewiesen: %v", err)
	}
}

// Der Lückenbericht muss aus der Nummer den Zählerstand zurücklesen, sonst
// vergleicht er nichts.
func TestParseInvoiceSequence(t *testing.T) {
	cases := []struct {
		number string
		format string
		want   int64
		ok     bool
	}{
		{"RE-2026-0007", DefaultInvoiceNumberFormat, 7, true},
		{"2026-000042", "{JAHR}-{NR:6}", 42, true},
		{"AR/2026/128", "AR/{JAHR}/{NR:3}", 128, true},
		// Ohne Trennzeichen zwischen Jahr und Zähler ist „20260007" eine
		// einzige Ziffernfolge. Nur das Format trennt sie richtig; die alte
		// Heuristik las daraus den Zähler 20260007 und machte aus jeder
		// vergebenen Nummer eine Lücke.
		{"20260007", "{JAHR}{NR:4}", 7, true},
		// Eine Nummer aus einem früheren Format passt nicht auf den heutigen
		// Ausdruck und wird trotzdem gelesen — sonst stünde sie als Lücke im
		// Bericht.
		{"AR/2026/128", DefaultInvoiceNumberFormat, 128, true},
		// Nur das Geschäftsjahr und sonst keine Ziffer: da ist kein Zähler.
		{"RE-2026", DefaultInvoiceNumberFormat, 0, false},
		{"ohne Ziffern", DefaultInvoiceNumberFormat, 0, false},
	}
	for _, c := range cases {
		got, ok := ParseInvoiceSequence(c.number, 2026, c.format)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseInvoiceSequence(%q, %q) = %d, %v — erwartet %d, %v",
				c.number, c.format, got, ok, c.want, c.ok)
		}
	}

	// Und der Lückenbericht mit demselben Format meldet keine Lücke.
	report := BuildNumberGapReport(2026, 3, []string{"20260001", "20260002"}, nil, "{JAHR}{NR:4}")
	if len(report.Gaps) != 0 {
		t.Errorf("der Bericht meldet %d Lücken, erwartet keine: %+v", len(report.Gaps), report.Gaps)
	}
}

// Der Bericht vergleicht Zählerstand und vergebene Nummern. Der Zähler ist der
// Maßstab: eine gelöschte Zeile ändert ihn nicht, und genau darum geht es.
func TestNumberGapReport(t *testing.T) {
	report := BuildNumberGapReport(2026, 5,
		[]string{"RE-2026-0001", "RE-2026-0003", "RE-2026-0004"},
		[]NumberGap{{Sequence: 2, Reason: NumberGapTest, Detail: "Probelauf", RecordedAt: "2026-03-01T10:00:00Z"}},
		DefaultInvoiceNumberFormat)

	if report.Issued != 4 {
		t.Errorf("ausgegeben = %d, erwartet 4", report.Issued)
	}
	if len(report.Gaps) != 1 {
		t.Fatalf("erwartet genau eine Lücke, erhalten %d", len(report.Gaps))
	}
	gap := report.Gaps[0]
	if gap.Sequence != 2 || gap.Number != "RE-2026-0002" {
		t.Errorf("die Lücke ist %d (%s), erwartet 2 (RE-2026-0002)", gap.Sequence, gap.Number)
	}
	if gap.Reason != NumberGapTest || gap.Label != "Probelauf" {
		t.Errorf("der dokumentierte Grund fehlt: %+v", gap)
	}

	// Ohne dokumentierten Grund bleibt die Lücke sichtbar — und als
	// unbegründet ausgewiesen. Das ist der Befund, um den es der Prüfung geht.
	plain := BuildNumberGapReport(2026, 3, []string{"RE-2026-0001"}, nil, DefaultInvoiceNumberFormat)
	if len(plain.Gaps) != 1 || plain.Gaps[0].Reason != NumberGapUnknown {
		t.Errorf("eine unbegründete Lücke muss als solche erscheinen: %+v", plain.Gaps)
	}
	// Eine leere Liste ist leer und nicht nil: sie geht als JSON ins Frontend.
	full := BuildNumberGapReport(2026, 2, []string{"RE-2026-0001"}, nil, DefaultInvoiceNumberFormat)
	if full.Gaps == nil {
		t.Error("die Lückenliste muss leer statt nil sein")
	}
}
