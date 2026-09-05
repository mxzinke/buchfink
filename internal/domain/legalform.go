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
