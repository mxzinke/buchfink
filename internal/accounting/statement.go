package accounting

import (
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Gliederung von Bilanz und Gewinn- und Verlustrechnung nach den §§ 266 und
// 275 HGB.
//
// Bisher entstanden beide im Frontend durch Filtern nach Kontenklasse. Das war
// keine Gliederung, sondern eine Sortierung: die Kontenklasse sagt, wo ein Konto
// im Kontenrahmen steht, nicht, unter welchem Posten es im Abschluss auszuweisen
// ist. Zwischen beidem liegt genau das, was § 266 HGB vorschreibt — und was
// jedes SKR04-Konto in seiner Position bereits mitbringt.
//
// Deshalb steht die Gliederung hier, im Backend, und nicht in der Ansicht: sie
// ist eine Rechtsfrage mit einer Antwort, keine Darstellungsvariante. Die
// Ansicht zeigt nur noch, was hier entsteht.

// lineDef ist eine Gliederungsposition der Vorlage.
type lineDef struct {
	// Key ist der stabile Schlüssel, an dem die Zuordnungstabelle und die
	// Taxonomie der E-Bilanz hängen.
	Key     string
	Ordinal string
	Label   string
	// Level ist die Ebene des Gesetzes: 1 Buchstabe, 2 römische Ziffer,
	// 3 arabische Ziffer. In der Staffel der GuV ist 1 die Nummer und 2 der
	// Kleinbuchstabe.
	Level   int
	Parent  string
	Section domain.StatementSection
	// Credit sagt, auf welcher Seite der natürliche Saldo dieser Position
	// steht. Daran entscheidet sich, ob ein Konto mit „falschem" Vorzeichen auf
	// die Gegenposition wandert (§ 265 Abs. 2 HGB verlangt Vergleichbarkeit,
	// nicht das Stehenlassen eines negativen Aktivpostens).
	Credit bool
	// Bidirectional kennzeichnet Positionen, deren Bezeichnung beide Richtungen
	// ausdrücklich nennt („Gewinnvortrag/Verlustvortrag"). Bei ihnen ist ein
	// Saldo auf der anderen Seite der Normalfall und kein Befund.
	Bidirectional bool
	Note          string
	Fallback      bool
	Subtotal      bool
}

// IsExpensePosition sagt, ob eine Position der Staffel ein Aufwandsposten ist
// (§ 275 Abs. 2 Nr. 5 bis 8, 12 bis 14, 16 HGB). In der Gliederung stehen
// Aufwendungen als negativer Beitrag zum Ergebnis; die E-Bilanz-Taxonomie
// erwartet sie als positiven Betrag. Wer den Betrag nach außen gibt, fragt hier
// nach, statt die Vorzeichenregel ein zweites Mal aufzuschreiben.
func IsExpensePosition(key string) bool {
	for _, def := range incomeLines {
		if def.Key == key {
			return !def.Credit && !def.Subtotal
		}
	}
	return false
}

// balanceLines ist die Bilanz in Kontoform nach § 266 Abs. 2 und 3 HGB.
// Die Bezeichnungen folgen dem Gesetzeswortlaut.
var balanceLines = []lineDef{
	// --- Aktivseite (§ 266 Abs. 2 HGB) ---
	{Key: "aktiva.0", Label: "Nicht eingeforderte ausstehende Einlagen auf das gezeichnete Kapital", Level: 1, Section: domain.SectionAssets,
		Note: "Vor dem Anlagevermögen auszuweisen (§ 272 Abs. 1 Satz 3 HGB); zählt nicht zur Bilanzsumme des § 267 Abs. 4a HGB."},

	{Key: "aktiva.A", Ordinal: "A.", Label: "Anlagevermögen", Level: 1, Section: domain.SectionAssets},
	{Key: "aktiva.A.I", Ordinal: "I.", Label: "Immaterielle Vermögensgegenstände", Level: 2, Parent: "aktiva.A", Section: domain.SectionAssets},
	{Key: "aktiva.A.I.1", Ordinal: "1.", Label: "Selbst geschaffene gewerbliche Schutzrechte und ähnliche Rechte und Werte", Level: 3, Parent: "aktiva.A.I", Section: domain.SectionAssets},
	{Key: "aktiva.A.I.2", Ordinal: "2.", Label: "entgeltlich erworbene Konzessionen, gewerbliche Schutzrechte und ähnliche Rechte und Werte sowie Lizenzen an solchen Rechten und Werten", Level: 3, Parent: "aktiva.A.I", Section: domain.SectionAssets},
	{Key: "aktiva.A.I.3", Ordinal: "3.", Label: "Geschäfts- oder Firmenwert", Level: 3, Parent: "aktiva.A.I", Section: domain.SectionAssets},
	{Key: "aktiva.A.I.4", Ordinal: "4.", Label: "geleistete Anzahlungen", Level: 3, Parent: "aktiva.A.I", Section: domain.SectionAssets},
	{Key: "aktiva.A.II", Ordinal: "II.", Label: "Sachanlagen", Level: 2, Parent: "aktiva.A", Section: domain.SectionAssets},
	{Key: "aktiva.A.II.1", Ordinal: "1.", Label: "Grundstücke, grundstücksgleiche Rechte und Bauten einschließlich der Bauten auf fremden Grundstücken", Level: 3, Parent: "aktiva.A.II", Section: domain.SectionAssets},
	{Key: "aktiva.A.II.2", Ordinal: "2.", Label: "technische Anlagen und Maschinen", Level: 3, Parent: "aktiva.A.II", Section: domain.SectionAssets},
	{Key: "aktiva.A.II.3", Ordinal: "3.", Label: "andere Anlagen, Betriebs- und Geschäftsausstattung", Level: 3, Parent: "aktiva.A.II", Section: domain.SectionAssets},
	{Key: "aktiva.A.II.4", Ordinal: "4.", Label: "geleistete Anzahlungen und Anlagen im Bau", Level: 3, Parent: "aktiva.A.II", Section: domain.SectionAssets},
	{Key: "aktiva.A.III", Ordinal: "III.", Label: "Finanzanlagen", Level: 2, Parent: "aktiva.A", Section: domain.SectionAssets},
	{Key: "aktiva.A.III.1", Ordinal: "1.", Label: "Anteile an verbundenen Unternehmen", Level: 3, Parent: "aktiva.A.III", Section: domain.SectionAssets},
	{Key: "aktiva.A.III.2", Ordinal: "2.", Label: "Ausleihungen an verbundene Unternehmen", Level: 3, Parent: "aktiva.A.III", Section: domain.SectionAssets},
	{Key: "aktiva.A.III.3", Ordinal: "3.", Label: "Beteiligungen", Level: 3, Parent: "aktiva.A.III", Section: domain.SectionAssets},
	{Key: "aktiva.A.III.4", Ordinal: "4.", Label: "Ausleihungen an Unternehmen, mit denen ein Beteiligungsverhältnis besteht", Level: 3, Parent: "aktiva.A.III", Section: domain.SectionAssets},
	{Key: "aktiva.A.III.5", Ordinal: "5.", Label: "Wertpapiere des Anlagevermögens", Level: 3, Parent: "aktiva.A.III", Section: domain.SectionAssets},
	{Key: "aktiva.A.III.6", Ordinal: "6.", Label: "sonstige Ausleihungen", Level: 3, Parent: "aktiva.A.III", Section: domain.SectionAssets, Fallback: true},

	{Key: "aktiva.B", Ordinal: "B.", Label: "Umlaufvermögen", Level: 1, Section: domain.SectionAssets},
	{Key: "aktiva.B.I", Ordinal: "I.", Label: "Vorräte", Level: 2, Parent: "aktiva.B", Section: domain.SectionAssets},
	{Key: "aktiva.B.I.1", Ordinal: "1.", Label: "Roh-, Hilfs- und Betriebsstoffe", Level: 3, Parent: "aktiva.B.I", Section: domain.SectionAssets},
	{Key: "aktiva.B.I.2", Ordinal: "2.", Label: "unfertige Erzeugnisse, unfertige Leistungen", Level: 3, Parent: "aktiva.B.I", Section: domain.SectionAssets},
	{Key: "aktiva.B.I.3", Ordinal: "3.", Label: "fertige Erzeugnisse und Waren", Level: 3, Parent: "aktiva.B.I", Section: domain.SectionAssets},
	{Key: "aktiva.B.I.4", Ordinal: "4.", Label: "geleistete Anzahlungen", Level: 3, Parent: "aktiva.B.I", Section: domain.SectionAssets},
	{Key: "aktiva.B.II", Ordinal: "II.", Label: "Forderungen und sonstige Vermögensgegenstände", Level: 2, Parent: "aktiva.B", Section: domain.SectionAssets},
	{Key: "aktiva.B.II.1", Ordinal: "1.", Label: "Forderungen aus Lieferungen und Leistungen", Level: 3, Parent: "aktiva.B.II", Section: domain.SectionAssets},
	{Key: "aktiva.B.II.2", Ordinal: "2.", Label: "Forderungen gegen verbundene Unternehmen", Level: 3, Parent: "aktiva.B.II", Section: domain.SectionAssets},
	{Key: "aktiva.B.II.3", Ordinal: "3.", Label: "Forderungen gegen Unternehmen, mit denen ein Beteiligungsverhältnis besteht", Level: 3, Parent: "aktiva.B.II", Section: domain.SectionAssets},
	{Key: "aktiva.B.II.4", Ordinal: "4.", Label: "sonstige Vermögensgegenstände", Level: 3, Parent: "aktiva.B.II", Section: domain.SectionAssets, Fallback: true},
	{Key: "aktiva.B.III", Ordinal: "III.", Label: "Wertpapiere", Level: 2, Parent: "aktiva.B", Section: domain.SectionAssets},
	{Key: "aktiva.B.III.1", Ordinal: "1.", Label: "Anteile an verbundenen Unternehmen", Level: 3, Parent: "aktiva.B.III", Section: domain.SectionAssets},
	{Key: "aktiva.B.III.2", Ordinal: "2.", Label: "sonstige Wertpapiere", Level: 3, Parent: "aktiva.B.III", Section: domain.SectionAssets, Fallback: true},
	{Key: "aktiva.B.IV", Ordinal: "IV.", Label: "Kassenbestand, Bundesbankguthaben, Guthaben bei Kreditinstituten und Schecks", Level: 2, Parent: "aktiva.B", Section: domain.SectionAssets},

	{Key: "aktiva.C", Ordinal: "C.", Label: "Rechnungsabgrenzungsposten", Level: 1, Section: domain.SectionAssets},
	{Key: "aktiva.D", Ordinal: "D.", Label: "Aktive latente Steuern", Level: 1, Section: domain.SectionAssets},
	{Key: "aktiva.E", Ordinal: "E.", Label: "Aktiver Unterschiedsbetrag aus der Vermögensverrechnung", Level: 1, Section: domain.SectionAssets},
	{Key: "aktiva.X", Label: "Nicht zugeordnet", Level: 1, Section: domain.SectionAssets,
		Note: "Konten mit Sollsaldo, für die die Gliederung keine Position kennt. Sie stehen hier, damit die Bilanz aufgeht — sie gehören dorthin nicht."},

	// --- Passivseite (§ 266 Abs. 3 HGB) ---
	{Key: "passiva.A", Ordinal: "A.", Label: "Eigenkapital", Level: 1, Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.A.I", Ordinal: "I.", Label: "Gezeichnetes Kapital", Level: 2, Parent: "passiva.A", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.A.II", Ordinal: "II.", Label: "Kapitalrücklage", Level: 2, Parent: "passiva.A", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.A.III", Ordinal: "III.", Label: "Gewinnrücklagen", Level: 2, Parent: "passiva.A", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.A.III.1", Ordinal: "1.", Label: "gesetzliche Rücklage", Level: 3, Parent: "passiva.A.III", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.A.III.2", Ordinal: "2.", Label: "Rücklage für Anteile an einem herrschenden oder mehrheitlich beteiligten Unternehmen", Level: 3, Parent: "passiva.A.III", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.A.III.3", Ordinal: "3.", Label: "satzungsmäßige Rücklagen", Level: 3, Parent: "passiva.A.III", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.A.III.4", Ordinal: "4.", Label: "andere Gewinnrücklagen", Level: 3, Parent: "passiva.A.III", Section: domain.SectionLiabilities, Credit: true, Fallback: true},
	{Key: "passiva.A.IV", Ordinal: "IV.", Label: "Gewinnvortrag/Verlustvortrag", Level: 2, Parent: "passiva.A", Section: domain.SectionLiabilities, Credit: true, Bidirectional: true},
	{Key: "passiva.A.V", Ordinal: "V.", Label: "Jahresüberschuss/Jahresfehlbetrag", Level: 2, Parent: "passiva.A", Section: domain.SectionLiabilities, Credit: true, Bidirectional: true,
		Note: "Ergebnis der Gewinn- und Verlustrechnung (§ 275 Abs. 2 Nr. 17 HGB). Es wird abgeleitet, nicht gebucht."},
	{Key: "passiva.A.VI", Ordinal: "VI.", Label: "Ergebnisverwendung", Level: 2, Parent: "passiva.A", Section: domain.SectionLiabilities, Credit: true, Bidirectional: true,
		Note: "Konten der Ergebnisverwendung (Entnahmen aus und Einstellungen in Rücklagen, Ausschüttungen). § 268 Abs. 1 HGB lässt die Bilanz unter Berücksichtigung der Verwendung zu; die Beträge gehören nicht in die Staffel des § 275 HGB."},
	{Key: "passiva.SP", Label: "Sonderposten mit Rücklageanteil", Level: 1, Section: domain.SectionLiabilities, Credit: true,
		Note: "§ 266 Abs. 3 HGB kennt den Posten seit dem BilMoG nicht mehr; der SKR04 führt ihn weiter. Er steht deshalb gesondert zwischen Eigenkapital und Rückstellungen."},

	{Key: "passiva.B", Ordinal: "B.", Label: "Rückstellungen", Level: 1, Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.B.1", Ordinal: "1.", Label: "Rückstellungen für Pensionen und ähnliche Verpflichtungen", Level: 3, Parent: "passiva.B", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.B.2", Ordinal: "2.", Label: "Steuerrückstellungen", Level: 3, Parent: "passiva.B", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.B.3", Ordinal: "3.", Label: "sonstige Rückstellungen", Level: 3, Parent: "passiva.B", Section: domain.SectionLiabilities, Credit: true, Fallback: true},

	{Key: "passiva.C", Ordinal: "C.", Label: "Verbindlichkeiten", Level: 1, Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.C.1", Ordinal: "1.", Label: "Anleihen", Level: 3, Parent: "passiva.C", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.C.2", Ordinal: "2.", Label: "Verbindlichkeiten gegenüber Kreditinstituten", Level: 3, Parent: "passiva.C", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.C.3", Ordinal: "3.", Label: "erhaltene Anzahlungen auf Bestellungen", Level: 3, Parent: "passiva.C", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.C.4", Ordinal: "4.", Label: "Verbindlichkeiten aus Lieferungen und Leistungen", Level: 3, Parent: "passiva.C", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.C.5", Ordinal: "5.", Label: "Verbindlichkeiten aus der Annahme gezogener Wechsel und der Ausstellung eigener Wechsel", Level: 3, Parent: "passiva.C", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.C.6", Ordinal: "6.", Label: "Verbindlichkeiten gegenüber verbundenen Unternehmen", Level: 3, Parent: "passiva.C", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.C.7", Ordinal: "7.", Label: "Verbindlichkeiten gegenüber Unternehmen, mit denen ein Beteiligungsverhältnis besteht", Level: 3, Parent: "passiva.C", Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.C.8", Ordinal: "8.", Label: "sonstige Verbindlichkeiten", Level: 3, Parent: "passiva.C", Section: domain.SectionLiabilities, Credit: true, Fallback: true},

	{Key: "passiva.D", Ordinal: "D.", Label: "Rechnungsabgrenzungsposten", Level: 1, Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.E", Ordinal: "E.", Label: "Passive latente Steuern", Level: 1, Section: domain.SectionLiabilities, Credit: true},
	{Key: "passiva.X", Label: "Nicht zugeordnet", Level: 1, Section: domain.SectionLiabilities, Credit: true,
		Note: "Konten mit Habensaldo, für die die Gliederung keine Position kennt. Sie stehen hier, damit die Bilanz aufgeht — sie gehören dorthin nicht."},
}

// incomeLines ist die Staffel des § 275 Abs. 2 HGB (Gesamtkostenverfahren) in
// der Fassung nach dem BilRUG.
//
// Alle Beträge stehen als Beitrag zum Ergebnis: Erträge positiv, Aufwendungen
// negativ. Damit ist jede Zwischensumme die Summe der Zeilen darüber, und
// Nummer 17 ist das Jahresergebnis — ohne dass die Ansicht die Vorzeichen
// nachrechnen müsste.
var incomeLines = []lineDef{
	{Key: "guv.1", Ordinal: "1.", Label: "Umsatzerlöse", Level: 1, Section: domain.SectionIncome, Credit: true},
	{Key: "guv.2", Ordinal: "2.", Label: "Erhöhung oder Verminderung des Bestands an fertigen und unfertigen Erzeugnissen", Level: 1, Section: domain.SectionIncome, Credit: true},
	{Key: "guv.3", Ordinal: "3.", Label: "andere aktivierte Eigenleistungen", Level: 1, Section: domain.SectionIncome, Credit: true},
	{Key: "guv.4", Ordinal: "4.", Label: "sonstige betriebliche Erträge", Level: 1, Section: domain.SectionIncome, Credit: true, Fallback: true},
	{Key: "guv.5", Ordinal: "5.", Label: "Materialaufwand", Level: 1, Section: domain.SectionIncome},
	{Key: "guv.5.a", Ordinal: "a)", Label: "Aufwendungen für Roh-, Hilfs- und Betriebsstoffe und für bezogene Waren", Level: 2, Parent: "guv.5", Section: domain.SectionIncome},
	{Key: "guv.5.b", Ordinal: "b)", Label: "Aufwendungen für bezogene Leistungen", Level: 2, Parent: "guv.5", Section: domain.SectionIncome},
	{Key: "guv.6", Ordinal: "6.", Label: "Personalaufwand", Level: 1, Section: domain.SectionIncome},
	{Key: "guv.6.a", Ordinal: "a)", Label: "Löhne und Gehälter", Level: 2, Parent: "guv.6", Section: domain.SectionIncome},
	{Key: "guv.6.b", Ordinal: "b)", Label: "soziale Abgaben und Aufwendungen für Altersversorgung und für Unterstützung", Level: 2, Parent: "guv.6", Section: domain.SectionIncome},
	{Key: "guv.7", Ordinal: "7.", Label: "Abschreibungen", Level: 1, Section: domain.SectionIncome},
	{Key: "guv.7.a", Ordinal: "a)", Label: "auf immaterielle Vermögensgegenstände des Anlagevermögens und Sachanlagen", Level: 2, Parent: "guv.7", Section: domain.SectionIncome},
	{Key: "guv.7.b", Ordinal: "b)", Label: "auf Vermögensgegenstände des Umlaufvermögens, soweit diese die in der Kapitalgesellschaft üblichen Abschreibungen überschreiten", Level: 2, Parent: "guv.7", Section: domain.SectionIncome},
	{Key: "guv.8", Ordinal: "8.", Label: "sonstige betriebliche Aufwendungen", Level: 1, Section: domain.SectionIncome, Fallback: true},
	{Key: "guv.9", Ordinal: "9.", Label: "Erträge aus Beteiligungen", Level: 1, Section: domain.SectionIncome, Credit: true},
	{Key: "guv.10", Ordinal: "10.", Label: "Erträge aus anderen Wertpapieren und Ausleihungen des Finanzanlagevermögens", Level: 1, Section: domain.SectionIncome, Credit: true},
	{Key: "guv.11", Ordinal: "11.", Label: "sonstige Zinsen und ähnliche Erträge", Level: 1, Section: domain.SectionIncome, Credit: true},
	{Key: "guv.11a", Label: "Erträge aus Verlustübernahme und auf Grund einer Gewinngemeinschaft erhaltene Gewinne", Level: 1, Section: domain.SectionIncome, Credit: true,
		Note: "Gesondert auszuweisen nach § 277 Abs. 3 Satz 2 HGB; die Staffel des § 275 Abs. 2 HGB gibt dafür keine Nummer."},
	{Key: "guv.12", Ordinal: "12.", Label: "Abschreibungen auf Finanzanlagen und auf Wertpapiere des Umlaufvermögens", Level: 1, Section: domain.SectionIncome},
	{Key: "guv.13", Ordinal: "13.", Label: "Zinsen und ähnliche Aufwendungen", Level: 1, Section: domain.SectionIncome},
	{Key: "guv.13a", Label: "Aufwendungen aus Verlustübernahme und abgeführte Gewinne", Level: 1, Section: domain.SectionIncome,
		Note: "Gesondert auszuweisen nach § 277 Abs. 3 Satz 2 HGB; die Staffel des § 275 Abs. 2 HGB gibt dafür keine Nummer."},
	{Key: "guv.14", Ordinal: "14.", Label: "Steuern vom Einkommen und vom Ertrag", Level: 1, Section: domain.SectionIncome},
	{Key: "guv.15", Ordinal: "15.", Label: "Ergebnis nach Steuern", Level: 1, Section: domain.SectionIncome, Subtotal: true},
	{Key: "guv.16", Ordinal: "16.", Label: "sonstige Steuern", Level: 1, Section: domain.SectionIncome},
	{Key: "guv.X", Label: "Nicht zugeordnet", Level: 1, Section: domain.SectionIncome,
		Note: "Erfolgskonten, für die die Gliederung keine Position kennt. Sie stehen unmittelbar vor Nummer 17 und gehen in das Jahresergebnis ein — sonst wiche die Staffel von dem Betrag ab, der als Posten A.V in der Bilanz steht. Wo sie in der Staffel hingehören, sagt erst die Klärung des Kontos."},
	{Key: "guv.17", Ordinal: "17.", Label: "Jahresüberschuss/Jahresfehlbetrag", Level: 1, Section: domain.SectionIncome, Subtotal: true},
}

// statisticalLines nimmt die Konten der Klasse 9 auf. Sie sind weder Bilanz-
// noch Erfolgskonten; die Vortragskonten gleichen sich nach einem
// vollständigen Saldenvortrag auf null aus. Bleibt dort ein Saldo stehen, geht
// die Bilanz nicht auf — und genau das soll man sehen.
var statisticalLines = []lineDef{
	{Key: "statistisch", Label: "Statistische Konten (nicht Bestandteil von Bilanz und Gewinn- und Verlustrechnung)", Level: 1, Section: domain.SectionStatistical},
}

// incomeResultKeys sind die Zeilen, die in das Ergebnis nach Steuern eingehen
// (§ 275 Abs. 2 Nr. 1 bis 14 HGB, zuzüglich der gesondert auszuweisenden
// Beträge des § 277 Abs. 3 Satz 2 HGB).
var incomeResultKeys = []string{
	"guv.1", "guv.2", "guv.3", "guv.4", "guv.5", "guv.6", "guv.7", "guv.8",
	"guv.9", "guv.10", "guv.11", "guv.11a", "guv.12", "guv.13", "guv.13a", "guv.14",
}

var (
	lineIndex    map[string]lineDef
	lineChildren map[string][]string
	allLines     []lineDef
)

func init() {
	allLines = make([]lineDef, 0, len(balanceLines)+len(incomeLines)+len(statisticalLines))
	allLines = append(allLines, balanceLines...)
	allLines = append(allLines, incomeLines...)
	allLines = append(allLines, statisticalLines...)

	lineIndex = make(map[string]lineDef, len(allLines))
	lineChildren = make(map[string][]string, len(allLines))
	for _, def := range allLines {
		lineIndex[def.Key] = def
		if def.Parent != "" {
			lineChildren[def.Parent] = append(lineChildren[def.Parent], def.Key)
		}
	}
}

// StatementLines liefert die Gliederungsvorlage als Schlüssel in ihrer
// gesetzlichen Reihenfolge. Die E-Bilanz prüft daran, ob ihre
// Taxonomie-Zuordnung vollständig ist.
func StatementLines() []domain.StatementLine {
	out := make([]domain.StatementLine, 0, len(allLines))
	for _, def := range allLines {
		out = append(out, toLine(def))
	}
	return out
}

// LineLabel nennt die Bezeichnung einer Gliederungsposition.
func LineLabel(key string) string {
	if def, ok := lineIndex[key]; ok {
		return def.Label
	}
	return key
}

func toLine(def lineDef) domain.StatementLine {
	return domain.StatementLine{
		Key: def.Key, Ordinal: def.Ordinal, Label: def.Label, Level: def.Level,
		Section: def.Section, Note: def.Note, IsSubtotal: def.Subtotal,
		IsFallback: def.Fallback,
	}
}

// positionTarget ist das Ziel einer SKR04-Position in der Gliederung.
type positionTarget struct {
	// Key ist die Position, in der das Konto steht, wenn sein Saldo in der
	// natürlichen Richtung dieser Position liegt.
	Key string
	// AltKey ist die Gegenposition. Der SKR04 nennt viele Posten „X oder Y":
	// dieselben Konten sind Forderung oder Verbindlichkeit, je nachdem, wie sie
	// stehen (Vorsteuer, Umsatzsteuer, Bank). Ohne die Gegenposition stünde ein
	// Bankkonto im Soll unter den Verbindlichkeiten und mit negativem Betrag.
	AltKey string
	// Deduction kennzeichnet Positionen, die das Gesetz offen von einer anderen
	// absetzen lässt: die nicht eingeforderten ausstehenden Einlagen (§ 272
	// Abs. 1 Satz 3 HGB) und die erworbenen eigenen Anteile (§ 272 Abs. 1a HGB)
	// stehen mit Sollsaldo auf einer Passivposition. Das ist dort der
	// vorgeschriebene Normalfall und kein Befund; ohne diesen Merker bekäme
	// jede GmbH mit ausstehenden Einlagen eine Warnung, die keine ist.
	Deduction bool
	Note      string
}

// balance ist der Saldo eines Kontos in Sollrichtung: positiv heißt Sollsaldo.
//
// Nicht Account.Balance: das ist der Saldo in der natürlichen Richtung der
// Kontoart und damit vorzeichenlos gegenüber der Frage, die die Gliederung
// stellt — auf welcher Seite steht das Konto tatsächlich.
func balance(acc domain.Account) domain.Cents { return acc.DebitSum - acc.CreditSum }

// BuildStatement gliedert die Salden eines Geschäftsjahres nach den §§ 266 und
// 275 HGB und stellt die Werte des Vorjahres daneben.
//
// Die Funktion ist rein: Konten mit Salden hinein, Gliederung heraus. Sie kennt
// weder Datenbank noch Geschäftsjahr-Entität — das ist die Aufgabe des
// StatementService.
//
// Sie schlägt fehl, wenn die Bilanz nicht aufgeht. Eine Bilanz, deren Seiten
// verschieden sind, ist keine Bilanz; sie mit einer Warnung anzuzeigen hieße,
// die Grundinvariante der doppelten Buchführung zur Empfehlung zu machen.
func BuildStatement(current, prior []domain.Account, depth domain.StatementDepth) (*domain.Statement, error) {
	if !depth.Valid() {
		return nil, fmt.Errorf("unbekannte Gliederungstiefe %q", depth)
	}

	b := &builder{
		own:      map[string]domain.Cents{},
		ownPrior: map[string]domain.Cents{},
		accounts: map[string][]domain.StatementAccount{},
		report:   domain.AssignmentReport{},
	}
	b.collect(current, prior)

	stmt := &domain.Statement{Depth: depth, HasPrior: len(prior) > 0}

	// Erst die Staffel: das Jahresergebnis der GuV ist der Posten A.V der
	// Passivseite, und ohne es geht die Bilanz nicht auf.
	stmt.Income = b.lines(incomeLines, depth.MaxIncomeLevel())
	b.applyIncomeSubtotals(stmt.Income)
	stmt.NetIncome, stmt.NetIncomePrior = amountOf(stmt.Income, "guv.17")
	stmt.Revenue, stmt.RevenuePrior = amountOf(stmt.Income, "guv.1")

	b.own["passiva.A.V"] += stmt.NetIncome
	b.ownPrior["passiva.A.V"] += stmt.NetIncomePrior

	balanceSheet := b.lines(balanceLines, depth.MaxBalanceLevel())
	for _, line := range balanceSheet {
		switch line.Section {
		case domain.SectionAssets:
			stmt.Assets = append(stmt.Assets, line)
			if line.Level == 1 {
				stmt.TotalAssets += line.Amount
				stmt.TotalAssetsPrior += line.PriorAmount
				// Die Bilanzsumme des § 267 Abs. 4a HGB ist die Summe der
				// Posten A bis E der Aktivseite: ausgenommen sind allein die
				// nicht eingeforderten ausstehenden Einlagen (aktiva.0, § 272
				// Abs. 1 Satz 3 HGB) und ein auf der Aktivseite ausgewiesener
				// Fehlbetrag nach § 268 Abs. 3 HGB. Die aktiven latenten
				// Steuern (aktiva.D) gehören dazu — die Spezifikation der
				// Welle 2 nimmt sie aus, das trifft den Gesetzeswortlaut nicht,
				// und die Größenklasse folgt hier dem Gesetz.
				//
				// Der Fehlbetrag des § 268 Abs. 3 HGB kommt in dieser
				// Gliederung nicht als Aktivposition vor; negatives
				// Eigenkapital steht als negativer Passivposten. Die
				// Bilanzsumme ist dieselbe, die Darstellung nicht — der
				// Ausweis auf der Aktivseite bleibt offen.
				if line.Key != "aktiva.0" {
					stmt.BalanceSheetTotal += line.Amount
					stmt.BalanceSheetTotalPrior += line.PriorAmount
				}
			}
		case domain.SectionLiabilities:
			stmt.Liabilities = append(stmt.Liabilities, line)
			if line.Level == 1 {
				stmt.TotalLiabilities += line.Amount
				stmt.TotalLiabilitiesPrior += line.PriorAmount
			}
		}
	}
	stmt.Statistical = b.lines(statisticalLines, 1)
	stmt.Assignment = b.finish()

	if diff := stmt.TotalAssets - stmt.TotalLiabilities; diff != 0 {
		return nil, imbalanceError(diff, stmt)
	}
	return stmt, nil
}

// imbalanceError benennt die Differenz und, wo möglich, ihre wahrscheinliche
// Ursache. Eine Fehlermeldung, die nur „geht nicht auf" sagt, lässt den Leser
// mit der Suche allein.
func imbalanceError(diff domain.Cents, stmt *domain.Statement) error {
	msg := fmt.Sprintf(
		"die Bilanz geht nicht auf: Aktiva %s €, Passiva %s €, Differenz %s €",
		stmt.TotalAssets, stmt.TotalLiabilities, diff)
	var statistical domain.Cents
	for _, line := range stmt.Statistical {
		statistical += line.Amount
	}
	if statistical != 0 {
		msg += fmt.Sprintf(
			"; auf den statistischen Konten der Klasse 9 stehen %s € — ein unvollständiger "+
				"Saldenvortrag lässt genau diesen Rest stehen", statistical)
	}
	return fmt.Errorf("%s", msg)
}

func amountOf(lines []domain.StatementLine, key string) (domain.Cents, domain.Cents) {
	for _, line := range lines {
		if line.Key == key {
			return line.Amount, line.PriorAmount
		}
	}
	return 0, 0
}

// builder sammelt die Salden auf den Gliederungspositionen ein.
type builder struct {
	own      map[string]domain.Cents
	ownPrior map[string]domain.Cents
	accounts map[string][]domain.StatementAccount
	report   domain.AssignmentReport
}

type accountBalances struct {
	account domain.Account
	current domain.Cents
	prior   domain.Cents
}

func (b *builder) collect(current, prior []domain.Account) {
	// Bereichskonten ("4400-4409 Erlöse 19 % USt") werden nicht übersprungen:
	// AccountingService.GetAccounts faltet die gebuchten Umsätze in genau die
	// Katalogzeile, die sie abdeckt — und das ist bei zehn zusammengefassten
	// Konten der Bereich selbst. Wer ihn hier ausließe, verlöre die Erlöse.
	merged := map[string]*accountBalances{}
	order := []string{}
	for _, acc := range current {
		v := balance(acc)
		merged[acc.Number] = &accountBalances{account: acc, current: v}
		order = append(order, acc.Number)
	}
	for _, acc := range prior {
		v := balance(acc)
		if existing, ok := merged[acc.Number]; ok {
			existing.prior = v
			continue
		}
		merged[acc.Number] = &accountBalances{account: acc, prior: v}
		order = append(order, acc.Number)
	}
	sort.Strings(order)

	for _, number := range order {
		item := merged[number]
		if item.current == 0 && item.prior == 0 {
			continue
		}
		b.assign(item)
	}
}

// assign entscheidet, unter welcher Position ein Konto ausgewiesen wird.
//
// Die Entscheidung fällt für jedes Jahr getrennt: ein Bankkonto, das im Vorjahr
// im Haben stand und in diesem Jahr im Soll, gehörte im Vorjahr unter die
// Verbindlichkeiten und gehört jetzt unter die flüssigen Mittel. Beides
// zusammen in eine Zeile zu zwingen hieße, eine der beiden Bilanzen zu fälschen.
func (b *builder) assign(item *accountBalances) {
	target, known := positionTargets[item.account.PositionID]
	if !known {
		entry := b.accountEntry(item, "", item.current, item.prior)
		b.report.Unassigned = append(b.report.Unassigned, entry)
		b.place(unassignedKey(item.account, item.current, item.prior), item, "", item.current, item.prior)
		return
	}

	currentKey := b.resolve(target, item, item.current)
	priorKey := currentKey
	if item.prior != 0 {
		priorKey = b.resolveQuiet(target, item.prior)
	}

	if currentKey == priorKey {
		b.place(currentKey, item, target.Note, item.current, item.prior)
		return
	}
	if item.current != 0 {
		b.place(currentKey, item, target.Note, item.current, 0)
	}
	b.place(priorKey, item, target.Note, 0, item.prior)
}

// resolve wählt zwischen Position und Gegenposition und protokolliert, was
// dabei auffällt.
func (b *builder) resolve(target positionTarget, item *accountBalances, value domain.Cents) string {
	def := lineIndex[target.Key]
	if value == 0 || !contradicts(def, value) {
		return target.Key
	}
	if target.AltKey == "" {
		if reportsWrongSign(def) && !target.Deduction {
			b.report.WrongSign = append(b.report.WrongSign,
				b.accountEntry(item, target.Note, value, 0))
		}
		return target.Key
	}
	b.report.SignSwitches = append(b.report.SignSwitches, domain.SignSwitch{
		Account: item.account.Number, Name: item.account.Name,
		From: target.Key, To: target.AltKey,
		Label:  LineLabel(target.AltKey),
		Amount: amountFor(lineIndex[target.AltKey], value),
	})
	return target.AltKey
}

// resolveQuiet trifft dieselbe Wahl für das Vorjahr, ohne sie noch einmal zu
// berichten: der Zuordnungsbericht gilt dem Jahr, das aufgestellt wird.
func (b *builder) resolveQuiet(target positionTarget, value domain.Cents) string {
	def := lineIndex[target.Key]
	if value == 0 || !contradicts(def, value) || target.AltKey == "" {
		return target.Key
	}
	return target.AltKey
}

func (b *builder) accountEntry(item *accountBalances, note string, value, priorValue domain.Cents) domain.StatementAccount {
	return domain.StatementAccount{
		Number: item.account.Number, Name: item.account.Name,
		PositionID: item.account.PositionID, Position: item.account.Posten,
		Note: note, Amount: value, PriorAmount: priorValue,
	}
}

func (b *builder) place(key string, item *accountBalances, note string, value, priorValue domain.Cents) {
	def := lineIndex[key]
	amount := amountFor(def, value)
	priorAmount := amountFor(def, priorValue)
	b.own[key] += amount
	b.ownPrior[key] += priorAmount
	entry := b.accountEntry(item, note, amount, priorAmount)
	b.accounts[key] = append(b.accounts[key], entry)
}

// reportsWrongSign sagt, ob ein Saldo gegen die Richtung einer Position eine
// Meldung im Zuordnungsbericht wert ist.
//
// Nur Bilanzpositionen haben eine Richtung, die verletzt werden kann. In der
// Staffel des § 275 HGB ist der umgekehrte Saldo der Normalfall: ein
// Aufwandskonto mit Habensaldo ist eine Erstattung, ein Erlöskonto mit
// Sollsaldo eine Gutschrift; beides ist richtig gebucht. Dasselbe gilt für die
// statistischen Konten der Klasse 9 und für Positionen, deren Bezeichnung
// ausdrücklich beide Richtungen nennt (Gewinn-/Verlustvortrag). Meldete der
// Bericht auch sie, hätte fast jeder Abschluss einen Befund — und ein Befund,
// den jeder Abschluss hat, wird nicht mehr gelesen.
func reportsWrongSign(def lineDef) bool {
	if def.Section != domain.SectionAssets && def.Section != domain.SectionLiabilities {
		return false
	}
	return !def.Bidirectional
}

// contradicts meldet, ob der Saldo der natürlichen Richtung der Position
// widerspricht.
func contradicts(def lineDef, value domain.Cents) bool {
	if def.Credit {
		return value > 0
	}
	return value < 0
}

// amountFor bringt einen Sollsaldo in die Darstellungsrichtung der Position:
// Aktivposten positiv im Soll, Passivposten positiv im Haben, und in der
// Staffel jede Zeile als Beitrag zum Ergebnis.
func amountFor(def lineDef, value domain.Cents) domain.Cents {
	if def.Section == domain.SectionIncome || def.Credit {
		return -value
	}
	return value
}

// unassignedKey wählt die Auffangzeile für ein Konto ohne Gliederungsposition.
//
// Ein Erfolgskonto gehört auch dann in die Staffel, wenn seine Position
// unbekannt ist. Legte man es auf die Bilanz, ginge die zwar auf, aber das
// Jahresergebnis der Gewinn- und Verlustrechnung enthielte den Saldo nicht —
// Nummer 17 und Posten A.V wichen voneinander ab, und ein Aufwand stünde als
// Vermögensgegenstand auf der Aktivseite.
func unassignedKey(acc domain.Account, current, prior domain.Cents) string {
	if acc.StatementType == "GuV" {
		return "guv.X"
	}
	value := current
	if value == 0 {
		value = prior
	}
	if value < 0 {
		return "passiva.X"
	}
	return "aktiva.X"
}

// lines baut die Zeilen eines Abschnitts, rollt die Werte der Unterpositionen
// nach oben und schneidet alles ab, was die Gliederungstiefe nicht zeigt.
func (b *builder) lines(defs []lineDef, maxLevel int) []domain.StatementLine {
	out := make([]domain.StatementLine, 0, len(defs))
	for _, def := range defs {
		if def.Level > maxLevel {
			continue
		}
		line := toLine(def)
		line.Amount = b.total(def.Key, b.own)
		line.PriorAmount = b.total(def.Key, b.ownPrior)
		line.Accounts = b.accountsBelow(def.Key, maxLevel)
		// § 265 Abs. 8 HGB: ein Posten ohne Betrag in beiden Jahren darf
		// entfallen. Die Zwischensummen der Staffel bleiben stehen, denn sie
		// tragen das Ergebnis auch dann, wenn es null ist.
		line.Omitted = !line.IsSubtotal && !b.hasValue(def.Key)
		out = append(out, line)
	}
	return out
}

// hasValue meldet, ob auf der Position selbst oder auf einer ihrer
// Unterpositionen in einem der beiden Jahre ein Betrag steht.
//
// Nicht der Wert der Position entscheidet, sondern ob unter ihr etwas steht:
// zwei Unterposten, die sich gegenseitig aufheben, ergeben in der Summe null,
// und die Obergruppe verschwände mit ihnen.
func (b *builder) hasValue(key string) bool {
	if b.own[key] != 0 || b.ownPrior[key] != 0 {
		return true
	}
	for _, child := range lineChildren[key] {
		if b.hasValue(child) {
			return true
		}
	}
	return false
}

// total ist der Wert einer Position einschließlich aller Unterpositionen.
func (b *builder) total(key string, values map[string]domain.Cents) domain.Cents {
	sum := values[key]
	for _, child := range lineChildren[key] {
		sum += b.total(child, values)
	}
	return sum
}

// accountsBelow sammelt die Konten, die zu dieser Position gehören — und die
// der Unterpositionen, die in dieser Tiefe nicht mehr eigens erscheinen. So
// bleibt der Weg von der Zeile zum Konto in jeder Tiefe offen.
func (b *builder) accountsBelow(key string, maxLevel int) []domain.StatementAccount {
	out := append([]domain.StatementAccount(nil), b.accounts[key]...)
	for _, child := range lineChildren[key] {
		if lineIndex[child].Level <= maxLevel {
			continue
		}
		out = append(out, b.accountsBelow(child, maxLevel)...)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}

// applyIncomeSubtotals setzt die beiden Zwischensummen der Staffel: Nr. 15
// Ergebnis nach Steuern und Nr. 17 Jahresüberschuss/Jahresfehlbetrag.
func (b *builder) applyIncomeSubtotals(lines []domain.StatementLine) {
	var afterTax, afterTaxPrior domain.Cents
	for _, key := range incomeResultKeys {
		amount, prior := amountOf(lines, key)
		afterTax += amount
		afterTaxPrior += prior
	}
	otherTaxes, otherTaxesPrior := amountOf(lines, "guv.16")
	// Die Auffangzeile geht in Nummer 17 ein, nicht schon in Nummer 15: wo ein
	// ungeklärtes Erfolgskonto in der Staffel steht, weiß niemand, und die
	// Zwischensumme „Ergebnis nach Steuern" soll nicht so tun, als wüsste sie es.
	// Nummer 17 muss den Betrag dagegen tragen, weil derselbe Betrag als Posten
	// A.V auf der Passivseite steht.
	unassigned, unassignedPrior := amountOf(lines, "guv.X")
	for i := range lines {
		switch lines[i].Key {
		case "guv.15":
			lines[i].Amount, lines[i].PriorAmount = afterTax, afterTaxPrior
		case "guv.17":
			lines[i].Amount = afterTax + otherTaxes + unassigned
			lines[i].PriorAmount = afterTaxPrior + otherTaxesPrior + unassignedPrior
		}
	}
}

// finish zählt die Auffangpositionen aus. § 265 Abs. 2 HGB verlangt
// Vergleichbarkeit; ein Abschluss, dessen Gewicht in den „sonstigen" Posten
// liegt, ist zwar richtig, aber wenig aussagekräftig — und das soll man sehen,
// bevor er das Haus verlässt.
func (b *builder) finish() domain.AssignmentReport {
	report := b.report
	for _, def := range allLines {
		if !def.Fallback {
			continue
		}
		entries := b.accounts[def.Key]
		if len(entries) == 0 {
			continue
		}
		var sum domain.Cents
		for _, entry := range entries {
			sum += entry.Amount
		}
		report.Fallbacks = append(report.Fallbacks, domain.FallbackCount{
			Key: def.Key, Label: def.Label, Accounts: len(entries), Amount: sum,
		})
	}
	return report
}

// AccountsWithBalance reduziert den Kontenplan auf die Konten, die einen Saldo
// tragen. Der Zuordnungsbericht und der Kontennachweis der E-Bilanz fragen
// beide danach.
func AccountsWithBalance(accounts []domain.Account) []domain.Account {
	out := make([]domain.Account, 0, len(accounts))
	for _, acc := range accounts {
		if balance(acc) == 0 {
			continue
		}
		out = append(out, acc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}

// StatementKeyForAccount nennt die Gliederungsposition eines Kontos, wie sie
// sich aus seinem Saldo ergibt. Die E-Bilanz braucht dieselbe Entscheidung wie
// die Bilanz — nicht eine zweite, die anders ausfallen könnte.
func StatementKeyForAccount(acc domain.Account) (string, bool) {
	target, ok := positionTargets[acc.PositionID]
	if !ok {
		return "", false
	}
	value := balance(acc)
	def := lineIndex[target.Key]
	if value != 0 && contradicts(def, value) && target.AltKey != "" {
		return target.AltKey, true
	}
	return target.Key, true
}

// PositionCount is the number of SKR04 positions the mapping table covers. It
// exists for the test that guards completeness against the catalog.
func PositionCount() int { return len(positionTargets) }
