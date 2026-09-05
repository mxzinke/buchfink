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

	// FirstFiscalYear ist das Geschäftsjahr des Zugangs, LastFiscalYear das
	// letzte Jahr des Berichtigungszeitraums.
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
// Das Zugangsjahr gehört nicht dazu: in ihm wurde der Vorsteuerabzug gewährt,
// berichtigt wird erst ab dem Jahr danach.
func (c *InputTaxCorrection) CoversYear(fiscalYear int) bool {
	return fiscalYear > c.FirstFiscalYear && fiscalYear <= c.LastFiscalYear
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
