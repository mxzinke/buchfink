package invoice

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Abgleich des Hybridformats (RECH-06 K4).
//
// Buchfink erzeugt PDF und XML aus demselben Datensatz — und liest das XML
// trotzdem zurück. Die Prüfung ist der Nachweis, dass die Erzeugung getan hat,
// was sie sollte; ein Dokument, dessen lesbarer und dessen strukturierter Teil
// verschiedene Beträge nennen, führt beim Empfänger zu einer anderen Buchung
// als beim Aussteller.
func TestEmbeddedRecordMatchesTheInvoice(t *testing.T) {
	inv := testInvoice()
	xml := renderFor(t, inv, testSeller(), testBuyer(), domain.EInvoiceProfileZUGFeRD)

	if err := VerifyEmbeddedRecord([]byte(xml), inv); err != nil {
		t.Fatalf("der unveränderte Datensatz muss durchgehen: %v", err)
	}
}

// Und mit einem manipulierten XML schlägt er an — vor der Ablage.
func TestManipulatedEmbeddedRecordIsRefused(t *testing.T) {
	inv := testInvoice()
	xml := renderFor(t, inv, testSeller(), testBuyer(), domain.EInvoiceProfileZUGFeRD)

	cases := []struct {
		name, from, to, expect string
	}{
		// Der Bruttobetrag: 1.800,00 € netto, 19 % Steuer, 2.142,00 € brutto.
		{"Bruttobetrag", ">2142.00<", ">2242.00<", "Bruttobetrag"},
		{"Rechnungsnummer", "RE-2026-0001", "RE-2026-0002", "Rechnungsnummer"},
		{"Rechnungsdatum", "20260301", "20260302", "Rechnungsdatum"},
	}
	for _, c := range cases {
		tampered := strings.Replace(xml, c.from, c.to, 1)
		if tampered == xml {
			t.Fatalf("%s: die Stelle %q steht nicht im erzeugten Datensatz", c.name, c.from)
		}
		err := VerifyEmbeddedRecord([]byte(tampered), inv)
		if err == nil {
			t.Errorf("%s: ein abweichender Datensatz muss vor der Ablage auffallen", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.expect) {
			t.Errorf("%s: die Meldung nennt die Abweichung nicht: %v", c.name, err)
		}
		if !strings.Contains(err.Error(), inv.InvoiceNumber) &&
			c.name != "Rechnungsnummer" {
			t.Errorf("%s: die Meldung nennt die Rechnung nicht: %v", c.name, err)
		}
	}

	// Gar kein Datensatz ist der schwerste Fall: ein Hybridformat ohne XML ist
	// keine E-Rechnung.
	if err := VerifyEmbeddedRecord(nil, inv); err == nil {
		t.Error("ein Dokument ohne Rechnungsdatensatz darf nicht durchgehen")
	}
}
