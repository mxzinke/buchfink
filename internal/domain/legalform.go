package domain

// Die Rechtsform, und was sie steuerlich nach sich zieht.
//
// Sie stand bisher als Freitext in den Stammdaten und wurde nur in die
// E-Bilanz durchgereicht. Damit war sie eine Angabe, aus der nichts folgte —
// und daneben stand für die Teilfreistellung nach § 20 InvStG eine zweite
// Frage, die der Nutzer in Rechtssätzen beantworten musste.
//
// Beides zusammenzuziehen ist die Vereinfachung: die Rechtsform wird ohnehin
// eingetragen, und aus ihr folgt in den allermeisten Fällen eindeutig, welche
// Anlegerstellung § 20 InvStG meint. Wo sie es nicht tut, wird gefragt — aber
// nur dort.
//
// Gespeichert bleibt der Klartext („GmbH & Co. KG") und kein Schlüssel: er
// steht so in der E-Bilanz, und ein Katalog, der die Schreibweise ändert,
// änderte den Export mit.

// LegalFormUG ist der Katalogeintrag der Unternehmergesellschaft. Er steht als
// Konstante, weil aus ihm eine Rechtsfolge folgt: § 5a Abs. 3 GmbHG bindet die
// Pflichtrücklage an diese Rechtsform, und ein Tippfehler im Vergleich ließe
// sie stillschweigend entfallen.
const LegalFormUG = "UG (haftungsbeschränkt)"

// LegalFormInfo is one entry of the Rechtsform catalog.
type LegalFormInfo struct {
	// Name ist die gespeicherte Schreibweise.
	Name string `json:"name"`
	// Investor ist die Anlegerstellung, die aus dieser Rechtsform folgt.
	// InvestorUnknown heißt: aus ihr allein folgt sie nicht.
	Investor InvestorType `json:"investor"`
	// Note sagt, warum — und wo die Rechtsform nicht entscheidet, was fehlt.
	Note string `json:"note"`
}

// Die drei Begründungen, die sich im Katalog wiederholen. Als Konstanten, damit
// nicht dreimal derselbe Satz leicht verschieden dasteht.
const (
	noteCorporate = "Eine Körperschaft unterliegt dem Körperschaftsteuergesetz. " +
		"Für Investmentanteile heißt das: 80 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 3 InvStG)."
	noteIndividual = "Das Unternehmen wird von einer natürlichen Person geführt. Für " +
		"Investmentanteile im Betriebsvermögen heißt das: 60 % Aktienteilfreistellung " +
		"(§ 20 Abs. 1 Satz 2 InvStG)."
	notePartnership = "Bei einer Personengesellschaft bestimmt sich die Teilfreistellung nach dem " +
		"einzelnen Gesellschafter (§ 20 Abs. 3a InvStG). Sind alle Gesellschafter natürliche " +
		"Personen, sind es 60 %; ist eine Körperschaft beteiligt, gilt für deren Anteil 80 % — " +
		"einen einheitlichen Satz gibt es dann nicht. Lege das fest, wenn du Investmentanteile hältst."
	noteUnknown = "Aus dieser Rechtsform folgt die Anlegerstellung nicht. Wenn du " +
		"Investmentanteile hältst, lege sie fest — davon hängt die Teilfreistellung nach " +
		"§ 20 InvStG ab."
)

// legalForms is the curated catalog.
//
// Kurz gehalten und nicht vollständig: er nennt die Rechtsformen, die ein
// bilanzierendes Unternehmen üblicherweise hat. Alles andere fällt unter
// „Sonstige", und dort wird gefragt statt geraten — eine KGaA und eine
// ausländische Rechtsform tragen Besonderheiten, die ein Katalogeintrag
// verschweigen würde.
var legalForms = []LegalFormInfo{
	{Name: "Einzelunternehmen", Investor: InvestorIndividualBusiness, Note: noteIndividual},
	{Name: "Eingetragener Kaufmann (e. K.)", Investor: InvestorIndividualBusiness, Note: noteIndividual},
	{Name: "Freiberufliche Praxis", Investor: InvestorIndividualBusiness, Note: noteIndividual},
	{Name: "GbR", Investor: InvestorUnknown, Note: notePartnership},
	{Name: "OHG", Investor: InvestorUnknown, Note: notePartnership},
	{Name: "KG", Investor: InvestorUnknown, Note: notePartnership},
	{Name: "GmbH & Co. KG", Investor: InvestorUnknown, Note: notePartnership},
	{Name: "Partnerschaftsgesellschaft", Investor: InvestorUnknown, Note: notePartnership},
	{Name: LegalFormUG, Investor: InvestorCorporate, Note: noteCorporate},
	{Name: "GmbH", Investor: InvestorCorporate, Note: noteCorporate},
	{Name: "AG", Investor: InvestorCorporate, Note: noteCorporate},
	{Name: "SE", Investor: InvestorCorporate, Note: noteCorporate},
	{Name: "eG", Investor: InvestorCorporate, Note: noteCorporate},
	{Name: "e. V.", Investor: InvestorCorporate, Note: noteCorporate},
	{Name: "Stiftung", Investor: InvestorCorporate, Note: noteCorporate},
	{Name: "Sonstige", Investor: InvestorUnknown, Note: noteUnknown},
}

// LegalFormCatalog returns the catalog for the input mask.
func LegalFormCatalog() []LegalFormInfo {
	out := make([]LegalFormInfo, len(legalForms))
	copy(out, legalForms)
	return out
}

// LookupLegalForm returns the catalog entry of a Rechtsform.
func LookupLegalForm(name string) (LegalFormInfo, bool) {
	for _, form := range legalForms {
		if form.Name == name {
			return form, true
		}
	}
	return LegalFormInfo{}, false
}

// InvestorTypeOrDerived is the Anlegerstellung for § 20 InvStG: the explicit
// choice where one was made, otherwise what the Rechtsform implies.
//
// Der zweite Rückgabewert ist die Begründung. Sie gehört dazu, weil ein Satz,
// der sich aus einer anderen Angabe ergibt, sonst wie eine Voreinstellung
// aussieht, die niemand getroffen hat.
func (s *CompanySettings) InvestorTypeOrDerived() (InvestorType, string) {
	if s.InvestorOverride.Valid() {
		return s.InvestorOverride, "Ausdrücklich festgelegt: " + s.InvestorOverride.Label() + "."
	}
	if form, ok := LookupLegalForm(s.LegalForm); ok {
		return form.Investor, form.Note
	}
	return InvestorUnknown, noteUnknown
}

// withdrawalLegalForms sind die Rechtsformen, bei denen es Entnahmen und
// Einlagen gibt.
//
// Das ist der Unterschied, der die Grenze des Funktionsumfangs markiert: bei
// einer Kapitalgesellschaft ist das Vermögen der Gesellschaft von dem der
// Gesellschafter getrennt, und eine Zahlung an den Gesellschafter ist eine
// Ausschüttung, ein Darlehen oder eine verdeckte Gewinnausschüttung. Bei
// Einzelunternehmen und Personengesellschaften ist sie eine Entnahme, sie läuft
// über Kapitalkonten, und § 4 Abs. 4a EStG kann den Schuldzinsenabzug kürzen.
// Nichts davon bildet Buchfink ab.
var withdrawalLegalForms = map[string]bool{
	"Einzelunternehmen":              true,
	"Eingetragener Kaufmann (e. K.)": true,
	"Freiberufliche Praxis":          true,
	"GbR":                            true,
	"OHG":                            true,
	"KG":                             true,
	"GmbH & Co. KG":                  true,
	"Partnerschaftsgesellschaft":     true,
}

// HasWithdrawals meldet, ob eine Rechtsform Entnahmen und Einlagen kennt.
func HasWithdrawals(legalForm string) bool { return withdrawalLegalForms[legalForm] }

// LegalFormLimitationNote ist der Hinweis auf die Grenze des Funktionsumfangs
// bei Rechtsformen mit Entnahmen. Leer heißt: der Hinweis trifft nicht zu.
//
// Er steht im Einrichtungsassistenten, in den Einstellungen und in der
// Verfahrensdokumentation. Ihn wegzulassen wäre der schlechteste Weg: die
// Rechtsform ist wählbar, es lässt sich damit buchen, und die Lücke fiele erst
// beim Jahresabschluss auf — dann, wenn sich nichts mehr daran ändern lässt.
func LegalFormLimitationNote(legalForm string) string {
	if !HasWithdrawals(legalForm) {
		return ""
	}
	return "Kapitalkonten, Entnahmen und Einlagen sowie die Zinsschranke für Überentnahmen " +
		"(§ 4 Abs. 4a EStG) sind in dieser Fassung nicht abgebildet. Buchfink führt die " +
		"Buchhaltung dieser Rechtsform, rechnet aber weder das Kapitalkonto fort noch prüft " +
		"es den Schuldzinsenabzug. Sprich das mit deinem steuerlichen Berater ab."
}

// TaxCaseHints sind die Hinweise zu den Steuerfällen, die Buchfink nicht
// abbildet.
//
// Sie stehen an den Einstellungen und in der Verfahrensdokumentation, weil eine
// Verfahrensdokumentation, die nur sagt, was das Verfahren kann, ihre wichtigste
// Aussage schuldig bleibt: wo es aufhört.
func TaxCaseHints() []string {
	return []string{
		"Der besondere Besteuerungsverfahren OSS und IOSS (§§ 18i bis 18k UStG) sind nicht " +
			"abgebildet. Wer Leistungen an Privatpersonen in anderen Mitgliedstaaten erbringt und " +
			"die Lieferschwelle überschreitet, meldet diese Umsätze außerhalb von Buchfink.",
		"Die Kleinunternehmerregelung (§ 19 UStG) ist auf der eigenen Seite nicht abgebildet: " +
			"Buchfink geht davon aus, dass das Unternehmen die Umsatzsteuer ausweist und " +
			"voranmeldet. Am Geschäftspartner lässt sich die Kleinunternehmereigenschaft " +
			"hinterlegen, weil sie für die E-Rechnungspflicht von Bedeutung ist.",
		"Kapitalkonten, Entnahmen und Einlagen sowie § 4 Abs. 4a EStG sind nicht abgebildet.",
		"Die Lohnbuchhaltung, die Anlage EÜR und die Reisekostenabrechnung sind nicht Teil " +
			"des Funktionsumfangs.",
	}
}
