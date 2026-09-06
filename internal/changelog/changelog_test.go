package changelog

import (
	"strings"
	"testing"
)

// Die Historie wird in der Oberfläche gezeigt und liegt dem Prüferpaket als
// Markdown bei. Beide Wege lesen dieselbe Datei; bricht das Zerlegen, steht in
// der Oberfläche eine leere Liste, während die Datei voll ist — und niemand
// merkt es, weil die Datei ja stimmt.
func TestEntriesLiestDieFassungen(t *testing.T) {
	entries := Entries()
	if len(entries) == 0 {
		t.Fatal("die Historie ist leer; die eingebettete Datei wurde nicht gelesen")
	}

	for _, entry := range entries {
		if entry.Version == "" {
			t.Errorf("Fassung ohne Bezeichnung: %+v", entry)
		}
		if entry.Date == "" {
			t.Errorf("Fassung %s ohne Datum", entry.Version)
		}
		if len(entry.Changes) == 0 {
			t.Errorf("Fassung %s ohne eine einzige Änderung", entry.Version)
		}
	}

	// Der Absatz über der Tabelle beschreibt die Datei und ist keine Fassung.
	// Ohne eigene Prüfung landete er als Zusammenfassung der ersten.
	if strings.Contains(entries[0].Summary, "UNV-06") {
		t.Errorf("die Beschreibung der Datei steht als Zusammenfassung der ersten Fassung: %q",
			entries[0].Summary)
	}
}

// Die Überschrift trennt Fassung und Datum. Ein Trennstrich, der nicht erkannt
// wird, machte aus „v0.1 — 2026-09-06" eine Fassung dieses Namens.
func TestSplitHeadline(t *testing.T) {
	cases := []struct {
		in      string
		version string
		date    string
	}{
		{"v0.1 — 2026-09-06", "v0.1", "2026-09-06"},
		{"v0.2 - 2026-10-01", "v0.2", "2026-10-01"},
		{"v0.3", "v0.3", ""},
	}
	for _, c := range cases {
		version, date := splitHeadline(c.in)
		if version != c.version || date != c.date {
			t.Errorf("%q → %q / %q, erwartet %q / %q", c.in, version, date, c.version, c.date)
		}
	}
}

// Ein Umbruch in der Datei ist keine neue Änderung: die Zeilen sind auf 80
// Zeichen gebrochen, und jeder Punkt läuft über mehrere davon.
func TestUmbruchIstKeineNeueAenderung(t *testing.T) {
	for _, entry := range Entries() {
		for _, change := range entry.Changes {
			if strings.HasSuffix(strings.TrimSpace(change), ",") {
				t.Errorf("abgeschnittene Änderung in %s: %q", entry.Version, change)
			}
		}
	}
}

// Markdown bleibt daneben bestehen: das Prüferpaket bekommt die Datei und
// nicht die zerlegte Fassung.
func TestMarkdownBleibtVollstaendig(t *testing.T) {
	if len(Markdown()) < 100 {
		t.Fatal("das eingebettete Markdown fehlt oder ist leer")
	}
}
