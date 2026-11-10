package repository

import (
	"context"
	"errors"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// Die gelernten Bankregeln.

type bankRuleRepositoryGorm struct {
	db *gorm.DB
}

// NewBankRuleRepository liefert die gelernten Zuordnungen.
func NewBankRuleRepository(db *gorm.DB) domain.BankRuleRepository {
	return &bankRuleRepositoryGorm{db: db}
}

func (r *bankRuleRepositoryGorm) FindAll(ctx context.Context) ([]domain.BankRule, error) {
	rules := make([]domain.BankRule, 0)
	// Die häufigste zuerst: in den Einstellungen steht oben, was das Programm
	// am sichersten vorschlägt.
	err := dbFrom(ctx, r.db).Order("hits desc, id asc").Find(&rules).Error
	return rules, err
}

func (r *bankRuleRepositoryGorm) FindByPattern(ctx context.Context, pattern string) (*domain.BankRule, error) {
	if pattern == "" {
		return nil, nil
	}
	var rule domain.BankRule
	err := dbFrom(ctx, r.db).Where("pattern = ?", pattern).First(&rule).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// Save legt die Regel an oder schreibt sie fort.
//
// Fortschreiben und nicht ein zweites Mal anlegen: dasselbe Muster ist derselbe
// wiederkehrende Vorgang, und zwei Regeln dafür wären zwei Vorschläge, von denen
// einer veraltet ist. Wer denselben Umsatz das nächste Mal gegen ein anderes
// Konto bucht, hat es sich anders überlegt — dann gilt das neue Konto.
func (r *bankRuleRepositoryGorm) Save(ctx context.Context, rule *domain.BankRule) error {
	if rule.Pattern == "" || rule.CounterAccount == "" {
		return errors.New("eine Bankregel braucht ein Muster und ein Gegenkonto")
	}
	now := time.Now().UTC()
	db := dbFrom(ctx, r.db)

	existing, err := r.FindByPattern(ctx, rule.Pattern)
	if err != nil {
		return err
	}
	if existing == nil {
		if rule.Hits <= 0 {
			rule.Hits = 1
		}
		rule.CreatedAt, rule.UpdatedAt = now, now
		return db.Create(rule).Error
	}

	existing.CounterAccount = rule.CounterAccount
	existing.PostingGroup = rule.PostingGroup
	existing.MoneyIn = rule.MoneyIn
	if rule.Label != "" {
		existing.Label = rule.Label
	}
	existing.Hits++
	existing.LastUsedAt = rule.LastUsedAt
	existing.UpdatedAt = now
	if err := db.Model(existing).Select(
		"CounterAccount", "PostingGroup", "MoneyIn", "Label", "Hits", "LastUsedAt", "UpdatedAt",
	).Updates(existing).Error; err != nil {
		return err
	}
	*rule = *existing
	return nil
}

func (r *bankRuleRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return dbFrom(ctx, r.db).Delete(&domain.BankRule{}, id).Error
}
