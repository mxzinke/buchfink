package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type journalRepositoryGorm struct {
	db *gorm.DB
}

// NewJournalRepository creates a GORM-backed JournalRepository.
func NewJournalRepository(db *gorm.DB) domain.JournalRepository {
	return &journalRepositoryGorm{db: db}
}

func (r *journalRepositoryGorm) scope(ctx context.Context, fiscalYear int) *gorm.DB {
	q := r.db.WithContext(ctx).Preload("Lines").Order("id asc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	return q
}

func (r *journalRepositoryGorm) FindAll(ctx context.Context, fiscalYear int) ([]domain.JournalEntry, error) {
	var entries []domain.JournalEntry
	err := r.scope(ctx, fiscalYear).Find(&entries).Error
	return entries, err
}

func (r *journalRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.JournalEntry, error) {
	var entry domain.JournalEntry
	if err := r.db.WithContext(ctx).Preload("Lines").First(&entry, id).Error; err != nil {
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
	err := r.db.WithContext(ctx).Preload("Lines").
		Where("id IN ?", ids).
		Order("booking_date asc, id asc").
		Find(&entries).Error
	return entries, err
}

func (r *journalRepositoryGorm) FindByContact(ctx context.Context, contactID uint, fiscalYear int) ([]domain.JournalEntry, error) {
	var entries []domain.JournalEntry
	q := r.db.WithContext(ctx).Preload("Lines").
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
	err := r.db.WithContext(ctx).Preload("Lines").Where("reversal_of_id = ?", entryID).First(&entry).Error
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
	q := r.db.WithContext(ctx).Preload("Lines").Order("id desc")
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
		err = tx.Preload("Lines").
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
