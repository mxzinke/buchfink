package domain

// SizeClassKind ist die Größenklasse einer Kapitalgesellschaft nach den
// §§ 267 und 267a HGB.
//
// Sie ist keine Beschreibung, sondern der Auslöser fast aller Erleichterungen
// des Abschlusses: Gliederungstiefe, Anhang, Lagebericht, Prüfung, Frist und
// Umfang der Offenlegung hängen daran. Deshalb wird sie berechnet und nicht
// eingestellt.
type SizeClassKind string

const (
	// SizeMicro ist die Kleinstkapitalgesellschaft des § 267a HGB.
	SizeMicro SizeClassKind = "micro"
	// SizeSmall ist die kleine Kapitalgesellschaft des § 267 Abs. 1 HGB.
	SizeSmall SizeClassKind = "small"
	// SizeMedium ist die mittelgroße des § 267 Abs. 2 HGB.
	SizeMedium SizeClassKind = "medium"
	// SizeLarge ist die große des § 267 Abs. 3 HGB.
	SizeLarge SizeClassKind = "large"
)

// Label benennt die Klasse für die Oberfläche.
func (k SizeClassKind) Label() string {
	switch k {
	case SizeMicro:
		return "Kleinstkapitalgesellschaft"
	case SizeSmall:
		return "Kleine Kapitalgesellschaft"
	case SizeMedium:
		return "Mittelgroße Kapitalgesellschaft"
	case SizeLarge:
		return "Große Kapitalgesellschaft"
	}
	return string(k)
}

// Reference nennt die Norm, aus der sich die Klasse ergibt.
func (k SizeClassKind) Reference() string {
	switch k {
	case SizeMicro:
		return "§ 267a Abs. 1 HGB"
	case SizeSmall:
		return "§ 267 Abs. 1 HGB"
	case SizeMedium:
		return "§ 267 Abs. 2 HGB"
	case SizeLarge:
		return "§ 267 Abs. 3 HGB"
	}
	return ""
}

// rank ordnet die Klassen der Größe nach. Damit lässt sich sagen, ob eine
// Pflicht „ab mittelgroß" greift.
func (k SizeClassKind) rank() int {
	switch k {
	case SizeMicro:
		return 0
	case SizeSmall:
		return 1
	case SizeMedium:
		return 2
	case SizeLarge:
		return 3
	}
	return -1
}

// AtLeast meldet, ob die Klasse mindestens so groß ist wie die andere.
func (k SizeClassKind) AtLeast(other SizeClassKind) bool {
	return k.rank() >= other.rank()
}

// SizeCriteria sind die drei Merkmale des § 267 Abs. 1 HGB zu einem Stichtag.
type SizeCriteria struct {
	// BalanceSheetTotal ist die Bilanzsumme nach § 267 Abs. 4a HGB.
	BalanceSheetTotal Cents `json:"balanceSheetTotal"`
	// Revenue sind die Umsatzerlöse der zwölf Monate vor dem Stichtag —
	// bei Buchfink die Nr. 1 der Staffel des abgelaufenen Geschäftsjahres.
	Revenue Cents `json:"revenue"`
	// Employees ist die durchschnittliche Zahl der Arbeitnehmer.
	Employees int `json:"employees"`
}

// SizeAssessment ist die Beurteilung eines einzelnen Abschlussstichtags.
//
// Sie steht für sich, weil § 267 Abs. 4 HGB zwei Stichtage vergleicht: die
// Rechtsfolge tritt erst ein, wenn zwei aufeinanderfolgende Stichtage dieselbe
// neue Klasse ergeben. Ohne die einzelne Beurteilung ließe sich das nicht
// begründen, sondern nur behaupten.
type SizeAssessment struct {
	Year        int           `json:"year"`
	ClosingDate string        `json:"closingDate"`
	Criteria    SizeCriteria  `json:"criteria"`
	Class       SizeClassKind `json:"class"`
	// Met benennt die Merkmale, die für die Klasse sprechen — nach § 267 Abs. 1
	// HGB müssen zwei der drei Größen unterschritten sein.
	Met []string `json:"met"`
	// Thresholds nennt die Schwellen, an denen gemessen wurde.
	Thresholds SizeThresholdSet `json:"thresholds"`
}

// SizeThresholdSet sind die Schwellenwerte einer Klasse zu einem Stichtag.
type SizeThresholdSet struct {
	ValidFrom string       `json:"validFrom"`
	Reference string       `json:"reference"`
	Micro     SizeCriteria `json:"micro"`
	Small     SizeCriteria `json:"small"`
	Medium    SizeCriteria `json:"medium"`
}

// SizeObligations sind die Folgen der Größenklasse.
type SizeObligations struct {
	Depth          StatementDepth `json:"depth"`
	DepthReference string         `json:"depthReference"`
	// NotesRequired: die Kleinstkapitalgesellschaft darf den Anhang weglassen,
	// wenn sie die Angaben unter der Bilanz macht (§ 264 Abs. 1 Satz 5 HGB).
	NotesRequired             bool   `json:"notesRequired"`
	NotesReference            string `json:"notesReference"`
	ManagementReport          bool   `json:"managementReport"`
	ManagementReportReference string `json:"managementReportReference"`
	AuditRequired             bool   `json:"auditRequired"`
	AuditReference            string `json:"auditReference"`
	// PreparationMonths ist die Frist zur Aufstellung nach § 264 Abs. 1 HGB.
	PreparationMonths        int    `json:"preparationMonths"`
	PreparationReference     string `json:"preparationReference"`
	DisclosureMonths         int    `json:"disclosureMonths"`
	DisclosureReference      string `json:"disclosureReference"`
	DisclosureScope          string `json:"disclosureScope"`
	DisclosureScopeReference string `json:"disclosureScopeReference"`
}

// SizeClass ist das Ergebnis der Einordnung eines Geschäftsjahres.
type SizeClass struct {
	Year        int           `json:"year"`
	ClosingDate string        `json:"closingDate"`
	Class       SizeClassKind `json:"class"`
	Criteria    SizeCriteria  `json:"criteria"`
	// Current ist die Beurteilung dieses Stichtags, Prior die des Vorjahres.
	Current SizeAssessment  `json:"current"`
	Prior   *SizeAssessment `json:"prior,omitempty"`
	// History sind die Beurteilungen, die in die Zweijahresregel eingegangen
	// sind, ältester Stichtag zuerst. Zwei Stichtage reichen dafür nicht: gilt
	// eine Klasse aus früheren Jahren fort, weil seither kein Paar
	// übereinstimmt, steht der Grund erst weiter hinten in dieser Kette.
	History []SizeAssessment `json:"history,omitempty"`
	// IsFirstYear meldet den Fall des § 267 Abs. 4 Satz 2 HGB: bei Neugründung
	// treten die Rechtsfolgen schon am ersten Abschlussstichtag ein.
	IsFirstYear bool `json:"isFirstYear"`
	// Reason begründet die Klasse in einem Satz, einschließlich der
	// Zweijahresregel.
	Reason      string          `json:"reason"`
	Obligations SizeObligations `json:"obligations"`
}
