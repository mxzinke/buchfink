package domain

// Die Beanstandungsliste einer Eingangsrechnung (RECH-02 K5, RECH-07 K2).
//
// Bisher lagen die Befunde in zwei getrennten Töpfen: die Regelverstöße der
// E-Rechnungsprüfung als JSON am Beleg (Receipt.ValidationFindings) und die
// Befunde des Vorsteuerabzugs im Buchungsweg. Beide beantworten dieselbe Frage
// — „was stimmt an dieser Rechnung nicht?" —, und wer sie beantworten muss,
// bekam sie an zwei Stellen und in zwei Formen.
//
// Zusammengeführt werden sie hier, und zwar klassifiziert. Die Klasse ist keine
// Sortierhilfe: sie sagt, wer den Fehler beheben kann und was er kostet. Ein
// Formatfehler ist ein Fehler des Lieferantensystems, ein Verstoß gegen eine
// Geschäftsregel ist ein Fehler der Rechnung, und eine fehlende Pflichtangabe
// des § 14 Abs. 4 UStG kostet den Vorsteuerabzug. Drei verschiedene Gespräche
// mit dem Lieferanten — und drei verschiedene Folgen für die eigene Buchung.

// ValidationFindingClass ist die Fehlerklasse eines Befunds.
type ValidationFindingClass string

const (
	// ValidationClassFormat ist der Formatfehler: der Datensatz ist nicht
	// lesbar oder verletzt das Schema seiner Syntax. Eine Rechnung, die sich
	// nicht lesen lässt, ist keine E-Rechnung im Sinne des § 14 Abs. 1 Satz 3
	// UStG — sie gilt als sonstige Rechnung, und ab 2028 genügt sie nicht mehr.
	ValidationClassFormat ValidationFindingClass = "format"
	// ValidationClassBusinessRule ist der Verstoß gegen eine Geschäftsregel der
	// EN 16931 (BR-…) oder der deutschen Ausprägung (BR-DE-…). Der Datensatz
	// ist lesbar, aber nicht normgerecht.
	ValidationClassBusinessRule ValidationFindingClass = "business_rule"
	// ValidationClassContent ist der Inhaltsfehler: eine Pflichtangabe der
	// §§ 14, 14a UStG fehlt oder widerspricht den Stammdaten. Das ist die
	// Klasse, die Geld kostet — ohne die Angabe gibt es keinen Vorsteuerabzug
	// (§ 15 Abs. 1 Satz 1 Nr. 1 UStG).
	ValidationClassContent ValidationFindingClass = "content"
)

// Label ist der Klartext für die Oberfläche.
func (c ValidationFindingClass) Label() string {
	switch c {
	case ValidationClassFormat:
		return "Formatfehler"
	case ValidationClassBusinessRule:
		return "Geschäftsregelfehler"
	case ValidationClassContent:
		return "Inhaltsfehler"
	default:
		return "Sonstiger Befund"
	}
}

// AllValidationFindingClasses listet die Klassen in fester Reihenfolge: vom
// Formalen zum Teuren.
func AllValidationFindingClasses() []ValidationFindingClass {
	return []ValidationFindingClass{
		ValidationClassFormat, ValidationClassBusinessRule, ValidationClassContent,
	}
}

// ValidationFinding ist ein Befund an einer Eingangsrechnung.
type ValidationFinding struct {
	Class ValidationFindingClass `json:"class"`
	// Rule ist die Kennung der Regel: „BR-DE-15", „input_tax_issuer_address"
	// oder — beim Formatfehler — die Syntax, an der das Lesen scheiterte.
	Rule string `json:"rule"`
	// Severity ist „fatal", „warning" oder „information", wie das Regelwerk sie
	// vergibt. Buchfink erfindet keine eigene Einstufung: die Norm sagt selbst,
	// welche ihrer Regeln nur ein Hinweis ist.
	Severity string `json:"severity"`
	// Where nennt die Stelle im Dokument, etwa „Position 3". Leer, wenn der
	// Befund das ganze Dokument betrifft.
	Where   string `json:"where,omitempty"`
	Message string `json:"message"`
	// Norm ist die Fundstelle: die Vorschrift oder die Norm, aus der die Regel
	// stammt. Ohne sie ist ein Befund eine Behauptung des Programms.
	Norm string `json:"norm,omitempty"`
	// InputTaxEffect sagt in einem Satz, was der Befund für den Vorsteuerabzug
	// bedeutet. Er steht am Befund und nicht in einer Legende, weil die Antwort
	// je Klasse verschieden ist und der Anwender sie dort braucht, wo er den
	// Befund liest.
	InputTaxEffect string `json:"inputTaxEffect"`
	// Blocking meldet, ob der Befund die Buchung mit Vorsteuer anhält.
	Blocking bool `json:"blocking"`
}

// ValidationFindingGroup sind die Befunde einer Klasse.
type ValidationFindingGroup struct {
	Class    ValidationFindingClass `json:"class"`
	Label    string                 `json:"label"`
	Findings []ValidationFinding    `json:"findings"`
}

// ReceiptFindings ist die Beanstandungsliste eines Belegs.
type ReceiptFindings struct {
	ReceiptID     uint   `json:"receiptId"`
	ReceiptNumber string `json:"receiptNumber"`
	// Checked meldet, ob der strukturierte Teil überhaupt geprüft wurde. Ohne
	// diese Unterscheidung sähe ein ungeprüfter Beleg aus wie ein fehlerfreier
	// — Schweigen als Zustimmung gelesen, obwohl nur nie gefragt wurde.
	Checked bool                     `json:"checked"`
	Groups  []ValidationFindingGroup `json:"groups"`
	// Total ist die Zahl aller Befunde, Blocking die der blockierenden.
	Total    int `json:"total"`
	Blocking int `json:"blocking"`
}

// EnsureLists macht aus nil-Scheiben leere. Die Liste geht als JSON an die
// Oberfläche, und `null.length` nähme dort den ganzen Baum mit.
func (f *ReceiptFindings) EnsureLists() {
	if f.Groups == nil {
		f.Groups = []ValidationFindingGroup{}
	}
	for i := range f.Groups {
		if f.Groups[i].Findings == nil {
			f.Groups[i].Findings = []ValidationFinding{}
		}
	}
}
