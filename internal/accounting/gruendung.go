package accounting

import (
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Was eine Kapitalgesellschaft aufbringen muss, bevor sie angemeldet werden darf
// — und welche Pflichten aus ihrer Gründung folgen.
//
// Der Katalog steht hier als Tabelle und nicht als Kette von if-Abfragen an den
// Aufrufstellen, aus demselben Grund wie der Rechtsformkatalog in
// domain/legalform.go: eine Regel, die an drei Stellen ausformuliert ist, ist
// drei Regeln, sobald sich eine ändert.
//
// Nur Kapitalgesellschaften. Bei einer Personengesellschaft gibt es weder eine
// Vorgesellschaft noch eine Unterbilanzhaftung — der Gründungsweg gilt dort
// nicht, und eine Rechtsform, die hier fehlt, bekommt ihn deshalb gar nicht erst
// angeboten.

// FoundationRules is what one legal form demands before the Anmeldung.
type FoundationRules struct {
	// LegalForm ist die Schreibweise aus domain.LegalFormCatalog.
	LegalForm string `json:"legalForm"`
	// MinShareCapital ist das gesetzliche Mindestkapital.
	MinShareCapital domain.Cents `json:"minShareCapital"`
	// PaidInPerShareQuota ist der Anteil jeder einzelnen Geldeinlage, der vor der
	// Anmeldung eingezahlt sein muss: ein Viertel bei GmbH und AG, alles bei der
	// UG.
	PaidInPerShareQuota float64 `json:"paidInPerShareQuota"`
	// PaidInFloor ist die Untergrenze für die Summe aller Einlagen, unabhängig
	// von der Aufteilung in Geschäftsanteile.
	//
	// Bei der GmbH ist das die Hälfte des *Mindest*stammkapitals, also stets
	// 12.500 € — nicht die Hälfte des tatsächlichen Stammkapitals
	// (§ 7 Abs. 2 Satz 2 GmbHG). Wer eine GmbH mit 100.000 € gründet, schuldet
	// vor der Anmeldung 25.000 € (ein Viertel je Anteil), nicht 50.000 €.
	PaidInFloor domain.Cents `json:"paidInFloor"`
	// PaidInFloorIsFullCapital sagt, dass die Untergrenze das volle Stammkapital
	// ist und deshalb nicht als fester Betrag danebenstehen kann (UG).
	PaidInFloorIsFullCapital bool `json:"paidInFloorIsFullCapital"`
	// CashOnly schließt die Sacheinlage aus (§ 5a Abs. 2 Satz 2 GmbHG).
	CashOnly bool `json:"cashOnly"`
	// LegalReserve verlangt die gesetzliche Rücklage nach § 5a Abs. 3 GmbHG.
	LegalReserve bool `json:"legalReserve"`
	// Reference ist die Fundstelle der Einzahlungsregel.
	Reference string `json:"reference"`
	// Note sagt, was die Rechtsform im Gründungsweg besonders macht.
	Note string `json:"note"`
}

// Der Katalog. Beträge in Cent, wie überall im System.
var foundationRules = []FoundationRules{
	{
		LegalForm:           "GmbH",
		MinShareCapital:     2_500_000,
		PaidInPerShareQuota: 0.25,
		PaidInFloor:         1_250_000,
		Reference:           "§ 7 Abs. 2 GmbHG",
		Note: "Auf jeden Geschäftsanteil ist ein Viertel des Nennbetrags einzuzahlen; " +
			"zusammen müssen die Einlagen die Hälfte des Mindeststammkapitals erreichen, " +
			"also 12.500 € (§ 7 Abs. 2 GmbHG). Sacheinlagen sind vor der Anmeldung " +
			"vollständig zu bewirken (§ 7 Abs. 3 GmbHG).",
	},
	{
		LegalForm:                "UG (haftungsbeschränkt)",
		MinShareCapital:          100,
		PaidInPerShareQuota:      1.0,
		PaidInFloorIsFullCapital: true,
		CashOnly:                 true,
		LegalReserve:             true,
		Reference:                "§ 5a Abs. 2 GmbHG",
		Note: "Die Anmeldung darf erst erfolgen, wenn das Stammkapital in voller Höhe " +
			"eingezahlt ist; Sacheinlagen sind ausgeschlossen (§ 5a Abs. 2 GmbHG). " +
			"In den Jahresabschluss ist ein Viertel des um einen Verlustvortrag " +
			"geminderten Jahresüberschusses in die gesetzliche Rücklage einzustellen " +
			"(§ 5a Abs. 3 GmbHG).",
	},
	{
		LegalForm:           "AG",
		MinShareCapital:     5_000_000,
		PaidInPerShareQuota: 0.25,
		Reference:           "§ 36a Abs. 1 AktG",
		Note: "Der eingeforderte Betrag muss bei Bareinlagen mindestens ein Viertel des " +
			"geringsten Ausgabebetrags umfassen (§ 36a Abs. 1 AktG); das Grundkapital " +
			"beträgt mindestens 50.000 € (§ 7 AktG). Sacheinlagen sind vollständig zu " +
			"leisten (§ 36a Abs. 2 AktG).",
	},
}

// FoundationRulesFor returns the rules of a Rechtsform.
//
// Der zweite Rückgabewert ist die Antwort auf die Frage, ob der Gründungsweg für
// diese Rechtsform überhaupt gilt. Er ist die einzige Stelle, an der das
// entschieden wird — die Oberfläche fragt hier, statt Rechtsformnamen zu
// vergleichen.
func FoundationRulesFor(legalForm string) (FoundationRules, bool) {
	for _, r := range foundationRules {
		if r.LegalForm == legalForm {
			return r, true
		}
	}
	return FoundationRules{}, false
}

// FoundationLegalForms returns the legal forms the Gründungsweg covers.
func FoundationLegalForms() []FoundationRules {
	out := make([]FoundationRules, len(foundationRules))
	copy(out, foundationRules)
	return out
}

// RequiredPaidIn computes what must be contributed before the Anmeldung.
//
// Gerechnet wird je Geschäftsanteil und dann gegen die Untergrenze geprüft, weil
// beides nebeneinander gilt: die Viertelregel je Anteil und der Gesamtbetrag.
// Eine Sacheinlage zählt dabei mit ihrem vollen Nennbetrag, weil sie vollständig
// zu bewirken ist und § 7 Abs. 2 Satz 2 GmbHG sie mit dem Gesamtnennbetrag in
// die Rechnung nimmt.
func (r FoundationRules) RequiredPaidIn(f *domain.Foundation) domain.Cents {
	var sum domain.Cents
	for _, s := range f.Shareholders {
		sum += r.RequiredPerShare(s)
	}

	floor := r.PaidInFloor
	if r.PaidInFloorIsFullCapital {
		floor = f.ShareCapital
	}
	if sum < floor {
		return floor
	}
	return sum
}

// RequiredPerShare is the minimum contribution on a single Geschäftsanteil.
func (r FoundationRules) RequiredPerShare(s domain.Shareholder) domain.Cents {
	if s.Kind == domain.ContributionInKind {
		// Vollständig zu bewirken, § 7 Abs. 3 GmbHG bzw. § 36a Abs. 2 AktG.
		return s.ShareCapital
	}
	return quotaOf(s.ShareCapital, r.PaidInPerShareQuota)
}

// quotaOf takes a fraction of an amount and rounds up to the next full cent.
//
// Aufgerundet, nicht kaufmännisch: die Viertelregel ist eine Untergrenze. Ein
// halber Cent zu wenig ist zu wenig.
func quotaOf(amount domain.Cents, quota float64) domain.Cents {
	if quota >= 1.0 {
		return amount
	}
	if quota <= 0 || amount <= 0 {
		return 0
	}
	// Ganzzahlig gerechnet, damit kein Fließkommarest die Grenze verschiebt.
	num := int64(quota * 1_000_000)
	v := (int64(amount)*num + 999_999) / 1_000_000
	return domain.Cents(v)
}

// -------------------------------------------------------------
// Umsatzsteuer-Voranmeldung im Gründungsfall
// -------------------------------------------------------------

// RecommendedVatPeriod is the Voranmeldungszeitraum a company founded in the
// given year starts with.
//
// § 18 Abs. 2 Satz 4 UStG verlangt im Gründungsjahr und im folgenden Jahr die
// monatliche Voranmeldung. Satz 6 hat das für die Besteuerungszeiträume 2021 bis
// 2026 ausgesetzt; dort gilt der Regelfall des Satzes 2, also das Kalender-
// vierteljahr. Ab dem Besteuerungszeitraum 2027 lebt die monatliche Pflicht
// wieder auf.
//
// Die Regel steht hier und nicht als Satz in der Oberfläche, weil sie ein
// Stichjahr hat: ein fest getippter Hinweis wäre seit 2021 falsch gewesen und
// wäre es ab 2027 wieder.
func RecommendedVatPeriod(foundingYear int) string {
	if foundingYear >= 2021 && foundingYear <= 2026 {
		return "quarter"
	}
	return "month"
}

// VatPeriodReason explains the recommendation in one sentence.
func VatPeriodReason(foundingYear int) string {
	if foundingYear >= 2021 && foundingYear <= 2026 {
		return fmt.Sprintf(
			"Für eine Gründung im Jahr %d gilt das Kalendervierteljahr: § 18 Abs. 2 Satz 6 UStG "+
				"setzt die monatliche Abgabepflicht des Satzes 4 für die Besteuerungszeiträume "+
				"2021 bis 2026 aus. Ab 2027 gilt sie wieder.",
			foundingYear,
		)
	}
	return fmt.Sprintf(
		"Für eine Gründung im Jahr %d ist die Voranmeldung monatlich abzugeben, im Gründungsjahr "+
			"und im folgenden Kalenderjahr (§ 18 Abs. 2 Satz 4 UStG).",
		foundingYear,
	)
}

// -------------------------------------------------------------
// Pflichten aus der Gründung
// -------------------------------------------------------------

// Die Schlüssel der Gründungspflichten. Sie stehen in der Datenbank und dürfen
// sich deshalb nicht mehr ändern.
const (
	DutyHandelsregister     = "handelsregister"
	DutyFragebogen          = "fragebogen"
	DutyGewerbeanmeldung    = "gewerbeanmeldung"
	DutyTransparenzregister = "transparenzregister"
	DutyEroeffnungsbilanz   = "eroeffnungsbilanz"
	DutyRuecklage           = "ruecklage"
	DutyOffenlegung         = "offenlegung"
)

// FoundationDuties beginnt nach Beurkundung, Kapitaleinzahlung und Registeranmeldung.
// Jahresabschlussaufgaben werden im Abschluss geführt.
func FoundationDuties(f *domain.Foundation, rules FoundationRules, done map[string]string) []domain.FoundationDuty {
	if f == nil || f.NotarizedOn == "" {
		return nil
	}
	letterReference := "§ 35a GmbHG"
	if rules.LegalForm == "AG" {
		letterReference = "§ 80 AktG"
	}
	duties := []domain.FoundationDuty{
		{
			Key: "stammdaten", Title: "Stammdaten vervollständigen",
			Anchor: domain.AnchorBeurkundung, Where: "Einstellungen in Buchfink",
			Deadline:    "Vor dem Erstellen der ersten Unterlagen",
			Description: "Für diesen Checkpunkt genügen Firma, vollständige Anschrift und Gesellschafter mit ihren Geschäftsanteilen. Buchfink übernimmt diese Angaben in Datenblätter und andere Unterlagen. Weitere Stammdaten können Sie später ergänzen.",
			Todo:        []string{"Firma und Anschrift in den Einstellungen ergänzen.", "Gesellschafter und Geschäftsanteile prüfen oder eintragen.", "Änderungen speichern."},
		},
		{
			AcceptsProof: true, Key: DutyFragebogen, Title: "Fragebogen zur steuerlichen Erfassung übermitteln",
			Anchor: domain.AnchorBeurkundung, Where: "Mein ELSTER",
			DueDate: addMonths(f.NotarizedOn, 1), Deadline: "Innerhalb eines Monats nach der Gründung",
			Reference:   "§§ 137, 138 Abs. 1b und 4 AO",
			Description: "Die Kapitalgesellschaft muss dem Finanzamt ihre Gründung und die steuerlich erheblichen Verhältnisse mitteilen. Der Fragebogen wird elektronisch übermittelt, auch wenn noch keine Steuernummer vorliegt.",
			Todo:        []string{"Bei Mein ELSTER anmelden oder einen Zugang einrichten.", "Den Fragebogen für Kapitalgesellschaften ausfüllen und übermitteln.", "Das Übermittlungsprotokoll ablegen; die Steuernummer später in den Einstellungen ergänzen."},
			Provides:    "Buchfink stellt ein Datenblatt mit den erfassten Angaben und noch offenen Feldern bereit.",
			ActionURL:   "https://www.elster.de/eportal/formulare-leistungen/alleformulare/fsekapg", ActionLabel: "Zu ELSTER",
		},
		{
			AcceptsProof: true, Key: "ust_id", Title: "Umsatzsteuer-ID beantragen",
			Anchor: domain.AnchorBeurkundung, Where: "Bundeszentralamt für Steuern",
			Condition: "Bei Bedarf, insbesondere für EU-Geschäfte",
			Deadline:  "Vor Geschäften, für die eine USt-IdNr. benötigt wird", Reference: "§ 27a UStG",
			Description: "Prüfen Sie, ob Sie für Ihre Geschäfte eine USt-IdNr. benötigen, insbesondere für Waren oder Dienstleistungen innerhalb der EU. Die USt-IdNr. ist eine eigene Nummer zusätzlich zur Steuernummer. Legen Sie den Vergabebescheid ab und tragen Sie die Nummer in den Einstellungen ein.",
			Todo:        []string{"Prüfen, ob die USt-IdNr. bereits beantragt oder erteilt wurde.", "Falls nötig, den Antrag beim BZSt stellen.", "Vergabebescheid ablegen und USt-IdNr. in den Stammdaten ergänzen."},
			ActionURL:   "https://online.portal.bzst.de/SharedDocs/Leistungsbeschreibung/DE/vergabe_der_umsatzsteuer-identifikationsnummer_nach_27_a_UStG.html", ActionLabel: "Zum BZSt",
		},
		{
			AcceptsProof: true, Key: DutyGewerbeanmeldung, Title: "Gewerbe anmelden",
			Anchor: domain.AnchorBeurkundung, Where: "Gewerbeamt am Betriebssitz",
			Deadline: "Bei Aufnahme des Gewerbebetriebs", Reference: "§ 14 GewO",
			Description: "Die Gewerbeanzeige ist gleichzeitig mit dem Beginn des Gewerbebetriebs abzugeben. Die zuständige Gemeinde nennt die erforderlichen Unterlagen und das Verfahren für eine Gesellschaft in Gründung.",
			Todo:        []string{"Zuständige Gemeinde im Verwaltungsportal auswählen.", "Tätigkeit und Betriebsbeginn angeben und die verlangten Unterlagen einreichen.", "Bestätigung der Gewerbeanmeldung ablegen."},
			ActionURL:   "https://verwaltung.bund.de/leistungsverzeichnis/de/leistung/99050012104000", ActionLabel: "Zum Gewerbeamt",
		},
		{
			AcceptsProof: true, Key: "unfallversicherung", Title: "Anmeldung beim Unfallversicherungsträger prüfen",
			Anchor: domain.AnchorBeurkundung, Where: "Zuständige Berufsgenossenschaft oder Unfallkasse",
			Deadline: "Binnen einer Woche nach Unternehmenseröffnung", Reference: "§ 192 Abs. 1 SGB VII",
			Description: "Die Mitteilungspflicht ist bereits erfüllt, wenn die Gewerbeanzeige binnen einer Woche nach Unternehmensbeginn erstattet wurde. Andernfalls das Unternehmen beim zuständigen Unfallversicherungsträger anmelden.",
			Todo:        []string{"Prüfen, ob die rechtzeitige Gewerbeanmeldung die Mitteilung bereits abdeckt.", "Falls nötig, das Unternehmen über das Serviceportal der Unfallversicherung anmelden.", "Bescheid und Unternehmensnummer ablegen."},
			ActionURL:   "https://serviceportal-uv.dguv.de/", ActionLabel: "Zur Unfallversicherung",
		},
		{
			AcceptsProof: true, Key: "rundfunkbeitrag", Title: "Rundfunkbeitrag anmelden",
			Anchor: domain.AnchorBeurkundung, Where: "ARD ZDF Deutschlandradio Beitragsservice",
			Condition: "Für Betriebsstätten und betriebliche Fahrzeuge prüfen",
			Deadline:  "Bei Beginn der Beitragspflicht", Reference: "§§ 5, 7, 8 RBStV",
			Description: "Prüfen Sie Betriebsstätten und nicht ausschließlich privat genutzte Fahrzeuge beim Beitragsservice. Auch eine Betriebsstätte in einer bereits angemeldeten Privatwohnung kann anmeldepflichtig sein, obwohl kein zusätzlicher Beitrag anfällt. Klären Sie die Voraussetzungen und legen Sie Anmeldung oder Bestätigung der Beitragsfreiheit ab.",
			Todo:        []string{"Betriebsstätten, Beschäftigtenzahl und betriebliche Fahrzeuge zusammenstellen.", "Anmeldung und mögliche Beitragsfreiheit beim Beitragsservice klären.", "Bestätigung und Beitragsnummer als Nachweis ablegen."},
			ActionURL:   "https://www.rundfunkbeitrag.de/anmelden", ActionLabel: "Zum Beitragsservice",
		},
		{
			AcceptsProof: true, Key: "ihk", Title: "IHK-Zugehörigkeit dokumentieren",
			Anchor: domain.AnchorBeurkundung, Where: "Zuständige Industrie- und Handelskammer",
			Condition: "Bei IHK-Zugehörigkeit",
			Deadline:  "Nach Eingang des IHK-Schreibens; dessen Fristen beachten", Reference: "§ 2 IHKG",
			Description: "Die IHK-Zugehörigkeit entsteht bei erfüllten Voraussetzungen automatisch. Gewerbeamt oder Registergericht informieren die IHK normalerweise. Prüfen Sie das Begrüßungsschreiben und den Beitragsbescheid und legen Sie beides ab. Bei Handwerksbetrieben kann stattdessen oder zusätzlich die Handwerkskammer zuständig sein.",
			Todo:        []string{"Zuständige Kammer und erfasste Unternehmensdaten prüfen.", "IHK-Schreiben, Mitgliedsnummer und Beitragsbescheid dokumentieren.", "Angeforderte Angaben innerhalb der Frist des Schreibens nachreichen; bei Unklarheiten die Kammer kontaktieren."},
			ActionURL:   "https://www.ihk.de/", ActionLabel: "IHK finden",
		},
		{
			AcceptsProof: true, Key: DutyTransparenzregister, Title: "Wirtschaftlich Berechtigte melden",
			Anchor: domain.AnchorEintragung, Where: "Transparenzregister",
			Deadline: "Unverzüglich nach der Eintragung", Reference: "§§ 3, 20 GwG",
			Description: "Die Gesellschaft muss ihre wirtschaftlich Berechtigten ermitteln und dem Transparenzregister mitteilen. Zu prüfen sind auch mittelbare Beteiligungen und Kontrolle auf andere Weise; die Gesellschafterliste allein genügt nicht immer.",
			Todo:        []string{"Wirtschaftlich Berechtigte und deren erforderliche Angaben ermitteln.", "Auf transparenzregister.de registrieren und die Meldung abgeben.", "Eingangsbestätigung der Meldung ablegen und spätere Änderungen nachmelden."},
			ActionURL:   "https://www.transparenzregister.de/", ActionLabel: "Zum Transparenzregister",
		},
		{
			AcceptsProof: true, Key: DutyEroeffnungsbilanz, Title: "Eröffnungsbilanz aufstellen",
			Anchor: domain.AnchorBeurkundung, Where: "In Buchfink und in geeigneter E-Bilanz-Software",
			Deadline: "Zu Beginn des Handelsgewerbes", Reference: "§ 242 Abs. 1 HGB, § 5b Abs. 1 EStG",
			Description: "Die Eröffnungsbilanz dokumentiert Vermögen und Schulden zu Beginn der Buchführung. Sie gehört zur Gründung. Für die steuerliche Übermittlung ist geeignete E-Bilanz-Software erforderlich; Buchfink übermittelt nicht direkt.",
			Todo:        []string{"Die bereits erfolgte Kapitalzeichnung und Einzahlung buchen.", "Eröffnungsbilanz prüfen und ablegen.", "Steuerliche Übermittlung über geeignete E-Bilanz-Software erledigen."},
			Provides:    "Buchfink erstellt eine Bilanzvorschau, ein PDF und eine noch nicht amtlich validierte XBRL-Arbeitsdatei.",
		},
		{
			Key: "geschaeftsbriefe", Title: "Pflichtangaben auf Geschäftsbriefen ergänzen",
			Anchor: domain.AnchorEintragung, Where: "Briefvorlagen, geschäftliche E-Mails und Unternehmensstammdaten",
			Deadline: "Ab Eintragung im Geschäftsverkehr", Reference: letterReference,
			Description: "Geschäftsbriefe müssen unter anderem Rechtsform, Sitz, Registergericht, Registernummer und die gesetzlich vorgeschriebenen Angaben zur Vertretung enthalten. Für eine AG gehören dazu auch Vorstand und Aufsichtsratsvorsitz. Vor der Eintragung muss der Gründungsstatus erkennbar bleiben.",
			Todo:        []string{"Register- und Vertretungsangaben in den Unternehmensstammdaten ergänzen.", "Briefvorlagen und geschäftliche E-Mail-Signaturen aktualisieren."},
		},
		{
			AcceptsProof: true, Key: "betriebsnummer", Title: "Betriebsnummer beantragen",
			Anchor: domain.AnchorBeurkundung, Where: "Bundesagentur für Arbeit",
			Condition: "Bei meldepflichtigen Beschäftigten, auch Minijobs",
			Deadline:  "Für die erste Anmeldung zur Sozialversicherung", Reference: "§ 18i SGB IV",
			Description: "Für die Meldung der ersten Beschäftigten benötigen Sie eine Betriebsnummer. Das gilt auch für Minijobs und Auszubildende. Für den Antrag wird die Unternehmensnummer der gesetzlichen Unfallversicherung benötigt.",
			Todo:        []string{"Unternehmensnummer beim Unfallversicherungsträger bereithalten.", "Betriebsnummer online beantragen.", "Vergabeschreiben ablegen und die Nummer an die Lohnabrechnung weitergeben."},
			ActionURL:   "https://www.arbeitsagentur.de/unternehmen/betriebsnummern-service/alles-wichtige/beantragung", ActionLabel: "Betriebsnummer beantragen",
		},
		{
			AcceptsProof: true, Key: "sozialversicherung", Title: "Beschäftigte zur Sozialversicherung anmelden",
			Anchor: domain.AnchorBeurkundung, Where: "Krankenkasse oder Minijob-Zentrale über die Lohnabrechnung bzw. das SV-Meldeportal",
			Condition: "Bei meldepflichtigen Beschäftigten, auch Minijobs",
			Deadline:  "Mit erster Abrechnung, spätestens nach 6 Wochen", Reference: "§ 6 DEÜV, § 28a Abs. 4 SGB IV",
			Description: "Melden Sie Beschäftigte bei der zuständigen Einzugsstelle an, Minijobs bei der Minijob-Zentrale. In Branchen mit Sofortmeldepflicht ist zusätzlich spätestens bei Beschäftigungsaufnahme eine Sofortmeldung nötig. Die reguläre Anmeldung bleibt erforderlich.",
			Todo:        []string{"Beschäftigungsart, Krankenkasse und Sozialversicherungsdaten klären.", "Sofortmeldepflicht prüfen und gegebenenfalls vor Arbeitsbeginn melden.", "Anmeldung über geeignete Lohnsoftware oder das SV-Meldeportal übermitteln und Protokoll ablegen."},
			ActionURL:   "https://app.sv-meldeportal.de/", ActionLabel: "Zum SV-Meldeportal",
		},
		{
			Key: "lohnabrechnung", Title: "Lohnabrechnung und Lohnsteuer einrichten",
			Anchor: domain.AnchorBeurkundung, Where: "Lohnsoftware oder Lohnbüro und Mein ELSTER",
			Condition: "Bei Beschäftigten mit Arbeitslohn",
			Deadline:  "Vor der ersten Lohnabrechnung", Reference: "§§ 39e, 41a EStG",
			Description: "Richten Sie die Lohnabrechnung ein und klären Sie das Lohnsteuerverfahren. Für den individuellen Lohnsteuerabzug werden die ELStAM abgerufen. Bei pauschal besteuerten Minijobs gelten andere Regeln. Lohnsteueranmeldungen und Zahlungen müssen anschließend fristgerecht erfolgen.",
			Todo:        []string{"Lohnsoftware oder Lohnbüro einrichten und Personaldaten erfassen.", "ELStAM abrufen, soweit erforderlich, und den Lohnsteuer-Anmeldungszeitraum klären.", "Anmeldungen und Zahlungen bis zum 10. Tag nach dem jeweiligen Anmeldungszeitraum organisieren."},
			ActionURL:   "https://www.elster.de/eportal/start?themaGlobal=help_arbeitgeber_eop", ActionLabel: "Zu ELSTER",
		},
		{
			AcceptsProof: true, Key: "arbeitsschutz", Title: "Arbeitsschutz organisieren und dokumentieren",
			Anchor: domain.AnchorBeurkundung, Where: "Im Betrieb mit Unterstützung des Unfallversicherungsträgers",
			Condition: "Bei Beschäftigten",
			Deadline:  "Vor Aufnahme der Tätigkeit", Reference: "§§ 5, 6, 12 ArbSchG",
			Description: "Beurteilen Sie die Gefährdungen der Arbeitsplätze, legen Sie Schutzmaßnahmen fest und dokumentieren Sie die Ergebnisse. Beschäftigte müssen vor Aufnahme ihrer Tätigkeit unterwiesen werden. Klären Sie auch die erforderliche arbeitsmedizinische und sicherheitstechnische Betreuung.",
			Todo:        []string{"Gefährdungsbeurteilung erstellen und Schutzmaßnahmen umsetzen.", "Erstunterweisung durchführen und dokumentieren.", "Betreuung und weitere Vorsorge mit dem Unfallversicherungsträger klären."},
			ActionURL:   "https://www.dguv.de/de/praevention/themen-a-z/gefaehrdungsbeurteilung/index.jsp", ActionLabel: "Zur Arbeitsschutz-Hilfe",
		},
	}
	for i := range duties {
		duties[i].Order = i + 1
		if duties[i].Anchor == domain.AnchorEintragung && f.RegisteredOn == "" {
			duties[i].IsPending = true
		}
		if day := done[duties[i].Key]; day != "" {
			duties[i].DoneOn = day
			duties[i].IsDone = true
			duties[i].IsPending = false
		}
	}
	return duties
}

// addMonths shifts an ISO date by whole months, clamping to the end of the
// target month. Ohne das Kappen würde aus dem 31. Januar plus einem Monat der
// 3. März, weil time.AddDate überläuft.
func addMonths(iso string, months int) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return ""
	}
	year, month, day := t.Date()
	target := time.Date(year, month+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	last := target.AddDate(0, 1, -1).Day()
	if day > last {
		day = last
	}
	return time.Date(target.Year(), target.Month(), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

// fiscalYearEndAfter is the 31 December of the founding year — the end of the
// Rumpfgeschäftsjahr.
func fiscalYearEndAfter(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d-12-31", t.Year())
}
