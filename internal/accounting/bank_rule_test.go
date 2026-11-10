package accounting

import (
	"strings"
	"testing"
)

// Das Muster einer gelernten Regel überlebt die wechselnden Teile des
// Verwendungszwecks.
func TestBankRulePatternIgnoresTheChangingParts(t *testing.T) {
	first := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro 03/2026 Vertrag 4711", false)
	second := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro 04/2026 Vertrag 4711", false)
	if first != second {
		t.Errorf("die Muster unterscheiden sich:\n%q\n%q", first, second)
	}
	if first == "" {
		t.Error("das Muster ist leer")
	}

	other := BankRulePattern("Stadtwerke", "Abschlag Strom 03/2026", false)
	if other == first {
		t.Error("zwei verschiedene Vorgänge dürfen nicht dasselbe Muster haben")
	}
}

// Der ausgeschriebene Monat wechselt und gehört deshalb nicht ins Muster.
//
// Der häufigste Fall überhaupt: Miete, Gehalt und Abschläge nennen den Monat im
// Klartext. Wer ihn ins Muster nähme, lernte jeden Monat eine neue Regel, die im
// Folgemonat nie wieder passt.
func TestBankRulePatternIgnoresWrittenMonthNames(t *testing.T) {
	march := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro März 2026", false)
	april := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro April 2026", false)
	if march != april {
		t.Errorf("die Muster unterscheiden sich:\n%q\n%q", march, april)
	}
	if march == "" {
		t.Fatal("das Muster ist leer")
	}
	// Abgekürzt ebenso, und der Vorgang bleibt im Muster stehen.
	if short := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro Sept 2026", false); short != march {
		t.Errorf("die abgekürzte Schreibweise ergibt ein anderes Muster:\n%q\n%q", short, march)
	}
	if !strings.Contains(march, "miete") {
		t.Errorf("das Muster nennt den Vorgang nicht: %q", march)
	}

	// Der Zahlungspartner lässt sich aus dem Muster zurücklesen — daran hängt
	// der Rückfall der Regelsuche.
	if got := BankRulePartnerOf(march); got != BankRulePartnerKey("Hausverwaltung Meier GmbH") {
		t.Errorf("Zahlungspartner aus dem Muster = %q, erwartet %q",
			got, BankRulePartnerKey("Hausverwaltung Meier GmbH"))
	}
	if BankRulePartnerOf(march) == march {
		t.Error("der Partnerteil darf nicht das ganze Muster sein")
	}

	// Die Geldrichtung gehört zum Schlüssel: die Rückerstattung desselben
	// Partners mit demselben Verwendungszweck ist ein anderer Vorgang und darf
	// die gelernte Regel des Ausgangs nicht überschreiben.
	incoming := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro März 2026", true)
	if incoming == march {
		t.Errorf("Geldeingang und Geldausgang haben dasselbe Muster: %q", march)
	}
	if BankRulePartnerOf(incoming) != BankRulePartnerKey("Hausverwaltung Meier GmbH") {
		t.Errorf("die Richtung verstellt den Partnerteil: %q", BankRulePartnerOf(incoming))
	}
	// Ohne Partner und ohne Verwendungszweck gibt es nichts zu lernen — die
	// Richtung allein ist kein Muster.
	if got := BankRulePattern("", "", true); got != "" {
		t.Errorf("Muster ohne kennzeichnenden Bestandteil = %q, erwartet leer", got)
	}
}
