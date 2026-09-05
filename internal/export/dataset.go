// Package export erzeugt die Datenüberlassung nach § 147 Abs. 6 AO.
//
// Die Schicht ist in zwei Hälften geteilt, und die Trennung ist der Zweck des
// Pakets: die Datenauswahl (Dataset mit Tabellen, Feldern und Zeilen) weiß
// nichts von einem Format, die Erzeuger (CSV, index.xml, Feldbeschreibung)
// wissen nichts von der Buchhaltung. Ein DATEV-Erzeuger kann später daneben
// entstehen, ohne die Auswahl anzufassen; und die Auswahl kann wachsen, ohne
// dass drei Erzeuger nachgezogen werden müssen.
//
// Alle Werte stehen als Zeichenkette in der Zeile. Das ist Absicht: die Zeile
// ist das, was in der Datei landet, und eine Umwandlung, die erst im Erzeuger
// geschieht, hätte je Format ein anderes Ergebnis. Betrag, Datum und Wahrheit
// werden deshalb an genau einer Stelle formatiert — hier.
package export

import (
	"fmt"
	"strconv"

	"github.com/buchfink/buchfink/internal/domain"
)

// FieldType ist der Datentyp einer Spalte, wie ihn der Beschreibungsstandard
// kennt: alphanumerisch, numerisch oder Datum. Mehr gibt es dort nicht.
type FieldType string

const (
	FieldAlphaNumeric FieldType = "alphanumeric"
	FieldNumeric      FieldType = "numeric"
	FieldDate         FieldType = "date"
)

// DateFormat ist das Datumsformat aller Datumsspalten. ISO 8601, weil es sich
// als einziges Format ohne Kenntnis der Herkunft eindeutig lesen lässt.
const DateFormat = "YYYY-MM-DD"

// Amount ist die Nachkommastellenzahl der Betragsspalten.
const amountAccuracy = 2

// Field beschreibt eine Spalte.
//
// Description ist Pflicht und keine Zier: eine Spalte namens „Betrag", zu der
// nirgends steht, ob sie brutto oder netto, Soll oder Haben ist, macht die
// Überlassung wertlos. Die Feldbeschreibung wird aus genau diesem Feld erzeugt.
type Field struct {
	Name        string
	Type        FieldType
	Length      int
	Description string
}

// Table ist eine Tabelle der Überlassung.
type Table struct {
	// Name ist der fachliche Name, der in index.xml steht.
	Name string
	// FileName ist der Dateiname im Zielordner. Er ist nicht aus dem Namen
	// abgeleitet, weil ein Tabellenname Umlaute tragen darf und ein Dateiname
	// über Betriebssysteme hinweg besser keine trägt.
	FileName    string
	Description string
	Fields      []Field
	Rows        [][]string
}

// Dataset ist die vollständige Auswahl einer Überlassung.
type Dataset struct {
	TenantName string
	// SupplierLocation ist der Sitz des Datenlieferanten, wie ihn der
	// Beschreibungsstandard im Element Location erwartet: der Ort, aus dem die
	// Bücher stammen, und nicht der Name des Programms.
	SupplierLocation string
	FiscalYear       int
	From             string
	To               string
	CreatedAt        string
	ProgramVersion   string
	Tables           []Table
}

// TableByName sucht eine Tabelle. Sie wird gebraucht, wo eine Auswertung eine
// einzelne Tabelle abgibt — der Journalexport eines Zeitraums etwa.
func (d *Dataset) TableByName(name string) (*Table, bool) {
	for i := range d.Tables {
		if d.Tables[i].Name == name {
			return &d.Tables[i], true
		}
	}
	return nil, false
}

// AddRow hängt eine Zeile an und prüft ihre Breite.
//
// Eine Zeile mit zu wenigen Werten wäre in der CSV nicht als Fehler zu
// erkennen — die Werte rutschen um eine Spalte, und die Prüfsoftware liest ein
// Datum als Betrag. Der Fehler gehört deshalb dorthin, wo er entsteht.
func (t *Table) AddRow(values ...string) error {
	if len(values) != len(t.Fields) {
		return fmt.Errorf("Tabelle %s: die Zeile hat %d Werte, die Tabelle aber %d Spalten",
			t.Name, len(values), len(t.Fields))
	}
	t.Rows = append(t.Rows, values)
	return nil
}

// FieldNames liefert die Spaltennamen in ihrer Reihenfolge.
func (t *Table) FieldNames() []string {
	names := make([]string, len(t.Fields))
	for i, f := range t.Fields {
		names[i] = f.Name
	}
	return names
}

// --- Formatierung der Werte ------------------------------------------------

// Amount formatiert einen Betrag als Dezimalzahl mit Punkt und zwei
// Nachkommastellen.
//
// Punkt und nicht Komma, und keine Tausendertrennung: die Überlassung geht an
// eine Prüfsoftware und nicht auf ein Blatt Papier. Die Zahl entsteht aus dem
// ganzzahligen Cent-Betrag und nicht aus einer Fließkommazahl, damit der
// ausgegebene Wert bitgenau der gebuchte ist.
func Amount(c domain.Cents) string { return c.Decimal() }

// Int formatiert eine Ganzzahl.
func Int(v int) string { return strconv.Itoa(v) }

// Int64 formatiert eine Ganzzahl.
func Int64(v int64) string { return strconv.FormatInt(v, 10) }

// Bool formatiert einen Wahrheitswert als 0 oder 1.
//
// Nicht „ja"/„nein" und nicht „true"/„false": beides ist sprachabhängig, und
// eine numerische Spalte lässt sich auswerten, ohne die Sprache zu kennen.
func Bool(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

// OptUint formatiert einen möglicherweise fehlenden Verweis. Leer heißt: kein
// Verweis.
func OptUint(v *uint) string {
	if v == nil {
		return ""
	}
	return strconv.FormatUint(uint64(*v), 10)
}

// Uint formatiert eine Kennung.
func Uint(v uint) string { return strconv.FormatUint(uint64(v), 10) }

// Rate formatiert einen Steuersatz in Basispunkten als Prozentzahl mit zwei
// Nachkommastellen, etwa "19.00".
func Rate(r domain.TaxRate) string {
	neg := r < 0
	v := int(r)
	if neg {
		v = -v
	}
	s := fmt.Sprintf("%d.%02d", v/100, v%100)
	if neg {
		return "-" + s
	}
	return s
}
