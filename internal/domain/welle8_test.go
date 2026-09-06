package domain

import "testing"

// Das Belegnummernformat folgt derselben Systematik wie das
// Rechnungsnummernformat (BEL-02 K4).
//
// Geprüft wird beides zusammen: die Vergabe und das Rücklesen. Ein Format, aus
// dem sich der Zähler nicht zurücklesen lässt, meldete im Lückenbericht jede
// vergebene Nummer als Lücke — und die Betriebsprüfung fragt nach genau diesen
// Zeilen.
func TestReceiptNumberFormat(t *testing.T) {
	cases := []struct {
		format string
		seq    int64
		want   string
	}{
		{"", 7, "ER-2026-0007"},
		{DefaultReceiptNumberFormat, 7, "ER-2026-0007"},
		{"BE-{JAHR}-{NR:5}", 7, "BE-2026-00007"},
		{"{JAHR}/AD/{NR:4}", 17, "2026/AD/0017"},
		{"BEL{NR}", 42, "BEL42"},
		// Ein untaugliches Format fällt auf die Voreinstellung zurück; die
		// Zurückweisung geschieht beim Speichern der Einstellung, nicht hier —
		// ein Beleg darf an einer Einstellung nicht scheitern.
		{"BE-{JAHR}", 7, "ER-2026-0007"},
	}
	for _, c := range cases {
		if got := FormatReceiptNumberWith(c.format, 2026, c.seq); got != c.want {
			t.Errorf("Format %q ergibt %q, erwartet %q", c.format, got, c.want)
		}
	}

	// Rücklesen: aus der Nummer wird wieder der Zähler.
	for _, c := range []struct {
		format, number string
		want           int64
	}{
		{DefaultReceiptNumberFormat, "ER-2026-0007", 7},
		{"BE-{JAHR}-{NR:5}", "BE-2026-00007", 7},
		{"{JAHR}/AD/{NR:4}", "2026/AD/0017", 17},
		// Der Zähler läuft über die Stellenzahl des Formats hinaus.
		{"BE-{JAHR}-{NR:4}", "BE-2026-12345", 12345},
	} {
		got, ok := ParseReceiptSequence(c.number, 2026, c.format)
		if !ok || got != c.want {
			t.Errorf("aus %q (Format %q) gelesen: %d (%v), erwartet %d",
				c.number, c.format, got, ok, c.want)
		}
	}

	// Ein Format ohne Zähler ist kein Nummernkreis.
	if err := ValidateNumberFormat("BE-{JAHR}"); err == nil {
		t.Error("ohne {NR} hätte jeder Beleg dieselbe Nummer")
	}
	if err := ValidateNumberFormat(""); err == nil {
		t.Error("ein leeres Format ist keines")
	}
	if err := ValidateNumberFormat(DefaultReceiptNumberFormat); err != nil {
		t.Errorf("die Voreinstellung muss gültig sein: %v", err)
	}
}

// Die Klärungsliste liefert leere Listen und nicht nil: die Oberfläche läuft
// über sie.
func TestReceiptFindingsEnsureLists(t *testing.T) {
	f := ReceiptFindings{}
	f.EnsureLists()
	if f.Groups == nil {
		t.Fatal("die Gruppen dürfen nicht nil sein")
	}
	f = ReceiptFindings{Groups: []ValidationFindingGroup{{Class: ValidationClassFormat}}}
	f.EnsureLists()
	if f.Groups[0].Findings == nil {
		t.Error("auch die Befunde einer Gruppe sind eine leere Liste")
	}
	for _, class := range AllValidationFindingClasses() {
		if class.Label() == "" {
			t.Errorf("die Klasse %q hat keinen Klartext", class)
		}
	}
}
