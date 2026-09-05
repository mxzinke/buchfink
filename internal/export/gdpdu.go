package export

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
)

// DTDFileName ist der Name, unter dem die Grammatik der Beschreibungsdatei
// mitgeliefert wird. Die index.xml nennt ihn im DOCTYPE.
//
// Name und Inhalt sind der amtlichen Fassung entnommen: Prüfsoftware erwartet
// genau diesen Namen, und eine Datei, die den amtlichen Namen trägt, muss auch
// den amtlichen Text führen — sonst prüfte der Prüfer gegen eine Grammatik, die
// niemand vereinbart hat.
const DTDFileName = "gdpdu-01-09-2004.dtd"

// DTDSHA256 ist die Prüfsumme des amtlichen Textes, wie er hier eingebettet
// liegt (Bundesministerium der Finanzen, Beschreibungsstandard für die
// Datenträgerüberlassung, Stand 01.09.2004, Zeilenenden CRLF wie ausgeliefert).
//
// Sie steht hier, damit eine Änderung an der Datei auffällt: die Grammatik ist
// vorgegeben und nicht zu pflegen. Wer sie anpassen wollte, um eine index.xml
// gültig zu bekommen, hätte die index.xml zu ändern und nicht die Grammatik.
const DTDSHA256 = "af3d4c5a19e991f2d8c53995bc708680bbd7ff9326fde539c55b7e2c63f848a2"

// IndexFileName ist der Name der Beschreibungsdatei. Prüfsoftware sucht genau
// diesen Namen im Wurzelverzeichnis des Datenträgers.
const IndexFileName = "index.xml"

// StandardVersion ist die Fassung des Beschreibungsstandards.
const StandardVersion = "1.0"

//go:embed gdpdu-01-09-2004.dtd
var dtdSource []byte

// DTD liefert die mitzuliefernde Grammatik: den amtlichen Text des
// Beschreibungsstandards für die Datenträgerüberlassung (Stand 01.09.2004).
//
// Er liegt jeder Überlassung bei, weil die index.xml ihn im DOCTYPE nennt: eine
// Beschreibungsdatei, deren Grammatik nur im Internet steht, lässt sich in zehn
// Jahren nicht mehr prüfen (§ 147 Abs. 1 i. V. m. Abs. 3 AO).
func DTD() []byte { return dtdSource }

// RenderIndexXML erzeugt die Beschreibungsdatei zu einem Dataset.
//
// Sie ist der eigentliche Gegenstand der Datenüberlassung: die CSV-Dateien sind
// ohne sie ein Haufen Semikolons. Für jede Spalte stehen hier Name, Erläuterung
// und Typ, und für jede Tabelle Trennzeichen, Dezimaltrennzeichen und der
// Zeitraum, für den die Daten gelten.
func RenderIndexXML(d *Dataset) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&b, "<!DOCTYPE DataSet SYSTEM %q>\n", DTDFileName)
	b.WriteString("<DataSet>\n")
	fmt.Fprintf(&b, "  <Version>%s</Version>\n", StandardVersion)

	b.WriteString("  <DataSupplier>\n")
	fmt.Fprintf(&b, "    <Name>%s</Name>\n", escape(d.TenantName))
	// Location ist der Standort des Datenlieferanten und nicht der Name des
	// Programms: der Prüfer soll der Beschreibungsdatei entnehmen können, aus
	// welchem Betrieb der Datenträger stammt. Die Programmfassung steht im
	// Comment, wo sie hingehört.
	fmt.Fprintf(&b, "    <Location>%s</Location>\n", escape(orUnknownLocation(d.SupplierLocation)))
	fmt.Fprintf(&b, "    <Comment>%s</Comment>\n", escape(fmt.Sprintf(
		"Datenüberlassung nach § 147 Abs. 6 AO, Geschäftsjahr %d, erzeugt am %s mit Buchfink %s. Die Feldbeschreibung steht in feldbeschreibung.md.",
		d.FiscalYear, d.CreatedAt, d.ProgramVersion)))
	b.WriteString("  </DataSupplier>\n")

	b.WriteString("  <Media>\n")
	fmt.Fprintf(&b, "    <Name>%s</Name>\n", escape(fmt.Sprintf("Buchführung %d", d.FiscalYear)))
	for i := range d.Tables {
		writeTable(&b, d, &d.Tables[i])
	}
	b.WriteString("  </Media>\n")
	b.WriteString("</DataSet>\n")
	return b.Bytes()
}

func writeTable(b *bytes.Buffer, d *Dataset, t *Table) {
	b.WriteString("    <Table>\n")
	fmt.Fprintf(b, "      <URL>%s</URL>\n", escape(t.FileName))
	fmt.Fprintf(b, "      <Name>%s</Name>\n", escape(t.Name))
	if t.Description != "" {
		fmt.Fprintf(b, "      <Description>%s</Description>\n", escape(t.Description))
	}
	if d.From != "" && d.To != "" {
		b.WriteString("      <Validity>\n")
		b.WriteString("        <Range>\n")
		fmt.Fprintf(b, "          <From>%s</From>\n", escape(d.From))
		fmt.Fprintf(b, "          <To>%s</To>\n", escape(d.To))
		b.WriteString("        </Range>\n")
		fmt.Fprintf(b, "        <Format>%s</Format>\n", DateFormat)
		b.WriteString("      </Validity>\n")
	}
	// Die Kodierungsangabe steht vor den Trennzeichen, weil das Inhaltsmodell
	// von Table sie dort erwartet: (URL, Name?, Description?, Validity?,
	// (ANSI | … | UTF8)?, (DecimalSymbol, DigitGroupingSymbol)?, …). Hinter den
	// Trennzeichen wäre die Datei gegen die amtliche Grammatik ungültig, und
	// eine Prüfsoftware, die validiert, wiese die Überlassung zurück.
	//
	// Ohne die Angabe wiederum läse eine Prüfsoftware die CSV-Dateien als ANSI,
	// und jeder Umlaut in Kontoname, Buchungstext oder Anschrift käme falsch
	// an. Die Dateien tragen keine BOM — die Angabe steht deshalb hier und
	// nirgends sonst.
	b.WriteString("      <UTF8/>\n")
	// Beide Trennzeichen oder keines: das Inhaltsmodell fasst sie zu einer
	// Gruppe zusammen.
	fmt.Fprintf(b, "      <DecimalSymbol>%s</DecimalSymbol>\n", DecimalSymbol)
	fmt.Fprintf(b, "      <DigitGroupingSymbol>%s</DigitGroupingSymbol>\n", DigitGroupingSymbol)

	b.WriteString("      <VariableLength>\n")
	fmt.Fprintf(b, "        <ColumnDelimiter>%s</ColumnDelimiter>\n", ColumnDelimiter)
	// CR und LF als Zeichenverweise: als rohe Bytes wären sie im XML nicht von
	// der Einrückung zu unterscheiden.
	b.WriteString("        <RecordDelimiter>&#13;&#10;</RecordDelimiter>\n")
	fmt.Fprintf(b, "        <TextEncapsulator>%s</TextEncapsulator>\n", escape(TextEncapsulator))
	for _, f := range t.Fields {
		writeColumn(b, f)
	}
	b.WriteString("      </VariableLength>\n")
	b.WriteString("    </Table>\n")
}

func writeColumn(b *bytes.Buffer, f Field) {
	b.WriteString("        <VariableColumn>\n")
	fmt.Fprintf(b, "          <Name>%s</Name>\n", escape(f.Name))
	if f.Description != "" {
		fmt.Fprintf(b, "          <Description>%s</Description>\n", escape(f.Description))
	}
	switch f.Type {
	case FieldNumeric:
		fmt.Fprintf(b, "          <Numeric><Accuracy>%d</Accuracy></Numeric>\n", amountAccuracy)
	case FieldDate:
		fmt.Fprintf(b, "          <Date><Format>%s</Format></Date>\n", DateFormat)
	default:
		// AlphaNumeric ist im Standard ein leeres Element, MaxLength sein
		// Geschwister und nicht sein Kind: VariableColumn (Name, Description?,
		// (Numeric | (AlphaNumeric, MaxLength?) | Date), Map*). Verschachtelt
		// wäre die Angabe gegen die amtliche Grammatik ungültig.
		b.WriteString("          <AlphaNumeric/>\n")
		if f.Length > 0 {
			fmt.Fprintf(b, "          <MaxLength>%d</MaxLength>\n", f.Length)
		}
	}
	b.WriteString("        </VariableColumn>\n")
}

// escape maskiert die fünf Zeichen, die in XML-Inhalten nicht roh stehen
// dürfen. Ein Firmenname mit einem „&" darf die Beschreibungsdatei nicht
// unlesbar machen.
func escape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	).Replace(s)
}

// orUnknownLocation füllt den Standort, wo die Unternehmensdaten keinen
// hergeben. Ein leeres Pflichtfeld wäre schlechter als ein ehrlicher Hinweis.
func orUnknownLocation(location string) string {
	if strings.TrimSpace(location) == "" {
		return "Sitz des Datenlieferanten nicht hinterlegt"
	}
	return location
}
