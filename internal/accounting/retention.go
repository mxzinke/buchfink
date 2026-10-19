package accounting

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/calendar"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Aufbewahrungsfristen, datiert.
//
// Datiert, weil sie sich geändert haben: das Vierte Bürokratieentlastungsgesetz
// hat die Frist für Buchungsbelege und Rechnungen zum 1.1.2025 von zehn auf acht
// Jahre verkürzt (§ 257 Abs. 4 HGB, § 147 Abs. 3 Satz 1 AO). Eine Konstante „8"
// wäre für einen Beleg aus 2012 falsch, eine Konstante „10" für einen aus 2025 —
// die Frist richtet sich nach dem Entstehungsjahr, und deshalb rechnet sie
// eine Funktion aus und keine Zahl im Code.
//
// Die Verkürzung wirkt auf laufende Fristen: für Unterlagen, deren zehnjährige
// Frist am 1.1.2025 noch nicht abgelaufen war, gilt die kürzere. Nur wo die
// alte Frist vorher schon abgelaufen war, bleibt es bei ihr — dort ist die
// Aufbewahrung ohnehin beendet, und eine rückwirkend verlängerte Frist wäre die
// falsche Auskunft.
// Die Zahlen stehen in retention_rules.json (ARC-01 K4);
// dieser Teil rechnet mit ihnen.

// yearsOfClass und legalBasisOf lesen die Frist und ihre Fundstelle aus der
// Ressource. Eine fehlende Klasse ergibt null Jahre und keinen erfundenen Wert:
// eine Frist, die niemand hinterlegt hat, ist keine Frist.
func yearsOfClass(class domain.RetentionClass) int {
	rules, ok := rulesForClass(class)
	if !ok {
		return 0
	}
	return rules.Years
}

func legalBasisOf(class domain.RetentionClass) string {
	rules, ok := rulesForClass(class)
	if !ok {
		return ""
	}
	return rules.LegalBasis
}

// classOf ordnet einer Objektart ihre Aufbewahrungsklasse zu.
//
// Die Zuordnung steht an einer Stelle, weil sie sonst zwischen Belegliste und
// Verfahrensdokumentation doppelt stünde und irgendwann verschieden.
func classOf(kind domain.RetentionKind) (domain.RetentionClass, string) {
	rules, ok := classForKind(kind)
	if !ok {
		return domain.RetentionClassNone, ""
	}
	return rules.Class, rules.LegalBasis
}

// LegalBasisForClass nennt die Fundstelle einer Aufbewahrungsklasse.
//
// Das Schlüsselverzeichnis der Datenüberlassung braucht sie ohne eine
// Objektart: dort steht die Klasse als Code, und der Prüfer soll die Norm
// daneben lesen können, ohne sich erst eine passende Objektart zu suchen.
func LegalBasisForClass(class domain.RetentionClass) string {
	return legalBasisOf(class)
}

// yearsFor liefert die Frist in Jahren für eine Klasse und ein Entstehungsjahr.
//
// Das Entstehungsjahr entscheidet, weil eine Fristverkürzung nicht rückwirkt,
// wo die alte Frist am Tag der Ablösung schon abgelaufen war: für einen Beleg
// aus 2013 endete die Zehnjahresfrist am 31.12.2023, und die seit dem 1.1.2025
// geltenden acht Jahre wären dort die falsche Auskunft über einen längst
// beendeten Vorgang.
func yearsFor(class domain.RetentionClass, originYear int) int {
	rules, ok := rulesForClass(class)
	if !ok {
		return 0
	}
	if prev := rules.Previous; prev != nil {
		if replaced := yearOf(prev.ReplacedFrom); replaced > 0 && originYear+prev.Years < replaced {
			return prev.Years
		}
	}
	return rules.Years
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
	// Der Hinweis auf die Verkürzung steht dort, wo sie gegriffen hat: an einem
	// Beleg, dessen Frist kürzer ist als die vorige. Sein Text kommt aus der
	// Ressource, damit die Anzeige nicht etwas anderes behauptet als die
	// Tabelle, nach der gerechnet wurde.
	if rules, ok := rulesForClass(class); ok && rules.Previous != nil && years == rules.Years {
		info.Note += " " + rules.Previous.Note
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
// aufbewahrt wird und aus welchem Rechtsgrund.
//
// Es steht im Code, weil es dieselben Fristen nennen muss, nach denen
// RetentionFor rechnet; die Verfahrensdokumentation übernimmt es von hier.
// Buchfink löscht nichts von sich aus, auch nach Fristablauf nicht.
func DeletionConcept() []DeletionConceptRow {
	return []DeletionConceptRow{
		{
			Category:   "Journal, Hauptbuch, Summen- und Saldenlisten",
			Class:      domain.RetentionClassBooks,
			Years:      yearsOfClass(domain.RetentionClassBooks),
			LegalBasis: legalBasisOf(domain.RetentionClassBooks),
			Note: "Die Buchungen sind Handelsbücher. Sie werden nicht gelöscht; eine " +
				"Korrektur geschieht über die Generalumkehr.",
		},
		{
			Category:   "Jahresabschlüsse, Eröffnungsbilanz, Inventare, Anhang",
			Class:      domain.RetentionClassBooks,
			Years:      yearsOfClass(domain.RetentionClassBooks),
			LegalBasis: legalBasisOf(domain.RetentionClassBooks),
			Note:       "Aufzubewahren im Original bzw. in der festgestellten Fassung.",
		},
		{
			Category:   "Festschreibungen, Zeitstempel, Änderungsprotokoll",
			Class:      domain.RetentionClassBooks,
			Years:      yearsOfClass(domain.RetentionClassBooks),
			LegalBasis: legalBasisOf(domain.RetentionClassBooks),
			Note: "Organisationsunterlagen zum Verständnis der Bücher. Sie werden zusammen " +
				"mit dem Geschäftsjahr aufbewahrt, auf das sie sich beziehen.",
		},
		{
			Category:   "Verfahrensdokumentation und ihre Fassungen",
			Class:      domain.RetentionClassBooks,
			Years:      yearsOfClass(domain.RetentionClassBooks),
			LegalBasis: legalBasisOf(domain.RetentionClassBooks),
			Note: "Jede Fassung wird über die Frist der Jahre aufbewahrt, in denen sie " +
				"gegolten hat (GoBD Rz. 151).",
		},
		{
			Category:   "Umsatzsteuer-Voranmeldungen, Zusammenfassende Meldungen",
			Class:      domain.RetentionClassBooks,
			Years:      yearsOfClass(domain.RetentionClassBooks),
			LegalBasis: legalBasisOf(domain.RetentionClassBooks),
			Note:       "Steuererklärungen und die Aufzeichnungen, aus denen sie abgeleitet sind.",
		},
		{
			Category:   "Anlagenverzeichnis",
			Class:      domain.RetentionClassBooks,
			Years:      yearsOfClass(domain.RetentionClassBooks),
			LegalBasis: legalBasisOf(domain.RetentionClassBooks),
			Note: "Der Anschaffungsbeleg trägt die Bemessungsgrundlage der Abschreibung und " +
				"wirkt über die gesamte Nutzungsdauer fort.",
		},
		{
			Category:   "Eingangs- und Ausgangsrechnungen, Buchungsbelege, Kontoauszüge, Eigenbelege",
			Class:      domain.RetentionClassVouchers,
			Years:      yearsOfClass(domain.RetentionClassVouchers),
			LegalBasis: legalBasisOf(domain.RetentionClassVouchers),
			Note: "Acht Jahre für Belege, deren Frist am 1.1.2025 noch lief; für früher " +
				"abgelaufene bleibt es bei zehn Jahren.",
		},
		{
			Category:   "Handelsbriefe und sonstige für die Besteuerung bedeutsame Unterlagen",
			Class:      domain.RetentionClassLetters,
			Years:      yearsOfClass(domain.RetentionClassLetters),
			LegalBasis: legalBasisOf(domain.RetentionClassLetters),
			Note:       "Verträge, Schriftwechsel, Bescheide.",
		},
		{
			Category:   "Stammdaten von Geschäftspartnern",
			Class:      domain.RetentionClassBooks,
			Years:      yearsOfClass(domain.RetentionClassBooks),
			LegalBasis: legalBasisOf(domain.RetentionClassBooks),
			Note: "Sie sind Bestandteil der Buchungen und werden mit ihnen aufbewahrt. Ein " +
				"Löschverlangen nach Art. 17 DSGVO greift währenddessen nicht (Art. 17 Abs. 3 " +
				"Buchst. b DSGVO); der Kontakt wird stattdessen gesperrt und ist für neue " +
				"Vorgänge nicht mehr wählbar.",
		},
	}
}

// SystemChangeNote beschreibt die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO:
// so lange ist ein Altsystem nach der Datenübernahme für den Datenzugriff
// verfügbar zu halten. Leerer Stichtag heißt: heute.
func SystemChangeNote(changeDate, today string) string {
	end := AddYearsISO(changeDate, 5)
	if end == "" {
		return ""
	}
	if today == "" {
		today = calendar.Today(nil)
	}
	if today > end {
		return fmt.Sprintf(
			"Die Umstellung aus dem Altsystem war am %s. Die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO ist am %s abgelaufen; das Altsystem muss für den Datenzugriff nicht mehr verfügbar gehalten werden.",
			changeDate, end)
	}
	return fmt.Sprintf(
		"Die Umstellung aus dem Altsystem war am %s. Bis zum %s ist das Altsystem für den Datenzugriff verfügbar zu halten (§ 147 Abs. 6 Satz 6 AO).",
		changeDate, end)
}

// BlockedContactAnswer ist die Antwort an eine betroffene Person, die die
// Löschung ihrer Daten verlangt.
//
// Sie steht als fertiger Text und nicht als Stichwortliste, weil eine falsch
// formulierte Ablehnung eines Löschverlangens teurer ist als jede Buchung, um
// die es geht. Wer ein Verlangen nicht erfüllt, nennt die Gründe und weist auf
// Beschwerde und Rechtsbehelf hin (Art. 12 Abs. 4 DSGVO). Buchfink löscht
// nicht; nach Ablauf der Frist löscht das Unternehmen als Verantwortlicher,
// und so verspricht es die Antwort.
func BlockedContactAnswer(name string) string {
	if name == "" {
		name = "Ihrer Person"
	}
	return fmt.Sprintf(
		"Die zu %s gespeicherten Daten stammen aus Geschäftsvorfällen, die wir buchhalterisch "+
			"erfasst haben. Sie unterliegen der handels- und steuerrechtlichen Aufbewahrungspflicht "+
			"(§ 257 HGB, § 147 AO). Nach Art. 17 Abs. 3 Buchst. b DSGVO besteht ein Anspruch auf "+
			"Löschung nicht, soweit die Verarbeitung zur Erfüllung einer rechtlichen Verpflichtung "+
			"erforderlich ist. Ihrem Löschverlangen können wir deshalb erst nach Ablauf der "+
			"Aufbewahrungsfrist nachkommen. Bis dahin haben wir die Daten gesperrt: wir verwenden "+
			"sie für keinen neuen Vorgang und bewahren sie nur noch auf, um diese Pflicht zu "+
			"erfüllen. Sobald die Frist abgelaufen ist, löschen wir sie. Sie können sich bei einer "+
			"Datenschutz-Aufsichtsbehörde beschweren (Art. 77 DSGVO) und gerichtlichen Rechtsschutz "+
			"in Anspruch nehmen (Art. 79 DSGVO).", name)
}

// RetentionForClass liefert die Frist einer ausdrücklich gewählten Klasse.
//
// Der Gegenstück zu RetentionFor: dort folgt die Klasse aus der Objektart, hier
// gibt sie jemand vor. Gebraucht wird das für die Verlängerung der Frist am
// einzelnen Beleg (ARC-01 K2) — sie ist eine Entscheidung des Unternehmers und
// keine Ableitung aus der Belegart.
func RetentionForClass(class domain.RetentionClass, originYear int) domain.RetentionInfo {
	info := domain.RetentionInfo{
		Class:      class,
		OriginYear: originYear,
		LegalBasis: legalBasisOf(class),
	}
	years := yearsFor(class, originYear)
	if years <= 0 || originYear <= 0 {
		info.Note = "Für diese Klasse ist keine Aufbewahrungsfrist hinterlegt."
		return info
	}
	info.Years = years
	info.RetentionEnd = fmt.Sprintf("%d-12-31", originYear+years)
	info.EarliestDeletion = fmt.Sprintf("%d-01-01", originYear+years+1)
	info.Note = fmt.Sprintf(
		"Die Frist beginnt mit dem Schluss des Jahres %d und läuft %d Jahre bis zum %s; gelöscht werden darf ab dem %s.",
		originYear, years, info.RetentionEnd, info.EarliestDeletion)
	return info
}
