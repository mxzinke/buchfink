package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// AppropriationService führt die Ergebnisverwendung und die Anhangtexte.
//
// Beides gehört zusammen, weil beides eine Willenserklärung ist und keine
// Rechnung: der Beschluss der Gesellschafter über das Ergebnis (§ 29 GmbHG) und
// die Angaben im Anhang, die aus keiner Buchung folgen. Buchfink rechnet hier
// die Kapitalertragsteuer auf die Ausschüttung und hält den Beschluss fest.
// Die Pflichtrücklage der UG entsteht bereits im Abschluss des Gewinnjahres.
type AppropriationService struct {
	appropriationRepo domain.AppropriationRepository
	notesRepo         domain.NotesTextRepository
	journalRepo       domain.JournalRepository
	journalSvc        *JournalService
	settingsRepo      domain.SettingsRepository
	auditRepo         domain.AuditRepository
	closingSvc        *ClosingService
	receipts          closingReceiptFiler
}

// NewAppropriationService wires die Ergebnisverwendung.
func NewAppropriationService(
	appropriationRepo domain.AppropriationRepository,
	notesRepo domain.NotesTextRepository,
	journalRepo domain.JournalRepository,
	journalSvc *JournalService,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	closingSvc *ClosingService,
) *AppropriationService {
	return &AppropriationService{
		appropriationRepo: appropriationRepo, notesRepo: notesRepo, journalRepo: journalRepo,
		journalSvc: journalSvc, settingsRepo: settingsRepo, auditRepo: auditRepo,
		closingSvc: closingSvc,
	}
}

// SetReceiptService gibt dem Dienst den Belegspeicher für den Eigenbeleg der
// Ergebnisverwendung und für das Beschlussdokument.
func (s *AppropriationService) SetReceiptService(r closingReceiptFiler) { s.receipts = r }

// AppropriationRequest ist der Beschluss über die Verwendung.
type AppropriationRequest struct {
	DecisionDate  string       `json:"decisionDate"`
	Text          string       `json:"text"`
	LegalReserve  domain.Cents `json:"legalReserve"`
	OtherReserves domain.Cents `json:"otherReserves"`
	Distribution  domain.Cents `json:"distribution"`
	ReceiptID     uint         `json:"receiptId,omitempty"`
}

// AppropriationPreview ist der Beschluss vor der Buchung.
type AppropriationPreview struct {
	Year          int                  `json:"year"`
	BookingYear   int                  `json:"bookingYear"`
	NetIncome     domain.Cents         `json:"netIncome"`
	Appropriation domain.Appropriation `json:"appropriation"`
	Lines         []domain.JournalLine `json:"lines"`
	BookingDate   string               `json:"bookingDate"`
	// YearResult ist der Jahresüberschuss des verwendeten Jahres. Er steht
	// neben NetIncome, weil beide auseinanderfallen: NetIncome ist der Saldo des
	// Vortragskontos und enthält auch, was frühere Jahre nicht verwendet haben.
	YearResult domain.Cents `json:"yearResult"`
	// RequiredLegalReserve remains zero for existing API clients. The mandatory
	// allocation is already included in ReservedInClosing and in the carryforward.
	RequiredLegalReserve domain.Cents `json:"requiredLegalReserve"`
	ReservedInClosing    domain.Cents `json:"reservedInClosing"`
	Explanation          string       `json:"explanation"`
	Warnings             []string     `json:"warnings"`
}

// PreviewAppropriation rechnet den Beschluss, ohne ihn zu buchen.
//
// Das verwendbare Ergebnis steht auf dem Vortragskonto des Folgejahres: der
// Saldenvortrag hat es dorthin gebracht, ausdrücklich „vor Verwendung". Erst der
// Beschluss verteilt es — und deshalb liest diese Vorschau das Folgejahr und
// nicht das Jahr, dessen Ergebnis verwendet wird.
func (s *AppropriationService) PreviewAppropriation(
	ctx context.Context, year int, req AppropriationRequest,
) (*AppropriationPreview, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	bookingYear := year + 1
	fy, err := s.closingSvc.PeriodOf(ctx, bookingYear)
	if err != nil {
		return nil, err
	}
	turnovers, err := s.journalRepo.AccountTurnovers(ctx, bookingYear)
	if err != nil {
		return nil, err
	}
	profit := turnovers[domain.AccountGewinnvortrag]
	loss := turnovers[domain.AccountVerlustvortrag]
	netIncome := (profit.Credit - profit.Debit) - (loss.Debit - loss.Credit)

	bookingDate := req.DecisionDate
	if bookingDate == "" {
		bookingDate = fy.StartDate
	}
	// Der Beschluss wird an seinem eigenen Datum gebucht, und eine Buchung
	// gehört in das Geschäftsjahr, in dem sie liegt. Ein Beschlussdatum
	// außerhalb des Buchungsjahres ergäbe eine Buchung, die im falschen Jahr
	// landet — oder gar nicht, weil die Periode gesperrt ist.
	if bookingDate < fy.StartDate || bookingDate > fy.EndDate {
		return nil, fmt.Errorf(
			"der Beschluss vom %s liegt nicht im Geschäftsjahr %d (%s bis %s). Die Verwendung des "+
				"Ergebnisses %d wird in dem Jahr gebucht, in dem sie beschlossen wurde",
			bookingDate, bookingYear, fy.StartDate, fy.EndDate, year)
	}

	// Das Jahresergebnis des verwendeten Jahres — nicht der Saldo des
	// Vortragskontos. Auf ihm liegen auch die nicht verwendeten Gewinne
	// früherer Jahre, und § 5a Abs. 3 GmbHG bemisst die Pflichtrücklage am
	// Jahresüberschuss des Jahres.
	var yearResult domain.Cents
	if priorTurnovers, err := s.journalRepo.AccountTurnovers(ctx, year); err == nil {
		if chart, err := s.journalSvc.Chart(ctx); err == nil {
			yearResult = netIncomeOf(priorTurnovers, chart)
		}
	}

	preview := &AppropriationPreview{
		Year: year, BookingYear: bookingYear, NetIncome: netIncome,
		YearResult:  yearResult,
		BookingDate: bookingDate, Lines: make([]domain.JournalLine, 0, 4),
		Warnings: make([]string, 0),
	}
	reserve, err := s.closingSvc.LegalReserve(ctx, year)
	if err != nil {
		return nil, err
	}
	// The mandatory amount was reserved before the year was closed. The
	// shareholders distribute only the remaining carryforward.
	if reserve.Applies && reserve.Difference != 0 {
		preview.Warnings = append(preview.Warnings, fmt.Sprintf("Im Abschluss %d fehlt die gesetzliche Rücklage. Bitte zuerst den Jahresabschluss berichtigen; eine Buchung im Folgejahr heilt den Abschluss nicht.", year))
	}
	preview.ReservedInClosing = reserve.Booked

	appropriation := domain.Appropriation{
		Year: year, DecisionDate: bookingDate, Text: req.Text, NetIncome: netIncome,
		LegalReserve: req.LegalReserve, OtherReserves: req.OtherReserves,
		Distribution: req.Distribution,
	}
	if req.ReceiptID != 0 {
		id := req.ReceiptID
		appropriation.ReceiptID = &id
	}
	appropriation.CarryForward = netIncome - appropriation.Distributed()

	if appropriation.Distribution > 0 {
		params, err := accounting.TaxParametersFor(bookingDate)
		if err != nil {
			return nil, err
		}
		appropriation.WithholdingTax = domain.MulRound(
			appropriation.Distribution, params.WithholdingTaxPermille, 1000)
		appropriation.SolidarityOnWithholding = domain.MulRound(
			appropriation.WithholdingTax, params.SolidarityPermille, 1000)
	}
	if err := appropriation.Validate(); err != nil {
		return nil, err
	}
	// Einstellung in Rücklagen: das Ergebnis verlässt den Vortrag und wird
	// gebundenes Eigenkapital.
	if appropriation.LegalReserve > 0 {
		preview.Lines = append(preview.Lines,
			domain.JournalLine{Side: domain.SideDebit, Account: domain.AccountGewinnvortrag,
				Amount: appropriation.LegalReserve, Text: "Einstellung in die gesetzliche Rücklage"},
			domain.JournalLine{Side: domain.SideCredit, Account: domain.AccountGesetzlicheRuecklage,
				Amount: appropriation.LegalReserve, Text: "Gesetzliche Rücklage"},
		)
	}
	if appropriation.OtherReserves > 0 {
		preview.Lines = append(preview.Lines,
			domain.JournalLine{Side: domain.SideDebit, Account: domain.AccountGewinnvortrag,
				Amount: appropriation.OtherReserves, Text: "Einstellung in andere Gewinnrücklagen"},
			domain.JournalLine{Side: domain.SideCredit, Account: domain.AccountAndereGewinnruecklagen,
				Amount: appropriation.OtherReserves, Text: "Andere Gewinnrücklagen"},
		)
	}
	// Ausschüttung: der Bruttobetrag verlässt das Eigenkapital, die
	// einbehaltene Kapitalertragsteuer wird eine Verbindlichkeit gegenüber dem
	// Finanzamt, der Rest eine gegenüber den Gesellschaftern.
	if appropriation.Distribution > 0 {
		withheld := appropriation.WithholdingTax + appropriation.SolidarityOnWithholding
		preview.Lines = append(preview.Lines,
			domain.JournalLine{Side: domain.SideDebit, Account: domain.AccountGewinnvortrag,
				Amount: appropriation.Distribution, Text: "Ausschüttung"},
			domain.JournalLine{Side: domain.SideCredit, Account: domain.AccountAusschuettung,
				Amount: appropriation.Distribution - withheld, Text: "Ausschüttung an Gesellschafter"},
		)
		if withheld > 0 {
			preview.Lines = append(preview.Lines, domain.JournalLine{
				Side: domain.SideCredit, Account: domain.AccountKapitalertragsteuer,
				Amount: withheld, Text: "Kapitalertragsteuer und Solidaritätszuschlag",
			})
			preview.Warnings = append(preview.Warnings,
				"Bei einer Gewinnausschüttung ist die einbehaltene Kapitalertragsteuer im Zeitpunkt "+
					"des Zuflusses anzumelden und abzuführen (§ 44 Abs. 1 Satz 5, § 45a Abs. 1 EStG). "+
					"Maßgeblich ist grundsätzlich der im Beschluss bestimmte Auszahlungstag; fehlt er, "+
					"gilt grundsätzlich der Tag nach dem Beschluss (§ 44 Abs. 2 EStG). "+
					"Buchfink erstellt die Steueranmeldung nicht.")
		}
	}
	preview.Appropriation = appropriation
	preview.Explanation = appropriationExplanation(&appropriation)
	return preview, nil
}

// isEntrepreneurialCompany meldet, ob die Rechtsform eine Unternehmergesellschaft
// ist.
func isEntrepreneurialCompany(legalForm string) bool {
	return legalForm == domain.LegalFormUG
}

func appropriationExplanation(a *domain.Appropriation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Verwendbares Ergebnis %s €. ", a.NetIncome)
	if a.LegalReserve > 0 {
		fmt.Fprintf(&b, "Gesetzliche Rücklage %s €. ", a.LegalReserve)
	}
	if a.OtherReserves > 0 {
		fmt.Fprintf(&b, "Andere Gewinnrücklagen %s €. ", a.OtherReserves)
	}
	if a.Distribution > 0 {
		fmt.Fprintf(&b, "Ausschüttung %s €, davon %s € Kapitalertragsteuer und %s € "+
			"Solidaritätszuschlag einbehalten. ",
			a.Distribution, a.WithholdingTax, a.SolidarityOnWithholding)
	}
	if a.CarryForward > 0 {
		fmt.Fprintf(&b, "Vortrag auf neue Rechnung %s € — dafür ist keine Buchung nötig, "+
			"der Betrag steht bereits auf dem Vortragskonto.", a.CarryForward)
	}
	return strings.TrimSpace(b.String())
}

// BookAppropriation hält den Beschluss fest und bucht ihn. Die Gesellschafter
// beschließen über die Verwendung des Ergebnisses, das der festgestellte
// Jahresabschluss ausweist (§ 42a Abs. 2, § 46 Nr. 1 GmbHG). Gebucht wird
// deshalb erst, wenn die Feststellung erfasst ist, und nicht mit einem
// Beschluss, der vor ihr liegt.
func (s *AppropriationService) BookAppropriation(
	ctx context.Context, year int, req AppropriationRequest,
) (*domain.Appropriation, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.Appropriation, error) {
		return s.bookAppropriation(ctx, year, req)
	})
}

func (s *AppropriationService) bookAppropriation(
	ctx context.Context, year int, req AppropriationRequest,
) (*domain.Appropriation, error) {
	preview, err := s.PreviewAppropriation(ctx, year, req)
	if err != nil {
		return nil, err
	}
	if err := s.closingSvc.EnsureLegalReserve(ctx, preview.Year); err != nil {
		return nil, err
	}
	if err := s.ensureAdopted(ctx, preview.Year, preview.BookingDate); err != nil {
		return nil, err
	}
	// Ein zweiter Beschluss zum selben Jahr überschriebe den ersten, während
	// seine Buchung stehen bliebe: aus einer Ausschüttung würden zwei, aus einer
	// Rücklage zwei — und der Verweis auf die erste Buchung wäre verloren. Wer
	// den Beschluss ändert, storniert erst seine Buchung.
	if stored, err := s.appropriationRepo.FindByYear(ctx, preview.Year); err == nil && stored != nil {
		// Nach dem Storno ist die Sperre aufgehoben: der Beschluss steht dann
		// nicht mehr in den Büchern, und der Weg, den die Meldung nennt, führt
		// sonst nirgendwohin.
		voided, err := entryIsReversed(ctx, s.journalRepo, stored.JournalEntryID)
		if err != nil {
			return nil, err
		}
		if stored.JournalEntryID != nil && !voided {
			return nil, fmt.Errorf(
				"für das Geschäftsjahr %d ist am %s bereits ein Beschluss erfasst und gebucht. "+
					"Zuerst ist seine Buchung zu stornieren", preview.Year, stored.DecisionDate)
		}
		// Ein Beschluss ohne Buchung — der reine Vortrag auf neue Rechnung —
		// wird fortgeschrieben: der Schlüssel ist das Geschäftsjahr, und es
		// gibt nichts zu stornieren.
	}
	appropriation := preview.Appropriation

	if len(preview.Lines) > 0 {
		entry := &domain.JournalEntry{
			BookingDate: preview.BookingDate, DocumentDate: preview.BookingDate,
			ServiceDateFrom: preview.BookingDate, ServiceDateTo: preview.BookingDate,
			Description: fmt.Sprintf("Ergebnisverwendung %d laut Gesellschafterbeschluss vom %s",
				preview.Year, preview.BookingDate),
			Source:             domain.EntrySourceClosing,
			DocumentNumber:     fmt.Sprintf("EV %d", preview.Year),
			PostingRuleVersion: accounting.PostingRuleVersion,
			Lines:              preview.Lines,
		}
		// Der Beleg der Ergebnisverwendung ist der Gesellschafterbeschluss.
		// Liegt er als Dokument vor, ist er der Beleg der Buchung; sonst tritt
		// der Eigenbeleg an seine Stelle, der den Beschluss mit seinem Datum,
		// seinem Wortlaut und seiner Verteilung festhält. Ohne beides wäre es
		// eine Buchung ohne Beleg (GoBD Rz. 61).
		receipt, err := selfIssuedVoucher(ctx, s.receipts, preview.BookingYear, closingVoucher{
			Kind: "ergebnisverwendung", FiscalYear: preview.BookingYear,
			Date: preview.BookingDate, Description: entry.Description,
			Explanation: preview.Explanation, Calculation: appropriation, Lines: preview.Lines,
		})
		if err != nil {
			return nil, err
		}
		reference := entry.DocumentNumber
		attachVoucher(entry, receipt)
		entry.DocumentNumber = reference
		// Das Beschlussdokument selbst geht dem Eigenbeleg vor: es ist der
		// äußere Nachweis, der Eigenbeleg nur die Rechnung dazu.
		if appropriation.ReceiptID != nil && s.receipts != nil {
			decision, err := s.receipts.Get(ctx, *appropriation.ReceiptID)
			if err != nil {
				return nil, fmt.Errorf(
					"das Beschlussdokument zum Beleg %d wurde im Belegspeicher nicht gefunden: %w",
					*appropriation.ReceiptID, err)
			}
			// Vor dem Schreiben: das Beschlussdokument belegt die Buchung und
			// wird hinter ihr versiegelt — ohne Kopfdaten bliebe sonst eine
			// geschriebene Buchung mit unverbundenem Beschluss zurück.
			if err := ensureOuterVoucherHeader(decision, "das Beschlussdokument"); err != nil {
				return nil, err
			}
			attachVoucher(entry, decision)
			entry.DocumentNumber = reference
		}

		created, err := postWithVoucher(ctx, s.journalSvc, s.receipts, entry, receipt)
		if err != nil {
			return nil, err
		}
		appropriation.JournalEntryID = &created.ID
		// Beide Belege gehören an die Buchung: der Beschluss als Nachweis, der
		// Eigenbeleg als Rechnung. Versiegelt wird deshalb auch der Beschluss —
		// sonst bliebe er als offener Beleg im Belegspeicher liegen.
		if appropriation.ReceiptID != nil && s.receipts != nil {
			if err := s.receipts.Seal(ctx, *appropriation.ReceiptID, created.ID); err != nil {
				return nil, fmt.Errorf("das Beschlussdokument ließ sich nicht mit der Buchung verbinden: %w", err)
			}
		}
	}
	if err := s.appropriationRepo.Save(ctx, &appropriation); err != nil {
		return nil, fmt.Errorf("der Beschluss konnte nicht gespeichert werden: %w", err)
	}
	if err := s.audit(ctx, "APPROPRIATION", preview.Year, fmt.Sprintf(
		"Ergebnisverwendung %d: %s", preview.Year, preview.Explanation)); err != nil {
		return nil, err
	}
	return &appropriation, nil
}

// ensureAdopted verlangt den festgestellten Abschluss des verwendeten Jahres
// und einen Beschluss, der nicht vor der Feststellung liegt. Beide dürfen auf
// denselben Tag fallen: meist beschließt dieselbe Versammlung über beides.
func (s *AppropriationService) ensureAdopted(ctx context.Context, year int, decisionDate string) error {
	fy, err := s.closingSvc.YearOf(ctx, year)
	if err != nil {
		return err
	}
	if !fy.IsAdopted() {
		return fmt.Errorf(
			"der Jahresabschluss %d ist nicht festgestellt (Stand: %s). Über die Verwendung des Ergebnisses "+
				"beschließen die Gesellschafter auf Grundlage des festgestellten Abschlusses (§ 42a Abs. 2, "+
				"§ 46 Nr. 1 GmbHG); die Feststellung erfasst der Mensch in der App", year, fy.Status.Label())
	}
	if decisionDate < fy.AdoptedOn {
		return fmt.Errorf("der Beschluss vom %s liegt vor der Feststellung des Abschlusses %d am %s",
			domain.GermanDate(decisionDate), year, domain.GermanDate(fy.AdoptedOn))
	}
	return nil
}

// Appropriation liefert den gespeicherten Beschluss eines Jahres, oder nil.
func (s *AppropriationService) Appropriation(ctx context.Context, year int) (*domain.Appropriation, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	return s.appropriationRepo.FindByYear(ctx, year)
}

// -------------------------------------------------------------------------
// Anhangtexte
// -------------------------------------------------------------------------

// NotesTextView ist ein Anhangabschnitt mit seinem Text. Der Typ steht in
// internal/domain, weil der Jahresabschluss den Anhang mitträgt.
type NotesTextView = domain.NotesSectionText

// NotesTexts liefert alle Abschnitte eines Jahres, auch die leeren: der Anhang
// ist eine Gliederung, und ein Abschnitt, der nur erscheint, wenn er gefüllt
// ist, wird nie gefüllt.
func (s *AppropriationService) NotesTexts(ctx context.Context, year int) ([]NotesTextView, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	stored, err := s.notesRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	bySection := map[domain.NotesSection]string{}
	for _, text := range stored {
		bySection[text.Section] = text.Text
	}
	out := make([]NotesTextView, 0, len(domain.AllNotesSections()))
	for _, def := range domain.AllNotesSections() {
		out = append(out, NotesTextView{NotesSectionDefinition: def, Text: bySection[def.Section]})
	}
	return out, nil
}

// SaveNotesText schreibt einen Abschnitt fort.
func (s *AppropriationService) SaveNotesText(
	ctx context.Context, year int, section domain.NotesSection, text string,
) ([]NotesTextView, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) ([]NotesTextView, error) {
		return s.saveNotesText(ctx, year, section, text)
	})
}

func (s *AppropriationService) saveNotesText(
	ctx context.Context, year int, section domain.NotesSection, text string,
) ([]NotesTextView, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	fy, err := s.closingSvc.YearOf(ctx, year)
	if err != nil {
		return nil, err
	}
	if fy.IsAdopted() {
		return nil, fmt.Errorf("der Abschluss %d ist festgestellt. Öffnen Sie das Jahr mit Begründung erneut, bevor Sie seine Abschlussangaben ändern", year)
	}
	entry := &domain.NotesText{Year: year, Section: section, Text: text}
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if err := s.notesRepo.Save(ctx, entry); err != nil {
		return nil, err
	}
	def, _ := domain.NotesSectionDefinitionFor(section)
	if err := s.audit(ctx, "NOTES", year, fmt.Sprintf("Anhangabschnitt %q im Geschäftsjahr %d geändert", def.Label, year)); err != nil {
		return nil, err
	}
	return s.NotesTexts(ctx, year)
}

// CopyNotesInto übernimmt die Anhangtexte des Vorjahres als Vorlage.
//
// Die Bilanzierungs- und Bewertungsmethoden ändern sich selten, und ein leerer
// Anhang im neuen Jahr führt dazu, dass die Angaben schlicht fehlen. Kopiert
// wird nur, was noch nicht dasteht: eine bereits geschriebene Fassung darf eine
// Vorlage nicht überschreiben.
func (s *AppropriationService) CopyNotesInto(ctx context.Context, toYear int) (int, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (int, error) {
		return s.copyNotesInto(ctx, toYear)
	})
}

func (s *AppropriationService) copyNotesInto(ctx context.Context, toYear int) (int, error) {
	if toYear <= 1 {
		return 0, nil
	}
	source, err := s.notesRepo.FindByYear(ctx, toYear-1)
	if err != nil || len(source) == 0 {
		return 0, err
	}
	target, err := s.notesRepo.FindByYear(ctx, toYear)
	if err != nil {
		return 0, err
	}
	present := map[domain.NotesSection]bool{}
	for _, text := range target {
		if strings.TrimSpace(text.Text) != "" {
			present[text.Section] = true
		}
	}
	copied := 0
	for _, text := range source {
		if present[text.Section] || strings.TrimSpace(text.Text) == "" {
			continue
		}
		entry := &domain.NotesText{Year: toYear, Section: text.Section, Text: text.Text}
		if err := s.notesRepo.Save(ctx, entry); err != nil {
			return copied, err
		}
		copied++
	}
	if copied > 0 {
		if err := s.audit(ctx, "NOTES", toYear, fmt.Sprintf(
			"%d Anhangtexte aus dem Geschäftsjahr %d als Vorlage übernommen", copied, toYear-1)); err != nil {
			return 0, err
		}
	}
	return copied, nil
}

func (s *AppropriationService) audit(ctx context.Context, entity string, year int, details string) error {
	return logAudit(ctx, s.auditRepo, domain.AuditActionUpdate, entity, fmt.Sprintf("%d", year), details)
}
