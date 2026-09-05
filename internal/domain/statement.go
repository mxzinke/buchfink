package domain

import "fmt"

// StatementDepth ist die Gliederungstiefe von Bilanz und Gewinn- und
// Verlustrechnung.
//
// Sie ist kein Anzeigegeschmack, sondern eine Rechtsfolge der Größenklasse:
// § 266 Abs. 1 Satz 3 HGB erlaubt kleinen Kapitalgesellschaften die verkürzte
// Bilanz mit Buchstaben und römischen Ziffern, Satz 4 erlaubt
// Kleinstkapitalgesellschaften die Bilanz allein mit den Buchstaben. Wer mehr
// zeigt, verstößt gegen nichts — wer weniger zeigt, schon. Deshalb ist die Tiefe
// ein Parameter des Aufbaus und keine Filterfunktion der Oberfläche.
type StatementDepth string

const (
	// DepthFull ist die volle Gliederung des § 266 Abs. 2 und 3 HGB.
	DepthFull StatementDepth = "full"
	// DepthShort ist die verkürzte Bilanz des § 266 Abs. 1 Satz 3 HGB:
	// Buchstaben und römische Ziffern.
	DepthShort StatementDepth = "short"
	// DepthLetters ist die Bilanz der Kleinstkapitalgesellschaft nach
	// § 266 Abs. 1 Satz 4 HGB: nur die Buchstaben.
	DepthLetters StatementDepth = "letters"
)

// Valid meldet, ob die Tiefe eine der drei bekannten ist.
func (d StatementDepth) Valid() bool {
	switch d {
	case DepthFull, DepthShort, DepthLetters:
		return true
	}
	return false
}

// Label benennt die Tiefe für die Oberfläche.
func (d StatementDepth) Label() string {
	switch d {
	case DepthFull:
		return "Vollgliederung (§ 266 Abs. 2 und 3 HGB)"
	case DepthShort:
		return "Verkürzte Gliederung (§ 266 Abs. 1 Satz 3 HGB)"
	case DepthLetters:
		return "Buchstabengliederung (§ 266 Abs. 1 Satz 4 HGB)"
	}
	return string(d)
}

// MaxBalanceLevel ist die tiefste Ebene, die die Bilanz in dieser Tiefe zeigt:
// 1 Buchstabe, 2 römische Ziffer, 3 arabische Ziffer.
func (d StatementDepth) MaxBalanceLevel() int {
	switch d {
	case DepthLetters:
		return 1
	case DepthShort:
		return 2
	default:
		return 3
	}
}

// MaxIncomeLevel ist die tiefste Ebene der Gewinn- und Verlustrechnung. Die
// Staffel des § 275 Abs. 2 HGB hat nur zwei: die Nummern und die Buchstaben
// unter Material-, Personalaufwand und Abschreibungen.
func (d StatementDepth) MaxIncomeLevel() int {
	if d == DepthFull {
		return 2
	}
	return 1
}

// ParseStatementDepth liest die Tiefe aus dem Wert, den die Oberfläche schickt.
// Leer heißt: volle Gliederung, denn das ist die Gliederung, die immer zulässig
// ist.
func ParseStatementDepth(value string) (StatementDepth, error) {
	if value == "" {
		return DepthFull, nil
	}
	d := StatementDepth(value)
	if !d.Valid() {
		return "", fmt.Errorf("unbekannte Gliederungstiefe %q", value)
	}
	return d, nil
}

// StatementSection ist der Abschnitt, in dem eine Gliederungsposition steht.
type StatementSection string

const (
	SectionAssets      StatementSection = "aktiva"
	SectionLiabilities StatementSection = "passiva"
	SectionIncome      StatementSection = "guv"
	// SectionStatistical sammelt die Konten der Klasse 9. Sie sind weder
	// Bilanz- noch Erfolgskonten und dürfen in keiner der beiden Summen
	// auftauchen — sie stehen hier, damit ein Saldo auf ihnen sichtbar bleibt
	// statt spurlos zu verschwinden.
	SectionStatistical StatementSection = "statistisch"
)

// StatementAccount ist ein Konto unter einer Gliederungsposition. Es trägt den
// Weg vom Posten zurück zum Kontoblatt (GOB-02: Drill-down).
type StatementAccount struct {
	Number     string `json:"number"`
	Name       string `json:"name"`
	PositionID string `json:"positionId"`
	// Position ist die Bezeichnung der SKR04-Position, aus der die Zuordnung
	// stammt — nicht die der Gliederungszeile, in der das Konto steht.
	Position string `json:"position"`
	// Note erklärt eine Zuordnung, die dem Kontonamen widerspricht — etwa ein
	// Kapitalkonto der Personenhandelsgesellschaft unter dem gezeichneten
	// Kapital.
	Note        string `json:"note,omitempty"`
	Amount      Cents  `json:"amount"`
	PriorAmount Cents  `json:"priorAmount"`
}

// StatementLine ist eine Gliederungsposition mit ihrem Wert.
type StatementLine struct {
	// Key ist der stabile Schlüssel, z. B. "aktiva.A.II.3". An ihm hängen die
	// Zuordnungstabelle und die Taxonomie-Zuordnung der E-Bilanz.
	Key string `json:"key"`
	// Ordinal ist die Ordnungszahl des Gesetzes ("A.", "II.", "3.", "a)").
	Ordinal string           `json:"ordinal"`
	Label   string           `json:"label"`
	Level   int              `json:"level"`
	Section StatementSection `json:"section"`
	// Note erklärt eine Zuordnung, die nicht selbsterklärend ist — etwa eine
	// Position der Personenhandelsgesellschaft in der Gliederung der
	// Kapitalgesellschaft.
	Note string `json:"note,omitempty"`
	// IsSubtotal kennzeichnet eine Zwischensumme der Staffel (§ 275 Abs. 2
	// Nr. 15 und 17 HGB). Auf ihr steht kein Konto.
	IsSubtotal bool `json:"isSubtotal"`
	// IsFallback kennzeichnet eine Auffangposition („sonstige …"). Der
	// Zuordnungsbericht zählt, wie viel dort landet.
	IsFallback bool `json:"isFallback"`
	// Omitted kennzeichnet einen Posten, den § 265 Abs. 8 HGB entfallen lässt:
	// er trägt in beiden Jahren keinen Betrag, und unter ihm steht auch nichts.
	//
	// Die Entscheidung fällt hier und nicht in der Ansicht. Träfe die Ansicht
	// sie selbst, zeigte der Bildschirm andere Zeilen als PDF und CSV — und
	// welche Posten ein Abschluss ausweist, ist eine Frage der Rechnungslegung
	// und keine der Darstellung.
	Omitted     bool               `json:"omitted"`
	Amount      Cents              `json:"amount"`
	PriorAmount Cents              `json:"priorAmount"`
	Accounts    []StatementAccount `json:"accounts,omitempty"`
}

// FallbackCount zählt, was in einer Auffangposition gelandet ist.
type FallbackCount struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Accounts int    `json:"accounts"`
	Amount   Cents  `json:"amount"`
}

// SignSwitch hält fest, dass ein Konto wegen seines Vorzeichens auf der
// Gegenposition ausgewiesen wird — der Fall des S/H-Kontos.
type SignSwitch struct {
	Account string `json:"account"`
	Name    string `json:"name"`
	From    string `json:"from"`
	To      string `json:"to"`
	Label   string `json:"label"`
	Amount  Cents  `json:"amount"`
}

// AssignmentReport ist der Zuordnungsbericht: was die Gliederung nicht
// einordnen konnte und was sie nur mit Vorbehalt eingeordnet hat.
type AssignmentReport struct {
	// Unassigned sind Konten mit Saldo, für die die Zuordnungstabelle keine
	// Gliederungsposition kennt. Sie stehen in der Position „Nicht zugeordnet".
	Unassigned []StatementAccount `json:"unassigned"`
	// WrongSign sind Konten, deren Saldo der Richtung ihrer Position
	// widerspricht und für die es keine Gegenposition gibt. Sie bleiben mit
	// negativem Betrag stehen — verschwiegen wären sie ein stiller Fehler.
	WrongSign    []StatementAccount `json:"wrongSign"`
	SignSwitches []SignSwitch       `json:"signSwitches"`
	Fallbacks    []FallbackCount    `json:"fallbacks"`
}

// HasFindings meldet, ob der Bericht etwas zu melden hat, das eine Prüfung
// erfordert.
func (r *AssignmentReport) HasFindings() bool {
	return len(r.Unassigned) > 0 || len(r.WrongSign) > 0
}

// Statement ist die Gliederung eines Geschäftsjahres mit Vorjahresspalte.
type Statement struct {
	FiscalYear  int              `json:"fiscalYear"`
	PriorYear   int              `json:"priorYear"`
	HasPrior    bool             `json:"hasPrior"`
	Depth       StatementDepth   `json:"depth"`
	Assets      []StatementLine  `json:"assets"`
	Liabilities []StatementLine  `json:"liabilities"`
	Income      []StatementLine  `json:"income"`
	Statistical []StatementLine  `json:"statistical"`
	Assignment  AssignmentReport `json:"assignment"`

	TotalAssets           Cents `json:"totalAssets"`
	TotalAssetsPrior      Cents `json:"totalAssetsPrior"`
	TotalLiabilities      Cents `json:"totalLiabilities"`
	TotalLiabilitiesPrior Cents `json:"totalLiabilitiesPrior"`
	// BalanceSheetTotal ist die Bilanzsumme des § 267 Abs. 4a HGB: die Summe
	// der Posten A bis E der Aktivseite. Die nicht eingeforderten ausstehenden
	// Einlagen stehen davor und zählen deshalb nicht mit.
	BalanceSheetTotal      Cents `json:"balanceSheetTotal"`
	BalanceSheetTotalPrior Cents `json:"balanceSheetTotalPrior"`

	NetIncome      Cents `json:"netIncome"`
	NetIncomePrior Cents `json:"netIncomePrior"`
	// Revenue ist Nr. 1 der Staffel — das Merkmal „Umsatzerlöse" des § 267 HGB.
	Revenue      Cents `json:"revenue"`
	RevenuePrior Cents `json:"revenuePrior"`
}

// Line sucht eine Gliederungsposition über alle Abschnitte.
func (s *Statement) Line(key string) *StatementLine {
	groups := [][]StatementLine{s.Assets, s.Liabilities, s.Income, s.Statistical}
	for _, group := range groups {
		for i := range group {
			if group[i].Key == key {
				return &group[i]
			}
		}
	}
	return nil
}

// MaturityRow ist eine Zeile der Restlaufzeitengliederung.
type MaturityRow struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	Total         Cents  `json:"total"`
	UpToOneYear   Cents  `json:"upToOneYear"`
	OverOneYear   Cents  `json:"overOneYear"`
	OverFiveYears Cents  `json:"overFiveYears"`
	Items         int    `json:"items"`
	// Undated zählt die Posten ohne Fälligkeit. Ohne Fälligkeit gibt es keine
	// Restlaufzeit; sie hier zu erfinden hieße, eine Angabe zu behaupten.
	Undated Cents  `json:"undated"`
	Note    string `json:"note,omitempty"`
}

// MaturityTable ist die Angabe unter der Bilanz nach § 268 Abs. 4 und 5 HGB:
// Forderungen mit einer Restlaufzeit von mehr als einem Jahr, Verbindlichkeiten
// mit einer Restlaufzeit bis zu einem Jahr und von mehr als fünf Jahren.
type MaturityTable struct {
	ClosingDate string        `json:"closingDate"`
	Rows        []MaturityRow `json:"rows"`
	Reference   string        `json:"reference"`
}

// Deadline ist ein Termin des Jahresabschlusses mit seiner Norm.
type Deadline struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	DueDate     string `json:"dueDate"`
	Period      string `json:"period"`
	Reference   string `json:"reference"`
	Description string `json:"description"`
	FiscalYear  int    `json:"fiscalYear"`
	IsDone      bool   `json:"isDone"`
	DoneOn      string `json:"doneOn,omitempty"`
}

// StatementHeader sind die Pflichtangaben des § 264 Abs. 1a HGB im Kopf des
// Abschlusses: Firma, Sitz, Registergericht und Registernummer.
type StatementHeader struct {
	CompanyName    string `json:"companyName"`
	LegalForm      string `json:"legalForm"`
	Seat           string `json:"seat"`
	RegisterCourt  string `json:"registerCourt"`
	RegisterNumber string `json:"registerNumber"`
	FiscalYear     int    `json:"fiscalYear"`
	StartDate      string `json:"startDate"`
	ClosingDate    string `json:"closingDate"`
	PriorYear      int    `json:"priorYear"`
	IsShortYear    bool   `json:"isShortYear"`
	Reference      string `json:"reference"`
	// Missing benennt die Pflichtangaben, die in den Einstellungen fehlen. Ein
	// leeres Feld im Kopf wäre sonst nur eine Lücke, die niemand einordnet.
	Missing []string `json:"missing"`
}

// FinancialStatement ist der Jahresabschluss, wie ihn die Oberfläche zeigt:
// Bilanz und GuV mit Vorjahr, Größenklasse, Angaben unter der Bilanz, Fristen
// und der Zuordnungsbericht.
type FinancialStatement struct {
	Header     StatementHeader `json:"header"`
	Statement  Statement       `json:"statement"`
	SizeClass  SizeClass       `json:"sizeClass"`
	Maturities MaturityTable   `json:"maturities"`
	Deadlines  []Deadline      `json:"deadlines"`
}
