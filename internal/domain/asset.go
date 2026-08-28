package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AssetClass separates the three blocks the Anlagevermögen is shown in
// (§ 266 Abs. 2 A HGB). It is not cosmetic: each block follows different
// valuation rules, and every account the Anlagenverzeichnis proposes hangs off
// this one field.
//
//   - Immaterielle Vermögensgegenstände werden planmäßig abgeschrieben, dürfen
//     aber nur angesetzt werden, wenn sie entgeltlich erworben sind
//     (§ 248 Abs. 2 HGB lässt selbst geschaffene nur wahlweise zu).
//   - Sachanlagen werden planmäßig über die betriebsgewöhnliche Nutzungsdauer
//     abgeschrieben (§ 7 Abs. 1 EStG) — mit der großen Ausnahme Grund und Boden,
//     der sich nicht abnutzt.
//   - Finanzanlagen werden überhaupt nicht planmäßig abgeschrieben. Bei ihnen
//     gibt es nur die außerplanmäßige Abschreibung und die Zuschreibung, und
//     für sie gilt zusätzlich das gemilderte Niederstwertprinzip
//     (§ 253 Abs. 3 Satz 6 HGB).
type AssetClass string

const (
	AssetClassIntangible AssetClass = "intangible" // A. I. Immaterielle Vermögensgegenstände
	AssetClassTangible   AssetClass = "tangible"   // A. II. Sachanlagen
	AssetClassFinancial  AssetClass = "financial"  // A. III. Finanzanlagen
)

// Label renders the class for the UI.
func (c AssetClass) Label() string {
	switch c {
	case AssetClassIntangible:
		return "Immaterielle Vermögensgegenstände"
	case AssetClassTangible:
		return "Sachanlagen"
	case AssetClassFinancial:
		return "Finanzanlagen"
	default:
		return string(c)
	}
}

// Valid reports whether the class is one of the three known blocks.
func (c AssetClass) Valid() bool {
	switch c {
	case AssetClassIntangible, AssetClassTangible, AssetClassFinancial:
		return true
	default:
		return false
	}
}

// DepreciationMethod is how the Anschaffungskosten are spread over time — or
// that they are not spread at all.
//
// The choice is made once, at the Zugang, and it is the first question of the
// whole Anlagenbuchhaltung: below the GWG-Grenze the item is an expense straight
// away, in the Sammelposten it is dissolved in fifths regardless of its actual
// life, and above it, it becomes an Anlagegut with a plan. Grund und Boden and
// Finanzanlagen have no plan at all.
type DepreciationMethod string

const (
	// DepreciationLinear spreads the cost evenly over the useful life
	// (§ 7 Abs. 1 EStG). The default, and the only method that always works.
	DepreciationLinear DepreciationMethod = "linear"
	// DepreciationDegressive writes off a constant percentage of the remaining
	// book value (§ 7 Abs. 2 EStG). Only for bewegliche Wirtschaftsgüter and
	// only inside the statutory window — see accounting.AfAParametersFor.
	DepreciationDegressive DepreciationMethod = "degressive"
	// DepreciationPool is the Sammelposten of § 6 Abs. 2a EStG: everything of one
	// fiscal year in one pool, dissolved by one fifth in the year it is formed
	// and each of the following four — without regard to the actual useful life
	// and without pro rata temporis.
	DepreciationPool DepreciationMethod = "pool"
	// DepreciationImmediate is the Sofortabzug of § 6 Abs. 2 EStG: the whole
	// amount is an expense in the year of acquisition. The item still belongs in
	// the register — from 250 € § 6 Abs. 2 Satz 4 EStG demands a record of it.
	DepreciationImmediate DepreciationMethod = "immediate"
	// DepreciationNone marks an asset that does not wear out: Grund und Boden,
	// Beteiligungen, Wertpapiere. It can still lose value, but only through an
	// außerplanmäßige Abschreibung, never through a plan.
	DepreciationNone DepreciationMethod = "none"
)

// Label renders the method for the UI.
func (m DepreciationMethod) Label() string {
	switch m {
	case DepreciationLinear:
		return "Linear (§ 7 Abs. 1 EStG)"
	case DepreciationDegressive:
		return "Degressiv (§ 7 Abs. 2 EStG)"
	case DepreciationPool:
		return "Sammelposten (§ 6 Abs. 2a EStG)"
	case DepreciationImmediate:
		return "Sofortabzug GWG (§ 6 Abs. 2 EStG)"
	case DepreciationNone:
		return "Keine planmäßige Abschreibung"
	default:
		return string(m)
	}
}

// IsPlanned reports whether the method produces a recurring AfA-Buchung. Only
// these assets appear in the yearly Abschreibungslauf.
func (m DepreciationMethod) IsPlanned() bool {
	return m == DepreciationLinear || m == DepreciationDegressive || m == DepreciationPool
}

// AssetStatus is the lifecycle of an Anlagegut. It is derived, never stored: an
// asset is disposed of exactly when it carries an Abgangsdatum.
type AssetStatus string

const (
	AssetStatusActive        AssetStatus = "active"         // im Bestand
	AssetStatusFullyDepr     AssetStatus = "fully_written"  // im Bestand, Buchwert null
	AssetStatusDisposed      AssetStatus = "disposed"       // abgegangen
	AssetStatusNotYetBooked  AssetStatus = "unbooked"       // erfasst, aber ohne Zugangsbuchung
	AssetStatusDepreciateDue AssetStatus = "depreciate_due" // AfA des Geschäftsjahres fehlt
)

// DisposalKind separates the two ways an asset leaves the books. They differ in
// the booking, not only in the wording: a sale produces revenue and usually
// Umsatzsteuer, a scrapping produces neither and drops the Restbuchwert straight
// into the expense.
type DisposalKind string

const (
	DisposalSale     DisposalKind = "sale"
	DisposalScrapped DisposalKind = "scrapped"
)

// AssetMovementKind names every event that changes the value of an Anlagegut.
//
// The list is deliberately closed. The Anlagenspiegel (§ 284 Abs. 3 HGB) needs
// each column filled from a defined source; a free-text movement kind would land
// in no column at all.
type AssetMovementKind string

const (
	// AssetMovementAcquisition is the Zugang with the Anschaffungs- oder
	// Herstellungskosten of § 255 Abs. 1 HGB.
	AssetMovementAcquisition AssetMovementKind = "acquisition"
	// AssetMovementSubsequentCost is a nachträgliche Anschaffungskosten — freight,
	// assembly, a later upgrade. It raises the AfA-Bemessungsgrundlage.
	AssetMovementSubsequentCost AssetMovementKind = "subsequent_cost"
	// AssetMovementCostReduction is an Anschaffungspreisminderung: Skonto, Rabatt,
	// Zuschuss. § 255 Abs. 1 Satz 3 HGB takes it off the Anschaffungskosten — on an
	// Anlagegut it therefore lowers the basis, it is not an Ertrag.
	AssetMovementCostReduction AssetMovementKind = "cost_reduction"
	// AssetMovementDepreciation is the planmäßige AfA of one fiscal year.
	AssetMovementDepreciation AssetMovementKind = "depreciation"
	// AssetMovementImpairment is the außerplanmäßige Abschreibung of
	// § 253 Abs. 3 Satz 5 HGB (Sach- und immaterielle Anlagen: nur bei
	// voraussichtlich dauernder Wertminderung) bzw. Satz 6 (Finanzanlagen: auch
	// bei nicht dauernder).
	AssetMovementImpairment AssetMovementKind = "impairment"
	// AssetMovementWriteUp is the Zuschreibung of § 253 Abs. 5 Satz 1 HGB. It is
	// a Gebot, not a Wahlrecht: fällt der Grund für eine frühere außerplanmäßige
	// Abschreibung weg, ist zuzuschreiben — höchstens bis zu den fortgeführten
	// Anschaffungskosten.
	AssetMovementWriteUp AssetMovementKind = "write_up"
	// AssetMovementDisposal takes both the Anschaffungskosten and the accumulated
	// depreciation out of the books. Bei Finanzanlagen auch anteilig: eine
	// Tranche von Anteilen, die Tilgung einer Ausleihung.
	AssetMovementDisposal AssetMovementKind = "disposal"
	// AssetMovementTransfer is eine Umbuchung zwischen zwei Anlagekonten.
	//
	// Der Regelfall ist die Fertigstellung: was als Anlage im Bau auf 0700 lag,
	// wandert auf sein endgültiges Konto, und erst dann beginnt die Abschreibung.
	// Sie entsteht immer paarweise — eine Bewegung ab dem alten Konto, eine auf
	// das neue —, weil der Anlagenspiegel beide Positionen getrennt ausweist.
	AssetMovementTransfer AssetMovementKind = "transfer"
)

// Label renders the movement kind for the UI.
func (k AssetMovementKind) Label() string {
	switch k {
	case AssetMovementAcquisition:
		return "Zugang"
	case AssetMovementSubsequentCost:
		return "Nachträgliche Anschaffungskosten"
	case AssetMovementCostReduction:
		return "Anschaffungskostenminderung"
	case AssetMovementDepreciation:
		return "Planmäßige Abschreibung"
	case AssetMovementImpairment:
		return "Außerplanmäßige Abschreibung"
	case AssetMovementWriteUp:
		return "Zuschreibung"
	case AssetMovementDisposal:
		return "Abgang"
	case AssetMovementTransfer:
		return "Umbuchung"
	default:
		return string(k)
	}
}

// AssetMovement is one value change of an Anlagegut.
//
// It carries two signed amounts rather than one, because the Anlagenspiegel
// tracks two columns that must not be netted: the Anschaffungs- und
// Herstellungskosten and the kumulierten Abschreibungen. A disposal touches both
// at once — it removes the full AHK *and* the accumulated depreciation — and a
// single amount could not express that.
//
//	Buchwert = Σ CostAmount − Σ DepreciationAmount
type AssetMovement struct {
	ID      uint              `gorm:"primaryKey" json:"id"`
	AssetID uint              `gorm:"index;not null" json:"assetId"`
	Kind    AssetMovementKind `gorm:"size:20;not null;index" json:"kind"`

	// Date is the day the movement belongs to, FiscalYear the year it is
	// reported in. Both are stored: the fiscal year cannot be derived from the
	// date without the company's fiscal year start month, and that setting may
	// change while old movements must keep their year.
	Date       string `gorm:"size:10;not null;index" json:"date"`
	FiscalYear int    `gorm:"index;not null" json:"fiscalYear"`

	// Account is the Anlagekonto this movement belongs to.
	//
	// Fast immer ist es das Konto des Anlageguts. Nach einer Umbuchung ist es das
	// nicht mehr: die Zugänge von damals gehören weiter zu dem Konto, auf dem sie
	// standen, sonst verschiebt eine Fertigstellung rückwirkend die Vorjahre des
	// Anlagenspiegels. Leer heißt: das aktuelle Konto des Anlageguts.
	Account string `gorm:"size:10;index" json:"account,omitempty"`

	// CostAmount changes the Anschaffungs- und Herstellungskosten, positive on a
	// Zugang, negative on a Minderung or Abgang.
	CostAmount Cents `gorm:"not null;default:0" json:"costAmount"`
	// DepreciationAmount changes the kumulierten Abschreibungen, positive on an
	// Abschreibung, negative on a Zuschreibung or Abgang.
	DepreciationAmount Cents `gorm:"not null;default:0" json:"depreciationAmount"`

	// JournalEntryID links the movement to the booking that carries it. The
	// reference has to bear in both directions (Anlagegut → Buchung and zurück),
	// which is why the journal entry keeps the Source "depreciation" and this
	// side keeps the id.
	JournalEntryID *uint  `gorm:"index" json:"journalEntryId,omitempty"`
	EntryNumber    string `gorm:"-" json:"entryNumber,omitempty"`

	// LifeExtensionMonths verlängert die Restnutzungsdauer ab dem Jahr dieser
	// Bewegung.
	//
	// Eine Erweiterung macht ein Wirtschaftsgut oft länger nutzbar — ein Anbau
	// hält so lange wie das Gebäude, ein Austauschmotor verlängert das Leben der
	// Maschine. Die Verlängerung gehört an die Bewegung und nicht in die
	// Stammdaten: dort würde sie auch die Jahre ändern, die längst gebucht sind.
	LifeExtensionMonths int `gorm:"default:0" json:"lifeExtensionMonths,omitempty"`

	// Note carries the reason. On an außerplanmäßige Abschreibung it is not
	// optional: ein Ermessensvorgang, dessen Begründung fehlt, ist später von
	// niemandem mehr nachvollziehbar.
	Note string `gorm:"size:500;serializer:encrypted" json:"note,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// BookValueEffect is what the movement does to the Buchwert.
func (m AssetMovement) BookValueEffect() Cents {
	return m.CostAmount - m.DepreciationAmount
}

// FixedAsset is one Anlagegut of the Anlagenkartei.
//
// The kartei is kept across fiscal years while the journal is organised per
// year. That is a requirement of the Anlagenspiegel, not a design preference:
// Zugänge, Abgänge und kumulierte Abschreibungen müssen über die Jahre erhalten
// bleiben, sonst ist die Entwicklung einer Position nicht mehr darstellbar.
type FixedAsset struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// InventoryNumber is the Inventarnummer. It is allocated once and never
	// reused — it is the handle an auditor uses to find the item in the
	// warehouse and the booking in the journal.
	InventoryNumber string `gorm:"size:30;uniqueIndex;not null" json:"inventoryNumber"`

	Name        string     `gorm:"size:255;not null;serializer:encrypted" json:"name"`
	Description string     `gorm:"size:500;serializer:encrypted" json:"description,omitempty"`
	Class       AssetClass `gorm:"size:20;not null;index" json:"class"`

	// Account is the Anlagekonto (Bestandskonto der Klasse 0),
	// DepreciationAccount the Aufwandskonto of the planmäßigen AfA. Both are
	// stored on the asset rather than derived at booking time, because the
	// catalog may gain entries later and a historical booking must stay
	// explainable.
	Account             string `gorm:"size:10;not null;index" json:"account"`
	DepreciationAccount string `gorm:"size:10" json:"depreciationAccount,omitempty"`

	AcquisitionDate string `gorm:"size:10;not null;index" json:"acquisitionDate"`
	// AcquisitionCost is the Zugangswert the asset was created with. The
	// authoritative figure is the sum of the movements; this field is what the
	// Zugangsbewegung was created from and what a later correction compares
	// against.
	AcquisitionCost Cents `gorm:"not null" json:"acquisitionCost"`

	// InServiceDate ist der Tag, ab dem abgeschrieben wird.
	//
	// Er weicht vom Anschaffungsdatum ab, wo zwischen beiden etwas liegt: eine
	// Anlage im Bau wird über Monate bezahlt und erst mit der Fertigstellung
	// betriebsbereit. Die AfA beginnt dann dort und nicht bei der ersten
	// Anzahlung. Leer heißt: mit der Anschaffung.
	InServiceDate string `gorm:"size:10" json:"inServiceDate,omitempty"`

	Method DepreciationMethod `gorm:"size:20;not null" json:"method"`
	// UsefulLifeMonths is the betriebsgewöhnliche Nutzungsdauer in months. It
	// comes from the AfA-Tabellen of the BMF, which bind the Finanzverwaltung and
	// not the Steuerpflichtigen — a begründete abweichende Nutzungsdauer is
	// allowed, so this stays freely editable.
	UsefulLifeMonths int `gorm:"not null;default:0" json:"usefulLifeMonths"`
	// PoolYear is the fiscal year of a Sammelposten. § 6 Abs. 2a EStG forms one
	// pool per Wirtschaftsjahr; two years never share one.
	PoolYear int `gorm:"index;default:0" json:"poolYear,omitempty"`

	// --- Finanzanlagen -----------------------------------------------------

	// Identifier is the ISIN, WKN or Handelsregisternummer — what identifies a
	// Finanzanlage the way an Inventarnummer identifies a machine.
	Identifier string `gorm:"size:60;serializer:encrypted" json:"identifier,omitempty"`
	// HoldingPermille is the Beteiligungsquote in Promille. § 271 Abs. 1 Satz 3
	// HGB vermutet eine Beteiligung ab einem Anteil von einem Fünftel — die Quote
	// entscheidet also mit darüber, ob der Anteil unter "Beteiligungen" oder
	// unter "Wertpapiere des Anlagevermögens" auszuweisen ist.
	HoldingPermille int `gorm:"default:0" json:"holdingPermille,omitempty"`
	// TaxPrivileged marks an Anteil an einer Kapitalgesellschaft, dessen
	// Veräußerungsgewinn dem Teileinkünfteverfahren (§ 3 Nr. 40 EStG) bzw.
	// § 8b Abs. 2 KStG unterliegt. Der SKR04 hat dafür eigene Abgangskonten;
	// ohne diese Angabe landet der Vorgang auf den falschen.
	TaxPrivileged bool `gorm:"default:false" json:"taxPrivileged,omitempty"`

	// --- Verknüpfungen -----------------------------------------------------

	ContactID *uint `gorm:"index" json:"contactId,omitempty"`
	// AcquisitionEntryID points at the Zugangsbuchung. It stays optional: die
	// Kartei kann einen Altbestand aufnehmen, dessen Zugang in keinem Journal
	// dieser Datenbank steht.
	AcquisitionEntryID *uint `gorm:"index" json:"acquisitionEntryId,omitempty"`

	// --- Abgang ------------------------------------------------------------

	DisposalDate     string       `gorm:"size:10;index" json:"disposalDate,omitempty"`
	DisposalKind     DisposalKind `gorm:"size:20" json:"disposalKind,omitempty"`
	DisposalProceeds Cents        `gorm:"default:0" json:"disposalProceeds,omitempty"`
	DisposalEntryID  *uint        `gorm:"index" json:"disposalEntryId,omitempty"`

	Notes     string    `gorm:"type:text;serializer:encrypted" json:"notes,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Movements []AssetMovement `gorm:"foreignKey:AssetID;constraint:OnDelete:CASCADE" json:"movements,omitempty"`

	// --- Abgeleitete Werte, nicht gespeichert -------------------------------

	AccountName string      `gorm:"-" json:"accountName,omitempty"`
	Cost        Cents       `gorm:"-" json:"cost"`                 // AHK zum Stichtag
	Accumulated Cents       `gorm:"-" json:"accumulated"`          // kumulierte Abschreibungen
	BookValue   Cents       `gorm:"-" json:"bookValue"`            // Buchwert zum Stichtag
	YearAmount  Cents       `gorm:"-" json:"yearAmount"`           // im Geschäftsjahr gebuchte AfA
	DueAmount   Cents       `gorm:"-" json:"dueAmount"`            // im Geschäftsjahr noch fällige AfA
	Status      AssetStatus `gorm:"-" json:"status"`               // abgeleitet, nie gespeichert
	StatusNote  string      `gorm:"-" json:"statusNote,omitempty"` // ein Satz zum Status
}

// IsDisposed reports whether the asset has left the books.
func (a *FixedAsset) IsDisposed() bool { return a.DisposalDate != "" }

// DepreciationStart is the day the AfA runs from: die Betriebsbereitschaft, und
// nur wo die nicht eigens vermerkt ist, die Anschaffung.
func (a *FixedAsset) DepreciationStart() string {
	if a.InServiceDate != "" {
		return a.InServiceDate
	}
	return a.AcquisitionDate
}

// Validate enforces what has to hold before an Anlagegut may be saved.
//
// The checks are the ones whose absence produces a silently wrong AfA later on:
// a missing Nutzungsdauer divides by zero, a Sammelposten without a year cannot
// be dissolved, and a Finanzanlage with a plan would be depreciated where the
// HGB forbids it.
func (a *FixedAsset) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("das Anlagegut braucht eine Bezeichnung")
	}
	if !a.Class.Valid() {
		return fmt.Errorf("unbekannte Anlagenklasse %q", a.Class)
	}
	if a.Account == "" {
		return fmt.Errorf("das Anlagekonto fehlt")
	}
	if len(a.AcquisitionDate) != 10 {
		return fmt.Errorf("das Anschaffungsdatum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if a.AcquisitionCost <= 0 {
		return fmt.Errorf("die Anschaffungskosten müssen größer als null sein")
	}

	switch a.Method {
	case DepreciationLinear, DepreciationDegressive:
		if a.UsefulLifeMonths <= 0 {
			return fmt.Errorf("für eine planmäßige Abschreibung braucht es die betriebsgewöhnliche Nutzungsdauer")
		}
		if a.DepreciationAccount == "" {
			return fmt.Errorf("für eine planmäßige Abschreibung braucht es ein Aufwandskonto")
		}
		if a.Class == AssetClassFinancial {
			return fmt.Errorf(
				"Finanzanlagen werden nicht planmäßig abgeschrieben. Sie nutzen sich nicht ab; " +
					"an Wert verlieren können sie nur außerplanmäßig (§ 253 Abs. 3 Satz 5 und 6 HGB)")
		}
	case DepreciationPool:
		if a.PoolYear <= 0 {
			return fmt.Errorf("ein Sammelposten gehört zu genau einem Wirtschaftsjahr (§ 6 Abs. 2a EStG)")
		}
		if a.DepreciationAccount == "" {
			return fmt.Errorf("für die Auflösung des Sammelpostens braucht es ein Aufwandskonto")
		}
	case DepreciationImmediate:
		// Der Sofortabzug ist mit der Zugangsbuchung erledigt; ein Aufwandskonto
		// braucht die Kartei dafür nicht mehr.
	case DepreciationNone:
	default:
		return fmt.Errorf("unbekannte Abschreibungsmethode %q", a.Method)
	}

	if a.DisposalDate != "" && a.DisposalDate < a.AcquisitionDate {
		return fmt.Errorf("der Abgang am %s läge vor der Anschaffung am %s", a.DisposalDate, a.AcquisitionDate)
	}
	if a.HoldingPermille < 0 || a.HoldingPermille > 1000 {
		return fmt.Errorf("die Beteiligungsquote liegt zwischen 0 und 100 %%")
	}
	return nil
}

// FormatInventoryNumber renders an Inventarnummer, e.g. "AN-2026-0007".
func FormatInventoryNumber(fiscalYear int, seq int64) string {
	return fmt.Sprintf("AN-%d-%04d", fiscalYear, seq)
}

// NumberRangeAsset is the counter of the Inventarnummern. Like the
// Personenkonten it runs across fiscal years, so an item keeps a unique number
// no matter when it is recorded.
const NumberRangeAsset NumberRangeKey = "asset"

// AssetRepository persists the Anlagenkartei.
//
// FindAll deliberately takes no fiscal year. The Anlagenkartei is the one part
// of the accounting that is *not* organised per year — an asset acquired in 2019
// is still in the 2026 Anlagenspiegel, and a query bounded to a year would show
// an empty register in every year but the one of acquisition.
type AssetRepository interface {
	FindAll(ctx context.Context) ([]FixedAsset, error)
	FindByClass(ctx context.Context, class AssetClass) ([]FixedAsset, error)
	FindByID(ctx context.Context, id uint) (*FixedAsset, error)
	FindPool(ctx context.Context, fiscalYear int) (*FixedAsset, error)
	Save(ctx context.Context, asset *FixedAsset) error
	Delete(ctx context.Context, id uint) error
	AddMovement(ctx context.Context, movement *AssetMovement) error
	DeleteMovement(ctx context.Context, id uint) error
	FindMovements(ctx context.Context, assetID uint) ([]AssetMovement, error)
	// LinkedEntryIDs returns the journal entries the Anlagenkartei already points
	// at — the Zugangsbuchungen. It is what keeps the "noch nicht erfasst" list
	// of acquisition candidates honest.
	LinkedEntryIDs(ctx context.Context) (map[uint]bool, error)
	Count(ctx context.Context) (int64, error)
}

// AnlagenspiegelRow is one line of the Anlagenspiegel (§ 284 Abs. 3 HGB): the
// development of one position from the start to the end of the fiscal year.
//
// Kleine Kapitalgesellschaften sind von der Aufstellung befreit (§ 288 Abs. 1
// Nr. 1 HGB) — die Auswertung ist deshalb kein Pflichtteil der Oberfläche,
// sondern eine Ansicht, die ohnehin vorhandene Daten zusammenstellt.
type AnlagenspiegelRow struct {
	Class       AssetClass `json:"class"`
	Account     string     `json:"account"`
	AccountName string     `json:"accountName"`
	AssetCount  int        `json:"assetCount"`

	CostOpening Cents `json:"costOpening"` // AHK zu Beginn des Geschäftsjahres
	Additions   Cents `json:"additions"`   // Zugänge (inkl. nachträglicher AK, abzüglich Minderungen)
	Disposals   Cents `json:"disposals"`   // Abgänge zu Anschaffungskosten
	// Transfers sind Umbuchungen: negativ, wo etwas abgeht (Anlage im Bau),
	// positiv, wo es ankommt. Über alle Positionen summieren sie sich zu null.
	Transfers   Cents `json:"transfers"`
	CostClosing Cents `json:"costClosing"` // AHK am Ende des Geschäftsjahres

	DepreciationOpening  Cents `json:"depreciationOpening"`  // kumulierte Abschreibungen zu Beginn
	DepreciationYear     Cents `json:"depreciationYear"`     // Abschreibungen des Geschäftsjahres
	WriteUpsYear         Cents `json:"writeUpsYear"`         // Zuschreibungen des Geschäftsjahres
	DepreciationDisposal Cents `json:"depreciationDisposal"` // mit dem Abgang ausgebuchte Abschreibungen
	DepreciationTransfer Cents `json:"depreciationTransfer"` // mit der Umbuchung mitgewanderte Abschreibungen
	DepreciationClosing  Cents `json:"depreciationClosing"`  // kumulierte Abschreibungen am Ende

	BookValueOpening Cents `json:"bookValueOpening"` // Buchwert zu Beginn (= Vorjahr)
	BookValueClosing Cents `json:"bookValueClosing"` // Buchwert am Ende
}

// Anlagenspiegel is the whole Auswertung for one fiscal year.
type Anlagenspiegel struct {
	FiscalYear int                 `json:"fiscalYear"`
	Rows       []AnlagenspiegelRow `json:"rows"`
	Totals     AnlagenspiegelRow   `json:"totals"`
	// ClassTotals carries one subtotal per Bilanzblock, in the order of
	// § 266 Abs. 2 A HGB.
	ClassTotals []AnlagenspiegelRow `json:"classTotals"`
}
