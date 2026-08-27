package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// AssetService is the Anlagenbuchhaltung: the Anlagenverzeichnis, the AfA and
// everything that leaves the Anlagevermögen again.
//
// It never writes to the journal itself. Every booking it produces goes through
// JournalService like every other booking in the system, which is what keeps the
// hash chain, the Festschreibung and the account checks in force for the AfA as
// well. What this service adds is the second record the journal cannot hold: the
// Anlagenkartei, which is kept across fiscal years while the journal is
// organised per year.
type AssetService struct {
	assetRepo    domain.AssetRepository
	journalRepo  domain.JournalRepository
	journalSvc   *JournalService
	numberRepo   domain.NumberRangeRepository
	contactRepo  domain.ContactRepository
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
	taxResolver  domain.TaxResolver
	fiscalYear   int
}

// NewAssetService wires the Anlagenbuchhaltung.
func NewAssetService(
	assetRepo domain.AssetRepository,
	journalRepo domain.JournalRepository,
	journalSvc *JournalService,
	numberRepo domain.NumberRangeRepository,
	contactRepo domain.ContactRepository,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *AssetService {
	return &AssetService{
		assetRepo:    assetRepo,
		journalRepo:  journalRepo,
		journalSvc:   journalSvc,
		numberRepo:   numberRepo,
		contactRepo:  contactRepo,
		settingsRepo: settingsRepo,
		auditRepo:    auditRepo,
		taxResolver:  journalSvc.TaxResolver(),
		fiscalYear:   fiscalYear,
	}
}

// SetFiscalYear updates the year every derived figure is computed for.
func (s *AssetService) SetFiscalYear(year int) { s.fiscalYear = year }

// FiscalYear returns the active fiscal year.
func (s *AssetService) FiscalYear() int { return s.fiscalYear }

// AssetSummary aggregates the register for the head of the view.
type AssetSummary struct {
	FiscalYear  int          `json:"fiscalYear"`
	Count       int          `json:"count"`
	Cost        domain.Cents `json:"cost"`
	Accumulated domain.Cents `json:"accumulated"`
	BookValue   domain.Cents `json:"bookValue"`
	YearAmount  domain.Cents `json:"yearAmount"`
	DueAmount   domain.Cents `json:"dueAmount"`
	DueCount    int          `json:"dueCount"`
}

// AssetDetail is one Anlagegut with everything that explains it: its plan, its
// movements and the notes that belong to its class.
type AssetDetail struct {
	Asset     domain.FixedAsset      `json:"asset"`
	Schedule  []AssetScheduleYear    `json:"schedule"`
	Movements []domain.AssetMovement `json:"movements"`
	// Notes are the sentences that apply to *this* asset — its method, its class,
	// its account. They are computed rather than written into the frontend, so
	// they cannot drift away from the rules the booking actually follows.
	Notes []string `json:"notes"`
}

// AssetScheduleYear is one year of the AfA plan, merged with what was actually
// booked. The difference between the two is the whole point of the view: it is
// what the Abschreibungslauf offers to book.
type AssetScheduleYear struct {
	accounting.AfAYear
	Booked domain.Cents `json:"booked"`
	Due    domain.Cents `json:"due"`
	Status string       `json:"status"` // "gebucht" | "offen" | "teilweise" | "geplant"
}

// DepreciationDue is one line of the yearly Abschreibungslauf.
type DepreciationDue struct {
	AssetID         uint         `json:"assetId"`
	InventoryNumber string       `json:"inventoryNumber"`
	Name            string       `json:"name"`
	Account         string       `json:"account"`
	ExpenseAccount  string       `json:"expenseAccount"`
	Method          string       `json:"method"`
	RateLabel       string       `json:"rateLabel"`
	Months          int          `json:"months"`
	Planned         domain.Cents `json:"planned"`
	Booked          domain.Cents `json:"booked"`
	Due             domain.Cents `json:"due"`
	BookValueBefore domain.Cents `json:"bookValueBefore"`
	BookValueAfter  domain.Cents `json:"bookValueAfter"`
	Note            string       `json:"note,omitempty"`
}

// DepreciationRun is the preview of the yearly AfA.
//
// AfA is an Abschlussbuchung zum Bilanzstichtag, kein laufender Geschäftsvorfall
// — sie entsteht deshalb nicht im Hintergrund, sondern hier, auf Ansage, mit
// Vorschau und Freigabe.
type DepreciationRun struct {
	FiscalYear  int               `json:"fiscalYear"`
	BookingDate string            `json:"bookingDate"`
	Due         []DepreciationDue `json:"due"`
	Total       domain.Cents      `json:"total"`
	// MissingPriorYears names fiscal years before this one whose AfA was never
	// booked. Sie hier nachzuholen wäre falsch — die Abschreibung gehört in ihr
	// eigenes Jahr —, aber sie zu verschweigen wäre schlimmer.
	MissingPriorYears []int `json:"missingPriorYears,omitempty"`
}

// BookDepreciationRequest books the AfA of one fiscal year.
type BookDepreciationRequest struct {
	FiscalYear  int    `json:"fiscalYear"`
	BookingDate string `json:"bookingDate"`
	// AssetIDs narrows the run. Empty means everything that is due.
	AssetIDs []uint `json:"assetIds"`
}

// DepreciationResult reports what the run wrote.
type DepreciationResult struct {
	Entries []domain.JournalEntry `json:"entries"`
	Total   domain.Cents          `json:"total"`
	Skipped []string              `json:"skipped,omitempty"`
}

// ImpairmentRequest is an außerplanmäßige Abschreibung
// (§ 253 Abs. 3 Sätze 5 und 6 HGB).
type ImpairmentRequest struct {
	AssetID uint         `json:"assetId"`
	Date    string       `json:"date"`
	Amount  domain.Cents `json:"amount"`
	// Permanent is the decisive question. Bei Sachanlagen und immateriellen
	// Vermögensgegenständen ist nur die voraussichtlich dauernde Wertminderung
	// ein Grund; Finanzanlagen dürfen auch bei einer nicht dauernden abgeschrieben
	// werden.
	Permanent bool `json:"permanent"`
	// Reason is mandatory: ein Ermessensvorgang ohne festgehaltene Begründung ist
	// später von niemandem mehr nachvollziehbar.
	Reason string `json:"reason"`
}

// WriteUpRequest is a Zuschreibung (§ 253 Abs. 5 Satz 1 HGB).
type WriteUpRequest struct {
	AssetID uint         `json:"assetId"`
	Date    string       `json:"date"`
	Amount  domain.Cents `json:"amount"`
	Reason  string       `json:"reason"`
}

// DisposalRequest is the Abgang of an Anlagegut.
type DisposalRequest struct {
	AssetID      uint                `json:"assetId"`
	Date         string              `json:"date"`
	Kind         domain.DisposalKind `json:"kind"`
	Proceeds     domain.Cents        `json:"proceeds"` // netto
	TaxTreatment domain.TaxTreatment `json:"taxTreatment"`
	TaxRate      domain.TaxRate      `json:"taxRate"`
	// Settlement says whether the Erlös lands on a Zahlungsmittelkonto or stays
	// an offener Posten on the buyer's Personenkonto.
	Settlement     SettlementKind `json:"settlement"`
	PaymentAccount string         `json:"paymentAccount,omitempty"`
	ContactID      uint           `json:"contactId,omitempty"`
	Note           string         `json:"note,omitempty"`
}

// DisposalPreview is what an Abgang would look like before it is written.
type DisposalPreview struct {
	CatchUpAmount domain.Cents                `json:"catchUpAmount"`
	CatchUpLines  []domain.JournalLine        `json:"catchUpLines,omitempty"`
	BookValue     domain.Cents                `json:"bookValue"`
	Result        domain.Cents                `json:"result"` // Buchgewinn (+) oder Buchverlust (−)
	IsGain        bool                        `json:"isGain"`
	Accounts      accounting.DisposalAccounts `json:"accounts"`
	Lines         []domain.JournalLine        `json:"lines"`
	Gross         domain.Cents                `json:"gross"`
	Tax           domain.Cents                `json:"tax"`
	Warnings      []string                    `json:"warnings,omitempty"`
}

// DisposalResult reports the bookings an Abgang produced.
type DisposalResult struct {
	CatchUpEntry  *domain.JournalEntry `json:"catchUpEntry,omitempty"`
	DisposalEntry *domain.JournalEntry `json:"disposalEntry,omitempty"`
	Asset         domain.FixedAsset    `json:"asset"`
	Message       string               `json:"message"`
}

// AcquisitionCandidate is a booking that sits on an Anlagekonto without an
// Anlagegut behind it.
//
// It is the bridge between the Belegbuchung and the Kartei: der Zugang wird über
// den Beleg gebucht — mit Vorsteuer, Lieferant und Belegverweis —, und was dabei
// auf einem Konto der Klasse 0 landet, muss anschließend im Verzeichnis stehen.
// Diese Liste zeigt, was noch fehlt.
type AcquisitionCandidate struct {
	EntryID     uint         `json:"entryId"`
	EntryNumber string       `json:"entryNumber"`
	BookingDate string       `json:"bookingDate"`
	Description string       `json:"description"`
	Account     string       `json:"account"`
	AccountName string       `json:"accountName"`
	Amount      domain.Cents `json:"amount"`
	ContactID   *uint        `json:"contactId,omitempty"`
}

// -------------------------------------------------------------------------
// Lesen
// -------------------------------------------------------------------------

// List returns the register of one class — or all of it — with every derived
// figure computed for the active fiscal year.
func (s *AssetService) List(ctx context.Context, class domain.AssetClass) ([]domain.FixedAsset, error) {
	var (
		assets []domain.FixedAsset
		err    error
	)
	if class == "" {
		assets, err = s.assetRepo.FindAll(ctx)
	} else {
		assets, err = s.assetRepo.FindByClass(ctx, class)
	}
	if err != nil {
		return nil, fmt.Errorf("das Anlagenverzeichnis konnte nicht gelesen werden: %w", err)
	}

	startMonth := s.fiscalYearStartMonth(ctx)
	chart, _ := s.journalSvc.Chart(ctx)
	for i := range assets {
		s.enrich(&assets[i], s.fiscalYear, startMonth)
		if chart != nil {
			assets[i].AccountName = chart.Name(assets[i].Account)
		}
	}
	return assets, nil
}

// Pool returns the Sammelposten of a fiscal year, or nil if none was formed.
// § 6 Abs. 2a EStG allows exactly one per Wirtschaftsjahr, and further items of
// that year are added to it rather than to a second one.
func (s *AssetService) Pool(ctx context.Context, fiscalYear int) (*domain.FixedAsset, error) {
	if fiscalYear == 0 {
		fiscalYear = s.fiscalYear
	}
	pool, err := s.assetRepo.FindPool(ctx, fiscalYear)
	if err != nil || pool == nil {
		return nil, err
	}
	startMonth := s.fiscalYearStartMonth(ctx)
	s.enrich(pool, s.fiscalYear, startMonth)
	return pool, nil
}

// Summary aggregates the register for one class.
func (s *AssetService) Summary(ctx context.Context, class domain.AssetClass) (AssetSummary, error) {
	assets, err := s.List(ctx, class)
	if err != nil {
		return AssetSummary{}, err
	}
	sum := AssetSummary{FiscalYear: s.fiscalYear}
	for _, a := range assets {
		if a.IsDisposed() && a.BookValue == 0 && a.Cost == 0 {
			// Ein vollständig ausgebuchter Abgang steht noch im Verzeichnis, aber
			// nicht mehr im Bestand.
			continue
		}
		sum.Count++
		sum.Cost += a.Cost
		sum.Accumulated += a.Accumulated
		sum.BookValue += a.BookValue
		sum.YearAmount += a.YearAmount
		sum.DueAmount += a.DueAmount
		if a.DueAmount > 0 {
			sum.DueCount++
		}
	}
	return sum, nil
}

// Get returns one Anlagegut with its plan, its movements and the notes that
// apply to it.
func (s *AssetService) Get(ctx context.Context, id uint) (*AssetDetail, error) {
	asset, err := s.assetRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", id, err)
	}
	startMonth := s.fiscalYearStartMonth(ctx)
	s.enrich(asset, s.fiscalYear, startMonth)
	if chart, err := s.journalSvc.Chart(ctx); err == nil {
		asset.AccountName = chart.Name(asset.Account)
	}

	rows, err := accounting.BuildAfASchedule(s.planFor(asset, startMonth))
	if err != nil {
		return nil, err
	}
	booked := bookedByYear(asset.Movements)

	schedule := make([]AssetScheduleYear, 0, len(rows))
	for _, r := range rows {
		row := AssetScheduleYear{AfAYear: r, Booked: booked[r.FiscalYear]}
		row.Due = r.Amount - row.Booked
		if row.Due < 0 || !asset.Method.IsPlanned() {
			// Sofortabzug und nicht abnutzbare Güter laufen nicht über den
			// Abschreibungslauf; ihr Plan ist eine Erklärung, keine offene Buchung.
			row.Due = 0
		}
		switch {
		case row.Booked == 0 && r.FiscalYear > s.fiscalYear:
			row.Status = "geplant"
		case row.Booked == 0:
			row.Status = "offen"
		case row.Due > 0:
			row.Status = "teilweise"
		default:
			row.Status = "gebucht"
		}
		schedule = append(schedule, row)
	}

	movements := append([]domain.AssetMovement(nil), asset.Movements...)
	s.fillEntryNumbers(ctx, movements)

	return &AssetDetail{
		Asset:     *asset,
		Schedule:  schedule,
		Movements: movements,
		Notes:     s.notesFor(asset),
	}, nil
}

// notesFor collects the sentences that explain this one asset. They are derived
// from the same fields the booking uses, so an asset can never be shown with an
// explanation its own data contradicts.
func (s *AssetService) notesFor(asset *domain.FixedAsset) []string {
	var notes []string

	switch asset.Class {
	case domain.AssetClassFinancial:
		notes = append(notes,
			"Finanzanlagen nutzen sich nicht ab und werden deshalb nicht planmäßig abgeschrieben. "+
				"Sie stehen mit ihren Anschaffungskosten in der Bilanz, bis ein Grund für eine "+
				"außerplanmäßige Abschreibung eintritt.",
			"Für Finanzanlagen gilt das gemilderte Niederstwertprinzip: bei voraussichtlich dauernder "+
				"Wertminderung ist abzuschreiben, bei einer nicht dauernden darf abgeschrieben werden "+
				"(§ 253 Abs. 3 Sätze 5 und 6 HGB).",
			"Fällt der Grund später weg, ist wieder zuzuschreiben — höchstens bis zu den "+
				"Anschaffungskosten (§ 253 Abs. 5 Satz 1 HGB). Das ist ein Gebot, kein Wahlrecht.")
		if asset.TaxPrivileged {
			notes = append(notes,
				"Der Anteil ist als Beteiligung an einer Kapitalgesellschaft gekennzeichnet. Gewinn und "+
					"Verlust aus seiner Veräußerung laufen deshalb über eigene Konten — § 8b Abs. 2 KStG "+
					"bzw. § 3 Nr. 40 EStG stellen sie außerbilanziell wieder glatt.")
		}
		if asset.HoldingPermille > 0 {
			notes = append(notes, fmt.Sprintf(
				"Beteiligungsquote %s %%. Ab 20 %% vermutet § 271 Abs. 1 Satz 3 HGB eine Beteiligung; "+
					"darunter gehören Anteile regelmäßig unter die Wertpapiere des Anlagevermögens.",
				permilleLabel(asset.HoldingPermille)))
		}
	case domain.AssetClassIntangible:
		notes = append(notes,
			"Immaterielle Vermögensgegenstände des Anlagevermögens dürfen nur angesetzt werden, wenn "+
				"sie entgeltlich erworben wurden; für selbst geschaffene besteht nach § 248 Abs. 2 HGB "+
				"lediglich ein handelsrechtliches Wahlrecht und steuerlich ein Ansatzverbot.")
	}

	switch asset.Method {
	case domain.DepreciationLinear:
		notes = append(notes,
			"Lineare Abschreibung über die betriebsgewöhnliche Nutzungsdauer (§ 7 Abs. 1 EStG), "+
				"zeitanteilig ab dem Anschaffungsmonat (§ 7 Abs. 1 Satz 4 EStG).")
	case domain.DepreciationDegressive:
		notes = append(notes,
			"Degressive Abschreibung vom jeweiligen Restbuchwert, höchstens das Dreifache des linearen "+
				"Satzes und höchstens 30 % (§ 7 Abs. 2 EStG). Buchfink schlägt den Übergang zur linearen "+
				"Abschreibung in dem Jahr vor, in dem sie höher ausfällt (§ 7 Abs. 3 EStG).")
	case domain.DepreciationPool:
		notes = append(notes, fmt.Sprintf(
			"Sammelposten des Wirtschaftsjahres %d: aufzulösen mit je einem Fünftel im Jahr der Bildung "+
				"und den folgenden vier Jahren, ohne Zeitanteil und unabhängig von der tatsächlichen "+
				"Nutzungsdauer (§ 6 Abs. 2a EStG). Ein Abgang einzelner Güter berührt den Posten nicht.",
			asset.PoolYear))
	case domain.DepreciationImmediate:
		notes = append(notes,
			"Sofortabzug als geringwertiges Wirtschaftsgut (§ 6 Abs. 2 EStG): der Aufwand ist mit dem "+
				"Anschaffungsjahr erledigt und entsteht über die Belegbuchung, nicht über den "+
				"Abschreibungslauf. Das Gut bleibt trotzdem im Verzeichnis — ab 250 € verlangt "+
				"§ 6 Abs. 2 Satz 4 EStG genau das.")
	case domain.DepreciationNone:
		if asset.Class == domain.AssetClassTangible {
			notes = append(notes,
				"Kein planmäßiger Werteverzehr: Grund und Boden sowie Anlagen im Bau werden nicht "+
					"abgeschrieben. Mit der Fertigstellung wird umgebucht, und dann beginnt die AfA.")
		}
	}

	if asset.IsDisposed() {
		notes = append(notes, "Das Anlagegut ist abgegangen. Es bleibt im Verzeichnis stehen, damit der "+
			"Anlagenspiegel des Abgangsjahres vollständig bleibt.")
	}
	return notes
}

// -------------------------------------------------------------------------
// Schreiben (ohne Buchung)
// -------------------------------------------------------------------------

// Save creates or updates an Anlagegut.
//
// On creation it allocates the Inventarnummer and writes the Zugangsbewegung; on
// an update it touches the master data only. Änderungen an den
// Anschaffungskosten laufen über eigene Bewegungen — nachträgliche
// Anschaffungskosten und Minderungen sind Vorgänge mit Datum und Beleg, keine
// Korrektur eines Feldes.
func (s *AssetService) Save(ctx context.Context, asset *domain.FixedAsset) (*domain.FixedAsset, error) {
	if asset == nil {
		return nil, fmt.Errorf("kein Anlagegut übergeben")
	}
	asset.Name = strings.TrimSpace(asset.Name)
	if asset.Method == "" {
		asset.Method = domain.DepreciationLinear
	}
	if err := asset.Validate(); err != nil {
		return nil, err
	}
	if err := s.validateAccounts(ctx, asset); err != nil {
		return nil, err
	}

	if asset.ID != 0 {
		existing, err := s.assetRepo.FindByID(ctx, asset.ID)
		if err != nil {
			return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", asset.ID, err)
		}
		// Bewegungen und Abgang gehören den Buchungen, nicht der Stammdatenmaske.
		asset.InventoryNumber = existing.InventoryNumber
		asset.DisposalDate = existing.DisposalDate
		asset.DisposalKind = existing.DisposalKind
		asset.DisposalProceeds = existing.DisposalProceeds
		asset.DisposalEntryID = existing.DisposalEntryID
		asset.CreatedAt = existing.CreatedAt

		if asset.AcquisitionCost != existing.AcquisitionCost {
			if err := s.adjustAcquisitionMovement(ctx, existing, asset.AcquisitionCost); err != nil {
				return nil, err
			}
		}
		if err := s.assetRepo.Save(ctx, asset); err != nil {
			return nil, fmt.Errorf("Anlagegut konnte nicht gespeichert werden: %w", err)
		}
		// Ein Wechsel der Methode oder ein geänderter Betrag muss den Sofortabzug
		// mitnehmen — sonst bliebe ein GWG mit einem Buchwert stehen, den es nicht
		// hat, oder ein aktiviertes Gut mit einer Abschreibung, die es nicht gab.
		if err := s.syncImmediateWriteOff(ctx, asset, existing.Movements); err != nil {
			return nil, err
		}
		s.audit(ctx, domain.AuditActionUpdate, asset.ID, fmt.Sprintf(
			"Anlagegut %s geändert: %s", asset.InventoryNumber, asset.Name))
		return s.reload(ctx, asset.ID)
	}

	number, err := s.nextInventoryNumber(ctx, asset.AcquisitionDate)
	if err != nil {
		return nil, err
	}
	asset.InventoryNumber = number
	asset.CreatedAt = time.Now().UTC()
	if err := s.assetRepo.Save(ctx, asset); err != nil {
		return nil, fmt.Errorf("Anlagegut konnte nicht gespeichert werden: %w", err)
	}

	movement := &domain.AssetMovement{
		AssetID:        asset.ID,
		Kind:           domain.AssetMovementAcquisition,
		Date:           asset.AcquisitionDate,
		FiscalYear:     domain.GetFiscalYearForDate(asset.AcquisitionDate, s.fiscalYearStartMonth(ctx)),
		CostAmount:     asset.AcquisitionCost,
		JournalEntryID: asset.AcquisitionEntryID,
		Note:           "Zugang",
	}
	if err := s.assetRepo.AddMovement(ctx, movement); err != nil {
		return nil, fmt.Errorf("die Zugangsbewegung konnte nicht gespeichert werden: %w", err)
	}

	if err := s.syncImmediateWriteOff(ctx, asset, nil); err != nil {
		return nil, err
	}

	s.audit(ctx, domain.AuditActionCreate, asset.ID, fmt.Sprintf(
		"Anlagegut %s angelegt: %s, Konto %s, Anschaffungskosten %s € am %s",
		asset.InventoryNumber, asset.Name, asset.Account, asset.AcquisitionCost, asset.AcquisitionDate))

	return s.reload(ctx, asset.ID)
}

// CostAdjustmentRequest records nachträgliche Anschaffungskosten or an
// Anschaffungspreisminderung.
type CostAdjustmentRequest struct {
	AssetID uint         `json:"assetId"`
	Date    string       `json:"date"`
	Amount  domain.Cents `json:"amount"` // immer positiv
	// Reduction switches from nachträglichen Anschaffungskosten to a Minderung.
	Reduction      bool   `json:"reduction"`
	Note           string `json:"note"`
	JournalEntryID *uint  `json:"journalEntryId,omitempty"`
}

// RecordCostAdjustment writes a cost movement without booking anything.
//
// The booking already exists: Fracht und Montage kommen über den Beleg auf das
// Anlagekonto, und ein Skonto mindert es über den Zahlungsflow. Was hier
// entsteht, ist die Fortschreibung der Kartei — und damit eine neue
// AfA-Bemessungsgrundlage für die Folgejahre.
func (s *AssetService) RecordCostAdjustment(ctx context.Context, req CostAdjustmentRequest) (*domain.FixedAsset, error) {
	asset, err := s.assetRepo.FindByID(ctx, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", req.AssetID, err)
	}
	if asset.IsDisposed() {
		return nil, fmt.Errorf("%s ist am %s abgegangen und kann keine weiteren Anschaffungskosten mehr aufnehmen",
			asset.InventoryNumber, asset.DisposalDate)
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("der Betrag muss größer als null sein")
	}
	date := req.Date
	if len(date) != 10 {
		return nil, fmt.Errorf("das Datum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}

	movement := &domain.AssetMovement{
		AssetID:        asset.ID,
		Kind:           domain.AssetMovementSubsequentCost,
		Date:           date,
		FiscalYear:     domain.GetFiscalYearForDate(date, s.fiscalYearStartMonth(ctx)),
		CostAmount:     req.Amount,
		JournalEntryID: req.JournalEntryID,
		Note:           req.Note,
	}
	if req.Reduction {
		movement.Kind = domain.AssetMovementCostReduction
		movement.CostAmount = -req.Amount
	}
	if err := s.assetRepo.AddMovement(ctx, movement); err != nil {
		return nil, fmt.Errorf("die Bewegung konnte nicht gespeichert werden: %w", err)
	}
	s.audit(ctx, domain.AuditActionUpdate, asset.ID, fmt.Sprintf(
		"%s zu %s: %s € am %s", movement.Kind.Label(), asset.InventoryNumber, req.Amount, date))

	return s.reload(ctx, asset.ID)
}

// Delete removes an Anlagegut that never carried a booking. Once a movement
// points at a journal entry the register has to keep the item: the booking
// stands, and a Kartei that loses its explanation is worse than none.
func (s *AssetService) Delete(ctx context.Context, id uint) error {
	asset, err := s.assetRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", id, err)
	}
	for _, m := range asset.Movements {
		if m.JournalEntryID != nil {
			return fmt.Errorf(
				"%s hängt an der Buchung zu %s und kann nicht gelöscht werden. "+
					"Ein Anlagegut, das gebucht wurde, verlässt das Verzeichnis nur über einen Abgang",
				asset.InventoryNumber, m.Date)
		}
	}
	if err := s.assetRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("Anlagegut konnte nicht gelöscht werden: %w", err)
	}
	s.audit(ctx, domain.AuditActionUpdate, id, fmt.Sprintf(
		"Anlagegut %s gelöscht (ohne Buchungen)", asset.InventoryNumber))
	return nil
}

// -------------------------------------------------------------------------
// Abschreibungslauf
// -------------------------------------------------------------------------

// Run computes the AfA of the active fiscal year without writing anything.
func (s *AssetService) Run(ctx context.Context) (*DepreciationRun, error) {
	assets, err := s.assetRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("das Anlagenverzeichnis konnte nicht gelesen werden: %w", err)
	}
	startMonth := s.fiscalYearStartMonth(ctx)

	run := &DepreciationRun{
		FiscalYear:  s.fiscalYear,
		BookingDate: fiscalYearEndDate(s.fiscalYear, startMonth),
	}
	missing := map[int]bool{}

	for i := range assets {
		asset := &assets[i]
		if !asset.Method.IsPlanned() {
			continue
		}
		rows, err := accounting.BuildAfASchedule(s.planFor(asset, startMonth))
		if err != nil {
			// Ein einzelner nicht rechenbarer Plan darf den ganzen Lauf nicht
			// blockieren; er wird als Zeile ohne Betrag mit der Begründung gezeigt.
			run.Due = append(run.Due, DepreciationDue{
				AssetID: asset.ID, InventoryNumber: asset.InventoryNumber,
				Name: asset.Name, Account: asset.Account, Note: err.Error(),
			})
			continue
		}
		booked := bookedByYear(asset.Movements)
		for _, r := range rows {
			if r.FiscalYear >= s.fiscalYear {
				continue
			}
			if r.Amount > booked[r.FiscalYear] {
				missing[r.FiscalYear] = true
			}
		}

		var row *accounting.AfAYear
		for i := range rows {
			if rows[i].FiscalYear == s.fiscalYear {
				row = &rows[i]
				break
			}
		}
		if row == nil {
			continue
		}
		due := row.Amount - booked[s.fiscalYear]
		if due <= 0 {
			continue
		}
		run.Due = append(run.Due, DepreciationDue{
			AssetID:         asset.ID,
			InventoryNumber: asset.InventoryNumber,
			Name:            asset.Name,
			Account:         asset.Account,
			ExpenseAccount:  asset.DepreciationAccount,
			Method:          string(row.Method),
			RateLabel:       row.RateLabel,
			Months:          row.Months,
			Planned:         row.Amount,
			Booked:          booked[s.fiscalYear],
			Due:             due,
			BookValueBefore: row.OpeningBookValue - booked[s.fiscalYear],
			BookValueAfter:  row.ClosingBookValue,
			Note:            row.Note,
		})
		run.Total += due
	}

	for year := range missing {
		run.MissingPriorYears = append(run.MissingPriorYears, year)
	}
	sort.Ints(run.MissingPriorYears)
	sort.Slice(run.Due, func(i, j int) bool {
		return run.Due[i].InventoryNumber < run.Due[j].InventoryNumber
	})
	return run, nil
}

// BookDepreciation writes the AfA of a fiscal year, one entry per Anlagegut.
//
// One entry per asset rather than one collective booking: der Bezug zwischen
// Anlagegut und Buchung muss in beide Richtungen tragen, und eine Sammelbuchung
// über zwanzig Anlagen trägt in keine.
func (s *AssetService) BookDepreciation(ctx context.Context, req BookDepreciationRequest) (*DepreciationResult, error) {
	year := req.FiscalYear
	if year == 0 {
		year = s.fiscalYear
	}
	if year != s.fiscalYear {
		return nil, fmt.Errorf(
			"die Abschreibung gehört in ihr eigenes Geschäftsjahr. Für %d wechsle zuerst das Geschäftsjahr", year)
	}

	run, err := s.Run(ctx)
	if err != nil {
		return nil, err
	}
	bookingDate := req.BookingDate
	if bookingDate == "" {
		bookingDate = run.BookingDate
	}
	if domain.GetFiscalYearForDate(bookingDate, s.fiscalYearStartMonth(ctx)) != year {
		return nil, fmt.Errorf(
			"das Buchungsdatum %s liegt nicht im Geschäftsjahr %d. Die AfA ist eine Abschlussbuchung "+
				"zum Bilanzstichtag und gehört in das Jahr, das sie betrifft", bookingDate, year)
	}

	selected := map[uint]bool{}
	for _, id := range req.AssetIDs {
		selected[id] = true
	}

	result := &DepreciationResult{}
	for _, due := range run.Due {
		if len(selected) > 0 && !selected[due.AssetID] {
			continue
		}
		if due.Due <= 0 {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s: %s", due.InventoryNumber, due.Note))
			continue
		}
		asset, err := s.assetRepo.FindByID(ctx, due.AssetID)
		if err != nil {
			return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", due.AssetID, err)
		}
		entry, err := s.postDepreciation(ctx, asset, due.Due, bookingDate, year, fmt.Sprintf(
			"AfA %d: %s", year, asset.Name))
		if err != nil {
			return nil, fmt.Errorf("%s (%s): %w", asset.InventoryNumber, asset.Name, err)
		}
		result.Entries = append(result.Entries, *entry)
		result.Total += due.Due
	}

	if len(result.Entries) == 0 && len(result.Skipped) == 0 {
		return nil, fmt.Errorf("für das Geschäftsjahr %d ist keine Abschreibung offen", year)
	}
	s.audit(ctx, domain.AuditActionCreate, 0, fmt.Sprintf(
		"Abschreibungslauf %d: %d Buchungen über %s €", year, len(result.Entries), result.Total))
	return result, nil
}

// postDepreciation writes one AfA booking and the movement that belongs to it.
func (s *AssetService) postDepreciation(
	ctx context.Context, asset *domain.FixedAsset, amount domain.Cents,
	bookingDate string, fiscalYear int, description string,
) (*domain.JournalEntry, error) {
	if asset.DepreciationAccount == "" {
		return nil, fmt.Errorf("es ist kein Aufwandskonto für die Abschreibung hinterlegt")
	}
	entry := &domain.JournalEntry{
		BookingDate:        bookingDate,
		DocumentDate:       bookingDate,
		ServiceDateFrom:    bookingDate,
		ServiceDateTo:      bookingDate,
		Description:        description,
		Source:             domain.EntrySourceDepreciation,
		DocumentNumber:     asset.InventoryNumber,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: asset.DepreciationAccount, Amount: amount, Text: asset.InventoryNumber},
			{Side: domain.SideCredit, Account: asset.Account, Amount: amount, Text: asset.InventoryNumber},
		},
	}
	created, err := s.journalSvc.Post(ctx, entry)
	if err != nil {
		return nil, err
	}
	movement := &domain.AssetMovement{
		AssetID:            asset.ID,
		Kind:               domain.AssetMovementDepreciation,
		Date:               bookingDate,
		FiscalYear:         fiscalYear,
		DepreciationAmount: amount,
		JournalEntryID:     &created.ID,
		Note:               description,
	}
	if err := s.assetRepo.AddMovement(ctx, movement); err != nil {
		// Die Buchung steht; nur die Kartei hat sie nicht mitbekommen. Das ehrlich
		// zu melden ist besser, als die Buchung als gescheitert auszugeben.
		return created, fmt.Errorf(
			"die Buchung %s wurde geschrieben, die Bewegung im Anlagenverzeichnis aber nicht: %w",
			created.EntryNumber, err)
	}
	return created, nil
}

// PendingDepreciation reports the assets whose AfA for a fiscal year is still
// missing. The Festschreibung of a year asks this before it locks the period:
// AfA ist eine Abschlussbuchung, und ein festgeschriebenes Jahr nimmt sie nicht
// mehr auf.
func (s *AssetService) PendingDepreciation(ctx context.Context, fiscalYear int) ([]DepreciationDue, error) {
	if fiscalYear != 0 && fiscalYear != s.fiscalYear {
		return nil, nil
	}
	run, err := s.Run(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]DepreciationDue, 0, len(run.Due))
	for _, d := range run.Due {
		if d.Due > 0 {
			out = append(out, d)
		}
	}
	return out, nil
}

// -------------------------------------------------------------------------
// Außerplanmäßige Abschreibung und Zuschreibung
// -------------------------------------------------------------------------

// BookImpairment writes an außerplanmäßige Abschreibung.
func (s *AssetService) BookImpairment(ctx context.Context, req ImpairmentRequest) (*domain.JournalEntry, error) {
	asset, err := s.assetRepo.FindByID(ctx, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", req.AssetID, err)
	}
	if asset.IsDisposed() {
		return nil, fmt.Errorf("%s ist bereits abgegangen", asset.InventoryNumber)
	}
	if strings.TrimSpace(req.Reason) == "" {
		return nil, fmt.Errorf(
			"eine außerplanmäßige Abschreibung ist ein Ermessensvorgang. Ohne festgehaltene Begründung " +
				"kann sie später niemand mehr nachvollziehen — der Grund ist deshalb Pflicht")
	}
	if len(req.Date) != 10 {
		return nil, fmt.Errorf("das Datum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}

	startMonth := s.fiscalYearStartMonth(ctx)
	s.enrich(asset, domain.GetFiscalYearForDate(req.Date, startMonth), startMonth)
	if req.Amount <= 0 {
		return nil, fmt.Errorf("der Abschreibungsbetrag muss größer als null sein")
	}
	if req.Amount > asset.BookValue {
		return nil, fmt.Errorf(
			"der Buchwert von %s beträgt %s €; mehr kann nicht abgeschrieben werden",
			asset.InventoryNumber, asset.BookValue)
	}

	expense, err := accounting.ImpairmentAccount(asset.Class, asset.Account, req.Permanent, asset.TaxPrivileged)
	if err != nil {
		return nil, err
	}

	entry := &domain.JournalEntry{
		BookingDate:        req.Date,
		DocumentDate:       req.Date,
		ServiceDateFrom:    req.Date,
		ServiceDateTo:      req.Date,
		Description:        fmt.Sprintf("Außerplanmäßige Abschreibung %s: %s", asset.Name, req.Reason),
		Source:             domain.EntrySourceClosing,
		DocumentNumber:     asset.InventoryNumber,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: expense, Amount: req.Amount, Text: asset.InventoryNumber},
			{Side: domain.SideCredit, Account: asset.Account, Amount: req.Amount, Text: asset.InventoryNumber},
		},
	}
	created, err := s.journalSvc.Post(ctx, entry)
	if err != nil {
		return nil, err
	}

	movement := &domain.AssetMovement{
		AssetID:            asset.ID,
		Kind:               domain.AssetMovementImpairment,
		Date:               req.Date,
		FiscalYear:         domain.GetFiscalYearForDate(req.Date, startMonth),
		DepreciationAmount: req.Amount,
		JournalEntryID:     &created.ID,
		Note:               req.Reason,
	}
	if err := s.assetRepo.AddMovement(ctx, movement); err != nil {
		return created, fmt.Errorf(
			"die Buchung %s wurde geschrieben, die Bewegung im Anlagenverzeichnis aber nicht: %w",
			created.EntryNumber, err)
	}
	s.audit(ctx, domain.AuditActionCreate, asset.ID, fmt.Sprintf(
		"Außerplanmäßige Abschreibung auf %s: %s € (%s)", asset.InventoryNumber, req.Amount, req.Reason))
	return created, nil
}

// BookWriteUp writes a Zuschreibung.
//
// Its ceiling is the reason this cannot be a free booking: zugeschrieben wird
// höchstens bis zu den fortgeführten Anschaffungskosten — also bis zu dem
// Buchwert, den das Anlagegut ohne die außerplanmäßige Abschreibung heute hätte
// (§ 253 Abs. 5 Satz 1 HGB).
func (s *AssetService) BookWriteUp(ctx context.Context, req WriteUpRequest) (*domain.JournalEntry, error) {
	asset, err := s.assetRepo.FindByID(ctx, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", req.AssetID, err)
	}
	if asset.IsDisposed() {
		return nil, fmt.Errorf("%s ist bereits abgegangen", asset.InventoryNumber)
	}
	if len(req.Date) != 10 {
		return nil, fmt.Errorf("das Datum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("der Zuschreibungsbetrag muss größer als null sein")
	}

	revenue, err := accounting.WriteUpAccount(asset.Class, asset.Account, asset.TaxPrivileged)
	if err != nil {
		return nil, err
	}

	startMonth := s.fiscalYearStartMonth(ctx)
	fiscalYear := domain.GetFiscalYearForDate(req.Date, startMonth)
	s.enrich(asset, fiscalYear, startMonth)

	ceiling, err := s.writeUpCeiling(ctx, asset, fiscalYear, startMonth)
	if err != nil {
		return nil, err
	}
	if req.Amount > ceiling {
		return nil, fmt.Errorf(
			"höchstens %s € dürfen zugeschrieben werden. Die Obergrenze sind die fortgeführten "+
				"Anschaffungskosten: der Buchwert, den %s ohne die außerplanmäßige Abschreibung heute "+
				"hätte (§ 253 Abs. 5 Satz 1 HGB)", ceiling, asset.InventoryNumber)
	}

	entry := &domain.JournalEntry{
		BookingDate:        req.Date,
		DocumentDate:       req.Date,
		ServiceDateFrom:    req.Date,
		ServiceDateTo:      req.Date,
		Description:        fmt.Sprintf("Zuschreibung %s: %s", asset.Name, req.Reason),
		Source:             domain.EntrySourceClosing,
		DocumentNumber:     asset.InventoryNumber,
		TaxTreatment:       domain.TaxTreatmentNotTaxable,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: asset.Account, Amount: req.Amount, Text: asset.InventoryNumber},
			{Side: domain.SideCredit, Account: revenue, Amount: req.Amount, Text: asset.InventoryNumber},
		},
	}
	created, err := s.journalSvc.Post(ctx, entry)
	if err != nil {
		return nil, err
	}

	movement := &domain.AssetMovement{
		AssetID:            asset.ID,
		Kind:               domain.AssetMovementWriteUp,
		Date:               req.Date,
		FiscalYear:         fiscalYear,
		DepreciationAmount: -req.Amount,
		JournalEntryID:     &created.ID,
		Note:               req.Reason,
	}
	if err := s.assetRepo.AddMovement(ctx, movement); err != nil {
		return created, fmt.Errorf(
			"die Buchung %s wurde geschrieben, die Bewegung im Anlagenverzeichnis aber nicht: %w",
			created.EntryNumber, err)
	}
	s.audit(ctx, domain.AuditActionCreate, asset.ID, fmt.Sprintf(
		"Zuschreibung auf %s: %s €", asset.InventoryNumber, req.Amount))
	return created, nil
}

// writeUpCeiling is the room a Zuschreibung has: the book value the asset would
// have today with its plan alone, minus what it actually carries.
func (s *AssetService) writeUpCeiling(
	ctx context.Context, asset *domain.FixedAsset, fiscalYear, startMonth int,
) (domain.Cents, error) {
	plan := s.planFor(asset, startMonth)
	plan.ImpairmentsByYear = nil // genau darum geht es: der Wert ohne die außerplanmäßige Abschreibung

	rows, err := accounting.BuildAfASchedule(plan)
	if err != nil {
		return 0, err
	}
	continued := plan.Cost
	for _, r := range rows {
		if r.FiscalYear > fiscalYear {
			break
		}
		continued -= r.Amount
	}
	ceiling := continued - asset.BookValue
	if ceiling < 0 {
		return 0, nil
	}
	return ceiling, nil
}

// -------------------------------------------------------------------------
// Abgang
// -------------------------------------------------------------------------

// PreviewDisposal computes an Abgang without writing it.
func (s *AssetService) PreviewDisposal(ctx context.Context, req DisposalRequest) (*DisposalPreview, error) {
	asset, err := s.assetRepo.FindByID(ctx, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", req.AssetID, err)
	}
	preview, _, _, err := s.buildDisposal(ctx, asset, req)
	return preview, err
}

// Dispose books the Abgang: first the AfA up to the month of disposal, then the
// disposal itself.
//
// Three things happen at once and in this order — die AfA wird nachgeholt, der
// Restbuchwert verschwindet, ein Erlös entsteht —, und die Differenz ist der
// Buchgewinn oder -verlust. Der SKR04 wählt das Erlöskonto nach diesem Ergebnis,
// weshalb es feststehen muss, bevor kontiert wird.
func (s *AssetService) Dispose(ctx context.Context, req DisposalRequest) (*DisposalResult, error) {
	asset, err := s.assetRepo.FindByID(ctx, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", req.AssetID, err)
	}
	preview, catchUpLines, disposalLines, err := s.buildDisposal(ctx, asset, req)
	if err != nil {
		return nil, err
	}

	startMonth := s.fiscalYearStartMonth(ctx)
	fiscalYear := domain.GetFiscalYearForDate(req.Date, startMonth)
	result := &DisposalResult{}

	// 1. AfA bis zum Abgangsmonat nachholen.
	if preview.CatchUpAmount > 0 && len(catchUpLines) > 0 {
		entry, err := s.postDepreciation(ctx, asset, preview.CatchUpAmount, req.Date, fiscalYear, fmt.Sprintf(
			"AfA bis zum Abgang: %s", asset.Name))
		if err != nil {
			return nil, fmt.Errorf("die AfA bis zum Abgangsmonat konnte nicht gebucht werden: %w", err)
		}
		result.CatchUpEntry = entry
	}

	// 2. Restbuchwert ausbuchen und den Erlös erfassen.
	if len(disposalLines) > 0 {
		description := fmt.Sprintf("Abgang %s (%s)", asset.Name, asset.InventoryNumber)
		if req.Kind == domain.DisposalScrapped {
			description = fmt.Sprintf("Verschrottung %s (%s)", asset.Name, asset.InventoryNumber)
		}
		entry := &domain.JournalEntry{
			BookingDate:        req.Date,
			DocumentDate:       req.Date,
			ServiceDateFrom:    req.Date,
			ServiceDateTo:      req.Date,
			Description:        description,
			Source:             domain.EntrySourceManual,
			DocumentNumber:     asset.InventoryNumber,
			TaxTreatment:       req.TaxTreatment,
			PostingRuleVersion: accounting.PostingRuleVersion,
			Lines:              disposalLines,
		}
		if req.ContactID != 0 {
			contactID := req.ContactID
			entry.ContactID = &contactID
		}
		created, err := s.journalSvc.Post(ctx, entry)
		if err != nil {
			if result.CatchUpEntry != nil {
				return nil, fmt.Errorf(
					"die nachgeholte AfA %s wurde bereits gebucht, der Abgang selbst aber nicht: %w",
					result.CatchUpEntry.EntryNumber, err)
			}
			return nil, err
		}
		result.DisposalEntry = created
	}

	// 3. Kartei fortschreiben: Anschaffungskosten und kumulierte Abschreibungen
	//    verlassen die Bücher gemeinsam.
	fresh, err := s.assetRepo.FindByID(ctx, asset.ID)
	if err != nil {
		return nil, err
	}
	s.enrich(fresh, fiscalYear, startMonth)

	movement := &domain.AssetMovement{
		AssetID:            fresh.ID,
		Kind:               domain.AssetMovementDisposal,
		Date:               req.Date,
		FiscalYear:         fiscalYear,
		CostAmount:         -fresh.Cost,
		DepreciationAmount: -fresh.Accumulated,
		Note:               req.Note,
	}
	if result.DisposalEntry != nil {
		movement.JournalEntryID = &result.DisposalEntry.ID
	}
	if err := s.assetRepo.AddMovement(ctx, movement); err != nil {
		return nil, fmt.Errorf("die Abgangsbewegung konnte nicht gespeichert werden: %w", err)
	}

	fresh.DisposalDate = req.Date
	fresh.DisposalKind = req.Kind
	fresh.DisposalProceeds = req.Proceeds
	if result.DisposalEntry != nil {
		fresh.DisposalEntryID = &result.DisposalEntry.ID
	}
	if err := s.assetRepo.Save(ctx, fresh); err != nil {
		return nil, fmt.Errorf("der Abgang konnte nicht gespeichert werden: %w", err)
	}

	reloaded, err := s.reload(ctx, fresh.ID)
	if err != nil {
		return nil, err
	}
	result.Asset = *reloaded
	switch {
	case preview.Result > 0:
		result.Message = fmt.Sprintf("Abgang gebucht. Buchgewinn %s €.", preview.Result)
	case preview.Result < 0:
		result.Message = fmt.Sprintf("Abgang gebucht. Buchverlust %s €.", -preview.Result)
	default:
		result.Message = "Abgang gebucht. Weder Buchgewinn noch Buchverlust."
	}

	s.audit(ctx, domain.AuditActionUpdate, fresh.ID, fmt.Sprintf(
		"Abgang %s am %s: Erlös %s €, Restbuchwert %s €, Ergebnis %s €",
		fresh.InventoryNumber, req.Date, req.Proceeds, preview.BookValue, preview.Result))
	return result, nil
}

// buildDisposal computes the whole Abgang: the AfA to catch up, the resulting
// Restbuchwert, the Buchgewinn or -verlust, the accounts that follow from it and
// the lines of both bookings. Preview and booking run through it, so the two
// cannot disagree.
func (s *AssetService) buildDisposal(
	ctx context.Context, asset *domain.FixedAsset, req DisposalRequest,
) (*DisposalPreview, []domain.JournalLine, []domain.JournalLine, error) {
	if asset.IsDisposed() {
		return nil, nil, nil, fmt.Errorf("%s ist bereits am %s abgegangen", asset.InventoryNumber, asset.DisposalDate)
	}
	if len(req.Date) != 10 {
		return nil, nil, nil, fmt.Errorf("das Abgangsdatum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if req.Date < asset.AcquisitionDate {
		return nil, nil, nil, fmt.Errorf("der Abgang am %s läge vor der Anschaffung am %s",
			req.Date, asset.AcquisitionDate)
	}
	if req.Proceeds < 0 {
		return nil, nil, nil, fmt.Errorf("der Erlös kann nicht negativ sein")
	}
	if req.Kind == domain.DisposalScrapped && req.Proceeds != 0 {
		return nil, nil, nil, fmt.Errorf("eine Verschrottung hat keinen Erlös")
	}
	if asset.Method == domain.DepreciationPool {
		return nil, nil, nil, fmt.Errorf(
			"ein Sammelposten geht nicht ab. Scheidet ein einzelnes Wirtschaftsgut aus dem " +
				"Betriebsvermögen aus, wird der Sammelposten nicht vermindert (§ 6 Abs. 2a Satz 4 EStG) — " +
				"er löst sich weiter mit einem Fünftel je Jahr auf")
	}

	startMonth := s.fiscalYearStartMonth(ctx)
	fiscalYear := domain.GetFiscalYearForDate(req.Date, startMonth)

	preview := &DisposalPreview{}

	// AfA bis einschließlich des Abgangsmonats.
	plan := s.planFor(asset, startMonth)
	plan.DisposalDate = req.Date
	rows, err := accounting.BuildAfASchedule(plan)
	if err != nil {
		return nil, nil, nil, err
	}
	booked := bookedByYear(asset.Movements)
	for _, r := range rows {
		if r.FiscalYear < fiscalYear && r.Amount > booked[r.FiscalYear] {
			preview.Warnings = append(preview.Warnings, fmt.Sprintf(
				"Für das Geschäftsjahr %d fehlt noch Abschreibung. Sie gehört in ihr eigenes Jahr und "+
					"wird hier nicht nachgeholt.", r.FiscalYear))
		}
		if r.FiscalYear == fiscalYear {
			if due := r.Amount - booked[fiscalYear]; due > 0 {
				preview.CatchUpAmount = due
			}
		}
	}

	var catchUpLines []domain.JournalLine
	if preview.CatchUpAmount > 0 {
		if asset.DepreciationAccount == "" {
			return nil, nil, nil, fmt.Errorf("es ist kein Aufwandskonto für die Abschreibung hinterlegt")
		}
		catchUpLines = []domain.JournalLine{
			{Side: domain.SideDebit, Account: asset.DepreciationAccount, Amount: preview.CatchUpAmount,
				Text: "AfA bis zum Abgangsmonat"},
			{Side: domain.SideCredit, Account: asset.Account, Amount: preview.CatchUpAmount,
				Text: "AfA bis zum Abgangsmonat"},
		}
	}
	preview.CatchUpLines = s.named(ctx, catchUpLines)

	s.enrich(asset, fiscalYear, startMonth)
	preview.BookValue = asset.BookValue - preview.CatchUpAmount
	if preview.BookValue < 0 {
		preview.BookValue = 0
	}
	preview.Result = req.Proceeds - preview.BookValue
	preview.IsGain = preview.Result > 0

	treatment := req.TaxTreatment
	if treatment == "" {
		treatment = domain.TaxTreatmentDomestic
		if asset.Class == domain.AssetClassFinancial {
			// Die Veräußerung von Anteilen ist nach § 4 Nr. 8 UStG steuerfrei.
			treatment = domain.TaxTreatmentExempt
		}
	}
	accounts, err := accounting.DisposalAccountsFor(asset.Class, treatment, preview.IsGain, asset.TaxPrivileged)
	if err != nil {
		return nil, nil, nil, err
	}
	preview.Accounts = accounts

	var lines []domain.JournalLine

	// Erlös mit Umsatzsteuer.
	if req.Proceeds > 0 {
		rate := req.TaxRate
		if treatment != domain.TaxTreatmentDomestic {
			rate = domain.TaxRateNone
		} else if rate == domain.TaxRateNone {
			rate = domain.TaxRateStandard
		}
		legs, err := s.taxResolver.Resolve(domain.DirectionOutgoing, treatment, rate, req.Proceeds)
		if err != nil {
			return nil, nil, nil, err
		}
		lines = append(lines, domain.JournalLine{
			Side: domain.SideCredit, Account: accounts.Revenue, Amount: req.Proceeds,
			Text: asset.InventoryNumber,
		})
		for _, leg := range legs {
			if leg.Amount == 0 {
				continue
			}
			preview.Tax += leg.Amount
			lines = append(lines, taxLegLine(leg))
		}

		gross := req.Proceeds + preview.Tax
		preview.Gross = gross
		settlement, err := s.disposalSettlementLine(ctx, req, gross)
		if err != nil {
			return nil, nil, nil, err
		}
		lines = append([]domain.JournalLine{settlement}, lines...)
	}

	// Restbuchwert ausbuchen.
	if preview.BookValue > 0 {
		lines = append(lines,
			domain.JournalLine{Side: domain.SideDebit, Account: accounts.BookValue,
				Amount: preview.BookValue, Text: "Restbuchwert " + asset.InventoryNumber},
			domain.JournalLine{Side: domain.SideCredit, Account: asset.Account,
				Amount: preview.BookValue, Text: "Restbuchwert " + asset.InventoryNumber},
		)
	}

	if len(lines) == 0 {
		preview.Warnings = append(preview.Warnings,
			"Der Buchwert ist null und es fließt kein Erlös: zu buchen gibt es nichts. Das Anlagegut "+
				"wird nur im Verzeichnis als abgegangen vermerkt.")
	}
	preview.Lines = s.named(ctx, lines)
	return preview, catchUpLines, lines, nil
}

// disposalSettlementLine books what the buyer actually pays: either straight
// onto a Zahlungsmittelkonto or as an offener Posten on their Personenkonto.
func (s *AssetService) disposalSettlementLine(
	ctx context.Context, req DisposalRequest, gross domain.Cents,
) (domain.JournalLine, error) {
	switch req.Settlement {
	case SettlementPaid:
		if req.PaymentAccount == "" {
			return domain.JournalLine{}, fmt.Errorf(
				"bei sofortiger Zahlung muss das Zahlungsmittel angegeben werden (Kasse, Bank)")
		}
		if !isLiquidAccount(req.PaymentAccount) {
			return domain.JournalLine{}, fmt.Errorf("Konto %s ist kein Zahlungsmittelkonto", req.PaymentAccount)
		}
		return domain.JournalLine{Side: domain.SideDebit, Account: req.PaymentAccount, Amount: gross}, nil
	default:
		if req.ContactID == 0 {
			return domain.JournalLine{}, fmt.Errorf(
				"ohne Käufer bleibt offen, auf wessen Personenkonto die Forderung steht. " +
					"Wähle den Kunden oder buche den Erlös als sofort bezahlt")
		}
		contact, err := s.contactRepo.FindByID(ctx, req.ContactID)
		if err != nil {
			return domain.JournalLine{}, fmt.Errorf("der Käufer konnte nicht geladen werden: %w", err)
		}
		if contact.Type != domain.ContactTypeCustomer {
			return domain.JournalLine{}, fmt.Errorf(
				"%s ist als Lieferant angelegt. Ein Verkaufserlös gehört auf ein Debitorenkonto", contact.Name)
		}
		return domain.JournalLine{
			Side: domain.SideDebit, Account: contact.LedgerAccount, ContactID: &contact.ID, Amount: gross,
		}, nil
	}
}

// -------------------------------------------------------------------------
// Anlagenspiegel
// -------------------------------------------------------------------------

// Anlagenspiegel builds the Entwicklung des Anlagevermögens for a fiscal year.
//
// It is an Auswertung, not a further booking — but one that only works because
// the Kartei is kept across the years. Zugänge, Abgänge und kumulierte
// Abschreibungen eines 2019 angeschafften Wirtschaftsguts stehen 2026 noch
// genauso da; das Journal allein könnte die Spalten nicht füllen.
func (s *AssetService) Anlagenspiegel(ctx context.Context) (*domain.Anlagenspiegel, error) {
	assets, err := s.assetRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("das Anlagenverzeichnis konnte nicht gelesen werden: %w", err)
	}
	chart, _ := s.journalSvc.Chart(ctx)
	year := s.fiscalYear

	byAccount := map[string]*domain.AnlagenspiegelRow{}
	order := []string{}

	for i := range assets {
		asset := &assets[i]
		row, ok := byAccount[asset.Account]
		if !ok {
			name := asset.Account
			if entry, found := accounting.LookupAssetAccount(asset.Account); found {
				name = entry.Name
			} else if chart != nil {
				name = chart.Name(asset.Account)
			}
			row = &domain.AnlagenspiegelRow{
				Class: asset.Class, Account: asset.Account, AccountName: name,
			}
			byAccount[asset.Account] = row
			order = append(order, asset.Account)
		}
		row.AssetCount++

		for _, m := range asset.Movements {
			switch {
			case m.FiscalYear < year:
				row.CostOpening += m.CostAmount
				row.DepreciationOpening += m.DepreciationAmount
			case m.FiscalYear == year:
				switch m.Kind {
				case domain.AssetMovementDisposal:
					row.Disposals += -m.CostAmount
					row.DepreciationDisposal += -m.DepreciationAmount
				case domain.AssetMovementWriteUp:
					row.WriteUpsYear += -m.DepreciationAmount
				case domain.AssetMovementDepreciation, domain.AssetMovementImpairment:
					row.DepreciationYear += m.DepreciationAmount
				default:
					row.Additions += m.CostAmount
				}
			}
		}
	}

	spiegel := &domain.Anlagenspiegel{FiscalYear: year}
	classTotals := map[domain.AssetClass]*domain.AnlagenspiegelRow{}

	for _, account := range order {
		row := byAccount[account]
		row.CostClosing = row.CostOpening + row.Additions - row.Disposals
		row.DepreciationClosing = row.DepreciationOpening + row.DepreciationYear -
			row.WriteUpsYear - row.DepreciationDisposal
		row.BookValueOpening = row.CostOpening - row.DepreciationOpening
		row.BookValueClosing = row.CostClosing - row.DepreciationClosing
		spiegel.Rows = append(spiegel.Rows, *row)

		total, ok := classTotals[row.Class]
		if !ok {
			total = &domain.AnlagenspiegelRow{
				Class: row.Class, AccountName: row.Class.Label(),
			}
			classTotals[row.Class] = total
		}
		addRow(total, row)
		addRow(&spiegel.Totals, row)
	}

	sort.Slice(spiegel.Rows, func(i, j int) bool {
		return spiegel.Rows[i].Account < spiegel.Rows[j].Account
	})
	spiegel.Totals.AccountName = "Anlagevermögen gesamt"
	for _, class := range []domain.AssetClass{
		domain.AssetClassIntangible, domain.AssetClassTangible, domain.AssetClassFinancial,
	} {
		if total, ok := classTotals[class]; ok {
			spiegel.ClassTotals = append(spiegel.ClassTotals, *total)
		}
	}
	return spiegel, nil
}

func addRow(into *domain.AnlagenspiegelRow, from *domain.AnlagenspiegelRow) {
	into.AssetCount += from.AssetCount
	into.CostOpening += from.CostOpening
	into.Additions += from.Additions
	into.Disposals += from.Disposals
	into.CostClosing += from.CostClosing
	into.DepreciationOpening += from.DepreciationOpening
	into.DepreciationYear += from.DepreciationYear
	into.WriteUpsYear += from.WriteUpsYear
	into.DepreciationDisposal += from.DepreciationDisposal
	into.DepreciationClosing += from.DepreciationClosing
	into.BookValueOpening += from.BookValueOpening
	into.BookValueClosing += from.BookValueClosing
}

// -------------------------------------------------------------------------
// Zugangsbuchungen ohne Anlagegut
// -------------------------------------------------------------------------

// AcquisitionCandidates lists bookings on Anlagekonten that no Anlagegut points
// at yet.
func (s *AssetService) AcquisitionCandidates(ctx context.Context) ([]AcquisitionCandidate, error) {
	entries, err := s.journalRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("das Journal konnte nicht gelesen werden: %w", err)
	}
	linked, err := s.assetRepo.LinkedEntryIDs(ctx)
	if err != nil {
		return nil, err
	}
	chart, _ := s.journalSvc.Chart(ctx)

	var out []AcquisitionCandidate
	for i := range entries {
		entry := &entries[i]
		if linked[entry.ID] || entry.Kind == domain.EntryKindReversal {
			continue
		}
		// Eine AfA-Buchung steht selbst auf einem Anlagekonto — sie ist aber kein
		// Zugang, sondern die Folge eines bereits erfassten.
		if entry.Source == domain.EntrySourceDepreciation {
			continue
		}
		for _, line := range entry.Lines {
			if line.Side != domain.SideDebit || !isFixedAssetAccount(line.Account) {
				continue
			}
			name := line.Account
			if chart != nil {
				name = chart.Name(line.Account)
			}
			out = append(out, AcquisitionCandidate{
				EntryID:     entry.ID,
				EntryNumber: entry.EntryNumber,
				BookingDate: entry.BookingDate,
				Description: entry.Description,
				Account:     line.Account,
				AccountName: name,
				Amount:      line.Amount,
				ContactID:   entry.ContactID,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BookingDate > out[j].BookingDate })
	return out, nil
}

// isFixedAssetAccount reports whether an account number belongs to
// Kontenklasse 0, the Anlagevermögen of the SKR04.
func isFixedAssetAccount(account string) bool {
	if len(account) != 4 {
		return false
	}
	n, err := strconv.Atoi(account)
	if err != nil {
		return false
	}
	return n >= 100 && n < 1000
}

// -------------------------------------------------------------------------
// Interne Helfer
// -------------------------------------------------------------------------

// enrich fills the derived figures of an asset as of one fiscal year.
func (s *AssetService) enrich(asset *domain.FixedAsset, fiscalYear, startMonth int) {
	asset.Cost, asset.Accumulated, asset.YearAmount = 0, 0, 0
	for _, m := range asset.Movements {
		if m.FiscalYear > fiscalYear {
			continue
		}
		asset.Cost += m.CostAmount
		asset.Accumulated += m.DepreciationAmount
		if m.FiscalYear == fiscalYear && m.Kind == domain.AssetMovementDepreciation {
			asset.YearAmount += m.DepreciationAmount
		}
	}
	asset.BookValue = asset.Cost - asset.Accumulated

	asset.DueAmount = 0
	if asset.Method.IsPlanned() && !disposedBefore(asset, fiscalYear) {
		if rows, err := accounting.BuildAfASchedule(s.planFor(asset, startMonth)); err == nil {
			if due := accounting.ScheduleAmountFor(rows, fiscalYear) - asset.YearAmount; due > 0 {
				asset.DueAmount = due
			}
		}
	}

	switch {
	case asset.IsDisposed():
		asset.Status = domain.AssetStatusDisposed
		asset.StatusNote = fmt.Sprintf("Abgegangen am %s.", asset.DisposalDate)
	case asset.DueAmount > 0:
		asset.Status = domain.AssetStatusDepreciateDue
		asset.StatusNote = fmt.Sprintf("Für %d ist noch Abschreibung offen.", fiscalYear)
	case asset.AcquisitionEntryID == nil && !hasBookedMovement(asset):
		asset.Status = domain.AssetStatusNotYetBooked
		asset.StatusNote = "Erfasst, aber mit keiner Buchung verknüpft. Der Zugang wird über den Beleg gebucht."
	case asset.BookValue == 0 && asset.Cost > 0:
		asset.Status = domain.AssetStatusFullyDepr
		asset.StatusNote = "Vollständig abgeschrieben, aber weiter im Bestand."
	default:
		asset.Status = domain.AssetStatusActive
	}
}

// planFor turns an asset into the input of the AfA computation.
func (s *AssetService) planFor(asset *domain.FixedAsset, startMonth int) accounting.AfAPlan {
	plan := accounting.AfAPlan{
		AcquisitionDate:      asset.AcquisitionDate,
		UsefulLifeMonths:     asset.UsefulLifeMonths,
		Method:               asset.Method,
		FiscalYearStartMonth: startMonth,
		PoolYear:             asset.PoolYear,
		DisposalDate:         asset.DisposalDate,
	}
	for _, m := range asset.Movements {
		plan.Cost += m.CostAmount
		if m.Kind == domain.AssetMovementImpairment {
			if plan.ImpairmentsByYear == nil {
				plan.ImpairmentsByYear = map[int]domain.Cents{}
			}
			plan.ImpairmentsByYear[m.FiscalYear] += m.DepreciationAmount
		}
		if m.Kind == domain.AssetMovementDisposal {
			// Der Abgang nimmt die Anschaffungskosten aus den Büchern; für den Plan
			// bleiben sie die Bemessungsgrundlage.
			plan.Cost -= m.CostAmount
		}
	}
	if plan.Cost == 0 {
		plan.Cost = asset.AcquisitionCost
	}
	return plan
}

// syncImmediateWriteOff keeps the write-off of a GWG in step with its master
// data.
//
// The Sofortabzug is not booked by the Abschreibungslauf: der Aufwand entsteht
// über die Belegbuchung auf 6260. Die Kartei hält ihn trotzdem als Bewegung
// fest, sonst stünde das geringwertige Wirtschaftsgut jahrelang mit einem
// Buchwert im Verzeichnis, den es nicht mehr hat. Die Bewegung trägt bewusst
// keine Journalbuchung: sie gehört zur Zugangsbuchung, die schon an der
// Zugangsbewegung hängt.
func (s *AssetService) syncImmediateWriteOff(
	ctx context.Context, asset *domain.FixedAsset, existing []domain.AssetMovement,
) error {
	var current *domain.AssetMovement
	for i := range existing {
		m := &existing[i]
		if m.Kind == domain.AssetMovementDepreciation && m.JournalEntryID == nil &&
			m.Date == asset.AcquisitionDate {
			current = m
			break
		}
	}

	if asset.Method != domain.DepreciationImmediate {
		if current == nil {
			return nil
		}
		if err := s.assetRepo.DeleteMovement(ctx, current.ID); err != nil {
			return fmt.Errorf("der frühere Sofortabzug konnte nicht entfernt werden: %w", err)
		}
		return nil
	}

	writeOff := &domain.AssetMovement{
		AssetID:            asset.ID,
		Kind:               domain.AssetMovementDepreciation,
		Date:               asset.AcquisitionDate,
		FiscalYear:         domain.GetFiscalYearForDate(asset.AcquisitionDate, s.fiscalYearStartMonth(ctx)),
		DepreciationAmount: asset.AcquisitionCost,
		Note:               "Sofortabzug nach § 6 Abs. 2 EStG, gebucht über den Beleg",
	}
	if current != nil {
		writeOff.ID = current.ID
	}
	if err := s.assetRepo.AddMovement(ctx, writeOff); err != nil {
		return fmt.Errorf("der Sofortabzug konnte nicht festgehalten werden: %w", err)
	}
	return nil
}

// adjustAcquisitionMovement rewrites the Zugangsbewegung when the acquisition
// cost is corrected before anything was booked. Once a booking exists the change
// has to be a movement of its own — a silently rewritten Zugang would make the
// Anlagenspiegel of an earlier year change behind the user's back.
func (s *AssetService) adjustAcquisitionMovement(
	ctx context.Context, existing *domain.FixedAsset, newCost domain.Cents,
) error {
	for _, m := range existing.Movements {
		if m.Kind != domain.AssetMovementAcquisition && m.JournalEntryID != nil {
			return fmt.Errorf(
				"zu %s sind bereits Buchungen erfasst. Ändere die Anschaffungskosten über eine "+
					"nachträgliche Anschaffungskostenposition oder eine Minderung, nicht über das Stammdatenfeld",
				existing.InventoryNumber)
		}
	}
	for i := range existing.Movements {
		m := &existing.Movements[i]
		if m.Kind != domain.AssetMovementAcquisition {
			continue
		}
		m.CostAmount = newCost
		return s.assetRepo.AddMovement(ctx, m)
	}
	return nil
}

func (s *AssetService) validateAccounts(ctx context.Context, asset *domain.FixedAsset) error {
	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return err
	}
	if err := chart.EnsurePostable(asset.Account); err != nil {
		return fmt.Errorf("Anlagekonto: %w", err)
	}
	if account, ok := chart.Lookup(asset.Account); ok && account.Kontenklasse != 0 {
		return fmt.Errorf(
			"Konto %s (%s) liegt in Kontenklasse %d. Das Anlagevermögen steht im SKR04 in Klasse 0",
			asset.Account, account.Name, account.Kontenklasse)
	}
	if asset.DepreciationAccount != "" {
		if err := chart.EnsurePostable(asset.DepreciationAccount); err != nil {
			return fmt.Errorf("Abschreibungskonto: %w", err)
		}
	}
	return nil
}

func (s *AssetService) nextInventoryNumber(ctx context.Context, acquisitionDate string) (string, error) {
	year := domain.GetFiscalYearForDate(acquisitionDate, s.fiscalYearStartMonth(ctx))
	if s.numberRepo == nil {
		return "", fmt.Errorf("der Nummernkreis für Inventarnummern ist nicht eingerichtet")
	}
	// Jahresübergreifender Kreis: eine Inventarnummer wird nie wiederverwendet,
	// auch nicht nach einem Jahreswechsel.
	seq, err := s.numberRepo.Allocate(ctx, domain.NumberRangeAsset, 0)
	if err != nil {
		return "", fmt.Errorf("die Inventarnummer konnte nicht vergeben werden: %w", err)
	}
	return domain.FormatInventoryNumber(year, seq), nil
}

func (s *AssetService) reload(ctx context.Context, id uint) (*domain.FixedAsset, error) {
	asset, err := s.assetRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	startMonth := s.fiscalYearStartMonth(ctx)
	s.enrich(asset, s.fiscalYear, startMonth)
	if chart, err := s.journalSvc.Chart(ctx); err == nil {
		asset.AccountName = chart.Name(asset.Account)
	}
	return asset, nil
}

// named fills the account names into preview lines, so the user reads names and
// not only numbers.
func (s *AssetService) named(ctx context.Context, lines []domain.JournalLine) []domain.JournalLine {
	out := make([]domain.JournalLine, len(lines))
	copy(out, lines)
	chart, err := s.journalSvc.Chart(ctx)
	for i := range out {
		out[i].Position = i + 1
		if err == nil {
			out[i].AccountName = chart.Name(out[i].Account)
		}
	}
	return out
}

func (s *AssetService) fillEntryNumbers(ctx context.Context, movements []domain.AssetMovement) {
	for i := range movements {
		if movements[i].JournalEntryID == nil {
			continue
		}
		entry, err := s.journalRepo.FindByID(ctx, *movements[i].JournalEntryID)
		if err == nil && entry != nil {
			movements[i].EntryNumber = entry.EntryNumber
		}
	}
}

func (s *AssetService) fiscalYearStartMonth(ctx context.Context) int {
	if s.settingsRepo == nil {
		return 1
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil || cfg.FiscalYearStartMonth <= 0 {
		return 1
	}
	return cfg.FiscalYearStartMonth
}

func (s *AssetService) audit(ctx context.Context, action domain.AuditAction, id uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "ANLAGE", fmt.Sprintf("%d", id), details)
}

// bookedByYear sums the planmäßige AfA already booked per fiscal year. Only the
// planmäßige counts here: eine außerplanmäßige Abschreibung erfüllt den Plan
// nicht, sie mindert nur den Wert, von dem er weiterrechnet.
func bookedByYear(movements []domain.AssetMovement) map[int]domain.Cents {
	booked := map[int]domain.Cents{}
	for _, m := range movements {
		if m.Kind == domain.AssetMovementDepreciation {
			booked[m.FiscalYear] += m.DepreciationAmount
		}
	}
	return booked
}

func hasBookedMovement(asset *domain.FixedAsset) bool {
	for _, m := range asset.Movements {
		if m.JournalEntryID != nil {
			return true
		}
	}
	return false
}

func disposedBefore(asset *domain.FixedAsset, fiscalYear int) bool {
	if !asset.IsDisposed() {
		return false
	}
	for _, m := range asset.Movements {
		if m.Kind == domain.AssetMovementDisposal {
			return m.FiscalYear < fiscalYear
		}
	}
	return false
}

// fiscalYearEndDate is the Bilanzstichtag of a fiscal year — the day the AfA is
// booked on.
func fiscalYearEndDate(fiscalYear, startMonth int) string {
	if startMonth <= 1 || startMonth > 12 {
		return fmt.Sprintf("%d-12-31", fiscalYear)
	}
	start := time.Date(fiscalYear, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
	return start.AddDate(1, 0, -1).Format("2006-01-02")
}

func permilleLabel(permille int) string {
	whole := permille / 10
	frac := permille % 10
	if frac == 0 {
		return strconv.Itoa(whole)
	}
	return fmt.Sprintf("%d,%d", whole, frac)
}
