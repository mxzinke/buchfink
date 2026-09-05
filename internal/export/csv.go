package export

import (
	"bytes"
	"strings"
)

// Die Festlegungen des CSV-Formats. Sie stehen hier und nicht verstreut, weil
// index.xml sie noch einmal ausdrücklich benennen muss: eine Datei, deren
// Trennzeichen nur die erzeugende Software kennt, ist keine Datenüberlassung.
const (
	// ColumnDelimiter ist das Semikolon. Das Komma scheidet aus, sobald ein
	// Feld einen Betrag oder einen deutschen Text trägt.
	ColumnDelimiter = ";"
	// RecordDelimiter ist CR LF nach RFC 4180.
	RecordDelimiter = "\r\n"
	// TextEncapsulator ist das doppelte Anführungszeichen.
	TextEncapsulator = `"`
	// DecimalSymbol trennt die Nachkommastellen.
	DecimalSymbol = "."
	// DigitGroupingSymbol muss der Standard nennen, obwohl Buchfink nicht
	// gruppiert. Ein Wert, der dem Dezimaltrennzeichen gleicht, machte jede
	// Zahl mehrdeutig — deshalb steht hier das Komma und nicht der Punkt.
	DigitGroupingSymbol = ","
)

// RenderCSV erzeugt die Datei einer Tabelle: eine Kopfzeile mit den
// Spaltennamen, danach die Zeilen.
//
// UTF-8 ohne BOM. Die BOM ist keine Kodierungsangabe, sondern ein Zeichen am
// Dateianfang: eine Prüfsoftware, die sie nicht erwartet, liest den ersten
// Spaltennamen mit drei unsichtbaren Bytes davor und findet die Spalte nicht
// wieder. Die Kodierung steht in index.xml, wo sie hingehört.
func RenderCSV(t Table) []byte {
	var buf bytes.Buffer
	writeRow(&buf, t.FieldNames())
	for _, row := range t.Rows {
		writeRow(&buf, row)
	}
	return buf.Bytes()
}

func writeRow(buf *bytes.Buffer, values []string) {
	for i, v := range values {
		if i > 0 {
			buf.WriteString(ColumnDelimiter)
		}
		buf.WriteString(quote(v))
	}
	buf.WriteString(RecordDelimiter)
}

// quote setzt ein Feld nach RFC 4180 in Anführungszeichen, wo es nötig ist, und
// verdoppelt darin enthaltene Anführungszeichen.
//
// Nötig ist es bei Trennzeichen, Zeilenumbruch und Anführungszeichen selbst.
// Führende oder folgende Leerzeichen kommen dazu: ohne Anführungszeichen ist
// nicht mehr zu erkennen, ob sie zum Wert gehören oder zur Formatierung.
func quote(v string) string {
	needsQuotes := strings.ContainsAny(v, ";\"\r\n") ||
		strings.HasPrefix(v, " ") || strings.HasSuffix(v, " ")
	if !needsQuotes {
		return v
	}
	return TextEncapsulator + strings.ReplaceAll(v, `"`, `""`) + TextEncapsulator
}
