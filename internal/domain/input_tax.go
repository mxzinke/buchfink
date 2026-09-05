package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Das Verzeichnis der Vorsteuerberichtigung nach § 15a UStG.
//
// § 22 Abs. 4 UStG verlangt Aufzeichnungen, aus denen sich die Berichtigung
// ergibt, und sie müssen über den ganzen Berichtigungszeitraum reichen — fünf
// Jahre bei beweglichen Wirtschaftsgütern, zehn bei Grundstücken. Aus dem
// Journal allein ist das nicht zu gewinnen: dort steht der gezogene Betrag, aber
// nicht der Anteil, zu dem er gezogen wurde, und schon gar nicht der Anteil des
// dritten Jahres. Deshalb eine eigene Kartei, die neben der Anlagenkartei
// herläuft und wie diese über Geschäftsjahre hinweg lebt.

// InputTaxCorrection ist ein Wirtschaftsgut im Verzeichnis.
type InputTaxCorrection struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// AssetID verweist auf das Anlagegut, aus dem der Eintrag stammt. Er bleibt
	// optional: ins Verzeichnis gehört auch ein Wirtschaftsgut, das nicht
	// aktiviert wurde — eine Großreparatur an einem Gebäude etwa.
	AssetID *uint `gorm:"index" json:"assetId,omitempty"`
	// ReceiptID und EntryID verweisen auf Beleg und Zugangsbuchung.
	ReceiptID *uint `gorm:"index" json:"receiptId,omitempty"`
	EntryID   *uint `gorm:"index" json:"entryId,omitempty"`

	Label string `gorm:"size:255;not null;serializer:encrypted" json:"label"`
	// Account ist das Konto, auf dem das Wirtschaftsgut steht. Es entscheidet
	// über die Beweglichkeit und damit über den Zeitraum und die
	// Berichtigungskonten.
	Account string `gorm:"size:10;index" json:"account,omitempty"`

	AcquisitionDate string `gorm:"size:10;not null;index" json:"acquisitionDate"`
	// NetAmount sind die Anschaffungskosten netto, InputTaxAmount die darauf
	// entfallende Vorsteuer in voller Höhe.
	NetAmount      Cents `gorm:"not null;default:0" json:"netAmount"`
	InputTaxAmount Cents `gorm:"not null" json:"inputTaxAmount"`
	// OriginalPermille ist der Anteil, mit dem die Vorsteuer beim Zugang gezogen
	// wurde.
	//
	// Ohne Vorgabewert in der Spalte: `default:1000` hätte GORM eine gespeicherte
	// Null als „nicht gesetzt" gelesen und die Spalte auf 1000 laufen lassen.
	// Null ist hier aber eine Aussage — die Anschaffung ohne Vorsteuerabzug, aus
	// der die Berichtigung nach oben entsteht. Der Regelfall wird im Dienst
	// gesetzt und nicht von der Datenbank.
	OriginalPermille int `gorm:"not null" json:"originalPermille"`
	// Immovable entscheidet über den Berichtigungszeitraum: zehn Jahre für
	// Grundstücke und Gebäude, fünf für alles andere (§ 15a Abs. 1 UStG).
	Immovable bool `gorm:"not null;default:false" json:"immovable"`
	// CorrectionPeriodYears ist der Zeitraum in Jahren. Er wird gespeichert und
	// nicht aus Immovable gerechnet: eine spätere Änderung der Ableitung darf
	// den Zeitraum eines laufenden Wirtschaftsguts nicht verschieben.
	CorrectionPeriodYears int `gorm:"not null" json:"correctionPeriodYears"`

	// PeriodEnd ist der letzte Tag des Berichtigungszeitraums, datumsgenau nach
	// § 15a Abs. 1 UStG i. V. m. § 45 UStDV.
	//
	// Der Zeitraum läuft ab der erstmaligen Verwendung und nicht ab dem
	// Jahresbeginn; bei einem Zugang mitten im Jahr reicht er in das sechste
	// (elfte) Kalenderjahr hinein. Er wird gespeichert und nicht bei jedem
	// Zugriff gerechnet: eine spätere Änderung der Ableitung darf den Zeitraum
	// eines laufenden Wirtschaftsguts nicht verschieben. Leer heißt: ein Eintrag
	// aus der Zeit vor dem datumsgenauen Zeitraum — dann leitet
	// PeriodEndDate() ihn aus dem Anschaffungsdatum ab.
	PeriodEnd string `gorm:"size:10" json:"periodEnd,omitempty"`

	// FirstFiscalYear ist das Geschäftsjahr des Zugangs, LastFiscalYear das
	// Geschäftsjahr, in das das Ende des Berichtigungszeitraums fällt.
	FirstFiscalYear int `gorm:"index;not null" json:"firstFiscalYear"`
	LastFiscalYear  int `gorm:"index;not null" json:"lastFiscalYear"`

	// ClosedReason schließt einen Eintrag vorzeitig ab — Abgang, Entnahme oder
	// die Feststellung, dass er nie hätte aufgenommen werden dürfen. Leer heißt:
	// der Eintrag läuft.
	ClosedReason string `gorm:"size:255;serializer:encrypted" json:"closedReason,omitempty"`
	ClosedOn     string `gorm:"size:10" json:"closedOn,omitempty"`

	Note      string    `gorm:"size:500;serializer:encrypted" json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Usages []InputTaxUsage `gorm:"foreignKey:CorrectionID;constraint:OnDelete:CASCADE" json:"usages"`
}

// InputTaxUsage ist der Verwendungsanteil eines Jahres samt der Berichtigung,
// die daraus folgt.
type InputTaxUsage struct {
	CorrectionID uint `gorm:"primaryKey;autoIncrement:false;index" json:"correctionId"`
	FiscalYear   int  `gorm:"primaryKey;autoIncrement:false" json:"fiscalYear"`

	// Permille ist der Anteil der Verwendung für zum Vorsteuerabzug berechtigende
	// Umsätze in diesem Jahr.
	Permille int `gorm:"not null" json:"permille"`
	// Confirmed sagt, dass der Anwender den Anteil bestätigt oder geändert hat.
	// Ohne Bestätigung gilt er als Vorschlag — der Jahreslauf legt ihn vor, und
	// eine ungeprüfte Übernahme wäre die Behauptung, jemand habe hingesehen.
	Confirmed bool `gorm:"not null;default:false" json:"confirmed"`

	// Amount ist der berechnete Berichtigungsbetrag mit Vorzeichen: positiv, wo
	// nachträglich Vorsteuer abziehbar wird, negativ, wo sie zurückzuzahlen ist.
	Amount Cents `gorm:"not null;default:0" json:"amount"`
	// Reason ist die Begründung aus der Bewertung — auch die, warum nicht
	// berichtigt wird.
	Reason string `gorm:"size:500;serializer:encrypted" json:"reason,omitempty"`

	// EntryID ist die Buchung der Berichtigung. Fehlt sie, ist noch nicht
	// gebucht.
	EntryID   *uint     `gorm:"index" json:"entryId,omitempty"`
	BookedOn  string    `gorm:"size:10" json:"bookedOn,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Booked meldet, ob die Berichtigung dieses Jahres gebucht ist.
func (u InputTaxUsage) Booked() bool { return u.EntryID != nil }

// UsageFor liefert den Eintrag eines Jahres.
func (c *InputTaxCorrection) UsageFor(fiscalYear int) (InputTaxUsage, bool) {
	for _, u := range c.Usages {
		if u.FiscalYear == fiscalYear {
			return u, true
		}
	}
	return InputTaxUsage{}, false
}

// Open meldet, ob der Eintrag noch läuft.
func (c *InputTaxCorrection) Open() bool { return c.ClosedReason == "" }

// CoversYear meldet, ob ein Geschäftsjahr in den Berichtigungszeitraum fällt.
//
// Das Zugangsjahr gehört dazu. In ihm wurde der Vorsteuerabzug nach der
// beabsichtigten Verwendung gewährt; weicht die tatsächliche Verwendung schon
// im Jahr der erstmaligen Verwendung davon ab, ist auch das ein Fall des
// § 15a UStG (UStAE 15a.2 Abs. 2). Und das letzte Jahr gehört ebenfalls dazu:
// der Zeitraum läuft ab der erstmaligen Verwendung, nicht ab dem Jahresbeginn.
func (c *InputTaxCorrection) CoversYear(fiscalYear int) bool {
	return fiscalYear >= c.FirstFiscalYear && fiscalYear <= c.LastFiscalYear
}

// PeriodEndDate liefert das gespeicherte Ende des Berichtigungszeitraums.
//
// Fehlt es — ein Eintrag aus der Zeit, in der Buchfink den Zeitraum in vollen
// Wirtschaftsjahren führte —, wird es aus dem Anschaffungsdatum abgeleitet.
// Sonst stünde für Altbestände weiterhin ein um bis zu ein Jahr zu kurzer
// Zeitraum, ohne dass es jemandem auffiele.
func (c *InputTaxCorrection) PeriodEndDate() string {
	if len(c.PeriodEnd) == 10 {
		return c.PeriodEnd
	}
	end, err := CorrectionPeriodEndDate(c.AcquisitionDate, c.CorrectionPeriodYears)
	if err != nil {
		return ""
	}
	return end
}

// CorrectionPeriodEndDate bestimmt den letzten Tag des Berichtigungszeitraums.
//
// § 15a Abs. 1 UStG rechnet ab dem Zeitpunkt der erstmaligen Verwendung: fünf
// bzw. zehn Jahre, also bis zum Vortag desselben Kalendertages. § 45 UStDV
// rundet dieses Ende auf einen ganzen Kalendermonat: endet der Zeitraum vor dem
// 16. eines Monats, bleibt der Monat unberücksichtigt; endet er nach dem 15.,
// ist er voll zu berücksichtigen. Erst dadurch ist das Ende ein Monatsende, und
// nur so lässt sich ein Jahr nach Monaten gewichten.
func CorrectionPeriodEndDate(acquisitionDate string, periodYears int) (string, error) {
	if periodYears <= 0 {
		return "", fmt.Errorf("der Berichtigungszeitraum fehlt")
	}
	start, err := time.Parse("2006-01-02", acquisitionDate)
	if err != nil {
		return "", fmt.Errorf(
			"das Anschaffungsdatum %q ist kein Datum (erwartet JJJJ-MM-TT)", acquisitionDate)
	}
	// Der Vortag des Jahrestages: ein am 15.01.2026 in Verwendung genommenes
	// Wirtschaftsgut hat seine fünf Jahre am 14.01.2031 hinter sich.
	last := start.AddDate(periodYears, 0, 0).AddDate(0, 0, -1)
	month := last.Month()
	year := last.Year()
	if last.Day() >= 16 {
		// Der angefangene Monat zählt voll: das Ende rückt auf sein Ende vor.
		month++
	}
	end := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	return end.Format("2006-01-02"), nil
}

// MonthsInWindow zählt die Kalendermonate des Berichtigungszeitraums, die in
// ein Zeitfenster fallen — regelmäßig ein Geschäftsjahr.
//
// Gezählt wird in ganzen Monaten und nicht in Tagen: § 45 UStDV macht aus dem
// Zeitraum eine Folge ganzer Kalendermonate, und der Monat der erstmaligen
// Verwendung zählt dabei voll mit.
func (c *InputTaxCorrection) MonthsInWindow(from, to string) int {
	periodFrom, err := monthOrdinal(c.AcquisitionDate)
	if err != nil {
		return 0
	}
	periodTo, err := monthOrdinal(c.PeriodEndDate())
	if err != nil {
		return 0
	}
	windowFrom, err := monthOrdinal(from)
	if err != nil {
		return 0
	}
	windowTo, err := monthOrdinal(to)
	if err != nil {
		return 0
	}
	first := periodFrom
	if windowFrom > first {
		first = windowFrom
	}
	last := periodTo
	if windowTo < last {
		last = windowTo
	}
	if last < first {
		return 0
	}
	return int(last - first + 1)
}

// monthOrdinal nummeriert Kalendermonate fortlaufend, damit sich zwei Monate
// über Jahresgrenzen hinweg vergleichen und subtrahieren lassen.
func monthOrdinal(date string) (int64, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, err
	}
	return int64(t.Year())*12 + int64(t.Month()) - 1, nil
}

// Validate prüft, was gelten muss, bevor ein Eintrag gespeichert wird.
func (c *InputTaxCorrection) Validate() error {
	if strings.TrimSpace(c.Label) == "" {
		return fmt.Errorf("der Eintrag braucht eine Bezeichnung")
	}
	if len(c.AcquisitionDate) != 10 {
		return fmt.Errorf("das Anschaffungsdatum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if c.InputTaxAmount <= 0 {
		return fmt.Errorf(
			"ohne Vorsteuer gibt es nichts zu berichtigen. § 15a UStG berichtigt den Abzug, den es " +
				"gegeben hat")
	}
	if c.OriginalPermille < 0 || c.OriginalPermille > 1000 {
		return fmt.Errorf("der ursprüngliche Verwendungsanteil liegt zwischen 0 und 100 %%")
	}
	if c.CorrectionPeriodYears <= 0 {
		return fmt.Errorf("der Berichtigungszeitraum fehlt")
	}
	if c.LastFiscalYear < c.FirstFiscalYear {
		return fmt.Errorf(
			"der Berichtigungszeitraum endet im Jahr %d und begänne im Jahr %d",
			c.LastFiscalYear, c.FirstFiscalYear)
	}
	return nil
}

// EnsureLists ersetzt nicht belegte Listen durch leere. Der Eintrag geht als
// JSON an die Oberfläche, und `null.map` nähme dort den Baum mit.
func (c *InputTaxCorrection) EnsureLists() {
	if c.Usages == nil {
		c.Usages = make([]InputTaxUsage, 0)
	}
}

// InputTaxCorrectionRepository persistiert das Verzeichnis.
//
// Wie die Anlagenkartei kennt es kein Geschäftsjahr: ein 2021 angeschafftes
// Grundstück wird 2030 noch berichtigt, und eine auf ein Jahr begrenzte Abfrage
// zeigte in jedem Jahr außer dem des Zugangs ein leeres Verzeichnis.
type InputTaxCorrectionRepository interface {
	FindAll(ctx context.Context) ([]InputTaxCorrection, error)
	FindByID(ctx context.Context, id uint) (*InputTaxCorrection, error)
	// FindByAsset liefert den Eintrag zu einem Anlagegut oder nil.
	FindByAsset(ctx context.Context, assetID uint) (*InputTaxCorrection, error)
	Save(ctx context.Context, correction *InputTaxCorrection) error
	Delete(ctx context.Context, id uint) error
	// SaveUsage legt den Verwendungsanteil eines Jahres an oder schreibt ihn
	// fort.
	SaveUsage(ctx context.Context, usage *InputTaxUsage) error
}
