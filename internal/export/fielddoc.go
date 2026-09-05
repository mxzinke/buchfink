package export

import (
	"bytes"
	"fmt"
	"strings"
)

// KeyDirectoryTable ist der Name der Tabelle mit den Schlüsselwerten. Die
// Feldbeschreibung liest sie, statt eine zweite Liste zu führen: zwei
// Verzeichnisse derselben Codes sind zwei Gelegenheiten, eines zu vergessen.
const KeyDirectoryTable = "schluesselverzeichnis"

// FieldDocFileName ist der Name der Feldbeschreibung.
const FieldDocFileName = "feldbeschreibung.md"

// canonicalFormDoc beschreibt die Kanonisierung der Hash-Chain so, dass sie
// außerhalb von Buchfink nachgebaut werden kann.
//
// Ohne diesen Abschnitt ist die Kette eine Behauptung: die Spalten Vorgängerhash
// und Eigenhash stehen in der Datei, aber niemand kann prüfen, ob sie zu den
// Daten passen. Der Text muss deshalb jede Eigenheit nennen, die das Ergebnis
// verändert — die Reihenfolge der Felder, die Längenpräfixe, die Sortierung der
// Zeilen und die Behandlung leerer Werte.
const canonicalFormDoc = `## Die Hash-Chain nachrechnen

Jede Buchung trägt einen Vorgängerhash und einen Eigenhash. Der Eigenhash ist
der SHA-256 über eine kanonische Form der Buchung; der Vorgängerhash ist der
Eigenhash der vorhergehenden Buchung desselben Geschäftsjahres. Die erste
Buchung eines Geschäftsjahres hat als Vorgängerhash 64 Nullen.

Die kanonische Form ist eine Folge von Feldern. Jedes Feld wird geschrieben als

    <Name>:<Länge des Wertes in Bytes>:<Wert>\n

Der Wert steht als UTF-8. Das Längenpräfix ist der Grund, warum sich die Form
nicht fälschen lässt: kein Wert kann so gewählt werden, dass er wie eine
Feldgrenze aussieht. Ein leerer Wert wird als ` + "`Name:0:\\n`" + ` geschrieben,
nicht weggelassen — „kein Wert" und „leerer Wert" müssen unterscheidbar bleiben.
Ganzzahlen stehen in Dezimalschreibweise ohne Trennzeichen, Beträge als
ganzzahlige Cent, fehlende Verweise als leerer Wert.

Die Felder folgen in genau dieser Reihenfolge (in Klammern die Spalte der
Tabelle ` + "`journal`" + `, aus der der Wert stammt):

 1. ` + "`prev`" + ` (Vorgaengerhash)
 2. ` + "`number`" + ` (Buchungsnummer)
 3. ` + "`fy`" + ` (Geschaeftsjahr)
 4. ` + "`booking_date`" + ` (Buchungsdatum)
 5. ` + "`document_date`" + ` (Belegdatum)
 6. ` + "`service_from`" + ` (Leistungsbeginn)
 7. ` + "`service_to`" + ` (Leistungsende)
 8. ` + "`value_date`" + ` (Valuta)
 9. ` + "`description`" + ` (Buchungstext)
10. ` + "`source`" + ` (Quelle)
11. ` + "`document_number`" + ` (Belegnummer)
12. ` + "`receipt_hash`" + ` (Beleg_SHA256)
13. ` + "`tax_treatment`" + ` (Steuerfall)
14. ` + "`contact`" + ` (Kontakt_ID)
15. ` + "`bank_tx`" + ` (Bankumsatz_ID)
16. ` + "`kind`" + ` (Buchungsart)
17. ` + "`reversal_of`" + ` (Storno_von_ID)
18. ` + "`reversal_reason`" + ` (Storno_Grund)
19. ` + "`currency`" + ` (Waehrung)
20. ` + "`rate_micros`" + ` (Kurs_Millionstel)
21. ` + "`rate_source`" + ` (Kursquelle)
22. ` + "`rate_date`" + ` (Kursdatum)
23. ` + "`rule_version`" + ` (Regelversion)
24. ` + "`created_at`" + ` (Erfassungszeitpunkt_UTC, Format RFC 3339)

Danach folgt ` + "`lines`" + ` mit der Anzahl der Zeilen der Buchung und für jede
Zeile — aufsteigend nach Zeilennummer sortiert — die acht Felder

    line_pos, line_side, line_account, line_amount,
    line_contact, line_tax_key, line_tax_base, line_text

aus den Spalten Zeilennummer, Seite, Konto, Betrag_Cent, Zeile_Kontakt_ID,
Steuerschluessel, Bemessungsgrundlage_Cent und Zeilentext.

Zuletzt die Bewirtungsaufzeichnung: ` + "`entertainment`" + ` mit dem Wert 1,
wenn die Tabelle ` + "`bewirtungen`" + ` eine Zeile zu dieser Buchung enthält,
gefolgt von ` + "`entertainment_place`, `entertainment_day`," + `
` + "`entertainment_participants`" + ` und ` + "`entertainment_occasion`" + `.
Gibt es keine Aufzeichnung, steht dort ` + "`entertainment`" + ` mit dem Wert 0
und sonst nichts.

Der SHA-256 über diese Bytefolge, hexadezimal in Kleinbuchstaben, ist der
Eigenhash.
`

// RenderFieldDoc erzeugt die Feldbeschreibung.
//
// Sie nennt jede Spalte jeder Tabelle im Klartext, dazu jeden Schlüsselwert und
// das Verfahren, mit dem sich die Hash-Chain außerhalb von Buchfink nachrechnen
// lässt. Eine Datenüberlassung ohne sie erfüllt § 147 Abs. 6 AO nicht: der
// Prüfer bekäme Zahlen, aber keine Auskunft darüber, was sie bedeuten.
func RenderFieldDoc(d *Dataset) []byte {
	var b bytes.Buffer

	fmt.Fprintf(&b, "# Feldbeschreibung der Datenüberlassung\n\n")
	fmt.Fprintf(&b, "Mandant: %s  \n", d.TenantName)
	fmt.Fprintf(&b, "Geschäftsjahr: %d  \n", d.FiscalYear)
	if d.From != "" && d.To != "" {
		fmt.Fprintf(&b, "Zeitraum: %s bis %s  \n", d.From, d.To)
	}
	fmt.Fprintf(&b, "Erzeugt am: %s  \n", d.CreatedAt)
	fmt.Fprintf(&b, "Programm: Buchfink %s  \n", d.ProgramVersion)
	fmt.Fprintf(&b, "Schnittstelle: Beschreibungsstandard für die Datenüberlassung, Version %s\n\n", StandardVersion)

	b.WriteString(`## Aufbau der Dateien

Jede Tabelle liegt als eigene CSV-Datei. Die erste Zeile trägt die
Spaltennamen, danach folgen die Daten.

| Festlegung | Wert |
| --- | --- |
| Zeichensatz | UTF-8 ohne BOM |
| Spaltentrennzeichen | Semikolon (` + "`;`" + `) |
| Zeilentrennzeichen | CR LF |
| Texterkennungszeichen | doppeltes Anführungszeichen (` + "`\"`" + `) |
| Dezimaltrennzeichen | Punkt, immer zwei Nachkommastellen |
| Tausendertrennzeichen | wird nicht verwendet |
| Datumsformat | ` + DateFormat + ` (ISO 8601) |
| Wahrheitswerte | 1 für ja, 0 für nein |
| Beträge | in Euro mit zwei Nachkommastellen, sofern die Spalte nicht ausdrücklich Cent nennt |

Ein Feld wird in Anführungszeichen gesetzt, sobald es ein Semikolon, ein
Anführungszeichen, einen Zeilenumbruch oder ein führendes bzw. folgendes
Leerzeichen enthält; enthaltene Anführungszeichen werden verdoppelt (RFC 4180).

Die Datei ` + "`" + IndexFileName + "`" + ` beschreibt dieselben Festlegungen
maschinenlesbar nach dem Beschreibungsstandard für die Datenüberlassung; die
zugehörige Grammatik liegt als ` + "`" + DTDFileName + "`" + ` bei. Es ist der
amtliche Text des Beschreibungsstandards für die Datenträgerüberlassung
(Bundesministerium der Finanzen, Stand 01.09.2004), unverändert übernommen;
` + "`" + IndexFileName + "`" + ` ist gegen ihn gültig. Die Datei liegt bei,
damit sich die Überlassung auch dann noch prüfen lässt, wenn die Grammatik
nirgends mehr im Netz steht — die Aufbewahrungsfrist läuft zehn Jahre
(§ 147 Abs. 1 i. V. m. Abs. 3 AO).

Ein tabellarischer Export als XLSX ist nicht enthalten. CSV und
` + "`" + IndexFileName + "`" + ` sind das Format, das IDEA und ACL einlesen;
eine Tabellenkalkulationsdatei wäre eine zweite Fassung derselben Daten in einem
Format, das Beträge und führende Nullen beim Öffnen verändert.

`)

	b.WriteString("## Tabellen\n\n")
	for _, t := range d.Tables {
		fmt.Fprintf(&b, "### %s (`%s`)\n\n", t.Name, t.FileName)
		if t.Description != "" {
			fmt.Fprintf(&b, "%s\n\n", t.Description)
		}
		fmt.Fprintf(&b, "Datensätze: %d\n\n", len(t.Rows))
		b.WriteString("| Spalte | Typ | Bedeutung |\n| --- | --- | --- |\n")
		for _, f := range t.Fields {
			fmt.Fprintf(&b, "| %s | %s | %s |\n",
				escapeCell(f.Name), typeLabel(f.Type), escapeCell(f.Description))
		}
		b.WriteString("\n")
	}

	if keys, ok := d.TableByName(KeyDirectoryTable); ok {
		b.WriteString("## Schlüsselwerte im Klartext\n\n")
		b.WriteString("Jeder Code, der in den Tabellen vorkommt, mit seiner Bedeutung. Dieselben Zeilen stehen maschinenlesbar in `" + keys.FileName + "`.\n\n")
		b.WriteString("| " + strings.Join(escapeCells(keys.FieldNames()), " | ") + " |\n")
		b.WriteString("| " + strings.Repeat("--- | ", len(keys.Fields)) + "\n")
		for _, row := range keys.Rows {
			b.WriteString("| " + strings.Join(escapeCells(row), " | ") + " |\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(canonicalFormDoc)
	return b.Bytes()
}

func typeLabel(t FieldType) string {
	switch t {
	case FieldNumeric:
		return "numerisch"
	case FieldDate:
		return "Datum"
	default:
		return "alphanumerisch"
	}
}

// escapeCell macht einen Wert für eine Markdown-Tabellenzelle unschädlich: ein
// senkrechter Strich im Text zerlegte die Zeile sonst in zusätzliche Spalten.
func escapeCell(v string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ", "\r", "").Replace(v)
}

func escapeCells(values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = escapeCell(v)
	}
	return out
}
