package service

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// AppropriationService führt die Ergebnisverwendung und die Anhangtexte.
//
// Beides gehört zusammen, weil beides eine Willenserklärung ist und keine
// Rechnung: der Beschluss der Gesellschafter über das Ergebnis (§ 29 GmbHG) und
// die Angaben im Anhang, die aus keiner Buchung folgen. Buchfink rechnet hier
// nur, was zwingend ist — die Pflichtrücklage der UG und die Kapitalertragsteuer
// auf die Ausschüttung —, und hält den Rest fest, wie er beschlossen wurde.
type AppropriationService struct {
	appropriationRepo domain.AppropriationRepository
	notesRepo         domain.NotesTextRepository
	journalRepo       domain.JournalRepository
	journalSvc        *JournalService
	settingsRepo      domain.SettingsRepository
	auditRepo         domain.AuditRepository
	closingSvc        *ClosingService
	receipts          closingReceiptFiler
	fiscalYear        int
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
	fiscalYear int,
) *AppropriationService {
	return &AppropriationService{
		appropriationRepo: appropriationRepo, notesRepo: notesRepo, journalRepo: journalRepo,
		journalSvc: journalSvc, settingsRepo: settingsRepo, auditRepo: auditRepo,
		closingSvc: closingSvc, fiscalYear: fiscalYear,
	}
}

// SetFiscalYear schaltet das aktive Geschäftsjahr um.
func (s *AppropriationService) SetFiscalYear(year int) { s.fiscalYear = year }

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
	// RequiredLegalReserve ist die Pflichtrücklage der UG (§ 5a Abs. 3 GmbHG);
	// null bei jeder anderen Rechtsform und bei der UG, deren Stammkapital
	// 25.000 Euro erreicht hat.
	RequiredLegalReserve domain.Cents `json:"requiredLegalReserve"`
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
	if year == 0 {
		year = s.fiscalYear - 1
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
	preview.RequiredLegalReserve = s.requiredLegalReserve(ctx, netIncome, yearResult, turnovers)

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
	// Die Pflichtrücklage wird vorgeschlagen und nicht erzwungen: die Vorschau
	// ist das leere Formular, das der Anwender vor sich hat, und sie ist der
	// einzige Weg, den Pflichtbetrag überhaupt zu erfahren. Bräche sie ab,
	// bekäme er beim ersten Aufruf eine Fehlermeldung statt der Zahl, die er
	// eintragen soll. Zurückgewiesen wird erst der Beschluss selbst
	// (BookAppropriation).
	if warning := legalReserveWarning(preview.RequiredLegalReserve, appropriation.LegalReserve); warning != "" {
		preview.Warnings = append(preview.Warnings, warning)
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
				"Die einbehaltene Kapitalertragsteuer ist bis zum zehnten Tag nach dem Zufluss "+
					"anzumelden und abzuführen (§ 44 Abs. 1 Satz 5 EStG). Buchfink erstellt die "+
					"Steueranmeldung nicht.")
		}
	}
	preview.Appropriation = appropriation
	preview.Explanation = appropriationExplanation(&appropriation)
	return preview, nil
}

// requiredLegalReserve rechnet die Pflichtrücklage der UG.
//
// § 5a Abs. 3 Satz 1 GmbHG verlangt „ein Viertel des um einen Verlustvortrag
// aus dem Vorjahr geminderten Jahresüberschusses". Beides steht nicht auf dem
// Vortragskonto: dort liegt zusätzlich, was frühere Jahre nicht verwendet
// haben. Wer daraus rechnete, verlangte bei einem Gewinnvortrag von 3.000 € und
// einem Jahresüberschuss von 1.000 € eine Rücklage von 1.000 € statt 250 €.
func (s *AppropriationService) requiredLegalReserve(
	ctx context.Context, netIncome, yearResult domain.Cents,
	turnovers map[string]domain.AccountTurnover,
) domain.Cents {
	if yearResult <= 0 || s.settingsRepo == nil {
		return 0
	}
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || settings == nil || !isEntrepreneurialCompany(settings.LegalForm) {
		return 0
	}
	// § 5a Abs. 3 Satz 1 GmbHG bindet die Pflicht an das Stammkapital: sie
	// entfällt, sobald es 25.000 Euro erreicht — dann ist aus der UG eine GmbH
	// geworden, auch wenn die Firma noch anders lautet.
	capital := turnovers[domain.AccountGezeichnetesKapital]
	if capital.Credit-capital.Debit >= 2500000 {
		return 0
	}
	// Was auf dem Vortragskonto über den Jahresüberschuss hinaus steht, stammt
	// aus früheren Jahren; ist es negativ, ist es der Verlustvortrag, der den
	// Jahresüberschuss mindert.
	base := yearResult
	if carried := netIncome - yearResult; carried < 0 {
		base += carried
	}
	if base <= 0 {
		return 0
	}
	return domain.MulRound(base, 1, 4)
}

// isEntrepreneurialCompany meldet, ob die Rechtsform eine Unternehmergesellschaft
// ist.
//
// Zuerst über den Katalog, denn dort steht die Schreibweise, die die
// Einstellungen anbieten. Für die freie Eingabe kommt der ausgeschriebene Name
// dazu und das Kürzel als eigenes Wort — „UG" als Teilzeichenkette zu suchen
// träfe irgendwann eine Rechtsform, die nichts damit zu tun hat.
func isEntrepreneurialCompany(legalForm string) bool {
	if form, ok := domain.LookupLegalForm(legalForm); ok {
		return form.Name == domain.LegalFormUG
	}
	lower := strings.ToLower(legalForm)
	if strings.Contains(lower, "unternehmergesellschaft") {
		return true
	}
	for _, word := range strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r)
	}) {
		if word == "ug" {
			return true
		}
	}
	return false
}

// legalReserveWarning nennt die Pflichtrücklage der UG, wenn der Beschluss
// weniger einstellt, als § 5a Abs. 3 GmbHG verlangt. Leerer Text heißt: alles
// in Ordnung.
func legalReserveWarning(required, planned domain.Cents) string {
	if required <= planned {
		return ""
	}
	return fmt.Sprintf(
		"§ 5a Abs. 3 GmbHG verlangt bei der UG (haftungsbeschränkt) eine Rücklage von einem Viertel "+
			"des Jahresüberschusses, hier %s €. Vorgesehen sind bisher %s €. Der Beschluss lässt "+
			"sich so nicht buchen.",
		required, planned)
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

// BookAppropriation hält den Beschluss fest und bucht ihn.
func (s *AppropriationService) BookAppropriation(
	ctx context.Context, year int, req AppropriationRequest,
) (*domain.Appropriation, error) {
	preview, err := s.PreviewAppropriation(ctx, year, req)
	if err != nil {
		return nil, err
	}
	// Hier — und erst hier — wird die Pflichtrücklage der UG erzwungen. Die
	// Vorschau nennt sie und warnt; gebucht wird ein Beschluss, der sie
	// unterschreitet, nicht: § 5a Abs. 3 Satz 1 GmbHG lässt der Gesellschaft
	// insoweit kein Wahlrecht.
	if warning := legalReserveWarning(
		preview.RequiredLegalReserve, preview.Appropriation.LegalReserve); warning != "" {
		return nil, fmt.Errorf("%s", warning)
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
					"Storniere zuerst seine Buchung", preview.Year, stored.DecisionDate)
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
				return nil, fmt.Errorf(
					"die Buchung %s wurde geschrieben, das Beschlussdokument aber nicht mit ihr "+
						"verbunden: %w", created.EntryNumber, err)
			}
		}
	}
	if err := s.appropriationRepo.Save(ctx, &appropriation); err != nil {
		return nil, fmt.Errorf("der Beschluss konnte nicht gespeichert werden: %w", err)
	}
	s.audit(ctx, "APPROPRIATION", preview.Year, fmt.Sprintf(
		"Ergebnisverwendung %d: %s", preview.Year, preview.Explanation))
	return &appropriation, nil
}

// Appropriation liefert den gespeicherten Beschluss eines Jahres, oder nil.
func (s *AppropriationService) Appropriation(ctx context.Context, year int) (*domain.Appropriation, error) {
	if year == 0 {
		year = s.fiscalYear - 1
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
	if year == 0 {
		year = s.fiscalYear
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
	if year == 0 {
		year = s.fiscalYear
	}
	entry := &domain.NotesText{Year: year, Section: section, Text: text}
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if err := s.notesRepo.Save(ctx, entry); err != nil {
		return nil, err
	}
	def, _ := domain.NotesSectionDefinitionFor(section)
	s.audit(ctx, "NOTES", year, fmt.Sprintf("Anhangabschnitt %q im Geschäftsjahr %d geändert", def.Label, year))
	return s.NotesTexts(ctx, year)
}

// CopyNotesInto übernimmt die Anhangtexte des Vorjahres als Vorlage.
//
// Die Bilanzierungs- und Bewertungsmethoden ändern sich selten, und ein leerer
// Anhang im neuen Jahr führt dazu, dass die Angaben schlicht fehlen. Kopiert
// wird nur, was noch nicht dasteht: eine bereits geschriebene Fassung darf eine
// Vorlage nicht überschreiben.
func (s *AppropriationService) CopyNotesInto(ctx context.Context, toYear int) (int, error) {
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
		s.audit(ctx, "NOTES", toYear, fmt.Sprintf(
			"%d Anhangtexte aus dem Geschäftsjahr %d als Vorlage übernommen", copied, toYear-1))
	}
	return copied, nil
}

func (s *AppropriationService) audit(ctx context.Context, entity string, year int, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, entity, fmt.Sprintf("%d", year), details)
}
