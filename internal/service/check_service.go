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

// CheckService ist der Prüfbericht vor der Festschreibung.
//
// Die Festschreibung ist der Punkt, ab dem sich nichts mehr ändern lässt. Was
// bis dahin fehlt — ein Beleg, eine Zuordnung, ein Saldo auf einem
// Interimskonto —, fehlt danach für immer und lässt sich nur noch über eine
// Korrekturbuchung im laufenden Zeitraum geradeziehen. Deshalb wird vorher
// geprüft und nicht hinterher, und deshalb blockieren die schweren Befunde.
//
// Übergehen bleibt möglich: es gibt Fälle, in denen ein Beleg nie kommt. Aber
// nur mit Begründung, und die Begründung steht am Lauf und im Protokoll — das
// ist der Kern eines internen Kontrollsystems (GoBD Rz. 100 ff.).
type CheckService struct {
	journalRepo        domain.JournalRepository
	receiptRepo        domain.ReceiptRepository
	bankRepo           domain.BankRepository
	invoiceRepo        domain.InvoiceRepository
	numberRepo         domain.NumberRangeRepository
	settingsRepo       domain.SettingsRepository
	festschreibungRepo domain.FestschreibungRepository
	vatReturnRepo      domain.VatReturnRepository
	checkRepo          domain.CheckRunRepository
	auditRepo          domain.AuditRepository

	accounts     AccountBalanceSource
	openItems    OpenItemSource
	depreciation PendingDepreciationSource
	provisions   ProvisionFindingSource
	closingSteps SkippedClosingStepSource
	// supplyEvidence und vatIDs tragen die beiden Regeln zur steuerfreien
	// innergemeinschaftlichen Lieferung: der Belegnachweis und die Bestätigung
	// der USt-IdNr. Ohne sie läuft der Prüflauf wie zuvor, nur ohne diese Regeln.
	supplyEvidence SupplyEvidenceSource
	vatIDs         VatIDStatusSource

	// now ist der Ausführungszeitpunkt des Laufs. Er steht als Feld, weil zwei
	// Regeln nicht nur den Stichtag, sondern auch den heutigen Tag brauchen —
	// und eine Regel, die „heute" fest verdrahtet, ließe sich nicht prüfen.
	now func() time.Time

	fiscalYear int
}

// AccountBalanceSource liefert den Kontenplan mit den Salden eines Jahres. Der
// Prüflauf braucht ihn für die Frage, ob jedes bebuchte Konto in der Bilanz
// ankommt.
type AccountBalanceSource interface {
	AccountsForYear(ctx context.Context, year int) ([]domain.Account, error)
}

// PendingDepreciationSource meldet die noch nicht gebuchte AfA eines Jahres.
type PendingDepreciationSource interface {
	PendingDepreciation(ctx context.Context, fiscalYear int) ([]DepreciationDue, error)
}

// ProvisionFindingSource meldet, wo die Abzinsung der Rückstellungen nicht mit
// dem Satz des Stichtagsmonats gerechnet werden konnte.
type ProvisionFindingSource interface {
	DiscountFindings(ctx context.Context, fiscalYear int) ([]domain.CheckFinding, error)
}

// SkippedClosingStepSource meldet die Bausteine des Abschlusses, die ein Jahr
// ausdrücklich übergeht.
type SkippedClosingStepSource interface {
	SkippedSteps(ctx context.Context, fiscalYear int) ([]SkippedClosingStep, error)
}

// NewCheckService wires the Prüflauf.
func NewCheckService(
	journalRepo domain.JournalRepository,
	receiptRepo domain.ReceiptRepository,
	bankRepo domain.BankRepository,
	invoiceRepo domain.InvoiceRepository,
	numberRepo domain.NumberRangeRepository,
	settingsRepo domain.SettingsRepository,
	festschreibungRepo domain.FestschreibungRepository,
	vatReturnRepo domain.VatReturnRepository,
	checkRepo domain.CheckRunRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *CheckService {
	return &CheckService{
		journalRepo:        journalRepo,
		receiptRepo:        receiptRepo,
		bankRepo:           bankRepo,
		invoiceRepo:        invoiceRepo,
		numberRepo:         numberRepo,
		settingsRepo:       settingsRepo,
		festschreibungRepo: festschreibungRepo,
		vatReturnRepo:      vatReturnRepo,
		checkRepo:          checkRepo,
		auditRepo:          auditRepo,
		now:                time.Now,
		fiscalYear:         fiscalYear,
	}
}

// SetAccountSource wires the chart with balances (Regel account_unmapped).
func (s *CheckService) SetAccountSource(src AccountBalanceSource) { s.accounts = src }

// SetOpenItemSource wires the open items (Regel duplicate_payment).
func (s *CheckService) SetOpenItemSource(src OpenItemSource) { s.openItems = src }

// SetDepreciationSource wires the Anlagenkartei (Regel depreciation_missing).
func (s *CheckService) SetDepreciationSource(src PendingDepreciationSource) { s.depreciation = src }

// SetProvisionSource wires die Rückstellungen (Regel provision_discount).
func (s *CheckService) SetProvisionSource(src ProvisionFindingSource) { s.provisions = src }

// SetClosingStepSource wires den Abschlussassistenten (Regel
// closing_step_skipped).
func (s *CheckService) SetClosingStepSource(src SkippedClosingStepSource) { s.closingSteps = src }

// SetFiscalYear updates the active fiscal year.
func (s *CheckService) SetFiscalYear(year int) { s.fiscalYear = year }

// CheckRequest ist der Auftrag an einen Prüflauf.
type CheckRequest struct {
	// CutoffDate ist der Stichtag: geprüft wird alles bis einschließlich hier.
	CutoffDate string
	// PeriodType ist der Anlass. "year" schaltet die Regeln zu, die nur vor der
	// Jahresfestschreibung gelten — die AfA und die Gliederungszuordnung sind
	// Fragen des Abschlusses und nicht des Monats.
	PeriodType string
	// OverrideReason ist die Begründung, mit der blockierende Befunde übergangen
	// werden. Sie wird am Lauf gespeichert.
	OverrideReason string
}

// Preview führt den Prüflauf aus, ohne ihn zu speichern.
//
// Der Bericht vor der Festschreibung ist eine Vorschau: er sagt, was die
// Festschreibung festhalten würde. Gespeichert wird der Lauf, an dem die
// Festschreibung hängt — sonst stünden je Festschreibung zwei Läufe im
// Protokoll, und die Begründung für ein Übergehen hinge an dem, den der
// Anwender gar nicht gesehen hat.
func (s *CheckService) Preview(ctx context.Context, req CheckRequest) (*domain.CheckRun, error) {
	return s.compute(ctx, req)
}

// Run führt den Prüflauf aus und speichert ihn.
func (s *CheckService) Run(ctx context.Context, req CheckRequest) (*domain.CheckRun, error) {
	run, err := s.compute(ctx, req)
	if err != nil {
		return nil, err
	}
	if s.checkRepo != nil {
		if err := s.checkRepo.Create(ctx, run); err != nil {
			return nil, fmt.Errorf("der Prüflauf konnte nicht gespeichert werden: %w", err)
		}
	}
	if s.auditRepo != nil {
		details := fmt.Sprintf("Prüflauf bis %s: %d Befunde (%d blockierend) über %d Buchungen, %d Belege, %d Bankumsätze",
			run.CutoffDate, len(run.Findings), len(run.Blocking()), run.CheckedEntries, run.CheckedReceipts, run.CheckedBankTx)
		if run.OverrideReason != "" {
			details += fmt.Sprintf(". Blockierende Befunde übergangen mit der Begründung: %s", run.OverrideReason)
		}
		_ = s.auditRepo.Log(ctx, domain.AuditActionIntegrityCheck, "CHECK_RUN", fmt.Sprintf("%d", run.ID), details)
	}
	return run, nil
}

// compute rechnet den Lauf, ohne ihn abzulegen.
func (s *CheckService) compute(ctx context.Context, req CheckRequest) (*domain.CheckRun, error) {
	if len(req.CutoffDate) != 10 {
		return nil, fmt.Errorf("der Prüflauf braucht einen Stichtag (erwartet JJJJ-MM-TT)")
	}
	run := &domain.CheckRun{
		FiscalYear: s.fiscalYear,
		CutoffDate: req.CutoffDate,
		PeriodType: req.PeriodType,
		// Der Zeitpunkt kommt aus derselben Uhr wie die Fristen des Laufs. Stünde
		// hier time.Now(), trüge ein Lauf, den ein Test auf einen bestimmten Tag
		// stellt, ein anderes Datum als seine eigenen Befunde.
		CreatedAt: s.clock(),
	}

	entries, err := s.journalRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("die Buchungen konnten nicht gelesen werden: %w", err)
	}
	upToCutoff := make([]domain.JournalEntry, 0, len(entries))
	for i := range entries {
		if entries[i].BookingDate <= req.CutoffDate {
			upToCutoff = append(upToCutoff, entries[i])
		}
	}
	run.CheckedEntries = len(upToCutoff)

	cfg, err := s.companySettings(ctx)
	if err != nil {
		return nil, err
	}

	receipts := s.allReceipts(ctx)
	reference := s.referenceDate(req.CutoffDate)

	// Leer statt nil: der Lauf geht als JSON an die Oberfläche, und ein nil-Slice
	// wird dort zu `null` — die Vorschau ohne Befund brächte die Ansicht zu Fall.
	findings := make([]domain.CheckFinding, 0, 8)
	findings = append(findings, s.checkEntriesWithoutReceipt(upToCutoff, entriesWithReceipt(receipts))...)
	findings = append(findings, s.checkDuplicateDocuments(upToCutoff)...)
	findings = append(findings, s.checkInterimBalances(upToCutoff, req.CutoffDate)...)

	run.CheckedReceipts = len(receipts)
	findings = append(findings, s.checkReceipts(receipts, req.CutoffDate, reference, cfg, s.pendingByDesign(ctx))...)

	bankFindings, bankCount := s.checkBank(ctx, req.CutoffDate)
	run.CheckedBankTx = bankCount
	findings = append(findings, bankFindings...)

	findings = append(findings, s.checkNumberGaps(ctx)...)
	findings = append(findings, s.checkOverpaidItems(ctx, req.CutoffDate)...)
	findings = append(findings, s.checkVatReturns(ctx, req.CutoffDate, reference, cfg)...)
	findings = append(findings, s.checkCommitOverdue(ctx, req.CutoffDate, cfg)...)
	findings = append(findings, s.checkSupplyEvidence(ctx)...)
	findings = append(findings, s.checkUnconfirmedSupplies(ctx)...)

	if req.PeriodType == "year" {
		findings = append(findings, s.checkAccountMapping(ctx)...)
		findings = append(findings, s.checkDepreciation(ctx)...)
		findings = append(findings, s.checkProvisionDiscounting(ctx)...)
		findings = append(findings, s.checkSkippedClosingSteps(ctx)...)
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity == domain.CheckBlocking
		}
		if findings[i].Rule != findings[j].Rule {
			return findings[i].Rule < findings[j].Rule
		}
		return findings[i].ObjectID < findings[j].ObjectID
	})
	run.Findings = findings
	run.EnsureLists()

	// Die Begründung gehört an den Lauf, der etwas zu übergehen hatte. Steht sie
	// an einem Lauf ohne blockierende Befunde, behauptet das Protokoll ein
	// Übergehen, das nie stattgefunden hat — und wer später prüft, sucht nach
	// einem Befund, den es nie gab.
	if run.HasBlocking() {
		run.OverrideReason = strings.TrimSpace(req.OverrideReason)
	}
	return run, nil
}

// EnsureCommittable führt den Prüflauf aus und entscheidet, ob festgeschrieben
// werden darf.
//
// Blockierende Befunde verhindern die Festschreibung. Übergehen bleibt möglich —
// es gibt Fälle, in denen ein Beleg nie kommt —, aber nur mit Begründung, und
// die steht am Lauf und im Protokoll. Ein Knopf „trotzdem" ohne Begründung wäre
// kein internes Kontrollsystem, sondern seine Umgehung.
func (s *CheckService) EnsureCommittable(ctx context.Context, req CheckRequest) (*domain.CheckRun, error) {
	run, err := s.Run(ctx, req)
	if err != nil {
		return nil, err
	}
	if run.HasBlocking() && strings.TrimSpace(req.OverrideReason) == "" {
		return run, fmt.Errorf(
			"der Prüfbericht zum %s hat %d blockierende Befunde: %s. "+
				"Behebe sie oder schreibe mit einer Begründung fest — die Begründung wird im Protokoll festgehalten",
			req.CutoffDate, len(run.Blocking()), run.BlockingSummary())
	}
	return run, nil
}

// Runs liefert die gespeicherten Läufe eines Jahres, der jüngste zuerst.
func (s *CheckService) Runs(ctx context.Context, year int) ([]domain.CheckRun, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	if s.checkRepo == nil {
		return []domain.CheckRun{}, nil
	}
	return s.checkRepo.FindByFiscalYear(ctx, year)
}

// Latest liefert den jüngsten Lauf des aktiven Jahres.
func (s *CheckService) Latest(ctx context.Context) (*domain.CheckRun, error) {
	if s.checkRepo == nil {
		return nil, nil
	}
	return s.checkRepo.Latest(ctx, s.fiscalYear)
}

// -------------------------------------------------------------
// Die einzelnen Regeln
// -------------------------------------------------------------

// receiptlessSources sind die Quellen, bei denen ein Beleg zu erwarten ist.
//
// Zahlung, AfA, Saldenvortrag und Abschlussbuchung sind Folgebuchungen: ihr
// Nachweis ist die Buchung, auf die sie sich beziehen, oder eine Berechnung, die
// das Programm selbst führt. Für sie einen Beleg zu verlangen, hieße, den
// Anwender Papier erfinden zu lassen.
func expectsReceipt(source domain.EntrySource) bool {
	switch source {
	case domain.EntrySourceManual, domain.EntrySourceReceipt, domain.EntrySourceInvoice:
		return true
	}
	return false
}

// entriesWithReceipt sammelt die Buchungen, auf die ein Beleg zeigt.
//
// Der Nachweis zählt in beide Richtungen. Der Regelfall ist die Buchung mit
// ReceiptID; der Ausgangsbeleg einer Rechnung wird aber erst nach der Buchung
// versiegelt und trägt seitdem deren Nummer. Für Bestände aus der Zeit vor
// dieser Prüfung ist das die einzige Verbindung — und ein Beleg, der auf die
// Buchung zeigt, ist ein Beleg zu ihr.
func entriesWithReceipt(receipts []domain.Receipt) map[uint]bool {
	out := map[uint]bool{}
	for i := range receipts {
		r := &receipts[i]
		if r.JournalEntryID != nil && r.Status != domain.ReceiptStatusDiscarded {
			out[*r.JournalEntryID] = true
		}
	}
	return out
}

func (s *CheckService) checkEntriesWithoutReceipt(entries []domain.JournalEntry, withReceipt map[uint]bool) []domain.CheckFinding {
	var out []domain.CheckFinding
	for i := range entries {
		e := &entries[i]
		if !expectsReceipt(e.Source) || e.ReceiptID != nil || withReceipt[e.ID] {
			continue
		}
		// Die Generalumkehr trägt den Beleg der Ursprungsbuchung; hat die
		// keinen, ist der Befund dort schon gemeldet.
		if e.Kind == domain.EntryKindReversal {
			continue
		}
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleEntryWithoutReceipt,
			Severity:   domain.CheckBlocking,
			ObjectType: "JOURNAL_ENTRY",
			ObjectID:   fmt.Sprintf("%d", e.ID),
			ObjectName: e.EntryNumber,
			Message: fmt.Sprintf("Buchung %s vom %s (%s) hat keinen Beleg",
				e.EntryNumber, e.BookingDate, e.Description),
			Reference: "§ 238 Abs. 1 Satz 3 HGB, GoBD Rz. 61",
		})
	}
	return out
}

// checkDuplicateDocuments findet dieselbe Rechnung zweimal.
//
// Gleicher Geschäftspartner, gleiche Belegnummer, gleicher Bruttobetrag: das ist
// keine zweite Leistung, das ist dieselbe Rechnung ein zweites Mal erfasst — und
// mit ihr ein zweiter Vorsteuerabzug.
//
// Verglichen werden nur Buchungen. domain.Receipt trägt weder Belegnummer des
// Ausstellers noch Geschäftspartner — zwei Bilder desselben Belegs sind an der
// Ablage nicht als Dubletten zu erkennen. Der doppelt erfasste Beleg fällt
// deshalb erst auf, wenn er gebucht ist; das ist auch der Zeitpunkt, an dem er
// schadet.
func (s *CheckService) checkDuplicateDocuments(entries []domain.JournalEntry) []domain.CheckFinding {
	type key struct {
		contactID uint
		document  string
		amount    domain.Cents
	}
	seen := map[key][]string{}
	order := make([]key, 0, len(entries))

	for i := range entries {
		e := &entries[i]
		if e.DocumentNumber == "" || e.ContactID == nil || e.Kind == domain.EntryKindReversal {
			continue
		}
		k := key{contactID: *e.ContactID, document: e.DocumentNumber, amount: e.GrossAmount()}
		if _, ok := seen[k]; !ok {
			order = append(order, k)
		}
		seen[k] = append(seen[k], e.EntryNumber)
	}

	var out []domain.CheckFinding
	for _, k := range order {
		numbers := seen[k]
		if len(numbers) < 2 {
			continue
		}
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleDuplicateReceipt,
			Severity:   domain.CheckWarning,
			ObjectType: "JOURNAL_ENTRY",
			ObjectID:   k.document,
			ObjectName: k.document,
			Message: fmt.Sprintf(
				"Belegnummer %s über %s € kommt %d-mal vor (%s) — möglicherweise doppelt erfasst",
				k.document, k.amount, len(numbers), strings.Join(numbers, ", ")),
			Reference: "§ 146 Abs. 1 AO",
		})
	}
	return out
}

// checkInterimBalances prüft die Durchgangskonten.
//
// Geldtransit (1460) und durchlaufende Posten (1370) sind Konten, die zum
// Stichtag leer sein müssen: sie halten einen Betrag nur zwischen zwei
// Buchungen. Ein Saldo darauf heißt, dass die zweite Buchung fehlt — und in der
// Bilanz stünde ein Betrag, den niemand einem Posten zuordnen kann.
func (s *CheckService) checkInterimBalances(entries []domain.JournalEntry, cutoff string) []domain.CheckFinding {
	balances := map[string]domain.Cents{}
	for i := range entries {
		for _, line := range entries[i].Lines {
			switch line.Account {
			case domain.AccountGeldtransit, domain.AccountDurchlaufendePosten:
				if line.Side == domain.SideDebit {
					balances[line.Account] += line.Amount
				} else {
					balances[line.Account] -= line.Amount
				}
			}
		}
	}

	var out []domain.CheckFinding
	for _, account := range []string{domain.AccountGeldtransit, domain.AccountDurchlaufendePosten} {
		balance := balances[account]
		if balance == 0 {
			continue
		}
		name := "Geldtransit"
		if account == domain.AccountDurchlaufendePosten {
			name = "Durchlaufende Posten"
		}
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleInterimBalance,
			Severity:   domain.CheckBlocking,
			ObjectType: "ACCOUNT",
			ObjectID:   account,
			ObjectName: name,
			Message: fmt.Sprintf(
				"Konto %s (%s) trägt am %s einen Saldo von %s € — ein Durchgangskonto muss zum Stichtag ausgeglichen sein",
				account, name, cutoff, balance),
			Reference: "§ 246 Abs. 1 HGB",
		})
	}
	return out
}

// allReceipts liest die Belege des Jahres einmal für den ganzen Lauf: zwei
// Regeln fragen dieselbe Liste.
func (s *CheckService) allReceipts(ctx context.Context) []domain.Receipt {
	if s.receiptRepo == nil {
		return nil
	}
	receipts, err := s.receiptRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return nil
	}
	return receipts
}

// pendingByDesign nennt die Ausgangsbelege, zu denen es noch keine Buchung
// geben kann.
//
// Es ist genau ein Fall, und er folgt aus dem Gesetz: bei einer
// Abschlagsrechnung entsteht die Steuer erst mit der Vereinnahmung
// (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG), und beim Ausstellen gibt es
// deshalb nichts zu buchen. Dasselbe gilt für ihr Stornodokument, solange die
// Abschlagsrechnung selbst ungebucht war: eine Buchung, die es nicht gibt,
// lässt sich nicht durch Generalumkehr zurücknehmen.
//
// Ohne diese Ausnahme meldete der Prüflauf jede Abschlagsrechnung als
// abgelegten, ungebuchten Beleg und sperrte damit die Festschreibung — für
// einen Zustand, der richtig ist.
func (s *CheckService) pendingByDesign(ctx context.Context) map[uint]string {
	out := map[uint]string{}
	if s.invoiceRepo == nil {
		return out
	}
	invoices, err := s.invoiceRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return out
	}
	byID := map[uint]*domain.Invoice{}
	for i := range invoices {
		byID[invoices[i].ID] = &invoices[i]
	}
	for i := range invoices {
		inv := &invoices[i]
		if inv.ReceiptID == nil || inv.JournalEntryID != nil {
			continue
		}
		switch inv.ResolvedKind() {
		case domain.InvoiceKindAdvance:
			out[*inv.ReceiptID] = fmt.Sprintf(
				"trägt die Abschlagsrechnung %s. Sie wird erst mit der Vereinnahmung gebucht "+
					"(§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG); bis dahin steht sie als offener Posten",
				inv.InvoiceNumber)
		case domain.InvoiceKindCancellation:
			if inv.CorrectsInvoiceID == nil {
				continue
			}
			original, ok := byID[*inv.CorrectsInvoiceID]
			if !ok || original.ResolvedKind() != domain.InvoiceKindAdvance || original.JournalEntryID != nil {
				continue
			}
			out[*inv.ReceiptID] = fmt.Sprintf(
				"storniert die ungebuchte Abschlagsrechnung %s; es gibt keine Buchung zurückzunehmen",
				original.InvoiceNumber)
		}
	}
	return out
}

func (s *CheckService) checkReceipts(
	receipts []domain.Receipt, cutoff, reference string, cfg *domain.CompanySettings,
	pendingByDesign map[uint]string,
) []domain.CheckFinding {
	captureDays := cfg.ReceiptCaptureDays
	if captureDays <= 0 {
		captureDays = 10
	}
	// Die Erfassungsfrist läuft gegen den Bezugstag und nicht gegen den
	// Stichtag: ein Beleg, der seit sieben Wochen liegt, ist überfällig, auch
	// wenn der Lauf einen zurückliegenden Monat prüft.
	overdueBefore := addDays(reference, -captureDays)

	var out []domain.CheckFinding
	for i := range receipts {
		r := &receipts[i]
		if r.Status != domain.ReceiptStatusFiled {
			continue
		}
		// Ein Kontoauszug ist ein Beleg ohne Buchungspflicht: gebucht werden
		// die einzelnen Umsätze daraus, nicht der Auszug. Ohne diese Zeile
		// meldete der Prüflauf jeden archivierten Auszug als ungebucht und
		// blockierte damit die Festschreibung.
		if !r.Kind.RequiresBooking() {
			continue
		}
		// Maßgeblich ist der Belegeingang, hilfsweise das Ablagedatum. Ein
		// Belegdatum trägt domain.Receipt nicht; es steht erst an der Buchung —
		// und die gibt es hier gerade nicht.
		relevant := r.ReceivedAt
		if relevant == "" {
			relevant = r.CreatedAt.Format("2006-01-02")
		}

		severity := domain.CheckWarning
		if relevant <= cutoff {
			// Ein Beleg, der in den festzuschreibenden Zeitraum gehört, muss vor
			// der Festschreibung gebucht sein — danach nimmt der Zeitraum keine
			// Buchung mehr auf.
			severity = domain.CheckBlocking
		}
		message := fmt.Sprintf("Beleg %s (Eingang %s) ist abgelegt, aber nicht gebucht",
			r.ReceiptNumber, relevant)
		reference := "GoBD Rz. 47"
		// Der Abschlagsfall ist keine Versäumnis, sondern die Rechtslage. Er
		// bleibt sichtbar — als Hinweis, nicht als Sperre.
		if note, ok := pendingByDesign[r.ID]; ok {
			severity = domain.CheckWarning
			message = fmt.Sprintf("Beleg %s %s", r.ReceiptNumber, note)
			reference = "§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG"
		}
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleReceiptUnbooked,
			Severity:   severity,
			ObjectType: "RECEIPT",
			ObjectID:   fmt.Sprintf("%d", r.ID),
			ObjectName: r.ReceiptNumber,
			Message:    message,
			Reference:  reference,
		})

		if _, byDesign := pendingByDesign[r.ID]; byDesign {
			continue
		}
		if relevant != "" && relevant < overdueBefore {
			out = append(out, domain.CheckFinding{
				Rule:       domain.CheckRuleReceiptOverdue,
				Severity:   domain.CheckWarning,
				ObjectType: "RECEIPT",
				ObjectID:   fmt.Sprintf("%d", r.ID),
				ObjectName: r.ReceiptNumber,
				Message: fmt.Sprintf(
					"Beleg %s liegt seit %s unbearbeitet — unbare Geschäftsvorfälle sind innerhalb von %d Tagen zu erfassen",
					r.ReceiptNumber, relevant, captureDays),
				Reference: "GoBD Rz. 47",
			})
		}
	}
	return out
}

func (s *CheckService) checkBank(ctx context.Context, cutoff string) ([]domain.CheckFinding, int) {
	if s.bankRepo == nil {
		return nil, 0
	}
	transactions, err := s.bankRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return nil, 0
	}
	var out []domain.CheckFinding
	for i := range transactions {
		tx := &transactions[i]
		if tx.BookingDate > cutoff || tx.MatchStatus != domain.MatchStatusUnmatched {
			continue
		}
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleBankUnmatched,
			Severity:   domain.CheckBlocking,
			ObjectType: "BANK_TX",
			ObjectID:   fmt.Sprintf("%d", tx.ID),
			ObjectName: tx.CounterpartyName,
			Message: fmt.Sprintf("Bankumsatz vom %s über %s € (%s) ist keiner Buchung zugeordnet",
				tx.BookingDate, tx.Amount, tx.CounterpartyName),
			Reference: "§ 146 Abs. 1 AO",
		})
	}
	return out, len(transactions)
}

// checkNumberGaps vergleicht den Zähler gegen die vorhandenen Nummern.
//
// Eine Lücke ist kein Fehler für sich — sie entsteht, wenn eine Vergabe
// zurückgerollt wurde. Sie muss aber erklärbar sein: § 14 Abs. 4 Nr. 4 UStG
// verlangt eine fortlaufende Rechnungsnummer, und eine Lücke, die niemand
// bemerkt hat, ist im Zweifel eine unterschlagene Rechnung.
func (s *CheckService) checkNumberGaps(ctx context.Context) []domain.CheckFinding {
	if s.numberRepo == nil {
		return nil
	}
	var out []domain.CheckFinding

	if s.receiptRepo != nil {
		receipts, err := s.receiptRepo.FindAll(ctx, s.fiscalYear)
		if err == nil {
			used := map[int64]bool{}
			prefix := fmt.Sprintf("ER-%d-", s.fiscalYear)
			for i := range receipts {
				if seq, ok := trailingNumber(receipts[i].ReceiptNumber, prefix); ok {
					used[seq] = true
				}
			}
			out = append(out, s.gapFinding(ctx, domain.NumberRangeReceipt, used, "Belegnummern",
				func(seq int64) string { return domain.FormatReceiptNumber(s.fiscalYear, seq) })...)
		}
	}

	if s.invoiceRepo != nil {
		invoices, err := s.invoiceRepo.FindAll(ctx, s.fiscalYear)
		if err == nil {
			used := map[int64]bool{}
			prefix := fmt.Sprintf("RE-%d-", s.fiscalYear)
			for i := range invoices {
				if seq, ok := trailingNumber(invoices[i].InvoiceNumber, prefix); ok {
					used[seq] = true
				}
			}
			out = append(out, s.gapFinding(ctx, domain.NumberRangeInvoice, used, "Rechnungsnummern",
				func(seq int64) string { return domain.FormatInvoiceNumber(s.fiscalYear, seq) })...)
		}
	}
	return out
}

func (s *CheckService) gapFinding(
	ctx context.Context, key domain.NumberRangeKey, used map[int64]bool, label string,
	format func(int64) string,
) []domain.CheckFinding {
	next, err := s.numberRepo.Peek(ctx, key, s.fiscalYear)
	if err != nil || next <= 1 {
		return nil
	}
	var missing []string
	for seq := int64(1); seq < next; seq++ {
		if !used[seq] {
			missing = append(missing, format(seq))
		}
	}
	if len(missing) == 0 {
		return nil
	}
	shown := missing
	suffix := ""
	if len(shown) > 5 {
		shown, suffix = shown[:5], fmt.Sprintf(" und %d weitere", len(missing)-5)
	}
	return []domain.CheckFinding{{
		Rule:       domain.CheckRuleNumberGap,
		Severity:   domain.CheckWarning,
		ObjectType: "NUMBER_RANGE",
		ObjectID:   string(key),
		ObjectName: label,
		Message: fmt.Sprintf("In den %s fehlen %d Nummern: %s%s",
			label, len(missing), strings.Join(shown, ", "), suffix),
		Reference: "§ 14 Abs. 4 Nr. 4 UStG, GoBD Rz. 36",
	}}
}

// checkOverpaidItems findet den offenen Posten, auf den mehr gezahlt wurde, als
// er ausmacht — der übliche Weg, auf dem eine Rechnung zweimal überwiesen wird.
func (s *CheckService) checkOverpaidItems(ctx context.Context, cutoff string) []domain.CheckFinding {
	if s.openItems == nil {
		return nil
	}
	items, err := s.openItems.OpenItemsAt(ctx, cutoff)
	if err != nil {
		return nil
	}
	var out []domain.CheckFinding
	for i := range items {
		item := &items[i]
		if item.SettledAmount <= item.GrossAmount {
			continue
		}
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleDuplicatePayment,
			Severity:   domain.CheckWarning,
			ObjectType: "JOURNAL_ENTRY",
			ObjectID:   fmt.Sprintf("%d", item.EntryID),
			ObjectName: item.EntryNumber,
			Message: fmt.Sprintf(
				"Auf den Posten %s (%s, %s €) sind %s € zugeordnet — %s € zu viel",
				item.EntryNumber, item.ContactName, item.GrossAmount,
				item.SettledAmount, item.SettledAmount-item.GrossAmount),
			Reference: "§ 146 Abs. 1 AO",
		})
	}
	return out
}

// checkVatReturns meldet fällige Voranmeldungszeiträume ohne bestätigte
// Übermittlung.
//
// Gemessen wird an der Fälligkeit und nicht am Ende des Zeitraums. Wer ein
// Quartal zu seinem letzten Tag festschreibt, konnte die Anmeldung noch gar
// nicht abgeben — und bestätigen kann er sie erst nach der Festschreibung. Ein
// Hinweis an dieser Stelle beanstandete die Reihenfolge, die Buchfink selbst
// vorschreibt.
func (s *CheckService) checkVatReturns(
	ctx context.Context, cutoff, reference string, cfg *domain.CompanySettings,
) []domain.CheckFinding {
	if s.vatReturnRepo == nil {
		return nil
	}
	// Wer vom Voranmeldungsverfahren befreit ist (§ 18 Abs. 2 Satz 3 UStG), gibt
	// keine Voranmeldung ab, sondern nur die Jahreserklärung. Ein Befund
	// „Voranmeldung fehlt" mahnte hier eine Pflicht an, die es nicht gibt — und
	// die Fristenliste (DeadlineService.vatDeadlines) schließt denselben Fall
	// bewusst aus. Prüflauf und Fristenliste dürfen sich nicht widersprechen.
	periodType := vatPeriodTypeOf(cfg)
	if periodType == domain.VatPeriodYear {
		return nil
	}
	saved, err := s.vatReturnRepo.FindByFiscalYear(ctx, s.fiscalYear)
	if err != nil {
		return nil
	}
	submitted := map[string]bool{}
	for i := range saved {
		if saved[i].Status == domain.VatReturnSubmitted {
			submitted[saved[i].PeriodKey] = true
		}
	}

	var out []domain.CheckFinding
	for _, p := range accounting.VatPeriodsOfYear(s.fiscalYear, periodType) {
		due := accounting.VatDueDate(p, cfg.PermanentExtension)
		if p.To > cutoff || submitted[p.Key] || due == "" || due >= reference {
			continue
		}
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleVatReturnMissing,
			Severity:   domain.CheckWarning,
			ObjectType: "VAT_PERIOD",
			ObjectID:   p.Key,
			ObjectName: p.Label,
			Message: fmt.Sprintf("Für %s ist keine übermittelte Umsatzsteuer-Voranmeldung erfasst (fällig am %s)",
				p.Label, due),
			Reference: "§ 18 Abs. 1 UStG",
		})
	}
	return out
}

// checkCommitOverdue meldet den Monat, dessen Folgemonat abgelaufen ist und der
// immer noch nicht festgeschrieben wurde.
func (s *CheckService) checkCommitOverdue(ctx context.Context, cutoff string, cfg *domain.CompanySettings) []domain.CheckFinding {
	if s.festschreibungRepo == nil {
		return nil
	}
	committed, err := s.festschreibungRepo.LatestCutoff(ctx, s.fiscalYear)
	if err != nil {
		return nil
	}
	for _, p := range accounting.VatPeriodsOfYear(s.fiscalYear, domain.VatPeriodMonth) {
		if committed >= p.To {
			continue
		}
		// Der Folgemonat muss abgelaufen sein, sonst wäre die Festschreibung
		// nicht überfällig, sondern bloß noch nicht dran.
		deadline := addDays(endOfNextMonth(p.To), cfg.CommitGraceDays)
		if cutoff <= deadline {
			continue
		}
		return []domain.CheckFinding{{
			Rule:       domain.CheckRuleCommitOverdue,
			Severity:   domain.CheckWarning,
			ObjectType: "PERIOD",
			ObjectID:   p.Key,
			ObjectName: p.Label,
			Message: fmt.Sprintf(
				"%s ist noch nicht festgeschrieben, obwohl der Folgemonat am %s abgelaufen ist",
				p.Label, deadline),
			Reference: "GoBD Rz. 107",
		}}
	}
	return nil
}

// checkAccountMapping meldet bebuchte Konten ohne Gliederungsposition. Vor der
// Jahresfestschreibung ist das blockierend: ein Konto ohne Position verschwindet
// aus der Bilanz, und der Abschluss wäre unvollständig.
func (s *CheckService) checkAccountMapping(ctx context.Context) []domain.CheckFinding {
	if s.accounts == nil {
		return nil
	}
	accounts, err := s.accounts.AccountsForYear(ctx, s.fiscalYear)
	if err != nil {
		return nil
	}
	var out []domain.CheckFinding
	for i := range accounts {
		acc := &accounts[i]
		if acc.DebitSum == 0 && acc.CreditSum == 0 {
			continue
		}
		if acc.StatementType == "Statistisch" || domain.IsCarryForwardAccount(acc.Number) {
			continue
		}
		if _, ok := accounting.StatementKeyForAccount(*acc); ok {
			continue
		}
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleAccountUnmapped,
			Severity:   domain.CheckBlocking,
			ObjectType: "ACCOUNT",
			ObjectID:   acc.Number,
			ObjectName: acc.Name,
			Message: fmt.Sprintf(
				"Konto %s (%s) trägt einen Saldo, hat aber keine Gliederungsposition — es erschiene in keinem Posten des Abschlusses",
				acc.Number, acc.Name),
			Reference: "§§ 266, 275 HGB",
		})
	}
	return out
}

// checkDepreciation überführt die bisherige Prüfung vor der Jahresfestschreibung
// in den Prüflauf. Die AfA ist eine Abschlussbuchung zum Bilanzstichtag und
// lässt sich nach der Festschreibung nicht mehr nachholen.
func (s *CheckService) checkDepreciation(ctx context.Context) []domain.CheckFinding {
	if s.depreciation == nil {
		return nil
	}
	pending, err := s.depreciation.PendingDepreciation(ctx, s.fiscalYear)
	if err != nil || len(pending) == 0 {
		return nil
	}
	out := make([]domain.CheckFinding, 0, len(pending))
	for _, p := range pending {
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleDepreciationMissing,
			Severity:   domain.CheckBlocking,
			ObjectType: "ASSET",
			ObjectID:   fmt.Sprintf("%d", p.AssetID),
			ObjectName: p.InventoryNumber,
			Message: fmt.Sprintf(
				"Für %s %s ist die Abschreibung von %s € noch nicht gebucht",
				p.InventoryNumber, p.Name, p.Due),
			Reference: "§ 253 Abs. 3 HGB, § 7 EStG",
		})
	}
	return out
}

// checkProvisionDiscounting nimmt die Befunde der Rückstellungsbuchhaltung auf.
//
// Sie entstehen bei der Bildung, wo sie in der Vorschau stehen — aber gelesen
// wird die Vorschau einmal, und geprüft wird das Jahr am Ende. Ein Hinweis, den
// nur sieht, wer gerade bucht, ist keine Prüfung.
func (s *CheckService) checkProvisionDiscounting(ctx context.Context) []domain.CheckFinding {
	if s.provisions == nil {
		return nil
	}
	findings, err := s.provisions.DiscountFindings(ctx, s.fiscalYear)
	if err != nil {
		// Ein Fehler ist kein „keine Befunde": stillschweigend nichts zu melden
		// hieße, dem Anwender einen geprüften Abschluss zu zeigen, der ungeprüft
		// ist. Der Fehler selbst wird deshalb zum Hinweis.
		return []domain.CheckFinding{{
			Rule:       domain.CheckRuleProvisionDiscount,
			Severity:   domain.CheckWarning,
			ObjectType: "PROVISION",
			Message: fmt.Sprintf(
				"Die Rückstellungen konnten nicht auf ihre Abzinsung geprüft werden: %v", err),
			Reference: "§ 253 Abs. 2 HGB",
		}}
	}
	return findings
}

// checkSkippedClosingSteps nennt die übersprungenen Bausteine des Abschlusses.
//
// Übergehen ist erlaubt — ein Jahr ohne Rückstellungen gibt es —, aber es ist
// eine Entscheidung mit Grund, und der Grund gehört dorthin, wo der Abschluss
// beurteilt wird. Ohne diese Regel stünde er nur in der Schrittliste, die nach
// der Festschreibung niemand mehr aufschlägt.
func (s *CheckService) checkSkippedClosingSteps(ctx context.Context) []domain.CheckFinding {
	if s.closingSteps == nil {
		return nil
	}
	skipped, err := s.closingSteps.SkippedSteps(ctx, s.fiscalYear)
	if err != nil || len(skipped) == 0 {
		return nil
	}
	out := make([]domain.CheckFinding, 0, len(skipped))
	for _, step := range skipped {
		message := fmt.Sprintf(
			"Der Abschlussschritt %q wurde übersprungen: %s", step.Label, step.Reason)
		if step.ChangedOn != "" {
			message = fmt.Sprintf(
				"Der Abschlussschritt %q wurde am %s übersprungen: %s",
				step.Label, step.ChangedOn, step.Reason)
		}
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleClosingStepSkipped,
			Severity:   domain.CheckWarning,
			ObjectType: "CLOSING_STEP",
			ObjectID:   string(step.Key),
			ObjectName: step.Label,
			Message:    message,
			Reference:  "GoBD Rz. 100 ff.",
		})
	}
	return out
}

// clock ist der Ausführungszeitpunkt des Laufs; ohne gesetzte Uhr die Systemzeit.
func (s *CheckService) clock() time.Time {
	if s.now == nil {
		return time.Now()
	}
	return s.now()
}

// referenceDate ist der Tag, an dem eine Frist gemessen wird: das spätere von
// Stichtag und Ausführungstag.
//
// Der Lauf ist eine Aussage zum Stichtag, „überfällig" aber eine Aussage über
// heute. Ein Lauf zum 31. März, der am 15. Mai läuft, muss den Beleg vom
// 25. März als überfällig melden — am Stichtag gemessen läge er erst sechs Tage.
func (s *CheckService) referenceDate(cutoff string) string {
	if s.now == nil {
		return cutoff
	}
	today := s.now().Format("2006-01-02")
	if today > cutoff {
		return today
	}
	return cutoff
}

func (s *CheckService) companySettings(ctx context.Context) (*domain.CompanySettings, error) {
	if s.settingsRepo == nil {
		return &domain.CompanySettings{VatPeriod: "quarter", ReceiptCaptureDays: 10}, nil
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("die Unternehmensdaten konnten nicht gelesen werden: %w", err)
	}
	return cfg, nil
}

// trailingNumber liest die laufende Nummer aus einer Nummer mit festem Präfix.
func trailingNumber(number, prefix string) (int64, bool) {
	if !strings.HasPrefix(number, prefix) {
		return 0, false
	}
	seq, err := strconv.ParseInt(strings.TrimPrefix(number, prefix), 10, 64)
	if err != nil {
		return 0, false
	}
	return seq, true
}

func addDays(iso string, days int) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

// endOfNextMonth ist der letzte Tag des Monats, der auf den Monat des Datums
// folgt.
func endOfNextMonth(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return first.AddDate(0, 2, -1).Format("2006-01-02")
}

// -------------------------------------------------------------------------
// Die beiden Regeln zur steuerfreien innergemeinschaftlichen Lieferung
// -------------------------------------------------------------------------

// SupplyEvidenceSource liefert den Nachweisstand der steuerfreien
// innergemeinschaftlichen Lieferungen eines Jahres.
type SupplyEvidenceSource interface {
	Report(ctx context.Context, year int) (*SupplyEvidenceReport, error)
}

// VatIDStatusSource beantwortet ohne Netzaufruf, ob für einen Geschäftspartner
// eine gültige Bestätigung der USt-IdNr. vorliegt.
type VatIDStatusSource interface {
	StatusForContact(ctx context.Context, contactID uint) (*VatIDStatus, error)
}

// SetSupplyEvidenceSource wires den Belegnachweis (Regel
// ic_supply_evidence_missing).
func (s *CheckService) SetSupplyEvidenceSource(src SupplyEvidenceSource) { s.supplyEvidence = src }

// SetVatIDStatusSource wires die Bestätigungsabfrage (Regel
// ic_supply_unconfirmed).
func (s *CheckService) SetVatIDStatusSource(src VatIDStatusSource) { s.vatIDs = src }

// checkSupplyEvidence meldet die steuerfreien ig. Lieferungen ohne
// vollständigen Belegnachweis.
//
// Als Warnung und nicht blockierend: der Nachweis darf nachgereicht werden, und
// er wird es in der Praxis auch — der Frachtbrief kommt nach der Rechnung. Was
// nicht passieren darf, ist dass er in Vergessenheit gerät, bis der Prüfer
// danach fragt und die Befreiung rückwirkend entfällt. Deshalb nennt die
// Meldung die Frist.
func (s *CheckService) checkSupplyEvidence(ctx context.Context) []domain.CheckFinding {
	if s.supplyEvidence == nil {
		return nil
	}
	report, err := s.supplyEvidence.Report(ctx, s.fiscalYear)
	if err != nil {
		// Ein Fehler ist kein „nichts gefunden": ein ungeprüfter Nachweisstand,
		// der wie ein geprüfter aussieht, ist schlimmer als gar keine Regel.
		return []domain.CheckFinding{{
			Rule:       domain.CheckRuleICSupplyEvidenceMissing,
			Severity:   domain.CheckWarning,
			ObjectType: "INVOICE",
			Message: fmt.Sprintf(
				"Der Belegnachweis der innergemeinschaftlichen Lieferungen ließ sich nicht prüfen: %v",
				err),
			Reference: "§ 17a UStDV",
		}}
	}
	out := make([]domain.CheckFinding, 0, report.Incomplete)
	for _, row := range report.Rows {
		if row.Status.Fulfilled {
			continue
		}
		missing := "der Nachweis ist nicht vollständig"
		if len(row.Status.Missing) > 0 {
			missing = "es fehlt " + strings.Join(row.Status.Missing, "; ")
		}
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleICSupplyEvidenceMissing,
			Severity:   domain.CheckWarning,
			ObjectType: "INVOICE",
			ObjectID:   fmt.Sprintf("%d", row.InvoiceID),
			ObjectName: row.InvoiceNumber,
			Message: fmt.Sprintf(
				"Die steuerfreie innergemeinschaftliche Lieferung %s an %s vom %s hat keinen "+
					"vollständigen Belegnachweis: %s. Er ist bis zur Abgabe der Voranmeldung des "+
					"Zeitraums zu führen, in dem die Lieferung ausgeführt wurde — fehlt er, ist die "+
					"Lieferung steuerpflichtig, und die Steuer schuldest du aus einer Rechnung, die "+
					"keine ausweist.",
				row.InvoiceNumber, row.ContactName, row.Date, missing),
			Reference: "§ 17a Abs. 1 UStDV",
		})
	}
	return out
}

// checkUnconfirmedSupplies meldet die steuerfreien ig. Lieferungen ohne
// bestätigte USt-IdNr. des Abnehmers.
//
// Das ist der zugesagte Folgebefund zur Übersteuerung: wer eine Rechnung
// ausstellt, während das Bundeszentralamt nicht antwortet, tut das mit einem
// festgehaltenen Grund — und findet die Rechnung im nächsten Prüflauf wieder,
// damit die Abfrage nachgeholt wird. § 6a Abs. 1 Satz 1 Nr. 4 UStG macht die
// gültige Nummer zur materiellen Voraussetzung; ein Grund ersetzt sie nicht.
func (s *CheckService) checkUnconfirmedSupplies(ctx context.Context) []domain.CheckFinding {
	if s.vatIDs == nil || s.invoiceRepo == nil {
		return nil
	}
	invoices, err := s.invoiceRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return nil
	}
	out := make([]domain.CheckFinding, 0, 2)
	for i := range invoices {
		inv := &invoices[i]
		if inv.TaxTreatment != domain.TaxTreatmentIntraCommunitySupply {
			continue
		}
		if inv.Status == domain.InvoiceStatusCancelled || inv.Status == domain.InvoiceStatusDraft {
			continue
		}
		status, err := s.vatIDs.StatusForContact(ctx, inv.ContactID)
		if err != nil {
			continue
		}
		// Eine gültige Bestätigung beendet den Befund — auch die nachgeholte.
		//
		// Vorher blieb eine mit Grund ausgestellte Rechnung stehen, bis das Jahr
		// vorbei war: der Lauf meldete „es liegt keine gültige Bestätigung vor"
		// und hängte im selben Satz an, die Nummer sei am soundsovielten
		// bestätigt worden. Der Befund ist die Aufforderung, die Abfrage
		// nachzuholen; wer ihr gefolgt ist, hat ihn erledigt.
		if status.Confirmed {
			continue
		}
		detail := status.Note
		if reason := strings.TrimSpace(inv.VatIDOverrideReason); reason != "" {
			detail = fmt.Sprintf("Ausgestellt mit dem Grund: %s. %s", reason, status.Note)
		}
		out = append(out, domain.CheckFinding{
			Rule:       domain.CheckRuleICSupplyUnconfirmed,
			Severity:   domain.CheckWarning,
			ObjectType: "INVOICE",
			ObjectID:   fmt.Sprintf("%d", inv.ID),
			ObjectName: inv.InvoiceNumber,
			Message: fmt.Sprintf(
				"Zur steuerfreien innergemeinschaftlichen Lieferung %s an %s liegt keine gültige "+
					"Bestätigung der USt-IdNr. vor. %s Hole die Bestätigungsanfrage nach: die gültige, "+
					"vom Bestimmungsland erteilte USt-IdNr. des Abnehmers ist materielle Voraussetzung "+
					"der Steuerbefreiung.",
				inv.InvoiceNumber, inv.ContactName, detail),
			Reference: "§ 6a Abs. 1 Satz 1 Nr. 4 UStG",
		})
	}
	return out
}
