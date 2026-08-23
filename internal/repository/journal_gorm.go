package repository

import (
	"context"
	"errors"
	"fmt"
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
	return r.db.WithContext(ctx).Preload("Lines").Preload("Entertainment").Order("id asc")
}

func (r *journalRepositoryGorm) scope(ctx context.Context, fiscalYear int) *gorm.DB {
	q := r.preloaded(ctx)
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	return q
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
	var entries []domain.JournalEntry
	err := r.scope(ctx, fiscalYear).Find(&entries).Error
	return entries, err
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

	// Gelesen wird in Stapeln, und das ist keine Vorsichtsmaßnahme: GORM lädt
	// die Zeilen über eine IN-Liste mit einer Bindvariablen je Buchung, und
	// SQLite bricht bei 32.766 davon ab. Über alle Jahre summiert ist diese
	// Zahl erreichbar — und dann läge nicht die Liste der offenen Posten
	// langsam, sondern sie käme mit einem Fehler zurück, und mit ihr jede
	// Zahlung. Je Stapel bleibt die Liste kurz, unabhängig davon, wie viele
	// Jahre der Mandant schon läuft.
	var entries []domain.JournalEntry
	batch := make([]domain.JournalEntry, 0, openItemBatchSize)
	err := q.FindInBatches(&batch, openItemBatchSize, func(*gorm.DB, int) error {
		entries = append(entries, batch...)
		return nil
	}).Error
	return entries, err
}

// openItemBatchSize keeps the preload's IN list far below SQLite's limit of
// 32766 bind variables, with room for the second preload on the same batch.
const openItemBatchSize = 2000

func (r *journalRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.JournalEntry, error) {
	var entry domain.JournalEntry
	if err := r.db.WithContext(ctx).Preload("Lines").Preload("Entertainment").First(&entry, id).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *journalRepositoryGorm) FindByAccount(ctx context.Context, account string, fiscalYear int) ([]domain.JournalEntry, error) {
	var ids []uint
	q := r.db.WithContext(ctx).Model(&domain.JournalLine{}).
		Joins("JOIN journal_entries e ON e.id = journal_lines.entry_id").
		Where("journal_lines.account = ?", account)
	if fiscalYear > 0 {
		q = q.Where("e.fiscal_year = ?", fiscalYear)
	}
	if err := q.Distinct("journal_lines.entry_id").Pluck("journal_lines.entry_id", &ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []domain.JournalEntry{}, nil
	}

	var entries []domain.JournalEntry
	err := r.db.WithContext(ctx).Preload("Lines").Preload("Entertainment").
		Where("id IN ?", ids).
		Order("booking_date asc, id asc").
		Find(&entries).Error
	return entries, err
}

func (r *journalRepositoryGorm) FindByContact(ctx context.Context, contactID uint, fiscalYear int) ([]domain.JournalEntry, error) {
	var entries []domain.JournalEntry
	q := r.db.WithContext(ctx).Preload("Lines").Preload("Entertainment").
		Where("contact_id = ?", contactID).
		Order("booking_date asc, id asc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Find(&entries).Error
	return entries, err
}

func (r *journalRepositoryGorm) FindReversalOf(ctx context.Context, entryID uint) (*domain.JournalEntry, error) {
	var entry domain.JournalEntry
	err := r.db.WithContext(ctx).Preload("Lines").Preload("Entertainment").Where("reversal_of_id = ?", entryID).First(&entry).Error
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
	q := r.db.WithContext(ctx).Preload("Lines").Preload("Entertainment").Order("id desc")
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
	err := r.db.WithContext(ctx).Preload("Lines").Preload("Entertainment").
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
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
	type row struct {
		Account string
		Side    string
		Total   int64
		Count   int
	}

	var rows []row
	q := r.db.WithContext(ctx).Model(&domain.JournalLine{}).
		Select("journal_lines.account as account, journal_lines.side as side, COALESCE(SUM(journal_lines.amount),0) as total, COUNT(*) as count").
		Joins("JOIN journal_entries e ON e.id = journal_lines.entry_id")
	if fiscalYear > 0 {
		q = q.Where("e.fiscal_year = ?", fiscalYear)
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
	q := r.db.WithContext(ctx).Model(&domain.JournalLine{}).
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
	q := r.db.WithContext(ctx).Model(&domain.JournalEntry{})
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Count(&count).Error
	return count, err
}

func (r *journalRepositoryGorm) GetAvailableFiscalYears(ctx context.Context) ([]int, error) {
	var years []int
	err := r.db.WithContext(ctx).Model(&domain.JournalEntry{}).
		Where("fiscal_year > 0").
		Distinct("fiscal_year").
		Order("fiscal_year asc").
		Pluck("fiscal_year", &years).Error
	return years, err
}
