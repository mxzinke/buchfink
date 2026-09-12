package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type bankRepositoryGorm struct {
	db *gorm.DB
}

// NewBankRepository creates a new GORM-backed BankRepository.
func NewBankRepository(db *gorm.DB) domain.BankRepository {
	return &bankRepositoryGorm{db: db}
}

func (r *bankRepositoryGorm) FindAll(ctx context.Context, fiscalYear int) ([]domain.BankTransaction, error) {
	var txs []domain.BankTransaction
	q := dbFrom(ctx, r.db).Order("booking_date desc, id desc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Find(&txs).Error
	return txs, err
}

func (r *bankRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.BankTransaction, error) {
	var tx domain.BankTransaction
	err := dbFrom(ctx, r.db).First(&tx, id).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *bankRepositoryGorm) CreateBatch(ctx context.Context, transactions []domain.BankTransaction) (int, error) {
	inserted := 0
	err := dbFrom(ctx, r.db).Transaction(func(db *gorm.DB) error {
		// Compare decrypted values. SQL equality against random ciphertext cannot
		// identify a reimport. Count occurrences to retain identical payments in
		// one statement when the bank supplies no unique booking reference.
		existing := make(map[string]int)
		queried := make(map[string]bool)
		seen := make(map[string]int)
		for _, tx := range transactions {
			scope := fmt.Sprintf("%s/%s/%d", tx.AccountIBAN, tx.BookingDate, tx.Amount)
			if !queried[scope] {
				var candidates []domain.BankTransaction
				if err := db.Where("account_iban = ? AND booking_date = ? AND amount = ?", tx.AccountIBAN, tx.BookingDate, tx.Amount).Find(&candidates).Error; err != nil {
					return fmt.Errorf("vorhandene Bankumsätze prüfen: %w", err)
				}
				for _, old := range candidates {
					existing[bankIdentity(old)]++
				}
				queried[scope] = true
			}
			key := bankIdentity(tx)
			seen[key]++
			if existing[key] >= seen[key] {
				continue
			}
			tx.ID = 0
			if err := db.Create(&tx).Error; err != nil {
				return fmt.Errorf("Bankumsatz speichern: %w", err)
			}
			inserted++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return inserted, nil
}

func bankIdentity(tx domain.BankTransaction) string {
	reference := func(s string) string {
		s = strings.TrimSpace(s)
		switch strings.ToUpper(s) {
		case "NOTPROVIDED", "NONREF", "N/A", "NOT PROVIDED":
			return ""
		}
		return s
	}
	currency := tx.Currency
	if currency == "" {
		currency = "EUR"
	}
	fields := []any{tx.AccountIBAN, tx.BookingDate, tx.Amount, currency}
	if ref := reference(tx.BankReference); ref != "" {
		fields = append(fields, "bank", ref)
	} else {
		fields = append(fields, "content", tx.ValueDate, reference(tx.EndToEndID), strings.ToUpper(strings.ReplaceAll(tx.CounterpartyIBAN, " ", "")), tx.CounterpartyName, tx.RemittanceInfo)
	}
	data, _ := json.Marshal(fields)
	return string(data)
}

func (r *bankRepositoryGorm) SetMatchStatus(ctx context.Context, id uint, status domain.MatchStatus) error {
	return dbFrom(ctx, r.db).Model(&domain.BankTransaction{}).
		Where("id = ?", id).
		Update("match_status", status).Error
}

func (r *bankRepositoryGorm) Count(ctx context.Context, fiscalYear int) (int64, error) {
	var count int64
	q := dbFrom(ctx, r.db).Model(&domain.BankTransaction{})
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Count(&count).Error
	return count, err
}
