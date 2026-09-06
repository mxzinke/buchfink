package accounting

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Aufbewahrungsfristen, datiert.
//
// Datiert, weil sie sich geändert haben: das Vierte Bürokratieentlastungsgesetz
// hat die Frist für Buchungsbelege und Rechnungen zum 1.1.2025 von zehn auf acht
// Jahre verkürzt (§ 257 Abs. 4 HGB, § 147 Abs. 3 Satz 1 AO). Eine Konstante „8"
// wäre für einen Beleg aus 2012 falsch, eine Konstante „10" für einen aus 2025 —
// die Frist hängt am Entstehungsjahr, und deshalb rechnet sie eine Funktion aus
// und keine Zahl im Code.
//
// Die Verkürzung wirkt auf laufende Fristen: für Unterlagen, deren zehnjährige
// Frist am 1.1.2025 noch nicht abgelaufen war, gilt die kürzere. Nur wo die
// alte Frist vorher schon abgelaufen war, bleibt es bei ihr — dort ist die
// Aufbewahrung ohnehin beendet, und eine rückwirkend verlängerte Frist wäre die
// falsche Auskunft.
const (
	// retentionYearsBooks ist die Frist der Handelsbücher und Abschlüsse.
	retentionYearsBooks = 10
	// retentionYearsVouchersFrom2025 ist die verkürzte Belegfrist.
	retentionYearsVouchersFrom2025 = 8
	// retentionYearsVouchersBefore ist die Belegfrist vor dem Vierten
	// Bürokratieentlastungsgesetz.
	retentionYearsVouchersBefore = 10
	// retentionYearsLetters ist die Frist der Handelsbriefe und der sonstigen
	// Unterlagen.
	retentionYearsLetters = 6
	// beg4EffectiveYear ist das Jahr, in dem die verkürzte Belegfrist in Kraft
	// getreten ist.
	beg4EffectiveYear = 2025
)

const (
	legalBasisBooks    = "§ 257 Abs. 1 Nr. 1, Abs. 4 HGB; § 147 Abs. 1 Nr. 1, Abs. 3 AO"
	legalBasisVouchers = "§ 257 Abs. 1 Nr. 4, Abs. 4 HGB; § 147 Abs. 1 Nr. 4, Abs. 3 AO"
	legalBasisLetters  = "§ 257 Abs. 1 Nr. 2 und 3, Abs. 4 HGB; § 147 Abs. 1 Nr. 2, 3 und 5, Abs. 3 AO"
)

// classOf ordnet einer Objektart ihre Aufbewahrungsklasse zu.
//
// Die Zuordnung steht an einer Stelle, weil sie sonst zwischen Belegliste,
// Löschbericht und Verfahrensdokumentation dreimal stünde und irgendwann
// dreimal verschieden.
func classOf(kind domain.RetentionKind) (domain.RetentionClass, string) {
	switch kind {
	case domain.RetentionKindJournal,
		domain.RetentionKindFestschreibung,
		domain.RetentionKindClosing,
		domain.RetentionKindVatReturn,
		domain.RetentionKindInventory,
		domain.RetentionKindOrganisation,
		// Anlagendokumente sind Organisationsunterlagen: der Anschaffungsbeleg
		// eines Anlageguts trägt die Bemessungsgrundlage der Abschreibung, und
		// die wirkt über die ganze Nutzungsdauer fort.
		domain.RetentionKindAssetDocument:
		return domain.RetentionClassBooks, legalBasisBooks
	case domain.RetentionKindReceiptInvoice,
		domain.RetentionKindReceiptStatement,
		domain.RetentionKindReceiptSelfIssue:
		return domain.RetentionClassVouchers, legalBasisVouchers
	case domain.RetentionKindReceiptLetter,
		domain.RetentionKindReceiptOther:
		return domain.RetentionClassLetters, legalBasisLetters
	default:
		return domain.RetentionClassNone, ""
	}
}

// LegalBasisForClass nennt die Fundstelle einer Aufbewahrungsklasse.
//
// Das Schlüsselverzeichnis der Datenüberlassung braucht sie ohne eine
// Objektart: dort steht die Klasse als Code, und der Prüfer soll die Norm
// daneben lesen können, ohne sich erst eine passende Objektart zu suchen.
func LegalBasisForClass(class domain.RetentionClass) string {
	switch class {
	case domain.RetentionClassBooks:
		return legalBasisBooks
	case domain.RetentionClassVouchers:
		return legalBasisVouchers
	case domain.RetentionClassLetters:
		return legalBasisLetters
	default:
		return ""
	}
}

// yearsFor liefert die Frist in Jahren für eine Klasse und ein Entstehungsjahr.
func yearsFor(class domain.RetentionClass, originYear int) int {
	switch class {
	case domain.RetentionClassBooks:
		return retentionYearsBooks
	case domain.RetentionClassLetters:
		return retentionYearsLetters
	case domain.RetentionClassVouchers:
		// Die alte Zehnjahresfrist lief bis zum 31.12. des Jahres
		// originYear+10. War sie beim Inkrafttreten am 1.1.2025 schon abgelaufen
		// — also originYear+10 < 2025 —, bleibt es bei ihr; sonst greift die
		// verkürzte.
		if originYear+retentionYearsVouchersBefore < beg4EffectiveYear {
			return retentionYearsVouchersBefore
		}
		return retentionYearsVouchersFrom2025
	default:
		return 0
	}
}

// RetentionFor liefert Klasse und frühestes Löschdatum eines Objekts.
//
// originYear ist das Jahr, in dem die Unterlage entstanden ist — bei einer
// Buchung das Geschäftsjahr, bei einem Beleg das Jahr seiner Ablage. Der
// Fristbeginn ist der Schluss dieses Kalenderjahres (§ 257 Abs. 5 HGB, § 147
// Abs. 4 AO), der letzte Aufbewahrungstag also der 31.12. des Jahres
// originYear+Jahre, und gelöscht werden darf ab dem 1.1. danach.
func RetentionFor(kind domain.RetentionKind, originYear int) domain.RetentionInfo {
	class, basis := classOf(kind)
	info := domain.RetentionInfo{
		Kind:       kind,
		Class:      class,
		OriginYear: originYear,
		LegalBasis: basis,
	}
	if class == domain.RetentionClassNone || originYear <= 0 {
		info.Note = "Für dieses Objekt ist keine Aufbewahrungsfrist hinterlegt."
		return info
	}

	years := yearsFor(class, originYear)
	info.Years = years
	info.RetentionEnd = fmt.Sprintf("%d-12-31", originYear+years)
	info.EarliestDeletion = fmt.Sprintf("%d-01-01", originYear+years+1)
	info.Note = fmt.Sprintf(
		"Die Frist beginnt mit dem Schluss des Jahres %d und läuft %d Jahre bis zum %s; gelöscht werden darf ab dem %s.",
		originYear, years, info.RetentionEnd, info.EarliestDeletion)
	if class == domain.RetentionClassVouchers && years == retentionYearsVouchersFrom2025 {
		info.Note += " Die Frist für Buchungsbelege und Rechnungen ist mit dem Vierten Bürokratieentlastungsgesetz von zehn auf acht Jahre verkürzt worden; die Verkürzung wirkt auf am 1.1.2025 noch laufende Fristen."
	}
	return info
}

// DeletionConceptRow ist eine Zeile des Löschkonzepts.
type DeletionConceptRow struct {
	Category   string                `json:"category"`
	Class      domain.RetentionClass `json:"class"`
	Years      int                   `json:"years"`
	LegalBasis string                `json:"legalBasis"`
	Note       string                `json:"note"`
}

// DeletionConcept ist das Löschkonzept: welche Datenkategorie wie lange
// aufbewahrt wird, aus welchem Rechtsgrund, und was danach geschieht.
//
// Es steht im Code und nicht in einem Dokument daneben, weil es beides sein
// muss: der Abschnitt der Verfahrensdokumentation, den ein Prüfer liest, und
// die Regel, nach der die Löschfunktion tatsächlich entscheidet. Zwei Fassungen
// desselben Konzepts laufen auseinander, und dann ist die geschriebene die,
// die nicht gilt.
func DeletionConcept() []DeletionConceptRow {
	return []DeletionConceptRow{
		{
			Category:   "Journal, Hauptbuch, Summen- und Saldenlisten",
			Class:      domain.RetentionClassBooks,
			Years:      retentionYearsBooks,
			LegalBasis: legalBasisBooks,
			Note: "Die Buchungen sind Handelsbücher. Sie werden nicht einzeln gelöscht, " +
				"sondern nur mit dem ganzen Geschäftsjahr und erst nach Fristablauf.",
		},
		{
			Category:   "Jahresabschlüsse, Eröffnungsbilanz, Inventare, Anhang",
			Class:      domain.RetentionClassBooks,
			Years:      retentionYearsBooks,
			LegalBasis: legalBasisBooks,
			Note:       "Aufzubewahren im Original bzw. in der festgestellten Fassung.",
		},
		{
			Category:   "Festschreibungen, Zeitstempel, Änderungsprotokoll",
			Class:      domain.RetentionClassBooks,
			Years:      retentionYearsBooks,
			LegalBasis: legalBasisBooks,
			Note: "Organisationsunterlagen zum Verständnis der Bücher. Sie werden zusammen " +
				"mit dem Geschäftsjahr aufbewahrt, auf das sie sich beziehen.",
		},
		{
			Category:   "Verfahrensdokumentation und ihre Fassungen",
			Class:      domain.RetentionClassBooks,
			Years:      retentionYearsBooks,
			LegalBasis: legalBasisBooks,
			Note: "Jede Fassung wird über die Frist der Jahre aufbewahrt, in denen sie " +
				"gegolten hat (GoBD Rz. 151).",
		},
		{
			Category:   "Umsatzsteuer-Voranmeldungen, Zusammenfassende Meldungen",
			Class:      domain.RetentionClassBooks,
			Years:      retentionYearsBooks,
			LegalBasis: legalBasisBooks,
			Note:       "Steuererklärungen und die Aufzeichnungen, aus denen sie abgeleitet sind.",
		},
		{
			Category:   "Anlagenverzeichnis und Anlagendokumente",
			Class:      domain.RetentionClassBooks,
			Years:      retentionYearsBooks,
			LegalBasis: legalBasisBooks,
			Note: "Der Anschaffungsbeleg trägt die Bemessungsgrundlage der Abschreibung und " +
				"wirkt über die gesamte Nutzungsdauer fort.",
		},
		{
			Category:   "Eingangs- und Ausgangsrechnungen, Buchungsbelege, Kontoauszüge, Eigenbelege",
			Class:      domain.RetentionClassVouchers,
			Years:      retentionYearsVouchersFrom2025,
			LegalBasis: legalBasisVouchers,
			Note: "Acht Jahre für Belege, deren Frist am 1.1.2025 noch lief; für früher " +
				"abgelaufene bleibt es bei zehn Jahren.",
		},
		{
			Category:   "Handelsbriefe und sonstige für die Besteuerung bedeutsame Unterlagen",
			Class:      domain.RetentionClassLetters,
			Years:      retentionYearsLetters,
			LegalBasis: legalBasisLetters,
			Note:       "Verträge, Schriftwechsel, Bescheide.",
		},
		{
			Category:   "Stammdaten von Geschäftspartnern",
			Class:      domain.RetentionClassBooks,
			Years:      retentionYearsBooks,
			LegalBasis: legalBasisBooks,
			Note: "Sie sind Bestandteil der Buchungen und werden mit ihnen aufbewahrt. Ein " +
				"Löschverlangen nach Art. 17 DSGVO greift währenddessen nicht (Art. 17 Abs. 3 " +
				"Buchst. b DSGVO); der Kontakt wird stattdessen gesperrt und ist für neue " +
				"Vorgänge nicht mehr wählbar.",
		},
	}
}

// BlockedContactAnswer ist die Antwort an eine betroffene Person, die die
// Löschung ihrer Daten verlangt.
//
// Sie steht als fertiger Text und nicht als Stichwortliste, weil die Antwort
// sonst jedes Mal neu formuliert würde — und weil eine falsch formulierte
// Ablehnung eines Löschverlangens teurer ist als jede Buchung, um die es geht.
func BlockedContactAnswer(name string) string {
	if name == "" {
		name = "Sie"
	}
	return fmt.Sprintf(
		"Die zu %s gespeicherten Daten stammen aus Geschäftsvorfällen, die wir buchhalterisch "+
			"erfasst haben. Sie unterliegen der handels- und steuerrechtlichen Aufbewahrungspflicht "+
			"(§ 257 HGB, § 147 AO). Nach Art. 17 Abs. 3 Buchst. b DSGVO besteht ein Anspruch auf "+
			"Löschung nicht, soweit die Verarbeitung zur Erfüllung einer rechtlichen Verpflichtung "+
			"erforderlich ist. Wir haben die Daten deshalb für jede weitere Verwendung gesperrt: "+
			"sie stehen in keiner Auswahl mehr zur Verfügung und werden nach Ablauf der "+
			"Aufbewahrungsfrist gelöscht. Die Verarbeitung beschränkt sich bis dahin auf die "+
			"Aufbewahrung (Art. 18 DSGVO).", name)
}
