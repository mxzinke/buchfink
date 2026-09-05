package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// Every read preloads the Aufzeichnung along with the lines. It is part of the
// entry's canonical form, so an entry read without it hashes differently than it
// was written — the integrity check would report every entertainment booking as
// broken.
type journalRepositoryGorm struct {
	db *gorm.DB
}

// NewJournalRepository creates a GORM-backed JournalRepository.
func NewJournalRepository(db *gorm.DB) domain.JournalRepository {
	return &journalRepositoryGorm{db: db}
}

// preloaded is the read every finder starts from. The preload set is not a
// convenience: an entry read without its Aufzeichnung hashes differently than it
// was written, so a finder that forgets it makes the integrity check report
// every Bewirtungsbuchung as broken. Written once, it cannot be forgotten.
func (r *journalRepositoryGorm) preloaded(ctx context.Context) *gorm.DB {
	return dbFrom(ctx, r.db).Preload("Lines").Preload("Entertainment").Order("id asc")
}

func (r *journalRepositoryGorm) scope(ctx context.Context, fiscalYear int) *gorm.DB {
	q := r.preloaded(ctx)
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	return q
}

// entryBatchSize keeps the preloads' IN lists far below SQLite's limit of 32766
// parameters per statement, with room for both preloads on the same batch.
const entryBatchSize = 2000

// findBatched runs a query in batches instead of one statement.
//
// This is not tuning, it is the difference between working and failing. GORM
// loads the lines of an entry through an IN list with one parameter per parent
// row, and SQLite refuses a statement with more than 32766 of them — a query
// does not get slow there, it returns an error. The readers on this path are the journal
// list, the Umsatzsteuer-Auswertung and the GoBD integrity check, so the failure
// would arrive as "the journal cannot be read" on a perfectly healthy database.
//
// The paging is by primary key, so every caller must be ordered by it — which
// preloaded already is. A query ordered by anything else pages wrong and has to
// sort in Go afterwards.
func findBatched(q *gorm.DB) ([]domain.JournalEntry, error) {
	var out []domain.JournalEntry
	batch := make([]domain.JournalEntry, 0, entryBatchSize)
	err := q.FindInBatches(&batch, entryBatchSize, func(*gorm.DB, int) error {
		out = append(out, batch...)
		return nil
	}).Error
	return out, err
}

// byBookingDate orders entries for display: chronologically, and within a day by
// the order they were written.
func byBookingDate(entries []domain.JournalEntry) []domain.JournalEntry {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].BookingDate != entries[j].BookingDate {
			return entries[i].BookingDate < entries[j].BookingDate
		}
		return entries[i].ID < entries[j].ID
	})
	return entries
}

// notReversed narrows a query to the entries that are still in force: neither a
// Generalumkehr itself nor cancelled by one.
//
// It is deliberately not bounded by fiscal year, and that is the whole point.
// A Generalumkehr is dated at the day of the correction, never backdated into
// the period it corrects, so it regularly lands in a later year than the entry
// it cancels. Asking the question inside a year window answers it wrong for
// exactly the entries that matter.
func notReversed(q *gorm.DB, alias string) *gorm.DB {
	return q.Where(alias+".kind <> ?", domain.EntryKindReversal).
		Where("NOT EXISTS (SELECT 1 FROM journal_entries gu WHERE gu.reversal_of_id = " + alias + ".id)")
}

func (r *journalRepositoryGorm) FindAll(ctx context.Context, fiscalYear int) ([]domain.JournalEntry, error) {
	return findBatched(r.scope(ctx, fiscalYear))
}

// FindByBookingDateRange narrows to a booking-date window inside a fiscal year.
// Empty bounds mean the whole year.
func (r *journalRepositoryGorm) FindByBookingDateRange(ctx context.Context, fiscalYear int, from, to string) ([]domain.JournalEntry, error) {
	q := r.scope(ctx, fiscalYear)
	if from != "" {
		q = q.Where("booking_date >= ?", from)
	}
	if to != "" {
		q = q.Where("booking_date <= ?", to)
	}
	return findBatched(q)
}

func (r *journalRepositoryGorm) FindOpenItemCandidates(ctx context.Context, fiscalYear int) ([]domain.JournalEntry, error) {
	q := notReversed(r.preloaded(ctx), "journal_entries").
		// Zahlungen begründen selbst keinen offenen Posten.
		Where("source <> ?", domain.EntrySourcePayment).
		// Und nur eine Buchung mit einem Personenkonto trägt überhaupt einen.
		// Die Vorauswahl ist absichtlich loser als domain.IsLedgerAccount: sie
		// darf nie strenger sein als die Regel, die anschließend im Go-Code
		// entscheidet, sonst fiele ein Posten stillschweigend heraus.
		Where("EXISTS (SELECT 1 FROM journal_lines l WHERE l.entry_id = journal_entries.id AND length(l.account) = 5)")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year <= ?", fiscalYear)
	}

	return findBatched(q)
}

// FindOpenItemCandidatesAt ist die Stichtagssicht auf dieselben Kandidaten.
//
// Zwei Grenzen statt einer: das Buchungsdatum der Buchung selbst und das
// Buchungsdatum ihrer Generalumkehr. Die zweite ist die eigentliche Arbeit —
// notReversed kennt kein Datum und schlösse eine Rechnung auch dann aus, wenn
// ihr Storno erst nach dem Bilanzstichtag gebucht wurde. Am Stichtag stand sie
// aber noch in den Büchern, und der Saldo des Personenkontos weist sie aus.
func (r *journalRepositoryGorm) FindOpenItemCandidatesAt(ctx context.Context, cutoff string) ([]domain.JournalEntry, error) {
	q := r.preloaded(ctx).
		Where("journal_entries.kind <> ?", domain.EntryKindReversal).
		Where("NOT EXISTS (SELECT 1 FROM journal_entries gu WHERE gu.reversal_of_id = journal_entries.id AND gu.booking_date <= ?)", cutoff).
		Where("source <> ?", domain.EntrySourcePayment).
		Where("booking_date <= ?", cutoff).
		Where("EXISTS (SELECT 1 FROM journal_lines l WHERE l.entry_id = journal_entries.id AND length(l.account) = 5)")
	return findBatched(q)
}

func (r *journalRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.JournalEntry, error) {
	var entry domain.JournalEntry
	if err := dbFrom(ctx, r.db).Preload("Lines").Preload("Entertainment").First(&entry, id).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *journalRepositoryGorm) FindByAccount(ctx context.Context, account string, fiscalYear int) ([]domain.JournalEntry, error) {
	// Ein EXISTS statt einer Liste eingesammelter Kennungen: die Liste wäre eine
	// zweite Abfrage und eine zweite Reihe von Abfrageparametern, und beide
	// wachsen mit der Zahl der Buchungen auf dem Konto.
	q := r.scope(ctx, fiscalYear).
		Where("EXISTS (SELECT 1 FROM journal_lines l WHERE l.entry_id = journal_entries.id AND l.account = ?)", account)
	entries, err := findBatched(q)
	if err != nil {
		return nil, err
	}
	return byBookingDate(entries), nil
}

// FindByAccountRange liefert das Kontoblatt eines Zeitraums, unabhängig vom
// Geschäftsjahr. Leere Grenzen heißen: keine Grenze.
func (r *journalRepositoryGorm) FindByAccountRange(ctx context.Context, account, from, to string) ([]domain.JournalEntry, error) {
	q := r.preloaded(ctx).
		Where("EXISTS (SELECT 1 FROM journal_lines l WHERE l.entry_id = journal_entries.id AND l.account = ?)", account)
	if from != "" {
		q = q.Where("booking_date >= ?", from)
	}
	if to != "" {
		q = q.Where("booking_date <= ?", to)
	}
	entries, err := findBatched(q)
	if err != nil {
		return nil, err
	}
	return byBookingDate(entries), nil
}

func (r *journalRepositoryGorm) FindByContact(ctx context.Context, contactID uint, fiscalYear int) ([]domain.JournalEntry, error) {
	entries, err := findBatched(r.scope(ctx, fiscalYear).Where("contact_id = ?", contactID))
	if err != nil {
		return nil, err
	}
	return byBookingDate(entries), nil
}

func (r *journalRepositoryGorm) FindReversalOf(ctx context.Context, entryID uint) (*domain.JournalEntry, error) {
	var entry domain.JournalEntry
	err := dbFrom(ctx, r.db).Preload("Lines").Preload("Entertainment").Where("reversal_of_id = ?", entryID).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *journalRepositoryGorm) GetLastEntry(ctx context.Context, fiscalYear int) (*domain.JournalEntry, error) {
	var entry domain.JournalEntry
	q := dbFrom(ctx, r.db).Preload("Lines").Preload("Entertainment").Order("id desc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// FindByReceipt returns the original booking that references a Beleg. A
// Generalumkehr points at the same Beleg but is not what sealed it, so it is
// skipped.
func (r *journalRepositoryGorm) FindByReceipt(ctx context.Context, receiptID uint) (*domain.JournalEntry, error) {
	var entry domain.JournalEntry
	err := dbFrom(ctx, r.db).Preload("Lines").Preload("Entertainment").
		Where("receipt_id = ? AND kind = ?", receiptID, domain.EntryKindNormal).
		Order("id asc").First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// Append allocates the Buchungsnummer, links the hash chain and inserts the
// entry with its lines in one transaction.
func (r *journalRepositoryGorm) Append(ctx context.Context, entry *domain.JournalEntry, hash domain.EntryHashFunc) error {
	// Läuft der Aufrufer bereits in einer Transaktion — die Ausgangsrechnung tut
	// das —, wird deren Handle benutzt: Nummer, Rechnung und Buchung sollen
	// zusammen gelingen oder zusammen ausbleiben.
	return dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		seq, err := allocateNumber(tx, domain.NumberRangeJournal, entry.FiscalYear)
		if err != nil {
			return err
		}
		entry.EntryNumber = domain.FormatJournalNumber(entry.FiscalYear, seq)

		// The chain head is read inside the transaction, so a second writer
		// cannot branch the chain by reading the same predecessor.
		var last domain.JournalEntry
		err = tx.Preload("Lines").Preload("Entertainment").
			Where("fiscal_year = ?", entry.FiscalYear).
			Order("id desc").First(&last).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			entry.PreviousHash = domain.GenesisHash
		case err != nil:
			return fmt.Errorf("Kettenkopf für Geschäftsjahr %d konnte nicht gelesen werden: %w", entry.FiscalYear, err)
		default:
			entry.PreviousHash = last.EntryHash
		}

		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = time.Now().UTC()
		}
		entry.EntryHash = hash(entry, entry.PreviousHash)

		return tx.Create(entry).Error
	})
}

func (r *journalRepositoryGorm) AccountTurnovers(ctx context.Context, fiscalYear int) (map[string]domain.AccountTurnover, error) {
	return r.AccountTurnoversUntil(ctx, fiscalYear, "")
}

// AccountTurnoversUntil summiert die Verkehrszahlen bis zu einem Stichtag.
func (r *journalRepositoryGorm) AccountTurnoversUntil(ctx context.Context, fiscalYear int, cutoff string) (map[string]domain.AccountTurnover, error) {
	type row struct {
		Account string
		Side    string
		Total   int64
		Count   int
	}

	var rows []row
	q := dbFrom(ctx, r.db).Model(&domain.JournalLine{}).
		Select("journal_lines.account as account, journal_lines.side as side, COALESCE(SUM(journal_lines.amount),0) as total, COUNT(*) as count").
		Joins("JOIN journal_entries e ON e.id = journal_lines.entry_id")
	if fiscalYear > 0 {
		q = q.Where("e.fiscal_year = ?", fiscalYear)
	}
	if cutoff != "" {
		q = q.Where("e.booking_date <= ?", cutoff)
	}
	if err := q.Group("journal_lines.account, journal_lines.side").Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[string]domain.AccountTurnover, len(rows))
	for _, x := range rows {
		t := result[x.Account]
		if domain.Side(x.Side) == domain.SideDebit {
			t.Debit += domain.Cents(x.Total)
		} else {
			t.Credit += domain.Cents(x.Total)
		}
		t.Count += x.Count
		result[x.Account] = t
	}
	return result, nil
}

func (r *journalRepositoryGorm) MonthlyCashflow(ctx context.Context, fiscalYear int, liquidAccounts []string) ([]domain.CashflowDataPoint, error) {
	if len(liquidAccounts) == 0 {
		return []domain.CashflowDataPoint{}, nil
	}

	type row struct {
		Month string
		Side  string
		Total int64
	}

	var rows []row
	q := dbFrom(ctx, r.db).Model(&domain.JournalLine{}).
		Select("substr(e.booking_date, 1, 7) as month, journal_lines.side as side, COALESCE(SUM(journal_lines.amount),0) as total").
		Joins("JOIN journal_entries e ON e.id = journal_lines.entry_id").
		Where("journal_lines.account IN ?", liquidAccounts)
	if fiscalYear > 0 {
		q = q.Where("e.fiscal_year = ?", fiscalYear)
	}
	if err := q.Group("month, side").Order("month asc").Scan(&rows).Error; err != nil {
		return nil, err
	}

	byMonth := make(map[string]*domain.CashflowDataPoint)
	var order []string
	for _, x := range rows {
		p, ok := byMonth[x.Month]
		if !ok {
			p = &domain.CashflowDataPoint{Month: x.Month, Label: monthLabel(x.Month)}
			byMonth[x.Month] = p
			order = append(order, x.Month)
		}
		// A debit on a liquid account is money coming in, a credit is money
		// going out.
		if domain.Side(x.Side) == domain.SideDebit {
			p.Inflow += domain.Cents(x.Total)
		} else {
			p.Outflow += domain.Cents(x.Total)
		}
	}

	result := make([]domain.CashflowDataPoint, 0, len(order))
	for _, m := range order {
		p := byMonth[m]
		p.Net = p.Inflow - p.Outflow
		result = append(result, *p)
	}
	return result, nil
}

var monthLabels = [...]string{"Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"}

func monthLabel(yyyymm string) string {
	if len(yyyymm) != 7 {
		return yyyymm
	}
	m := int(yyyymm[5]-'0')*10 + int(yyyymm[6]-'0')
	if m < 1 || m > 12 {
		return yyyymm
	}
	return monthLabels[m-1]
}

func (r *journalRepositoryGorm) Count(ctx context.Context, fiscalYear int) (int64, error) {
	var count int64
	q := dbFrom(ctx, r.db).Model(&domain.JournalEntry{})
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Count(&count).Error
	return count, err
}

func (r *journalRepositoryGorm) GetAvailableFiscalYears(ctx context.Context) ([]int, error) {
	var years []int
	err := dbFrom(ctx, r.db).Model(&domain.JournalEntry{}).
		Where("fiscal_year > 0").
		Distinct("fiscal_year").
		Order("fiscal_year asc").
		Pluck("fiscal_year", &years).Error
	return years, err
}
