package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// InputTaxService führt das Verzeichnis der Vorsteuerberichtigung (§ 15a UStG).
//
// Der Vorsteuerabzug eines Wirtschaftsguts wird im Jahr der Anschaffung nach der
// beabsichtigten Verwendung gewährt. Ändert sich die Verwendung innerhalb des
// Berichtigungszeitraums — fünf Jahre bei beweglichen Wirtschaftsgütern, zehn
// bei Grundstücken —, ist der Abzug für jedes betroffene Jahr anteilig zu
// berichtigen.
//
// Das Verzeichnis ist keine Auswertung, sondern eine Kartei. Aus dem Journal
// allein ist es nicht zu gewinnen: dort steht der gezogene Betrag, aber nicht
// der Anteil, zu dem er gezogen wurde, und schon gar nicht der Anteil des
// dritten Jahres. § 22 Abs. 4 UStG verlangt genau diese Aufzeichnungen.
type InputTaxService struct {
	repo         domain.InputTaxCorrectionRepository
	journalSvc   *JournalService
	journalRepo  domain.JournalRepository
	settingsRepo domain.SettingsRepository
	closingSvc   *ClosingService
	auditRepo    domain.AuditRepository
	fiscalYear   int
	// txRunner klammert die Buchung der Berichtigung und den Vermerk am
	// Verzeichnis. Fehlt er, laufen beide nacheinander — die Suche nach einer
	// stehenden Buchung darunter fängt den halben Zustand auch dann ab, sie kann
	// ihn nur nicht mehr zurückrollen.
	txRunner domain.TxRunner
}

// SetTxRunner koppelt die Transaktionsklammer an die Jahresbuchung.
func (s *InputTaxService) SetTxRunner(r domain.TxRunner) { s.txRunner = r }

// runInTx führt fn in einer Transaktion aus, wo eine eingerichtet ist.
func (s *InputTaxService) runInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.txRunner == nil {
		return fn(ctx)
	}
	return s.txRunner.RunInTx(ctx, fn)
}

// NewInputTaxService wires das Verzeichnis.
func NewInputTaxService(
	repo domain.InputTaxCorrectionRepository,
	journalSvc *JournalService,
	journalRepo domain.JournalRepository,
	settingsRepo domain.SettingsRepository,
	closingSvc *ClosingService,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *InputTaxService {
	return &InputTaxService{
		repo: repo, journalSvc: journalSvc, journalRepo: journalRepo,
		settingsRepo: settingsRepo, closingSvc: closingSvc, auditRepo: auditRepo,
		fiscalYear: fiscalYear,
	}
}

// SetFiscalYear schaltet das aktive Geschäftsjahr um.
func (s *InputTaxService) SetFiscalYear(year int) { s.fiscalYear = year }

// -------------------------------------------------------------------------
// Aufnahme ins Verzeichnis
// -------------------------------------------------------------------------

// RegisterRequest nimmt ein Wirtschaftsgut ins Verzeichnis auf.
type RegisterInputTaxRequest struct {
	AssetID         uint         `json:"assetId,omitempty"`
	ReceiptID       uint         `json:"receiptId,omitempty"`
	EntryID         uint         `json:"entryId,omitempty"`
	Label           string       `json:"label"`
	Account         string       `json:"account,omitempty"`
	AcquisitionDate string       `json:"acquisitionDate"`
	NetAmount       domain.Cents `json:"netAmount"`
	InputTaxAmount  domain.Cents `json:"inputTaxAmount"`
	// OriginalPermille ist der Anteil, mit dem die Vorsteuer gezogen wurde.
	//
	// Ein Zeiger, weil hier drei Zustände auseinanderzuhalten sind und nicht
	// zwei: keine Angabe (dann gilt volle Verwendung, der Regelfall), ein Anteil
	// dazwischen — und null. Null ist die Anschaffung ohne Vorsteuerabzug, und
	// aus ihr wird der häufigste Fall der Berichtigung nach oben: das
	// Wirtschaftsgut wird später für abzugsberechtigende Umsätze verwendet
	// (§ 15a Abs. 1 UStG). Als Zahl mit `omitempty` war dieser Fall nicht
	// ausdrückbar — er wurde still zu voller Verwendung.
	OriginalPermille *int   `json:"originalPermille,omitempty"`
	Immovable        bool   `json:"immovable,omitempty"`
	Note             string `json:"note,omitempty"`
}

// Register nimmt ein Wirtschaftsgut ins Verzeichnis auf oder schreibt seinen
// Eintrag fort.
//
// Die Bagatellgrenze des § 44 Abs. 1 UStDV wird hier nicht angewandt: sie
// entscheidet über die *Berichtigung*, nicht über die Aufzeichnung. Ein
// Wirtschaftsgut mit 900 € Vorsteuer wird nie berichtigt, aber wer das später
// nachvollziehen will, muss den Betrag im Verzeichnis finden — sonst steht dort
// eine Lücke, die wie ein Versäumnis aussieht.
func (s *InputTaxService) Register(
	ctx context.Context, req RegisterInputTaxRequest,
) (*domain.InputTaxCorrection, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("das Verzeichnis nach § 15a UStG ist nicht eingerichtet")
	}
	permille := 1000
	if req.OriginalPermille != nil {
		permille = *req.OriginalPermille
	}
	immovable := req.Immovable
	if entry, ok := accounting.LookupAssetAccount(req.Account); ok && entry.Immovable {
		immovable = true
	}
	startMonth := s.fiscalYearStartMonth(ctx)
	first := domain.GetFiscalYearForDate(req.AcquisitionDate, startMonth)
	period := accounting.CorrectionPeriodYears(immovable)
	// Das Ende des Zeitraums wird datumsgenau gerechnet und nicht als
	// „Zugangsjahr plus Zeitraum minus eins": ein am 20.12. angeschafftes
	// Wirtschaftsgut läuft bis in den Dezember des sechsten Kalenderjahres, und
	// eine Änderung der Verwendung in diesem letzten Jahr ist zu berichtigen.
	periodEnd, err := domain.CorrectionPeriodEndDate(req.AcquisitionDate, period)
	if err != nil {
		return nil, err
	}

	correction := &domain.InputTaxCorrection{
		Label:                 strings.TrimSpace(req.Label),
		Account:               req.Account,
		AcquisitionDate:       req.AcquisitionDate,
		NetAmount:             req.NetAmount,
		InputTaxAmount:        req.InputTaxAmount,
		OriginalPermille:      permille,
		Immovable:             immovable,
		CorrectionPeriodYears: period,
		PeriodEnd:             periodEnd,
		FirstFiscalYear:       first,
		LastFiscalYear:        domain.GetFiscalYearForDate(periodEnd, startMonth),
		Note:                  req.Note,
	}
	if req.AssetID != 0 {
		id := req.AssetID
		correction.AssetID = &id
		existing, err := s.repo.FindByAsset(ctx, id)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			correction.ID = existing.ID
			correction.CreatedAt = existing.CreatedAt
		}
	}
	if req.ReceiptID != 0 {
		id := req.ReceiptID
		correction.ReceiptID = &id
	}
	if req.EntryID != 0 {
		id := req.EntryID
		correction.EntryID = &id
	}
	if err := correction.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, correction); err != nil {
		return nil, fmt.Errorf("der Eintrag ließ sich nicht speichern: %w", err)
	}
	s.audit(ctx, correction.ID, fmt.Sprintf(
		"Ins Verzeichnis nach § 15a UStG aufgenommen: %s, Vorsteuer %s €, Anteil %s, Zeitraum "+
			"%d Jahre bis zum %s",
		correction.Label, correction.InputTaxAmount,
		accounting.PermilleLabel(int64(correction.OriginalPermille)), period,
		germanDay(correction.PeriodEnd)))
	correction.EnsureLists()
	return correction, nil
}

// Close schließt einen Eintrag vorzeitig ab — Abgang, Entnahme oder eine
// Aufnahme, die nicht hätte sein sollen.
func (s *InputTaxService) Close(ctx context.Context, id uint, reason, date string) (*domain.InputTaxCorrection, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("zum vorzeitigen Abschluss eines Eintrags gehört sein Grund")
	}
	correction, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("der Eintrag %d wurde nicht gefunden: %w", id, err)
	}
	correction.ClosedReason = strings.TrimSpace(reason)
	correction.ClosedOn = date
	if err := s.repo.Save(ctx, correction); err != nil {
		return nil, err
	}
	s.audit(ctx, id, "Eintrag abgeschlossen: "+reason)
	correction.EnsureLists()
	return correction, nil
}

// -------------------------------------------------------------------------
// Der Jahreslauf
// -------------------------------------------------------------------------

// InputTaxCorrectionRow ist ein Wirtschaftsgut im Jahreslauf.
type InputTaxCorrectionRow struct {
	Correction domain.InputTaxCorrection `json:"correction"`
	// InPeriod sagt, ob das Jahr in den Berichtigungszeitraum fällt.
	InPeriod bool `json:"inPeriod"`
	// Permille ist der Verwendungsanteil, mit dem gerechnet wurde: der bestätigte
	// des Jahres, sonst der ursprüngliche als Vorschlag.
	Permille  int  `json:"permille"`
	Confirmed bool `json:"confirmed"`
	// MonthsInYear ist die Zahl der Monate des Berichtigungszeitraums, die in
	// dieses Geschäftsjahr fallen — zwölf im Regelfall, weniger im Anfangs- und
	// im Schlussjahr eines mitten im Jahr begonnenen Zeitraums.
	MonthsInYear int `json:"monthsInYear"`
	// Assessment ist die Bewertung des Jahres.
	Assessment accounting.InputTaxCorrectionAssessment `json:"assessment"`
	// Booked sagt, ob die Berichtigung dieses Jahres bereits gebucht ist.
	Booked      bool   `json:"booked"`
	EntryNumber string `json:"entryNumber,omitempty"`
}

// InputTaxCorrectionYear ist das Verzeichnis mit Blick auf ein Geschäftsjahr.
type InputTaxCorrectionYear struct {
	FiscalYear  int                     `json:"fiscalYear"`
	BookingDate string                  `json:"bookingDate"`
	Rows        []InputTaxCorrectionRow `json:"rows"`
	// TotalAmount ist die Summe der zu buchenden Berichtigungen mit Vorzeichen.
	TotalAmount domain.Cents `json:"totalAmount"`
	// Unconfirmed zählt die Wirtschaftsgüter, deren Verwendungsanteil noch
	// niemand bestätigt hat. Solange einer offen ist, wird nicht gebucht.
	Unconfirmed int    `json:"unconfirmed"`
	Note        string `json:"note"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (y *InputTaxCorrectionYear) EnsureLists() {
	if y.Rows == nil {
		y.Rows = make([]InputTaxCorrectionRow, 0)
	}
}

// Year stellt das Verzeichnis für ein Geschäftsjahr zusammen.
func (s *InputTaxService) Year(ctx context.Context, year int) (*InputTaxCorrectionYear, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	out := &InputTaxCorrectionYear{FiscalYear: year}
	out.EnsureLists()
	if s.repo == nil {
		return out, nil
	}

	bookingDate, err := s.bookingDate(ctx, year)
	if err != nil {
		return nil, err
	}
	out.BookingDate = bookingDate
	params, err := accounting.TaxParametersFor(bookingDate)
	if err != nil {
		return nil, err
	}

	windowFrom, windowTo := s.fiscalYearWindow(ctx, year)

	corrections, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	entryNumbers, err := s.entryNumbers(ctx, year)
	if err != nil {
		return nil, err
	}
	reversals := newReversalIndex(s.journalRepo)

	for i := range corrections {
		c := &corrections[i]
		c.EnsureLists()
		months := c.MonthsInWindow(windowFrom, windowTo)
		row := InputTaxCorrectionRow{
			Correction: *c,
			// Ein Jahr ohne einen Monat des Zeitraums ist kein Berichtigungsjahr,
			// auch wenn es zwischen dem ersten und dem letzten Geschäftsjahr liegt
			// — bei einem Rumpfgeschäftsjahr kann das auseinanderfallen.
			InPeriod:     c.Open() && c.CoversYear(year) && months > 0,
			MonthsInYear: months,
			Permille:     c.OriginalPermille,
		}
		if usage, ok := c.UsageFor(year); ok {
			row.Permille = usage.Permille
			row.Confirmed = usage.Confirmed
			booked, err := s.usageStands(ctx, reversals, usage)
			if err != nil {
				return nil, err
			}
			row.Booked = booked
			if usage.EntryID != nil && booked {
				row.EntryNumber = entryNumbers[*usage.EntryID]
			}
		}
		if !row.InPeriod {
			row.Assessment = accounting.InputTaxCorrectionAssessment{Reason: s.outOfPeriodReason(c, year)}
			out.Rows = append(out.Rows, row)
			continue
		}
		assessment, err := accounting.AssessInputTaxCorrection(accounting.InputTaxCorrectionRequest{
			InputTaxAmount:   c.InputTaxAmount,
			OriginalPermille: int64(c.OriginalPermille),
			CurrentPermille:  int64(row.Permille),
			PeriodYears:      c.CorrectionPeriodYears,
			MonthsInYear:     months,
			Immovable:        c.Immovable,
		}, params)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.Label, err)
		}
		row.Assessment = assessment
		if assessment.Required && !row.Booked {
			out.TotalAmount += assessment.Amount
		}
		if !row.Confirmed {
			out.Unconfirmed++
		}
		out.Rows = append(out.Rows, row)
	}

	sort.SliceStable(out.Rows, func(i, j int) bool {
		if out.Rows[i].InPeriod != out.Rows[j].InPeriod {
			return out.Rows[i].InPeriod
		}
		return out.Rows[i].Correction.AcquisitionDate < out.Rows[j].Correction.AcquisitionDate
	})
	out.Note = fmt.Sprintf(
		"Bestätige für jedes Wirtschaftsgut den Verwendungsanteil des Jahres %d oder ändere ihn. "+
			"Berichtigt wird die Differenz zum ursprünglichen Anteil, verteilt auf den "+
			"Berichtigungszeitraum. Gebucht wird zum %s mit dem Steuerschlüssel %s; er läuft in die "+
			"Kennziffer %s der Voranmeldung. Der Zeitraum läuft ab der erstmaligen Verwendung "+
			"(§ 15a Abs. 1 UStG) und endet mit dem Kalendermonat nach § 45 UStDV; im Anfangs- und im "+
			"Schlussjahr geht deshalb nur der Teil des Jahres ein, der in den Zeitraum fällt.",
		year, germanDay(bookingDate), accounting.TaxKeyInputTaxCorrection,
		accounting.VatCodeInputTaxCorrection)
	return out, nil
}

// usageStands meldet, ob die Berichtigung eines Jahres gebucht ist *und* die
// Buchung noch steht.
//
// Am Vermerk allein lässt sich das nicht ablesen. BookYear schreibt die
// Buchungskennung an den Verwendungsanteil, und nichts nimmt sie je wieder
// heraus — der Weg zurück ist die Generalumkehr, und die berührt die Kartei
// nicht. Ohne diese Frage wäre der Rat „nimm die Buchung zurück" ein Rat in die
// Sackgasse: nach dem Storno meldete das Verzeichnis weiter „gebucht", der
// Jahreslauf fände nichts zu tun, und die Berichtigung dieses Jahres käme nie
// mehr in die Kennziffer 64.
func (s *InputTaxService) usageStands(
	ctx context.Context, index *reversalIndex, usage domain.InputTaxUsage,
) (bool, error) {
	if !usage.Booked() {
		return false, nil
	}
	voided, err := index.reversed(ctx, usage.EntryID)
	if err != nil {
		return false, err
	}
	return !voided, nil
}

// outOfPeriodReason sagt in einem Satz, warum ein Eintrag in diesem Jahr nicht
// zu berichtigen ist.
func (s *InputTaxService) outOfPeriodReason(c *domain.InputTaxCorrection, year int) string {
	switch {
	case !c.Open():
		return "Der Eintrag ist abgeschlossen: " + c.ClosedReason
	case year < c.FirstFiscalYear:
		return fmt.Sprintf(
			"Der Berichtigungszeitraum beginnt erst mit der erstmaligen Verwendung am %s.",
			germanDay(c.AcquisitionDate))
	default:
		return fmt.Sprintf(
			"Der Berichtigungszeitraum lief vom %s bis zum %s und ist abgelaufen.",
			germanDay(c.AcquisitionDate), germanDay(c.PeriodEndDate()))
	}
}

// SaveUsageRequest bestätigt oder ändert den Verwendungsanteil eines Jahres.
type SaveInputTaxUsageRequest struct {
	CorrectionID uint `json:"correctionId"`
	FiscalYear   int  `json:"fiscalYear"`
	// Permille ist der Verwendungsanteil des Jahres.
	Permille int    `json:"permille"`
	Reason   string `json:"reason,omitempty"`
}

// SaveUsage hält den Verwendungsanteil eines Jahres fest.
func (s *InputTaxService) SaveUsage(
	ctx context.Context, req SaveInputTaxUsageRequest,
) (*InputTaxCorrectionYear, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("das Verzeichnis nach § 15a UStG ist nicht eingerichtet")
	}
	correction, err := s.repo.FindByID(ctx, req.CorrectionID)
	if err != nil {
		return nil, fmt.Errorf("der Eintrag %d wurde nicht gefunden: %w", req.CorrectionID, err)
	}
	year := req.FiscalYear
	if year == 0 {
		year = s.fiscalYear
	}
	if !correction.CoversYear(year) {
		return nil, fmt.Errorf(
			"das Jahr %d liegt nicht im Berichtigungszeitraum von %s (%s bis %s)",
			year, correction.Label, germanDay(correction.AcquisitionDate),
			germanDay(correction.PeriodEndDate()))
	}
	if req.Permille < 0 || req.Permille > 1000 {
		return nil, fmt.Errorf("der Verwendungsanteil liegt zwischen 0 und 100 %%")
	}
	if usage, ok := correction.UsageFor(year); ok {
		// Gefragt wird nicht nach dem Vermerk, sondern nach dem Journal: eine per
		// Generalumkehr zurückgenommene Berichtigung ist keine gebuchte, und wer
		// dem Rat der Meldung gefolgt ist, muss den Anteil danach ändern können.
		stands, err := s.usageStands(ctx, newReversalIndex(s.journalRepo), usage)
		if err != nil {
			return nil, err
		}
		if stands {
			return nil, fmt.Errorf(
				"die Berichtigung des Jahres %d ist für %s bereits gebucht. Nimm die Buchung zurück, "+
					"bevor du den Anteil änderst", year, correction.Label)
		}
	}

	bookingDate, err := s.bookingDate(ctx, year)
	if err != nil {
		return nil, err
	}
	params, err := accounting.TaxParametersFor(bookingDate)
	if err != nil {
		return nil, err
	}
	windowFrom, windowTo := s.fiscalYearWindow(ctx, year)
	assessment, err := accounting.AssessInputTaxCorrection(accounting.InputTaxCorrectionRequest{
		InputTaxAmount:   correction.InputTaxAmount,
		OriginalPermille: int64(correction.OriginalPermille),
		CurrentPermille:  int64(req.Permille),
		PeriodYears:      correction.CorrectionPeriodYears,
		MonthsInYear:     correction.MonthsInWindow(windowFrom, windowTo),
		Immovable:        correction.Immovable,
	}, params)
	if err != nil {
		return nil, err
	}

	reason := assessment.Reason
	if strings.TrimSpace(req.Reason) != "" {
		reason = strings.TrimSpace(req.Reason) + " — " + reason
	}
	if err := s.repo.SaveUsage(ctx, &domain.InputTaxUsage{
		CorrectionID: correction.ID, FiscalYear: year,
		Permille: req.Permille, Confirmed: true,
		Amount: assessment.Amount, Reason: reason,
	}); err != nil {
		return nil, fmt.Errorf("der Verwendungsanteil ließ sich nicht speichern: %w", err)
	}
	s.audit(ctx, correction.ID, fmt.Sprintf(
		"Verwendungsanteil %d für %s bestätigt: %s (%s)",
		year, correction.Label, accounting.PermilleLabel(int64(req.Permille)), assessment.Reason))
	return s.Year(ctx, year)
}

// BookYear bucht die Berichtigungen eines Geschäftsjahres.
//
// In einer Buchung und nicht in einer je Wirtschaftsgut: die Berichtigung ist
// ein Vorgang des Jahresabschlusses, sie steht in einer Kennziffer der
// Voranmeldung, und n Buchungen über je zwölf Euro machen das Journal nicht
// klarer. Welches Wirtschaftsgut welchen Anteil beisteuert, steht im
// Verzeichnis — dafür gibt es das Verzeichnis.
func (s *InputTaxService) BookYear(ctx context.Context, year int) (*InputTaxCorrectionYear, error) {
	if s.journalSvc == nil {
		return nil, fmt.Errorf("der Buchungsweg ist nicht eingerichtet")
	}
	if year == 0 {
		year = s.fiscalYear
	}
	view, err := s.Year(ctx, year)
	if err != nil {
		return nil, err
	}
	if view.Unconfirmed > 0 {
		return nil, fmt.Errorf(
			"für %d Wirtschaftsgut(er) ist der Verwendungsanteil des Jahres %d noch nicht bestätigt. "+
				"Die Berichtigung ist eine Aussage über die tatsächliche Verwendung; sie ungeprüft zu "+
				"buchen hieße, sie zu erfinden", view.Unconfirmed, year)
	}

	type pending struct {
		row        InputTaxCorrectionRow
		assessment accounting.InputTaxCorrectionAssessment
	}
	todo := make([]pending, 0, len(view.Rows))
	for _, row := range view.Rows {
		if row.InPeriod && row.Assessment.Required && !row.Booked {
			todo = append(todo, pending{row: row, assessment: row.Assessment})
		}
	}
	if len(todo) == 0 {
		view.Note = "Für dieses Geschäftsjahr ist nichts zu berichtigen. " + view.Note
		return view, nil
	}

	lines := make([]domain.JournalLine, 0, len(todo)*2)
	var total domain.Cents
	for _, p := range todo {
		amount := p.assessment.Amount
		magnitude := amount.Abs()
		total += amount
		if amount < 0 {
			// Zurückzuzahlende Vorsteuer: sie mindert den Abzug und ist Aufwand.
			lines = append(lines,
				domain.JournalLine{Side: domain.SideDebit, Account: nonDeductibleInputTaxExpense,
					Amount: magnitude, Text: p.row.Correction.Label},
				domain.JournalLine{Side: domain.SideCredit, Account: p.assessment.Account,
					Amount: magnitude, TaxKey: accounting.TaxKeyInputTaxCorrection,
					Text: p.row.Correction.Label})
			continue
		}
		lines = append(lines,
			domain.JournalLine{Side: domain.SideDebit, Account: p.assessment.Account,
				Amount: magnitude, TaxKey: accounting.TaxKeyInputTaxCorrection,
				Text: p.row.Correction.Label},
			domain.JournalLine{Side: domain.SideCredit, Account: inputTaxCorrectionIncome,
				Amount: magnitude, Text: p.row.Correction.Label})
	}

	// Zwei Schutzschichten um denselben Fehler: die Berichtigung eines Jahres
	// darf nicht zweimal in Kennziffer 64 landen.
	//
	// Die erste ist die Suche nach einer stehenden Buchung unter der Belegnummer
	// des Jahres — sie hält auch dann, wenn der Vermerk am Verzeichnis fehlt,
	// weil ein früherer Lauf zwischen Buchung und Vermerk abgebrochen ist. Die
	// zweite ist die Transaktion darum: dann kommt es gar nicht erst dazu.
	document := inputTaxCorrectionDocument(year)
	if standing, err := s.standingCorrectionEntry(ctx, year, document); err != nil {
		return nil, err
	} else if standing != nil {
		return nil, fmt.Errorf(
			"die Berichtigung des Vorsteuerabzugs %d ist mit der Buchung %s bereits gebucht. Nimm "+
				"sie zurück, bevor du sie neu rechnest — zweimal gebucht stünde sie doppelt in "+
				"Kennziffer 64", year, standing.EntryNumber)
	}

	var entry *domain.JournalEntry
	err = s.runInTx(ctx, func(ctx context.Context) error {
		posted, err := s.journalSvc.Post(ctx, &domain.JournalEntry{
			BookingDate: view.BookingDate, DocumentDate: view.BookingDate,
			ServiceDateFrom: view.BookingDate, ServiceDateTo: view.BookingDate,
			Description: fmt.Sprintf(
				"Berichtigung des Vorsteuerabzugs %d nach § 15a UStG (%d Wirtschaftsgüter)",
				year, len(todo)),
			Source:             domain.EntrySourceClosing,
			DocumentNumber:     document,
			TaxTreatment:       domain.TaxTreatmentDomestic,
			PostingRuleVersion: accounting.PostingRuleVersion,
			Lines:              lines,
		})
		if err != nil {
			return err
		}
		entry = posted
		for _, p := range todo {
			usage := domain.InputTaxUsage{
				CorrectionID: p.row.Correction.ID, FiscalYear: year,
				Permille: p.row.Permille, Confirmed: true,
				Amount: p.assessment.Amount, Reason: p.assessment.Reason,
				EntryID: &posted.ID, BookedOn: view.BookingDate,
			}
			if err := s.repo.SaveUsage(ctx, &usage); err != nil {
				return fmt.Errorf(
					"der Vermerk am Eintrag %s ließ sich nicht schreiben; die Berichtigung wird "+
						"deshalb nicht gebucht: %w", p.row.Correction.Label, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.audit(ctx, 0, fmt.Sprintf(
		"Vorsteuerberichtigung %d gebucht: %s € über %d Wirtschaftsgüter (%s)",
		year, total, len(todo), entry.EntryNumber))
	return s.Year(ctx, year)
}

// Die beiden Gegenkonten der Berichtigung.
//
// Die zurückzuzahlende Vorsteuer ist Aufwand des Jahres, in dem sich die
// Verwendung geändert hat, die nachträglich abziehbare ein Ertrag. Der SKR04
// hält dafür keine eigenen Konten bereit — die sonstigen betrieblichen
// Aufwendungen und Erträge sind die richtige Stelle, und der Steuerschlüssel an
// der Gegenzeile sagt, woraus der Betrag stammt.
const (
	nonDeductibleInputTaxExpense = "6300"
	inputTaxCorrectionIncome     = "4830"
)

// bookingDate ist der Tag, zu dem die Berichtigung gebucht wird: das Ende des
// Geschäftsjahres.
//
// § 44 Abs. 3 UStDV lässt die Berichtigung bis 6.000 € erst bei der
// Steuerberechnung für das Kalenderjahr zu, und über 6.000 € wäre sie im
// Voranmeldungszeitraum der Änderung vorzunehmen. Buchfink bucht beides zum
// Jahresende: die Änderung der Verwendung wird ohnehin erst im Jahreslauf
// festgestellt, und ein rückdatierter Voranmeldungszeitraum wäre eine
// Berichtigung einer bereits abgegebenen Anmeldung.
func (s *InputTaxService) bookingDate(ctx context.Context, year int) (string, error) {
	_, end := s.fiscalYearWindow(ctx, year)
	return end, nil
}

// fiscalYearWindow liefert den ersten und den letzten Tag eines
// Geschäftsjahres.
//
// Das Fenster und nicht bloß sein Ende: der Anteil eines Jahres am
// Berichtigungszeitraum bemisst sich nach den Monaten, die in beides fallen,
// und ein Rumpfgeschäftsjahr hat weniger als zwölf davon.
func (s *InputTaxService) fiscalYearWindow(ctx context.Context, year int) (string, string) {
	if s.closingSvc != nil {
		if fy, err := s.closingSvc.PeriodOf(ctx, year); err == nil && fy != nil {
			return fy.StartDate, fy.EndDate
		}
	}
	month := s.fiscalYearStartMonth(ctx)
	// Der letzte Tag des Wirtschaftsjahres und nicht der erste des folgenden:
	// ein Wirtschaftsjahr, das im Juli beginnt, endet am 30. Juni — und eine
	// Berichtigung, die auf den 1. Juli datiert wäre, fiele in das nächste Jahr
	// und in dessen Voranmeldung.
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0).AddDate(0, 0, -1)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

// entryNumbers löst die Buchungskennungen eines Jahres in ihre Nummern auf.
func (s *InputTaxService) entryNumbers(ctx context.Context, year int) (map[uint]string, error) {
	out := map[uint]string{}
	if s.journalRepo == nil {
		return out, nil
	}
	entries, err := s.journalRepo.FindAll(ctx, year)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		out[entries[i].ID] = entries[i].EntryNumber
	}
	return out, nil
}

func (s *InputTaxService) fiscalYearStartMonth(ctx context.Context) int {
	if s.settingsRepo == nil {
		return 1
	}
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || settings == nil {
		return 1
	}
	if settings.FiscalYearStartMonth < 1 || settings.FiscalYearStartMonth > 12 {
		return 1
	}
	return settings.FiscalYearStartMonth
}

func (s *InputTaxService) audit(ctx context.Context, id uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "INPUT_TAX_CORRECTION",
		fmt.Sprintf("%d", id), details)
}

// standingCorrectionEntry sucht die Buchung der Berichtigung eines Jahres, die
// noch steht — eine von einer Generalumkehr zurückgenommene zählt nicht.
func (s *InputTaxService) standingCorrectionEntry(
	ctx context.Context, year int, document string,
) (*domain.JournalEntry, error) {
	if s.journalRepo == nil {
		return nil, nil
	}
	entries, err := s.journalRepo.FindAll(ctx, year)
	if err != nil {
		return nil, err
	}
	// Gefragt wird je Buchung und nicht über eine Liste der Generalumkehren des
	// Jahres: die Umkehr trägt den Tag ihrer Erstellung und liegt deshalb
	// regelmäßig in einem späteren Geschäftsjahr als die Buchung, die sie
	// zurücknimmt.
	index := newReversalIndex(s.journalRepo)
	for i := range entries {
		e := &entries[i]
		if e.DocumentNumber != document || e.Kind == domain.EntryKindReversal {
			continue
		}
		voided, err := index.reversed(ctx, &e.ID)
		if err != nil {
			return nil, err
		}
		if voided {
			continue
		}
		return e, nil
	}
	return nil, nil
}

// inputTaxCorrectionDocument ist die Belegnummer, unter der die Berichtigung
// eines Jahres im Journal steht. Der Abschlussbaustein findet sie darüber.
func inputTaxCorrectionDocument(fiscalYear int) string {
	return fmt.Sprintf("15A-%d", fiscalYear)
}
