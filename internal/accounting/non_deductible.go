package accounting

// Die Kategorien der beschränkt abziehbaren Betriebsausgaben (§ 4 Abs. 5 EStG).
//
// Sie stehen hier und nicht im Gruppenkatalog, weil sie eine Auswertung sind und
// keine Eingabe: der Bericht führt je Kategorie und Jahr auf, was abziehbar
// gebucht wurde und was nicht. Die Konten sind dieselben wie im Gruppenkatalog —
// TestNonDeductibleCategoriesMatchGroups hält beide zusammen, damit eine neue
// Kategorie nicht in der einen Liste erscheint und in der anderen fehlt.

// NonDeductibleCategory ist eine Kategorie mit ihren beiden Konten.
type NonDeductibleCategory struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Reference nennt die Vorschrift, aus der die Beschränkung folgt.
	Reference string `json:"reference"`
	// DeductibleAccount trägt den abziehbaren Teil, NonDeductibleAccount den
	// nicht abziehbaren. Eines von beiden kann leer sein: was in keiner Höhe
	// abziehbar ist, hat kein abziehbares Konto.
	DeductibleAccount    string `json:"deductibleAccount,omitempty"`
	NonDeductibleAccount string `json:"nonDeductibleAccount,omitempty"`
	// Note ist der Satz, der im Bericht unter der Zeile steht.
	Note string `json:"note"`
}

// NonDeductibleCategories liefert die Kategorien in fester Reihenfolge.
func NonDeductibleCategories() []NonDeductibleCategory {
	return []NonDeductibleCategory{
		{
			Key: "gifts", Label: "Geschenke an Geschäftspartner",
			Reference:         "§ 4 Abs. 5 Satz 1 Nr. 1 EStG",
			DeductibleAccount: "6610", NonDeductibleAccount: "6620",
			Note: "Freigrenze je Empfänger und Wirtschaftsjahr. Wird sie überschritten, sind sämtliche " +
				"Geschenke an diesen Empfänger nicht abziehbar; der Vorsteuerabzug entfällt mit ihnen " +
				"(§ 15 Abs. 1a UStG).",
		},
		{
			Key: "gifts_business", Label: "Geschenke zur ausschließlich betrieblichen Nutzung",
			Reference:         "§ 4 Abs. 5 Satz 1 Nr. 1 EStG",
			DeductibleAccount: "6625",
			Note: "Ein Gegenstand, den der Empfänger nur betrieblich nutzen kann, fällt nicht unter die " +
				"Freigrenze und bleibt unbegrenzt abziehbar.",
		},
		{
			Key: "entertainment", Label: "Bewirtung von Geschäftspartnern",
			Reference:         "§ 4 Abs. 5 Satz 1 Nr. 2 EStG",
			DeductibleAccount: "6640", NonDeductibleAccount: "6644",
			Note: "70 % der angemessenen Aufwendungen sind abziehbar. Die Vorsteuer bleibt in voller Höhe " +
				"abziehbar (§ 15 Abs. 1a Satz 2 UStG).",
		},
		{
			Key: "representation", Label: "Gästehaus, Jagd, Fischerei, Yacht",
			Reference:            "§ 4 Abs. 5 Satz 1 Nr. 3 und 4 EStG",
			NonDeductibleAccount: "6645",
			Note: "In keiner Höhe abziehbar. § 15 Abs. 1a UStG nimmt diesen Aufwendungen auch den " +
				"Vorsteuerabzug, die Umsatzsteuer gehört deshalb zum Aufwand.",
		},
	}
}

// NonDeductibleCategoryForAccount findet die Kategorie eines Kontos.
func NonDeductibleCategoryForAccount(account string) (NonDeductibleCategory, bool) {
	for _, c := range NonDeductibleCategories() {
		if c.DeductibleAccount == account || c.NonDeductibleAccount == account {
			return c, true
		}
	}
	return NonDeductibleCategory{}, false
}
