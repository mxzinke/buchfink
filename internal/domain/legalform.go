package domain

// Die Rechtsform. Buchfink führt die Bücher einer GmbH oder einer
// UG (haftungsbeschränkt); andere Rechtsformen nimmt es nicht an.
//
// Gespeichert wird der Klartext und kein Schlüssel: er steht so in der
// E-Bilanz.

const (
	// LegalFormGmbH ist die Gesellschaft mit beschränkter Haftung.
	LegalFormGmbH = "GmbH"
	// LegalFormUG ist die Unternehmergesellschaft. § 5a Abs. 3 GmbHG bindet die
	// Pflichtrücklage an diese Rechtsform.
	LegalFormUG = "UG (haftungsbeschränkt)"
)

// LegalFormInfo is one entry of the Rechtsform catalog.
type LegalFormInfo struct {
	// Name ist die gespeicherte Schreibweise.
	Name string `json:"name"`
	// Investor ist die Anlegerstellung für § 20 InvStG, die aus dieser
	// Rechtsform folgt.
	Investor InvestorType `json:"investor"`
	// Note sagt, warum.
	Note string `json:"note"`
}

const noteCorporate = "Eine Körperschaft unterliegt dem Körperschaftsteuergesetz. " +
	"Für Investmentanteile heißt das: 80 % Aktienteilfreistellung (§ 20 Abs. 1 Satz 3 InvStG)."

var legalForms = []LegalFormInfo{
	{Name: LegalFormGmbH, Investor: InvestorCorporate, Note: noteCorporate},
	{Name: LegalFormUG, Investor: InvestorCorporate, Note: noteCorporate},
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

// IsSupportedLegalForm meldet, ob Buchfink die Rechtsform führt.
func IsSupportedLegalForm(name string) bool {
	_, ok := LookupLegalForm(name)
	return ok
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
	return InvestorUnknown, "Ohne Rechtsform folgt die Anlegerstellung nicht. Tragen Sie die " +
		"Rechtsform in den Stammdaten ein, wenn Sie Investmentanteile halten — davon hängt die " +
		"Teilfreistellung nach § 20 InvStG ab."
}

// TaxCaseHints sind die Hinweise zu den Steuerfällen, die Buchfink nicht
// abbildet.
//
// Sie stehen an den Einstellungen und in der Verfahrensdokumentation, weil eine
// Verfahrensdokumentation, die nur sagt, was das Verfahren kann, ihre wichtigste
// Aussage schuldig bleibt: wo es aufhört.
func TaxCaseHints() []string {
	return []string{
		"Die besonderen Besteuerungsverfahren OSS und IOSS (§§ 18i bis 18k UStG) sind nicht " +
			"abgebildet. Wer Leistungen an Privatpersonen in anderen Mitgliedstaaten erbringt und " +
			"die Lieferschwelle überschreitet, meldet diese Umsätze außerhalb von Buchfink.",
		"Die Kleinunternehmerregelung (§ 19 UStG) ist auf der eigenen Seite nicht abgebildet: " +
			"Buchfink geht davon aus, dass das Unternehmen die Umsatzsteuer ausweist und " +
			"voranmeldet. Am Geschäftspartner lässt sich die Kleinunternehmereigenschaft " +
			"hinterlegen, weil sie für die E-Rechnungspflicht von Bedeutung ist.",
		"Die Lohnbuchhaltung und die Reisekostenabrechnung sind nicht Teil des Funktionsumfangs.",
	}
}
