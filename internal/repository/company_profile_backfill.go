package repository

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// BackfillCompanyProfile überführt die Stammdaten in ihre Form ab
// Schemaversion 11.
//
// Die Zeile „PLZ und Ort" wird zerlegt, die frei eingetragene Landesangabe zum
// Ländercode. Die IBAN der Einstellungen wird zum Bankkonto für Rechnungen. Eine
// erfasste Gründung füllt Gründungsdatum, Stammkapital und Gesellschafterliste,
// soweit sie leer sind.
//
// Was sich nicht sicher überführen lässt — ein unbekanntes Land, eine IBAN mit
// falscher Prüfziffer, kein freies Bankkonto 1800 bis 1850 —, bleibt unter
// seinem alten Schlüssel stehen und wird nicht gelöscht.
func BackfillCompanyProfile(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		items := map[string]string{}
		var rows []domain.SettingItem
		if err := tx.Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			items[row.Key] = row.Value
		}
		b := profileBackfill{tx: tx, items: items}
		for _, step := range []func() error{b.address, b.invoiceAccount, b.founding} {
			if err := step(); err != nil {
				return err
			}
		}
		return nil
	})
}

type profileBackfill struct {
	tx    *gorm.DB
	items map[string]string
}

func (b profileBackfill) set(key, value string) error {
	b.items[key] = value
	return b.tx.Save(&domain.SettingItem{Key: key, Value: value, UpdatedAt: time.Now().UTC()}).Error
}

func (b profileBackfill) drop(keys ...string) error {
	return b.tx.Where("key IN ?", keys).Delete(&domain.SettingItem{}).Error
}

func (b profileBackfill) address() error {
	if zipCity, ok := b.items["zip_city"]; ok {
		if b.items["postal_code"] == "" && b.items["city"] == "" {
			postalCode, city := domain.SplitLegacyPostalLine(zipCity)
			if err := b.set("postal_code", postalCode); err != nil {
				return err
			}
			if err := b.set("city", city); err != nil {
				return err
			}
		}
		if err := b.drop("zip_city"); err != nil {
			return err
		}
	}
	if country, ok := b.items["country"]; ok {
		code := domain.CountryCodeFromName(country)
		if code == "" {
			return nil
		}
		if b.items["country_code"] == "" {
			if err := b.set("country_code", code); err != nil {
				return err
			}
		}
		return b.drop("country")
	}
	return nil
}

func (b profileBackfill) invoiceAccount() error {
	iban := domain.NormalizeIBAN(b.items["iban"])
	if iban == "" {
		return b.drop("iban", "bic", "bank_name")
	}
	if !domain.ValidIBAN(iban) {
		return nil
	}
	var accounts []domain.BankAccount
	if err := b.tx.Find(&accounts).Error; err != nil {
		return err
	}
	bic := strings.ToUpper(strings.Join(strings.Fields(b.items["bic"]), ""))
	if !domain.ValidBIC(bic) {
		bic = ""
	}
	bankName := strings.TrimSpace(b.items["bank_name"])

	var account *domain.BankAccount
	used := map[string]bool{}
	for i := range accounts {
		used[accounts[i].LedgerAccount] = true
		if accounts[i].IBAN == iban {
			account = &accounts[i]
		}
	}
	if account == nil {
		var history []domain.BankTransaction
		if err := b.tx.Select("account_iban", "ledger_account").Where("ledger_account <> ''").Find(&history).Error; err != nil {
			return err
		}
		ledger := ""
		for _, t := range history {
			used[t.LedgerAccount] = true
			if domain.NormalizeIBAN(t.AccountIBAN) == iban {
				ledger = t.LedgerAccount
			}
		}
		for _, candidate := range []string{"1800", "1810", "1820", "1830", "1840", "1850"} {
			if ledger == "" && !used[candidate] {
				ledger = candidate
			}
		}
		if ledger == "" {
			return nil
		}
		name := bankName
		if name == "" {
			name = "Geschäftskonto"
		}
		account = &domain.BankAccount{IBAN: iban, Name: name, LedgerAccount: ledger, Currency: "EUR"}
	}
	if account.BIC == "" {
		account.BIC = bic
	}
	if account.BankName == "" {
		account.BankName = bankName
	}
	account.IsInvoiceAccount = true
	if err := b.tx.Save(account).Error; err != nil {
		return err
	}
	return b.drop("iban", "bic", "bank_name")
}

func (b profileBackfill) founding() error {
	var f domain.Foundation
	err := b.tx.Preload("Shareholders", func(db *gorm.DB) *gorm.DB { return db.Order("id asc") }).
		Order("id asc").First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if b.items["founded_on"] == "" && f.NotarizedOn != "" {
		if err := b.set("founded_on", f.NotarizedOn); err != nil {
			return err
		}
	}
	if capital, _ := strconv.ParseInt(b.items["share_capital"], 10, 64); capital == 0 && f.ShareCapital > 0 {
		if err := b.set("share_capital", strconv.FormatInt(int64(f.ShareCapital), 10)); err != nil {
			return err
		}
	}
	var count int64
	if err := b.tx.Model(&domain.CompanyShareholder{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	for i, sh := range f.Shareholders {
		row := domain.CompanyShareholder{Position: i, Name: sh.Name, ShareCapital: sh.ShareCapital}
		if err := b.tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
