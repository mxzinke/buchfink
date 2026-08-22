package einvoice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// examplesDir returns the corpus of official CII invoices, or skips.
//
// Die Dateien liegen nicht im Repository: sie gehören zum Validierungsartefakt
// der EU-Kommission und stehen unter EUPL-1.2, Buchfink unter MIT. `task
// test:en16931` holt sie und setzt die Umgebungsvariable.
func examplesDir(t *testing.T) string {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("EN16931_CII_EXAMPLES"))
	if dir == "" {
		t.Skip("EN16931_CII_EXAMPLES ist nicht gesetzt — `task test:en16931` holt die Beispiele")
	}
	return dir
}

func officialCIIFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(examplesDir(t), "*.xml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("keine Beispieldateien in %s gefunden (%v)", examplesDir(t), err)
	}
	return files
}

// Jede offizielle Beispieldatei muss lesbar sein. Ein Leser, der an einer
// normkonformen Rechnung scheitert, ist kein Leser.
func TestOfficialExamplesParse(t *testing.T) {
	for _, path := range officialCIIFiles(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Datei lesen: %v", err)
			}
			inv, err := ParseCII(data)
			if err != nil {
				t.Fatalf("XML lesen: %v", err)
			}

			// Die Pflichtangaben müssen ankommen — sonst hat der Leser sie
			// zwar überflogen, aber nicht abgebildet.
			if inv.Number == "" {
				t.Error("die Rechnungsnummer (BT-1) ist nicht angekommen")
			}
			if !inv.IssueDate.Present() {
				t.Error("das Rechnungsdatum (BT-2) ist nicht angekommen")
			}
			if inv.TypeCode == "" {
				t.Error("der Rechnungstyp (BT-3) ist nicht angekommen")
			}
			if inv.Currency == "" {
				t.Error("die Währung (BT-5) ist nicht angekommen")
			}
			if inv.Seller.Name == "" || inv.Buyer.Name == "" {
				t.Errorf("die Beteiligten fehlen: Verkäufer %q, Erwerber %q",
					inv.Seller.Name, inv.Buyer.Name)
			}
			if len(inv.Lines) == 0 {
				t.Error("keine Position gelesen")
			}
			if len(inv.VATBreakdowns()) == 0 {
				t.Error("keine Steueraufschlüsselung gelesen")
			}
			if !inv.Totals.GrandTotal.Present() {
				t.Error("der Gesamtbetrag (BT-112) ist nicht angekommen")
			}
			if inv.Syntax != SyntaxCII {
				t.Errorf("Syntax = %q", inv.Syntax)
			}
		})
	}
}

// Was der Korpus an Feldern trägt, wird auch gelesen. Der Test zählt, wie oft
// jedes Feld belegt ist, und hält fest, welche überhaupt vorkommen — sonst
// bliebe unbemerkt, dass eine ganze Gruppe nie ankommt.
func TestCorpusFillsTheModelBroadly(t *testing.T) {
	seen := map[string]int{}
	note := func(name string, ok bool) {
		if ok {
			seen[name]++
		}
	}

	files := officialCIIFiles(t)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Datei lesen: %v", err)
		}
		inv, err := ParseCII(data)
		if err != nil {
			t.Fatalf("%s: %v", filepath.Base(path), err)
		}

		note("BT-10 Käuferreferenz", inv.BuyerReference != "")
		note("BT-13 Bestellnummer", inv.OrderReference != "")
		note("BT-12 Vertragsnummer", inv.ContractReference != "")
		note("BT-9 Fälligkeitsdatum", inv.DueDate.Present())
		note("BT-20 Zahlungsbedingungen", inv.PaymentTermsNote != "")
		note("BG-1 Bemerkungen", len(inv.Notes) > 0)
		note("BG-3 Rechnungsbezug", len(inv.PrecedingInvoices) > 0)
		note("BG-14 Rechnungszeitraum", inv.Period.Present())
		note("BG-16 Zahlungsanweisung", len(inv.PaymentMeans) > 0)
		note("BG-20 Nachlässe", len(inv.Allowances) > 0)
		note("BG-21 Zuschläge", len(inv.Charges) > 0)
		note("BG-24 Unterlagen", len(inv.SupportingDocs) > 0)
		note("BT-31 USt-IdNr. Verkäufer", inv.Seller.VATIdentifier != "")
		note("BT-32 Steuernummer Verkäufer", inv.Seller.TaxRegistration != "")
		note("BT-30 Registernummer", inv.Seller.LegalRegistration.Present())
		note("BG-6 Kontakt", inv.Seller.Contact != nil)
		note("BT-34 elektronische Adresse", inv.Seller.ElectronicAddress.Present())
		note("BG-11 Steuervertreter", inv.TaxRepresentative != nil)
		note("BG-10 Zahlungsempfänger", inv.Payee != nil)
		note("BG-13 Lieferinformationen", inv.Delivery != nil)
		note("BT-6 Abrechnungswährung", inv.TaxCurrency != "")
		note("BT-111 Steuer in Abrechnungswährung", inv.Totals.TaxTotalInTaxCurr.Present())

		for _, l := range inv.Lines {
			note("BT-129 Menge", l.Quantity.Present())
			note("BT-130 Mengeneinheit", l.UnitCode != "")
			note("BT-146 Nettopreis", l.Price.NetPrice.Present())
			note("BT-148 Bruttopreis", l.Price.GrossPrice.Present())
			note("BT-149 Preisbasismenge", l.Price.BaseQuantity.Present())
			note("BG-26 Positionszeitraum", l.Period.Present())
			note("BG-27 Positionsnachlässe", len(l.Allowances) > 0)
			note("BG-28 Positionszuschläge", len(l.Charges) > 0)
			note("BT-155 Artikelnummer Verkäufer", l.Item.SellerID != "")
			note("BT-157 Standardkennung", l.Item.StandardID.Present())
			note("BT-158 Artikelklassifizierung", len(l.Item.Classifications) > 0)
			note("BT-159 Ursprungsland", l.Item.OriginCountryCode != "")
			note("BG-32 Artikelattribute", len(l.Item.Attributes) > 0)
			note("BT-127 Positionsfreitext", l.Note != "")
			note("BT-133 Buchungsreferenz", l.AccountingCost != "")
		}
	}

	t.Logf("%d Beispieldateien gelesen", len(files))
	for _, name := range sortedKeys(seen) {
		t.Logf("  %-38s %d Vorkommen", name, seen[name])
	}

	// Positionsfelder werden je Position gezählt, Kopffelder je Datei — die
	// Zahl sagt, wie oft der Korpus das Feld überhaupt vorführt.
	//
	// Die Gruppen, die der Buchungsflow heute nicht anfasst, sind der Grund für
	// dieses Modul. Kommen sie im Korpus vor und landen nicht im Modell, ist der
	// Leser unvollständig — und genau das würde erst bei einem echten Beleg
	// auffallen.
	for _, required := range []string{
		"BG-16 Zahlungsanweisung",
		"BG-20 Nachlässe",
		"BT-13 Bestellnummer",
		"BT-129 Menge",
		"BT-146 Nettopreis",
	} {
		if seen[required] == 0 {
			t.Errorf("%s kommt im Korpus vor, wird aber nie gelesen", required)
		}
	}
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
