package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// JournalService is the single write path into the journal.
//
// Every booking in the system goes through Post, which enforces the rules that
// make the journal usable as accounting evidence: balanced entries, existing and
// bookable accounts, tax accounts reserved for the tax automation, gapless
// numbering, an unbroken hash chain and respect for committed periods.
type JournalService struct {
	journalRepo        domain.JournalRepository
	accountRepo        domain.AccountRepository
	contactRepo        domain.ContactRepository
	auditRepo          domain.AuditRepository
	settingsRepo       domain.SettingsRepository
	festschreibungRepo domain.FestschreibungRepository

	hashChain   *accounting.HashChain
	taxResolver domain.TaxResolver

	chart      *accounting.Chart
	fiscalYear int
}

// NewJournalService wires the journal write path.
func NewJournalService(
	journalRepo domain.JournalRepository,
	accountRepo domain.AccountRepository,
	contactRepo domain.ContactRepository,
	auditRepo domain.AuditRepository,
	settingsRepo domain.SettingsRepository,
	fiscalYear int,
) *JournalService {
	return &JournalService{
		journalRepo:  journalRepo,
		accountRepo:  accountRepo,
		contactRepo:  contactRepo,
		auditRepo:    auditRepo,
		settingsRepo: settingsRepo,
		hashChain:    accounting.NewHashChain(),
		taxResolver:  accounting.NewSKR04TaxResolver(),
		fiscalYear:   fiscalYear,
	}
}

// SetFestschreibungRepo wires period-commitment enforcement. Optional.
func (s *JournalService) SetFestschreibungRepo(r domain.FestschreibungRepository) {
	s.festschreibungRepo = r
}

// SetFiscalYear updates the active fiscal year filter.
func (s *JournalService) SetFiscalYear(year int) { s.fiscalYear = year }

// FiscalYear returns the active fiscal year.
func (s *JournalService) FiscalYear() int { return s.fiscalYear }

// Chart returns the cached chart of accounts resolver.
func (s *JournalService) Chart(ctx context.Context) (*accounting.Chart, error) {
	if s.chart != nil {
		return s.chart, nil
	}
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("Kontenplan konnte nicht geladen werden: %w", err)
	}
	s.chart = accounting.NewChart(accounts)
	return s.chart, nil
}

// TaxResolver exposes the SKR04 tax resolution used by the posting rules.
func (s *JournalService) TaxResolver() domain.TaxResolver { return s.taxResolver }

// Post validates and appends a journal entry.
func (s *JournalService) Post(ctx context.Context, entry *domain.JournalEntry) (*domain.JournalEntry, error) {
	s.applyDefaults(ctx, entry)

	if err := s.ensureAccrualTaxation(ctx); err != nil {
		return nil, err
	}
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if d := entry.Entertainment; d != nil {
		if err := d.Validate(); err != nil {
			return nil, err
		}
	}
	if err := s.validateAccounts(ctx, entry); err != nil {
		return nil, err
	}
	if err := s.ensurePeriodOpen(ctx, entry); err != nil {
		return nil, err
	}

	entry.CreatedAt = time.Now().UTC()

	if err := s.journalRepo.Append(ctx, entry, s.hashChain.CalculateHash); err != nil {
		return nil, fmt.Errorf("Buchung konnte nicht gespeichert werden: %w", err)
	}

	s.audit(ctx, domain.AuditActionCreate, entry.ID, fmt.Sprintf(
		"Buchung %s: %s, %s € (GJ %d, %d Zeilen)",
		entry.EntryNumber, entry.Description, entry.GrossAmount(), entry.FiscalYear, len(entry.Lines),
	))

	return entry, nil
}

// Reverse cancels a booking by Generalumkehr: the same accounts on the same
// sides with negated amounts.
//
// A side-swapped Storno would also produce a zero balance, but it inflates the
// Verkehrszahlen of every account involved — a 1.000 € expense corrected that
// way leaves the account showing 1.000 € Soll and 1.000 € Haben. The Summen- und
// Saldenliste and the VAT figures derived from turnover would then be wrong. The
// Generalumkehr returns the turnover to zero and is what DATEV records as "GU".
func (s *JournalService) Reverse(ctx context.Context, entryID uint, reason string) (*domain.JournalEntry, error) {
	original, err := s.journalRepo.FindByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("Buchung %d wurde nicht gefunden: %w", entryID, err)
	}
	if original.Kind == domain.EntryKindReversal {
		return nil, fmt.Errorf("Buchung %s ist selbst eine Generalumkehr und kann nicht erneut storniert werden", original.EntryNumber)
	}

	existing, err := s.journalRepo.FindReversalOf(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("bestehende Stornos zu Buchung %d konnten nicht geprüft werden: %w", entryID, err)
	}
	if existing != nil {
		return nil, fmt.Errorf("Buchung %s wurde bereits durch %s storniert", original.EntryNumber, existing.EntryNumber)
	}
	if reason == "" {
		return nil, fmt.Errorf("für eine Stornierung ist ein Grund anzugeben")
	}

	// The correction is dated at the time of correction, never backdated into
	// the original period: that is what keeps a committed period untouched.
	today := time.Now().Format("2006-01-02")

	lines := make([]domain.JournalLine, 0, len(original.Lines))
	for _, l := range original.Lines {
		lines = append(lines, domain.JournalLine{
			Position:  l.Position,
			Side:      l.Side, // unchanged — this is what makes it a Generalumkehr
			Account:   l.Account,
			ContactID: l.ContactID,
			Amount:    -l.Amount,
			TaxKey:    l.TaxKey,
			TaxBase:   -l.TaxBase,
			Text:      l.Text,
		})
	}

	reversal := &domain.JournalEntry{
		FiscalYear:         domain.GetFiscalYearForDate(today, s.fiscalYearStartMonth(ctx)),
		BookingDate:        today,
		DocumentDate:       original.DocumentDate,
		ServiceDateFrom:    original.ServiceDateFrom,
		ServiceDateTo:      original.ServiceDateTo,
		ValueDate:          original.ValueDate,
		Description:        fmt.Sprintf("Storno zu %s: %s", original.EntryNumber, original.Description),
		Source:             original.Source,
		DocumentNumber:     original.DocumentNumber,
		ReceiptID:          original.ReceiptID,
		ReceiptHash:        original.ReceiptHash,
		TaxTreatment:       original.TaxTreatment,
		ContactID:          original.ContactID,
		BankTxID:           original.BankTxID,
		Kind:               domain.EntryKindReversal,
		ReversalOfID:       &original.ID,
		ReversalReason:     reason,
		Currency:           original.Currency,
		ExchangeRateMicros: original.ExchangeRateMicros,
		ExchangeRateSource: original.ExchangeRateSource,
		ExchangeRateDate:   original.ExchangeRateDate,
		PostingRuleVersion: original.PostingRuleVersion,
		Lines:              lines,
	}
	// The Generalumkehr points at the same Beleg and carries the same
	// Aufzeichnung: it corrects the booking, it does not undo the meal.
	if d := original.Entertainment; d != nil {
		reversal.Entertainment = &domain.EntertainmentDetail{
			Place: d.Place, Day: d.Day, Participants: d.Participants, Occasion: d.Occasion,
		}
	}

	created, err := s.Post(ctx, reversal)
	if err != nil {
		return nil, err
	}

	s.audit(ctx, domain.AuditActionStorno, created.ID, fmt.Sprintf(
		"Generalumkehr %s storniert Buchung %s (Grund: %s)",
		created.EntryNumber, original.EntryNumber, reason,
	))

	return created, nil
}

// VerifyIntegrity re-computes the hash chain of the active fiscal year.
func (s *JournalService) VerifyIntegrity(ctx context.Context) (domain.IntegrityCheckResult, error) {
	entries, err := s.journalRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return domain.IntegrityCheckResult{}, err
	}

	result := s.hashChain.VerifyChain(entries)
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionIntegrityCheck, "HASH_CHAIN",
			fmt.Sprintf("GJ_%d", s.fiscalYear), result.Message)
	}
	return result, nil
}

func (s *JournalService) applyDefaults(ctx context.Context, e *domain.JournalEntry) {
	if e.BookingDate == "" {
		e.BookingDate = time.Now().Format("2006-01-02")
	}
	if e.DocumentDate == "" {
		e.DocumentDate = e.BookingDate
	}
	if e.ServiceDateFrom == "" {
		e.ServiceDateFrom = e.DocumentDate
	}
	if e.ServiceDateTo == "" {
		e.ServiceDateTo = e.ServiceDateFrom
	}
	if e.Kind == "" {
		e.Kind = domain.EntryKindNormal
	}
	if e.Source == "" {
		e.Source = domain.EntrySourceManual
	}
	if e.Currency == "" {
		e.Currency = "EUR"
	}
	if e.ExchangeRateMicros == 0 {
		e.ExchangeRateMicros = 1_000_000
	}
	if e.FiscalYear == 0 {
		e.FiscalYear = domain.GetFiscalYearForDate(e.BookingDate, s.fiscalYearStartMonth(ctx))
	}
	for i := range e.Lines {
		if e.Lines[i].Position == 0 {
			e.Lines[i].Position = i + 1
		}
	}
}

// ensureAccrualTaxation refuses to book while the company is set to
// Istversteuerung.
//
// The whole flow — record the invoice, book it at once, settle it later —
// presumes taxation on agreed consideration (§ 16 Abs. 1 Satz 1 UStG). Under
// Istversteuerung the tax only arises with the receipt of payment
// (§ 13 Abs. 1 Nr. 1 Buchst. b UStG) and the bookings would look different. The
// setting has existed since the setup wizard shipped, but nothing ever checked
// it — so the option quietly produced wrong VAT returns. Refusing is the honest
// behaviour until the second booking path exists.
func (s *JournalService) ensureAccrualTaxation(ctx context.Context) error {
	if s.settingsRepo == nil {
		return nil
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil || cfg.TaxationType == "" {
		return nil
	}
	if strings.EqualFold(cfg.TaxationType, "SOLL") {
		return nil
	}
	return fmt.Errorf(
		"Buchfink unterstützt derzeit nur die Sollversteuerung (§ 16 Abs. 1 Satz 1 UStG). Bei Istversteuerung entsteht die Steuer erst mit der Vereinnahmung des Entgelts (§ 13 Abs. 1 Nr. 1 Buchst. b UStG), und die Buchungen sähen anders aus. Bitte in den Einstellungen auf Sollversteuerung umstellen")
}

func (s *JournalService) fiscalYearStartMonth(ctx context.Context) int {
	if s.settingsRepo == nil {
		return 1
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil || cfg.FiscalYearStartMonth <= 0 {
		return 1
	}
	return cfg.FiscalYearStartMonth
}

// validateAccounts rejects bookings that reference accounts which cannot carry
// them. Without this check the journal accepts numbers that exist nowhere in the
// chart of accounts, and the resulting balances silently omit them.
func (s *JournalService) validateAccounts(ctx context.Context, e *domain.JournalEntry) error {
	chart, err := s.Chart(ctx)
	if err != nil {
		return err
	}

	for i, l := range e.Lines {
		if domain.IsLedgerAccount(l.Account) {
			if err := s.validateLedgerAccount(ctx, l); err != nil {
				return fmt.Errorf("Zeile %d: %w", i+1, err)
			}
			continue
		}

		if err := chart.EnsurePostable(l.Account); err != nil {
			return fmt.Errorf("Zeile %d: %w", i+1, err)
		}

		// Tax accounts carry the figures of the Umsatzsteuer-Voranmeldung. They
		// may only be written by the tax automation, which stamps a TaxKey on
		// the line it generates.
		if s.taxResolver.IsTaxAccount(l.Account) && l.TaxKey == "" {
			return fmt.Errorf(
				"Zeile %d: Konto %s ist ein Steuerkonto und darf nur über die Steuerautomatik bebucht werden",
				i+1, l.Account,
			)
		}

		// Offene Posten gehören auf das Personenkonto des Geschäftspartners.
		// Eine Buchung direkt auf das Sammelkonto stünde zwar in der Bilanz,
		// aber in keiner OPOS-Liste.
		if kind, ok := domain.CollectiveAccounts()[l.Account]; ok {
			partner := "Kunden"
			if kind == domain.ContactTypeVendor {
				partner = "Lieferanten"
			}
			return fmt.Errorf(
				"Zeile %d: Konto %s ist das Sammelkonto für die Bilanz und wird nicht direkt bebucht. "+
					"Buche den offenen Posten auf das Personenkonto des %s – die Bilanzposition verdichtet sich daraus",
				i+1, l.Account, partner,
			)
		}
	}

	return nil
}

func (s *JournalService) validateLedgerAccount(ctx context.Context, l domain.JournalLine) error {
	if s.contactRepo == nil {
		return fmt.Errorf("Personenkonto %s kann ohne Stammdaten nicht geprüft werden", l.Account)
	}
	contact, err := s.contactRepo.FindByLedgerAccount(ctx, l.Account)
	if err != nil || contact == nil {
		return fmt.Errorf("Personenkonto %s gehört zu keinem angelegten Geschäftspartner", l.Account)
	}
	if l.ContactID != nil && *l.ContactID != contact.ID {
		return fmt.Errorf("Personenkonto %s gehört zu %s, die Buchung verweist aber auf einen anderen Geschäftspartner", l.Account, contact.Name)
	}
	return nil
}

// ensurePeriodOpen blocks bookings backdated into a committed period. A
// correction is dated at correction time and therefore never lands in one.
func (s *JournalService) ensurePeriodOpen(ctx context.Context, e *domain.JournalEntry) error {
	if s.festschreibungRepo == nil {
		return nil
	}
	cutoff, err := s.festschreibungRepo.LatestCutoff(ctx, e.FiscalYear)
	if err != nil {
		return fmt.Errorf("Festschreibungsstand konnte nicht geprüft werden: %w", err)
	}
	if cutoff != "" && e.BookingDate <= cutoff {
		return fmt.Errorf(
			"Der Zeitraum bis %s ist festgeschrieben. Eine Buchung zum %s ist nicht mehr möglich – bitte über eine Stornierung korrigieren",
			cutoff, e.BookingDate,
		)
	}
	return nil
}

func (s *JournalService) audit(ctx context.Context, action domain.AuditAction, id uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "JOURNAL", fmt.Sprintf("%d", id), details)
}
