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

// AccrualService ist die Rechnungsabgrenzung (§ 250 HGB).
//
// Der Gedanke dahinter ist der Grundsatz der periodengerechten Abgrenzung
// (§ 252 Abs. 1 Nr. 5 HGB): Aufwendungen und Erträge gehören in das Jahr, in
// dem sie wirtschaftlich verursacht wurden, und nicht in das, in dem gezahlt
// wurde. Die Versicherungsprämie, die im Dezember für zwölf Monate abgeht, ist
// zu elf Zwölfteln Aufwand des nächsten Jahres.
//
// Wie überall schreibt der Dienst nicht selbst ins Journal: jede Buchung geht
// durch den JournalService und trägt damit Nummer, Hash und Periodenprüfung wie
// jede andere.
type AccrualService struct {
	accrualRepo  domain.AccrualRepository
	journalRepo  domain.JournalRepository
	journalSvc   *JournalService
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
	closingSvc   *ClosingService
	receipts     closingReceiptFiler
	fiscalYear   int
}

// NewAccrualService wires die Rechnungsabgrenzung.
func NewAccrualService(
	accrualRepo domain.AccrualRepository,
	journalRepo domain.JournalRepository,
	journalSvc *JournalService,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	closingSvc *ClosingService,
	fiscalYear int,
) *AccrualService {
	return &AccrualService{
		accrualRepo:  accrualRepo,
		journalRepo:  journalRepo,
		journalSvc:   journalSvc,
		settingsRepo: settingsRepo,
		auditRepo:    auditRepo,
		closingSvc:   closingSvc,
		fiscalYear:   fiscalYear,
	}
}

// SetFiscalYear schaltet das aktive Geschäftsjahr um.
func (s *AccrualService) SetFiscalYear(year int) { s.fiscalYear = year }

// SetReceiptService gibt dem Dienst den Belegspeicher für die Eigenbelege.
func (s *AccrualService) SetReceiptService(r closingReceiptFiler) { s.receipts = r }

// -------------------------------------------------------------------------
// Einstellungen
// -------------------------------------------------------------------------

// Method liefert das Verteilungsverfahren des Mandanten.
func (s *AccrualService) Method(ctx context.Context) domain.AccrualMethod {
	if s.settingsRepo == nil {
		return domain.AccrualMonthly
	}
	value, err := s.settingsRepo.Get(ctx, domain.SettingAccrualMethod)
	if err == nil && domain.AccrualMethod(value).Valid() {
		return domain.AccrualMethod(value)
	}
	return domain.AccrualMonthly
}

// ReleaseSchedule liefert den Auflösungstakt des Mandanten.
func (s *AccrualService) ReleaseSchedule(ctx context.Context) domain.AccrualReleaseSchedule {
	if s.settingsRepo == nil {
		return domain.AccrualReleaseYearly
	}
	value, err := s.settingsRepo.Get(ctx, domain.SettingAccrualRelease)
	if err == nil && domain.AccrualReleaseSchedule(value).Valid() {
		return domain.AccrualReleaseSchedule(value)
	}
	return domain.AccrualReleaseYearly
}

// Threshold liefert die Vorschlagsschwelle.
func (s *AccrualService) Threshold(ctx context.Context) domain.Cents {
	if s.settingsRepo == nil {
		return domain.DefaultAccrualThreshold
	}
	value, err := s.settingsRepo.Get(ctx, domain.SettingAccrualThreshold)
	if err != nil || strings.TrimSpace(value) == "" {
		return domain.DefaultAccrualThreshold
	}
	cents, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || cents < 0 {
		return domain.DefaultAccrualThreshold
	}
	return domain.Cents(cents)
}

// -------------------------------------------------------------------------
// Vorschlag
// -------------------------------------------------------------------------

// AccrualProposalItem ist eine Buchung, deren Leistung über den Stichtag
// hinausreicht.
type AccrualProposalItem struct {
	EntryID     uint   `json:"entryId"`
	EntryNumber string `json:"entryNumber"`
	BookingDate string `json:"bookingDate"`
	Description string `json:"description"`

	Kind        domain.AccrualKind `json:"kind"`
	Account     string             `json:"account"`
	AccountName string             `json:"accountName"`

	ServiceFrom string       `json:"serviceFrom"`
	ServiceTo   string       `json:"serviceTo"`
	TotalAmount domain.Cents `json:"totalAmount"`
	// DeferredAmount ist der Teil nach dem Stichtag — der Betrag, der
	// abzugrenzen wäre.
	DeferredAmount domain.Cents `json:"deferredAmount"`

	// BelowThreshold sagt, dass der Betrag unter der Vorschlagsschwelle liegt.
	// Der Posten wird trotzdem angezeigt: die Schwelle ist ein steuerliches
	// Wahlrecht und keine handelsrechtliche Grenze.
	BelowThreshold bool `json:"belowThreshold"`
	// AlreadyBooked sagt, dass zu dieser Buchung schon eine Abgrenzung besteht.
	AlreadyBooked bool `json:"alreadyBooked"`
}

// AccrualProposal ist der Vorschlag des Bausteins zum Bilanzstichtag.
type AccrualProposal struct {
	FiscalYear int                   `json:"fiscalYear"`
	Cutoff     string                `json:"cutoff"`
	Method     domain.AccrualMethod  `json:"method"`
	Threshold  domain.Cents          `json:"threshold"`
	Items      []AccrualProposalItem `json:"items"`
	// Note ist der Erklärtext: was die Schwelle bedeutet und was nicht.
	Note string `json:"note"`
}

// Propose sucht die Buchungen des Jahres, deren Leistung über den Stichtag
// hinausreicht.
//
// Gesucht wird über das Leistungsende (ServiceDateTo): es steht an jeder
// Buchung, weil § 14 Abs. 4 Nr. 6 UStG den Leistungszeitpunkt zur
// Pflichtangabe macht. Damit ist die Abgrenzung keine Schätzung, sondern ergibt
// sich aus dem, was ohnehin erfasst wurde.
func (s *AccrualService) Propose(ctx context.Context, year int) (*AccrualProposal, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	method := s.Method(ctx)
	threshold := s.Threshold(ctx)
	startMonth := s.fiscalYearStartMonth(ctx)

	entries, err := s.journalRepo.FindAll(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("die Buchungen des Geschäftsjahres %d konnten nicht gelesen werden: %w", year, err)
	}
	existing, err := s.liveAccruals(ctx)
	if err != nil {
		return nil, err
	}
	booked := map[uint]bool{}
	for i := range existing {
		if existing[i].SourceEntryID != nil {
			booked[*existing[i].SourceEntryID] = true
		}
	}

	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return nil, err
	}

	proposal := &AccrualProposal{
		FiscalYear: year, Cutoff: fy.EndDate, Method: method, Threshold: threshold,
		Items: make([]AccrualProposalItem, 0),
		Note: fmt.Sprintf(
			"Vorgeschlagen wird ab %s €. Die Schwelle stammt aus § 5 Abs. 5 Satz 2 EStG und ist ein "+
				"steuerliches Wahlrecht für geringwertige Beträge — das Handelsrecht kennt keine Grenze, "+
				"§ 250 Abs. 1 HGB verlangt die Abgrenzung ohne Rücksicht auf die Höhe. Kleinere Beträge "+
				"stehen deshalb in der Liste und sind nur nicht vorausgewählt.", threshold),
	}

	reversals := newReversalIndex(s.journalRepo)
	for i := range entries {
		entry := &entries[i]
		if entry.Kind == domain.EntryKindReversal || entry.Source == domain.EntrySourceOpening {
			continue
		}
		if entry.Source == domain.EntrySourceClosing {
			continue
		}
		if entry.ServiceDateTo == "" || entry.ServiceDateTo <= fy.EndDate {
			continue
		}
		// Eine per Generalumkehr aufgehobene Rechnung ist keine Ausgabe mehr;
		// abzugrenzen ist an ihr nichts. Gefragt wird erst hier und nicht über
		// alle Buchungen des Jahres: nach dem Leistungsende bleibt eine Handvoll
		// übrig, davor wären es Tausende Abfragen.
		voided, err := reversals.reversed(ctx, &entry.ID)
		if err != nil {
			return nil, err
		}
		if voided {
			continue
		}
		from := entry.ServiceDateFrom
		if from == "" {
			from = entry.BookingDate
		}

		for _, line := range entry.Lines {
			acc, ok := chart.Lookup(line.Account)
			if !ok {
				continue
			}
			var kind domain.AccrualKind
			switch {
			case acc.Type == domain.AccountTypeExpense && line.Side == domain.SideDebit:
				kind = domain.AccrualActive
			case acc.Type == domain.AccountTypeRevenue && line.Side == domain.SideCredit:
				kind = domain.AccrualPassive
			default:
				continue
			}
			deferred, err := accounting.AccrualShare(
				line.Amount, from, entry.ServiceDateTo, fy.EndDate, method, startMonth)
			if err != nil || deferred <= 0 {
				continue
			}
			proposal.Items = append(proposal.Items, AccrualProposalItem{
				EntryID: entry.ID, EntryNumber: entry.EntryNumber, BookingDate: entry.BookingDate,
				Description: entry.Description, Kind: kind,
				Account: line.Account, AccountName: acc.Name,
				ServiceFrom: from, ServiceTo: entry.ServiceDateTo,
				TotalAmount: line.Amount, DeferredAmount: deferred,
				BelowThreshold: deferred < threshold,
				AlreadyBooked:  booked[entry.ID],
			})
		}
	}

	sort.Slice(proposal.Items, func(i, j int) bool {
		if proposal.Items[i].DeferredAmount != proposal.Items[j].DeferredAmount {
			return proposal.Items[i].DeferredAmount > proposal.Items[j].DeferredAmount
		}
		return proposal.Items[i].EntryNumber < proposal.Items[j].EntryNumber
	})
	return proposal, nil
}

// -------------------------------------------------------------------------
// Vorschau und Buchung
// -------------------------------------------------------------------------

// AccrualRequest ist die Anlage eines Abgrenzungspostens.
type AccrualRequest struct {
	FiscalYear    int                `json:"fiscalYear"`
	Kind          domain.AccrualKind `json:"kind"`
	SourceEntryID uint               `json:"sourceEntryId,omitempty"`
	Text          string             `json:"text"`
	TotalAmount   domain.Cents       `json:"totalAmount"`
	StartDate     string             `json:"startDate"`
	EndDate       string             `json:"endDate"`
	Account       string             `json:"account"`
	// DeferredAmount überschreibt den gerechneten Anteil. Null heißt: rechnen.
	// Der Anwender darf ihn setzen, weil eine Leistung nicht immer gleichmäßig
	// über die Zeit verteilt ist.
	DeferredAmount domain.Cents `json:"deferredAmount,omitempty"`
}

// AccrualPreview ist die Vorschau eines Postens vor der Freigabe.
type AccrualPreview struct {
	Accrual     domain.Accrual              `json:"accrual"`
	Lines       []domain.JournalLine        `json:"lines"`
	Releases    []accounting.AccrualRelease `json:"releases"`
	BookingDate string                      `json:"bookingDate"`
	Explanation string                      `json:"explanation"`
	Warnings    []string                    `json:"warnings"`
}

// Preview rechnet den Posten, ohne ihn zu speichern.
func (s *AccrualService) Preview(ctx context.Context, req AccrualRequest) (*AccrualPreview, error) {
	year := req.FiscalYear
	if year == 0 {
		year = s.fiscalYear
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	method := s.Method(ctx)
	startMonth := s.fiscalYearStartMonth(ctx)

	if !req.Kind.Valid() {
		return nil, fmt.Errorf("unbekannte Art der Abgrenzung %q", req.Kind)
	}
	if req.StartDate == "" || req.EndDate == "" {
		return nil, fmt.Errorf("die Abgrenzung braucht Beginn und Ende des Zeitraums")
	}
	if req.Kind == domain.AccrualDisagio {
		account, err := disagioAccount(req.Account)
		if err != nil {
			return nil, err
		}
		req.Account = account
	}

	deferred := req.DeferredAmount
	if deferred == 0 {
		deferred, err = accounting.AccrualShare(
			req.TotalAmount, req.StartDate, req.EndDate, fy.EndDate, method, startMonth)
		if err != nil {
			return nil, err
		}
	}

	accrual := domain.Accrual{
		FiscalYear: year, Kind: req.Kind, Text: req.Text,
		TotalAmount: req.TotalAmount, DeferredAmount: deferred,
		StartDate: req.StartDate, EndDate: req.EndDate, CutoffDate: fy.EndDate,
		Account: req.Account, Method: method,
	}
	if req.SourceEntryID != 0 {
		id := req.SourceEntryID
		accrual.SourceEntryID = &id
	}
	if err := accrual.Validate(); err != nil {
		return nil, err
	}

	plan, err := accounting.AccrualReleasePlanFor(
		deferred, req.StartDate, req.EndDate, fy.EndDate, method, startMonth, s.ReleaseSchedule(ctx))
	if err != nil {
		return nil, err
	}
	for _, release := range plan {
		accrual.Releases = append(accrual.Releases, domain.AccrualRelease{
			FiscalYear: release.FiscalYear, Date: release.Date, Amount: release.Amount,
		})
	}

	preview := &AccrualPreview{
		Accrual: accrual, Lines: accrualLines(&accrual, false),
		Releases: plan, BookingDate: fy.EndDate,
		Warnings: make([]string, 0),
	}
	preview.Explanation = accrualExplanation(&accrual, method)
	if deferred < s.Threshold(ctx) {
		preview.Warnings = append(preview.Warnings, fmt.Sprintf(
			"Der Betrag liegt unter der Vorschlagsschwelle von %s €. Handelsrechtlich ist trotzdem "+
				"abzugrenzen (§ 250 Abs. 1 HGB); steuerlich lässt § 5 Abs. 5 Satz 2 EStG die Wahl.",
			s.Threshold(ctx)))
	}
	return preview, nil
}

// Book legt den Posten an und bucht seine Bildung.
func (s *AccrualService) Book(ctx context.Context, req AccrualRequest) (*domain.Accrual, error) {
	preview, err := s.Preview(ctx, req)
	if err != nil {
		return nil, err
	}
	accrual := preview.Accrual

	// Zu einer Buchung gehört höchstens eine Abgrenzung. Ohne diese Prüfung
	// bucht ein zweiter Griff in die Vorschlagsliste denselben Betrag ein
	// zweites Mal auf 1900 — und legt einen zweiten Auflösungsplan an, den der
	// Saldenvortrag im Folgejahr ebenfalls bucht.
	//
	// Ist die Bildungsbuchung dagegen storniert, ist die Sperre aufgehoben: der
	// Posten, auf den sie sich beruft, steht nicht mehr in den Büchern.
	if accrual.SourceEntryID != nil {
		existing, err := s.liveAccruals(ctx)
		if err != nil {
			return nil, err
		}
		for i := range existing {
			other := &existing[i]
			if other.SourceEntryID == nil || *other.SourceEntryID != *accrual.SourceEntryID {
				continue
			}
			return nil, fmt.Errorf(
				"zu dieser Buchung besteht bereits die Abgrenzung %q über %s € vom %s. "+
					"Storniere sie, wenn der Betrag nicht stimmt",
				other.Text, other.DeferredAmount, other.CutoffDate)
		}
	}

	entry := &domain.JournalEntry{
		BookingDate:        preview.BookingDate,
		DocumentDate:       preview.BookingDate,
		ServiceDateFrom:    accrual.StartDate,
		ServiceDateTo:      accrual.EndDate,
		Description:        fmt.Sprintf("Rechnungsabgrenzung: %s", accrual.Text),
		Source:             domain.EntrySourceClosing,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              preview.Lines,
	}

	receipt, err := selfIssuedVoucher(ctx, s.receipts, accrual.FiscalYear, closingVoucher{
		Kind: "rechnungsabgrenzung", FiscalYear: accrual.FiscalYear, Date: preview.BookingDate,
		Description: entry.Description, Explanation: preview.Explanation,
		Calculation: accrual, Lines: preview.Lines,
	})
	if err != nil {
		return nil, err
	}
	attachVoucher(entry, receipt)

	created, err := postWithVoucher(ctx, s.journalSvc, s.receipts, entry, receipt)
	if err != nil {
		return nil, err
	}
	accrual.FormationEntryID = &created.ID
	if err := s.accrualRepo.Save(ctx, &accrual); err != nil {
		return nil, fmt.Errorf(
			"die Buchung %s wurde geschrieben, der Abgrenzungsposten aber nicht gespeichert: %w",
			created.EntryNumber, err)
	}
	s.audit(ctx, domain.AuditActionCreate, accrual.ID, fmt.Sprintf(
		"Rechnungsabgrenzung %s über %s € (%s bis %s, %s)",
		accrual.Text, accrual.DeferredAmount, accrual.StartDate, accrual.EndDate, accrual.Method.Label()))
	return &accrual, nil
}

// accrualLines baut den Buchungssatz. release=false ist die Bildung, true die
// Auflösung — sie ist derselbe Satz mit vertauschten Seiten.
func accrualLines(accrual *domain.Accrual, release bool) []domain.JournalLine {
	amount := accrual.DeferredAmount
	balance := accrual.Kind.BalanceAccount()
	balanceSide, accountSide := domain.SideDebit, domain.SideCredit
	if accrual.Kind == domain.AccrualPassive {
		balanceSide, accountSide = domain.SideCredit, domain.SideDebit
	}
	if release {
		balanceSide, accountSide = accountSide, balanceSide
	}
	return []domain.JournalLine{
		{Side: balanceSide, Account: balance, Amount: amount, Text: accrual.Text},
		{Side: accountSide, Account: accrual.Account, Amount: amount, Text: accrual.Text},
	}
}

// disagioAccount prüft das Gegenkonto des Disagios und schlägt es vor.
//
// Die Auflösung eines Damnums ist Zinsaufwand und nichts anderes: § 250 Abs. 3
// HGB lässt den Unterschiedsbetrag zwischen Erfüllungs- und Ausgabebetrag
// aktivieren, weil er wirtschaftlich vorausgezahlter Zins ist. Der SKR04 führt
// ihn im Bereich 7300 bis 7399; ein Konto außerhalb wiese ihn in der GuV an
// einer Stelle aus, an die er nicht gehört (§ 275 Abs. 2 Nr. 13 HGB).
func disagioAccount(account string) (string, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return domain.AccountZinsaufwandLangfristig, nil
	}
	number, err := strconv.Atoi(account)
	if err != nil || number < 7300 || number > 7399 {
		return "", fmt.Errorf(
			"das Disagio wird über Zinsaufwand aufgelöst (§ 250 Abs. 3 HGB). Das Konto %s gehört "+
				"nicht dazu; vorgesehen sind die Konten 7300 bis 7399, voreingestellt %s",
			account, domain.AccountZinsaufwandLangfristig)
	}
	return account, nil
}

func accrualExplanation(accrual *domain.Accrual, method domain.AccrualMethod) string {
	if accrual.Kind == domain.AccrualDisagio {
		return fmt.Sprintf(
			"Das Disagio von %s € wird nach § 250 Abs. 3 HGB über die Laufzeit des Darlehens "+
				"(%s bis %s) verteilt und dabei %s aufgeteilt.",
			accrual.DeferredAmount, accrual.StartDate, accrual.EndDate, method.Label())
	}
	side := "Aufwand"
	if accrual.Kind == domain.AccrualPassive {
		side = "Ertrag"
	}
	return fmt.Sprintf(
		"Von %s € entfallen %s € auf die Zeit nach dem %s. Dieser Teil ist %s der Folgejahre und wird "+
			"nach § 250 HGB abgegrenzt; verteilt wird %s.",
		accrual.TotalAmount, accrual.DeferredAmount, accrual.CutoffDate, side, method.Label())
}

// -------------------------------------------------------------------------
// Auflösung im Folgejahr
// -------------------------------------------------------------------------

// Der Auflösungsplan entsteht bei der Bildung und wird gespeichert. Wie fein er
// ist, entscheidet die Einstellung `accrual_release`.
//
// Sie ist von `accrual_method` zu trennen: die Methode sagt, *wie* der Anteil
// gerechnet wird (monatsgenau nach Zwölfteln oder taggenau), der Takt, in *wie
// vielen Buchungen* er zurückkommt. Voreingestellt ist eine Auflösung je
// Geschäftsjahr am ersten Tag — für Bilanz und GuV des Jahres macht es keinen
// Unterschied, und die Zahl der Abschlussbuchungen bleibt klein. Wer unterjährig
// auswertet, stellt auf „monatlich" um: sonst trägt der Januar den gesamten
// Vorjahresaufwand, und jede BWA und jeder Zwischenabschluss des Jahres sind
// verzerrt.
//
// Gebucht werden in beiden Fällen alle Auflösungen des Zieljahres auf einmal —
// beim Saldenvortrag, jede mit ihrem eigenen Datum.

// AccrualReleaseDue ist eine im Zieljahr fällige Auflösung.
type AccrualReleaseDue struct {
	AccrualID uint               `json:"accrualId"`
	ReleaseID uint               `json:"releaseId"`
	Kind      domain.AccrualKind `json:"kind"`
	Text      string             `json:"text"`
	Account   string             `json:"account"`
	Date      string             `json:"date"`
	Amount    domain.Cents       `json:"amount"`
}

// PendingReleases nennt die Auflösungen, die in einem Geschäftsjahr fällig und
// noch nicht gebucht sind.
func (s *AccrualService) PendingReleases(ctx context.Context, toYear int) ([]AccrualReleaseDue, error) {
	accruals, err := s.liveAccruals(ctx)
	if err != nil {
		return nil, err
	}
	due := make([]AccrualReleaseDue, 0)
	for i := range accruals {
		a := &accruals[i]
		if !a.IsBooked() {
			continue
		}
		for _, r := range a.Releases {
			if r.FiscalYear != toYear || r.JournalEntryID != nil || r.Amount == 0 {
				continue
			}
			due = append(due, AccrualReleaseDue{
				AccrualID: a.ID, ReleaseID: r.ID, Kind: a.Kind, Text: a.Text,
				Account: a.Account, Date: r.Date, Amount: r.Amount,
			})
		}
	}
	sort.Slice(due, func(i, j int) bool { return due[i].Date < due[j].Date })
	return due, nil
}

// ReleaseInto bucht die im Zieljahr fälligen Auflösungen.
//
// Aufgerufen wird sie vom Saldenvortrag: die Auflösung gehört in das neue Jahr
// und würde ohne diesen Anstoß bis zum nächsten Jahresabschluss liegenbleiben —
// also bis zu dem Zeitpunkt, an dem sie längst hätte gebucht sein müssen.
func (s *AccrualService) ReleaseInto(ctx context.Context, toYear int) ([]domain.JournalEntry, error) {
	due, err := s.PendingReleases(ctx, toYear)
	if err != nil {
		return nil, err
	}
	created := make([]domain.JournalEntry, 0, len(due))
	for _, item := range due {
		accrual, err := s.accrualRepo.FindByID(ctx, item.AccrualID)
		if err != nil || accrual == nil {
			return created, fmt.Errorf("der Abgrenzungsposten %d wurde nicht gefunden", item.AccrualID)
		}
		// Für die Zeilen zählt der Betrag dieser Auflösung, nicht der ganze
		// Posten: ein Posten über drei Jahre löst sich in drei Schritten auf.
		part := *accrual
		part.DeferredAmount = item.Amount
		lines := accrualLines(&part, true)
		entry := &domain.JournalEntry{
			BookingDate:        item.Date,
			DocumentDate:       item.Date,
			ServiceDateFrom:    item.Date,
			ServiceDateTo:      item.Date,
			Description:        fmt.Sprintf("Auflösung Rechnungsabgrenzung: %s", accrual.Text),
			Source:             domain.EntrySourceClosing,
			PostingRuleVersion: accounting.PostingRuleVersion,
			Lines:              lines,
		}
		// Auch die Auflösung ist eine Buchung und braucht ihren Beleg
		// (GoBD Rz. 61). Der Eigenbeleg nennt den Posten, aus dem sie stammt,
		// und den Teil des Plans, den sie erfüllt — sonst stünde im Folgejahr
		// eine Aufwandsbuchung ohne jede Herkunft.
		explanation := accrualReleaseExplanation(accrual, item)
		receipt, err := selfIssuedVoucher(ctx, s.receipts, toYear, closingVoucher{
			Kind: "rechnungsabgrenzung-aufloesung", FiscalYear: toYear, Date: item.Date,
			Description: entry.Description, Explanation: explanation,
			Calculation: item, Lines: lines,
		})
		if err != nil {
			return created, err
		}
		attachVoucher(entry, receipt)

		out, err := postWithVoucher(ctx, s.journalSvc, s.receipts, entry, receipt)
		if err != nil {
			return created, fmt.Errorf(
				"die Auflösung der Abgrenzung %q konnte nicht gebucht werden: %w", accrual.Text, err)
		}
		release := domain.AccrualRelease{
			ID: item.ReleaseID, AccrualID: item.AccrualID, FiscalYear: toYear,
			Date: item.Date, Amount: item.Amount, JournalEntryID: &out.ID,
		}
		if err := s.accrualRepo.SaveRelease(ctx, &release); err != nil {
			return created, fmt.Errorf(
				"die Buchung %s wurde geschrieben, die Auflösung aber nicht vermerkt: %w",
				out.EntryNumber, err)
		}
		created = append(created, *out)
	}
	if len(created) > 0 {
		s.audit(ctx, domain.AuditActionCreate, 0, fmt.Sprintf(
			"Auflösung der Rechnungsabgrenzung im Geschäftsjahr %d: %d Buchungen", toYear, len(created)))
	}
	return created, nil
}

// accrualReleaseExplanation sagt in einem Satz, woher die Auflösung kommt.
func accrualReleaseExplanation(accrual *domain.Accrual, item AccrualReleaseDue) string {
	side := "Aufwand"
	if accrual.Kind == domain.AccrualPassive {
		side = "Ertrag"
	}
	return fmt.Sprintf(
		"Auflösung des zum %s gebildeten Abgrenzungspostens über %s € (%s bis %s, %s). Auf diesen "+
			"Teil des Auflösungsplans entfallen %s €; sie werden am %s als %s der Periode "+
			"nachgeholt (§ 250 HGB).",
		accrual.CutoffDate, accrual.DeferredAmount, accrual.StartDate, accrual.EndDate,
		accrual.Method.Label(), item.Amount, item.Date, side)
}

// -------------------------------------------------------------------------
// Bestand und Bericht
// -------------------------------------------------------------------------

// List liefert die im Geschäftsjahr gebildeten Posten.
func (s *AccrualService) List(ctx context.Context, year int) ([]domain.Accrual, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	accruals, err := s.accrualRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	return liveAccruals(ctx, s.journalRepo, accruals)
}

// liveAccruals sind alle Posten, deren Bildungsbuchung noch steht.
func (s *AccrualService) liveAccruals(ctx context.Context) ([]domain.Accrual, error) {
	accruals, err := s.accrualRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	return liveAccruals(ctx, s.journalRepo, accruals)
}

// AccrualReportRow ist eine Zeile des Abgrenzungsberichts.
type AccrualReportRow struct {
	AccrualID uint               `json:"accrualId"`
	Kind      domain.AccrualKind `json:"kind"`
	KindLabel string             `json:"kindLabel"`
	Text      string             `json:"text"`
	Account   string             `json:"account"`
	StartDate string             `json:"startDate"`
	EndDate   string             `json:"endDate"`

	DeferredAmount domain.Cents `json:"deferredAmount"`
	Released       domain.Cents `json:"released"`
	Remaining      domain.Cents `json:"remaining"`
	// RemainingDays ist die Restlaufzeit ab dem Stichtag in Kalendertagen.
	RemainingDays int `json:"remainingDays"`
}

// AccrualReport ist der Bestand aller Abgrenzungen zu einem Stichtag.
type AccrualReport struct {
	Cutoff       string             `json:"cutoff"`
	Rows         []AccrualReportRow `json:"rows"`
	TotalActive  domain.Cents       `json:"totalActive"`
	TotalPassive domain.Cents       `json:"totalPassive"`
}

// Report stellt den Bestand zu einem Stichtag zusammen.
func (s *AccrualService) Report(ctx context.Context, cutoff string) (*AccrualReport, error) {
	if cutoff == "" {
		fy, err := s.closingSvc.PeriodOf(ctx, s.fiscalYear)
		if err != nil {
			return nil, err
		}
		cutoff = fy.EndDate
	}
	// Ein Posten, dessen Bildung storniert wurde, hat keinen Bestand: er steht
	// weder auf 1900 noch auf 3900, und der Bericht führte ihn sonst als
	// Vermögen oder Schuld, die es nicht gibt.
	accruals, err := s.liveAccruals(ctx)
	if err != nil {
		return nil, err
	}
	report := &AccrualReport{Cutoff: cutoff, Rows: make([]AccrualReportRow, 0, len(accruals))}
	for i := range accruals {
		a := &accruals[i]
		if !a.IsBooked() || a.CutoffDate > cutoff {
			continue
		}
		// Zum Stichtag zählt, was bis dahin aufgelöst wurde. Eine Auflösung aus
		// einem späteren Jahr gehört nicht in einen früheren Bestand.
		var released domain.Cents
		for _, r := range a.Releases {
			if r.JournalEntryID != nil && r.Date <= cutoff {
				released += r.Amount
			}
		}
		remaining := a.DeferredAmount - released
		if remaining <= 0 {
			continue
		}
		row := AccrualReportRow{
			AccrualID: a.ID, Kind: a.Kind, KindLabel: a.Kind.Label(), Text: a.Text,
			Account: a.Account, StartDate: a.StartDate, EndDate: a.EndDate,
			DeferredAmount: a.DeferredAmount, Released: released, Remaining: remaining,
			RemainingDays: calendarDaysBetween(cutoff, a.EndDate),
		}
		report.Rows = append(report.Rows, row)
		if a.Kind == domain.AccrualPassive {
			report.TotalPassive += remaining
		} else {
			report.TotalActive += remaining
		}
	}
	sort.Slice(report.Rows, func(i, j int) bool { return report.Rows[i].EndDate < report.Rows[j].EndDate })
	return report, nil
}

func (s *AccrualService) fiscalYearStartMonth(ctx context.Context) int {
	if s.settingsRepo == nil {
		return 1
	}
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || settings == nil || settings.FiscalYearStartMonth <= 0 {
		return 1
	}
	return settings.FiscalYearStartMonth
}

func (s *AccrualService) audit(ctx context.Context, action domain.AuditAction, id uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "ACCRUAL", fmt.Sprintf("%d", id), details)
}

// calendarDaysBetween ist die Zahl der Kalendertage zwischen zwei Tagen; nie
// negativ, weil eine abgelaufene Restlaufzeit keine ist.
func calendarDaysBetween(from, to string) int {
	start, err1 := time.Parse("2006-01-02", from)
	end, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil || !end.After(start) {
		return 0
	}
	return int(end.Sub(start).Hours() / 24)
}
