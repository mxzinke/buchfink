package einvoice

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Der Rundlauf: jede offizielle Rechnung wird gelesen, wieder geschrieben und
// erneut gelesen. Was dabei verloren geht, ginge auch beim Erzeugen einer
// eigenen Rechnung verloren — nur würde es dort niemandem auffallen.
func TestCIIRoundTripKeepsTheModel(t *testing.T) {
	for _, path := range officialCIIFiles(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Datei lesen: %v", err)
			}
			first, err := ParseCII(data)
			if err != nil {
				t.Fatalf("erstes Lesen: %v", err)
			}

			written, err := RenderCII(first)
			if err != nil {
				t.Fatalf("schreiben: %v", err)
			}
			second, err := ParseCII(written)
			if err != nil {
				t.Fatalf("zweites Lesen: %v\n%s", err, written)
			}

			for _, diff := range compareInvoices(first, second) {
				t.Errorf("Rundlauf verliert %s", diff)
			}
		})
	}
}

// Was Buchfink schreibt, muss die eigene Prüfung bestehen. Täte es das nicht,
// wäre entweder der Schreiber falsch oder die Prüfung — und man wüsste nicht,
// welches von beidem.
func TestWhatWeWriteStillValidates(t *testing.T) {
	for _, path := range officialCIIFiles(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Datei lesen: %v", err)
			}
			inv, err := ParseCII(data)
			if err != nil {
				t.Fatalf("lesen: %v", err)
			}
			written, err := RenderCII(inv)
			if err != nil {
				t.Fatalf("schreiben: %v", err)
			}
			reread, err := ParseCII(written)
			if err != nil {
				t.Fatalf("zurücklesen: %v", err)
			}

			before := fatalRules(Validate(inv))
			after := fatalRules(Validate(reread))
			if !reflect.DeepEqual(before, after) {
				t.Errorf("die Beurteilung ändert sich durch das Schreiben: vorher %v, nachher %v",
					before, after)
			}
		})
	}
}

// Eine UBL-Rechnung, durch das Modell nach CII geschrieben, muss dieselbe
// Beurteilung bekommen. Das ist die Zusage des semantischen Modells, und sie
// ist nachzuweisen statt zu behaupten.
func TestUBLConvertsToCIIWithoutChangingTheVerdict(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(ublExamplesDir(t), "*.[xX][mM][lL]"))
	if err != nil || len(files) == 0 {
		t.Fatalf("keine UBL-Beispiele (%v)", err)
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Datei lesen: %v", err)
			}
			fromUBL, err := ParseUBL(data)
			if err != nil {
				t.Fatalf("UBL lesen: %v", err)
			}
			written, err := RenderCII(fromUBL)
			if err != nil {
				t.Fatalf("als CII schreiben: %v", err)
			}
			fromCII, err := ParseCII(written)
			if err != nil {
				t.Fatalf("als CII zurücklesen: %v", err)
			}

			before := fatalRules(Validate(fromUBL))
			after := fatalRules(Validate(fromCII))
			if !reflect.DeepEqual(before, after) {
				t.Errorf("die Beurteilung wechselt mit der Syntax: als UBL %v, als CII %v",
					before, after)
			}
		})
	}
}

func fatalRules(r Result) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range r.Findings {
		if f.Severity == SeverityFatal && !seen[f.Rule] {
			seen[f.Rule] = true
			out = append(out, f.Rule)
		}
	}
	sortStrings(out)
	return out
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

// compareInvoices names the business terms that did not survive the round trip.
//
// It compares the model, not the XML: the writer is free to lay the document
// out differently, but not to lose or invent a value.
func compareInvoices(before, after *Invoice) []string {
	var diffs []string
	check := func(name string, a, b any) {
		if !reflect.DeepEqual(a, b) {
			diffs = append(diffs, name)
		}
	}

	check("BT-1 Rechnungsnummer", before.Number, after.Number)
	check("BT-2 Rechnungsdatum", before.IssueDate.ISO(), after.IssueDate.ISO())
	check("BT-3 Rechnungstyp", before.TypeCode, after.TypeCode)
	check("BT-5 Währung", before.Currency, after.Currency)
	check("BT-6 Abrechnungswährung", before.TaxCurrency, after.TaxCurrency)
	check("BT-9 Fälligkeitsdatum", before.DueDate.ISO(), after.DueDate.ISO())
	check("BT-10 Käuferreferenz", before.BuyerReference, after.BuyerReference)
	check("BT-11 Projektnummer", before.ProjectReference, after.ProjectReference)
	check("BT-12 Vertragsnummer", before.ContractReference, after.ContractReference)
	check("BT-13 Bestellnummer", before.OrderReference, after.OrderReference)
	check("BT-14 Auftragsnummer", before.SalesOrderReference, after.SalesOrderReference)
	check("BT-15 Wareneingangsmeldung", before.ReceivingAdviceReference, after.ReceivingAdviceReference)
	check("BT-16 Lieferavis", before.DespatchAdviceReference, after.DespatchAdviceReference)
	check("BT-17 Ausschreibungsnummer", before.TenderReference, after.TenderReference)
	check("BT-18 Objektkennung", before.ObjectIdentifier, after.ObjectIdentifier)
	check("BT-19 Buchungsreferenz", before.AccountingCost, after.AccountingCost)
	check("BT-20 Zahlungsbedingungen", before.PaymentTermsNote, after.PaymentTermsNote)
	check("BT-24 Spezifikation", before.SpecificationID, after.SpecificationID)
	check("BG-1 Bemerkungen", before.Notes, after.Notes)
	check("BG-3 Rechnungsbezug", before.PrecedingInvoices, after.PrecedingInvoices)
	check("BG-4 Verkäufer", before.Seller, after.Seller)
	check("BG-7 Erwerber", before.Buyer, after.Buyer)
	check("BG-10 Zahlungsempfänger", before.Payee, after.Payee)
	check("BG-11 Steuervertreter", before.TaxRepresentative, after.TaxRepresentative)
	check("BG-13 Lieferinformationen", before.Delivery, after.Delivery)
	check("BG-14 Rechnungszeitraum", before.Period, after.Period)
	check("BG-20 Nachlässe", before.Allowances, after.Allowances)
	check("BG-21 Zuschläge", before.Charges, after.Charges)
	check("BG-22 Summen", before.Totals, after.Totals)
	check("BG-23 Steueraufschlüsselung", before.VATBreakdown, after.VATBreakdown)
	check("BG-24 Unterlagen", before.SupportingDocs, after.SupportingDocs)

	if len(before.Lines) != len(after.Lines) {
		return append(diffs, "BG-25 Anzahl der Positionen")
	}
	for i := range before.Lines {
		if !reflect.DeepEqual(before.Lines[i], after.Lines[i]) {
			diffs = append(diffs, lineDiff(i, before.Lines[i], after.Lines[i]))
		}
	}
	return diffs
}

func lineDiff(i int, before, after Line) string {
	var parts []string
	add := func(name string, a, b any) {
		if !reflect.DeepEqual(a, b) {
			parts = append(parts, name)
		}
	}
	add("BT-126 Nummer", before.ID, after.ID)
	add("BT-127 Freitext", before.Note, after.Note)
	add("BT-128 Objektkennung", before.ObjectIdentifier, after.ObjectIdentifier)
	add("BT-129 Menge", before.Quantity, after.Quantity)
	add("BT-130 Einheit", before.UnitCode, after.UnitCode)
	add("BT-131 Nettobetrag", before.NetAmount, after.NetAmount)
	add("BT-132 Bestellposition", before.OrderLineID, after.OrderLineID)
	add("BT-133 Buchungsreferenz", before.AccountingCost, after.AccountingCost)
	add("BG-26 Zeitraum", before.Period, after.Period)
	add("BG-27 Nachlässe", before.Allowances, after.Allowances)
	add("BG-28 Zuschläge", before.Charges, after.Charges)
	add("BG-29 Preis", before.Price, after.Price)
	add("BG-30 Steuer", before.VAT, after.VAT)
	add("BG-31 Artikel", before.Item, after.Item)
	return "Position " + itoa(i+1) + ": " + strings.Join(parts, ", ")
}
