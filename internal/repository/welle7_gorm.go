package repository

import (
	"context"
	"errors"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Die Ablagen der Welle 7: die gelernten Bankregeln, die Mahnschreiben und der
// Basiszinssatz.

// --- Gelernte Bankregeln --------------------------------------------------

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

// --- Mahnschreiben --------------------------------------------------------

type dunningRepositoryGorm struct {
	db *gorm.DB
}

// NewDunningRepository liefert die Ablage der Mahnschreiben.
func NewDunningRepository(db *gorm.DB) domain.DunningRepository {
	return &dunningRepositoryGorm{db: db}
}

func (r *dunningRepositoryGorm) Create(ctx context.Context, notice *domain.DunningNotice) error {
	if notice.CreatedAt.IsZero() {
		notice.CreatedAt = time.Now().UTC()
	}
	return dbFrom(ctx, r.db).Create(notice).Error
}

func (r *dunningRepositoryGorm) FindByContact(ctx context.Context, contactID uint) ([]domain.DunningNotice, error) {
	notices := make([]domain.DunningNotice, 0)
	db := dbFrom(ctx, r.db).Preload("Items").Order("notice_date desc, id desc")
	if contactID != 0 {
		db = db.Where("contact_id = ?", contactID)
	}
	if err := db.Find(&notices).Error; err != nil {
		return nil, err
	}
	for i := range notices {
		notices[i].EnsureLists()
	}
	return notices, nil
}

func (r *dunningRepositoryGorm) LevelByOpenItem(ctx context.Context) (map[uint]int, error) {
	var items []domain.DunningNoticeItem
	if err := dbFrom(ctx, r.db).Find(&items).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]int, len(items))
	for _, item := range items {
		if item.Level > out[item.OpenItemEntryID] {
			out[item.OpenItemEntryID] = item.Level
		}
	}
	return out, nil
}

// LumpSumChargedByOpenItem meldet die Posten, für die die Pauschale des § 288
// Abs. 5 BGB schon einmal angesetzt wurde.
//
// Gelesen wird der Posten und nicht das Schreiben: ein Schreiben über drei
// Rechnungen enthält die Pauschale nur für die, bei denen der Verzug
// eingetreten war, und der nächste Lauf muss die übrigen noch ansetzen können.
func (r *dunningRepositoryGorm) LumpSumChargedByOpenItem(ctx context.Context) (map[uint]bool, error) {
	var items []domain.DunningNoticeItem
	if err := dbFrom(ctx, r.db).Where("lump_sum_amount > 0").Find(&items).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]bool, len(items))
	for _, item := range items {
		out[item.OpenItemEntryID] = true
	}
	return out, nil
}

func (r *dunningRepositoryGorm) LastNoticeByOpenItem(ctx context.Context) (map[uint]string, error) {
	type row struct {
		OpenItemEntryID uint
		NoticeDate      string
	}
	var rows []row
	err := dbFrom(ctx, r.db).
		Table("dunning_notice_items").
		Select("dunning_notice_items.open_item_entry_id, dunning_notices.notice_date").
		Joins("join dunning_notices on dunning_notices.id = dunning_notice_items.dunning_notice_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint]string, len(rows))
	for _, r := range rows {
		if r.NoticeDate > out[r.OpenItemEntryID] {
			out[r.OpenItemEntryID] = r.NoticeDate
		}
	}
	return out, nil
}

// --- Basiszinssatz --------------------------------------------------------

type baseRateRepositoryGorm struct {
	db *gorm.DB
}

// NewBaseRateRepository liefert die gepflegte Basiszinstabelle.
func NewBaseRateRepository(db *gorm.DB) domain.BaseRateRepository {
	return &baseRateRepositoryGorm{db: db}
}

func (r *baseRateRepositoryGorm) FindAll(ctx context.Context) ([]domain.BaseRate, error) {
	rates := make([]domain.BaseRate, 0)
	err := dbFrom(ctx, r.db).Order("valid_from asc").Find(&rates).Error
	return rates, err
}

// Save schreibt einen Satz. Der Stichtag ist der Schlüssel: eine Bekanntgabe
// zum 1. Juli gibt es nur einmal, und wer sie berichtigt, ersetzt sie.
func (r *baseRateRepositoryGorm) Save(ctx context.Context, rate *domain.BaseRate) error {
	if rate.ValidFrom == "" {
		return errors.New("ein Basiszinssatz braucht den Tag, ab dem er gilt")
	}
	rate.UpdatedAt = time.Now().UTC()
	return dbFrom(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "valid_from"}},
		UpdateAll: true,
	}).Create(rate).Error
}
