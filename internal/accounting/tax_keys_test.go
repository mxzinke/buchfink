package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Das Verzeichnis der Steuerschlüssel muss genau die Schlüssel enthalten, die
// der Steuerrechner erzeugt.
//
// Ein Schlüssel, der im Journal steht und im Verzeichnis fehlt, ist für einen
// Prüfer eine Zeichenkette ohne Bedeutung — und § 147 Abs. 6 AO verlangt, dass
// sich jede Verschlüsselung auflösen lässt. Umgekehrt beschriebe ein Eintrag
// ohne Schlüssel etwas, das es nicht gibt.
func TestTaxKeyCatalogCoversEveryProducedKey(t *testing.T) {
	resolver := NewSKR04TaxResolver()

	// Zwei Schlüssel entstehen nicht aus einem Steuerfall und stehen deshalb von
	// vornherein in der erwarteten Menge: der § 14c-Betrag wird ohne
	// steuerpflichtigen Umsatz geschuldet, und die Vorsteuerberichtigung nach
	// § 15a UStG bewertet die Verwendung eines früheren Jahres neu. Beide sind
	// trotzdem Schlüssel, die im Journal stehen — und damit auflösbar sein müssen.
	produced := map[string]bool{TaxKeyUnlawful: true, TaxKeyInputTaxCorrection: true}
	for _, dir := range []domain.Direction{domain.DirectionIncoming, domain.DirectionOutgoing} {
		for _, treatment := range domain.AllTaxTreatments() {
			for _, rate := range []domain.TaxRate{domain.TaxRateNone, domain.TaxRateReduced, domain.TaxRateStandard} {
				legs, err := resolver.Resolve(dir, treatment, rate, 100_00)
				if err != nil {
					continue
				}
				for _, leg := range legs {
					produced[leg.Key] = true
				}
			}
		}
	}
	if len(produced) < 5 {
		t.Fatalf("der Steuerrechner erzeugt nur %d Schlüssel — der Test prüft nichts", len(produced))
	}

	catalog := map[string]TaxKeyInfo{}
	for _, info := range TaxKeyCatalog() {
		catalog[info.Key] = info
	}

	for key := range produced {
		info, ok := catalog[key]
		if !ok {
			t.Errorf("der Steuerschlüssel %s fehlt im Verzeichnis", key)
			continue
		}
		if info.Label == "" {
			t.Errorf("der Steuerschlüssel %s hat keinen Klartext", key)
		}
		if info.Account == "" {
			t.Errorf("der Steuerschlüssel %s nennt kein Steuerkonto", key)
		}
	}
	for key := range catalog {
		if !produced[key] {
			t.Errorf("das Verzeichnis führt %s, den der Steuerrechner nie erzeugt", key)
		}
	}
}

// Jeder Schlüssel läuft in eine Kennziffer des Vordrucks — und in dieselbe, in
// die ihn auch die Voranmeldung einordnet. Zwei Zuordnungen, die auseinander
// laufen, ergäben eine Überlassung, die der eigenen Anmeldung widerspricht.
func TestTaxKeyCatalogMatchesVatReturn(t *testing.T) {
	for _, info := range TaxKeyCatalog() {
		if info.VatCode == "" {
			t.Errorf("der Steuerschlüssel %s nennt keine Kennziffer", info.Key)
			continue
		}
		if info.VatCodeLabel == "" {
			t.Errorf("die Kennziffer %s des Schlüssels %s hat keinen Wortlaut im Vordruck",
				info.VatCode, info.Key)
		}
	}

	// Die Zuordnung wird gegen die Auswertung gehalten: eine Buchung mit dem
	// Schlüssel muss in derselben Kennziffer landen, die das Verzeichnis nennt.
	for _, info := range TaxKeyCatalog() {
		if info.Key == TaxKeyUnlawful {
			continue
		}
		side := domain.SideCredit
		if info.Side == domain.SideDebit {
			side = domain.SideDebit
		}
		entry := domain.JournalEntry{
			BookingDate: "2026-03-01", DocumentDate: "2026-03-01",
			FiscalYear: 2026,
			Lines: []domain.JournalLine{{
				Position: 1, Side: side, Account: info.Account,
				Amount: 1900, TaxKey: info.Key, TaxBase: 10000,
			}},
		}
		period, err := VatPeriodOf("2026-03-01", "month")
		if err != nil {
			t.Fatalf("Zeitraum: %v", err)
		}
		ret := BuildVatReturn(period, VatReturnSource{Entries: []domain.JournalEntry{entry}})

		hit := false
		for _, line := range ret.Figures {
			if line.Code != info.VatCode {
				continue
			}
			if line.Tax != 0 || line.Base != 0 {
				hit = true
			}
		}
		if !hit {
			t.Errorf("eine Buchung mit dem Schlüssel %s landet nicht in der Kennziffer %s, die das Verzeichnis nennt",
				info.Key, info.VatCode)
		}
	}
}
