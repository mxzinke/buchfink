// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package accounting

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/buchfink/buchfink/internal/domain"
)

//go:embed skr04_2026.json
var skr04JSONData []byte

var (
	catalogOnce   sync.Once
	cachedCatalog *SKR04Catalog
	catalogErr    error
)

// SKR04Metadata contains document info according to official DATEV SKR04 (BilRUG 2026).
type SKR04Metadata struct {
	Title         string `json:"title"`
	Subtitle      string `json:"subtitle"`
	ValidityFrom  string `json:"validity_from"`
	Version       string `json:"version"`
	ArticleNumber string `json:"article_number"`
	SourceFile    string `json:"source_file"`
	Description   string `json:"description"`
	GeneratedAt   string `json:"generated_at"`
}

// SKR04Legend holds explanation texts for functions, VAT keys, and footnotes.
type SKR04Legend struct {
	Hauptfunktionen    map[string]string `json:"hauptfunktionen"`
	Zusatzfunktionen   map[string]string `json:"zusatzfunktionen"`
	Abschlusszweck     map[string]string `json:"abschlusszweck"`
	Programmverbindung map[string]string `json:"programmverbindung"`
	Footnotes          map[string]string `json:"footnotes"`
}

// SKR04Statistics holds account and position metrics.
type SKR04Statistics struct {
	TotalAccounts          int            `json:"total_accounts"`
	ActiveAccounts         int            `json:"active_accounts"`
	ReservedAccounts       int            `json:"reserved_accounts"`
	RangeAccounts          int            `json:"range_accounts"`
	TotalPositions         int            `json:"total_positions"`
	AccountsByType         map[string]int `json:"accounts_by_type"`
	AccountsByKontenklasse map[string]int `json:"accounts_by_kontenklasse"`
	PositionsBySide        map[string]int `json:"positions_by_side"`
}

// SKR04Kontenklasse represents a class 0-9.
type SKR04Kontenklasse struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
}

// SKR04Bilanzierung contains HGB § 266 / § 275 balance and P&L classification.
type SKR04Bilanzierung struct {
	StatementType string `json:"statement_type"`
	BalanceSide   string `json:"balance_side"`
	HGBCode       string `json:"hgb_code"`
	Posten        string `json:"posten"`
	AccountType   string `json:"account_type"`
	PositionID    string `json:"position_id"`
}

// SKR04SteuerFunktion contains DATEV automatic tax calculation and program integration tags.
type SKR04SteuerFunktion struct {
	Hauptfunktion             *string  `json:"hauptfunktion"`
	HauptfunktionDescription  *string  `json:"hauptfunktion_description"`
	Zusatzfunktion            *string  `json:"zusatzfunktion"`
	ZusatzfunktionDescription *string  `json:"zusatzfunktion_description"`
	Abschlusszweck            *string  `json:"abschlusszweck"`
	Programmverbindung        []string `json:"programmverbindung"`
}

// SKR04AccountEntry represents an individual account or account range in the official JSON.
type SKR04AccountEntry struct {
	Number         string              `json:"number"`
	Name           string              `json:"name"`
	Category       string              `json:"category"`
	Subcategory    string              `json:"subcategory"`
	Kontenklasse   SKR04Kontenklasse   `json:"kontenklasse"`
	PositionID     string              `json:"position_id"`
	IsRange        bool                `json:"is_range"`
	RangeStart     string              `json:"range_start"`
	RangeEnd       string              `json:"range_end"`
	IsReserved     bool                `json:"is_reserved"`
	Page           int                 `json:"page"`
	Bilanzierung   SKR04Bilanzierung   `json:"bilanzierung"`
	SteuerFunktion SKR04SteuerFunktion `json:"steuer_funktion"`
	Footnotes      []int               `json:"footnotes"`
	Description    string              `json:"description,omitempty"`
}

// SKR04Position represents an HGB position node (e.g. Anlagevermögen -> Sachanlagen).
type SKR04Position struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	StatementType  string            `json:"statement_type"`
	BalanceSide    string            `json:"balance_side"`
	HGBCode        string            `json:"hgb_code"`
	Group          string            `json:"group"`
	MainGroup      string            `json:"main_group"`
	AccountType    string            `json:"account_type"`
	Kontenklasse   SKR04Kontenklasse `json:"kontenklasse"`
	AccountNumbers []string          `json:"account_numbers"`
	AccountsCount  int               `json:"accounts_count"`
}

// SKR04Catalog represents the entire dataset of SKR04 (2026).
type SKR04Catalog struct {
	Metadata   SKR04Metadata          `json:"metadata"`
	Legend     SKR04Legend            `json:"legend"`
	Statistics SKR04Statistics        `json:"statistics"`
	Positions  []SKR04Position        `json:"positions"`
	Accounts   []SKR04AccountEntry    `json:"accounts"`
	Hierarchy  map[string]interface{} `json:"hierarchy"`
}

// GetSKR04Catalog loads and caches the complete SKR04 2026 catalog from the embedded JSON.
func GetSKR04Catalog() (*SKR04Catalog, error) {
	catalogOnce.Do(func() {
		var cat SKR04Catalog
		if err := json.Unmarshal(skr04JSONData, &cat); err != nil {
			catalogErr = fmt.Errorf("failed to parse embedded skr04_2026.json: %w", err)
			return
		}
		cachedCatalog = &cat
	})
	return cachedCatalog, catalogErr
}

// DefaultSKR04Accounts returns all structured accounts from the official SKR04 2026 JSON.
func DefaultSKR04Accounts() []domain.Account {
	cat, err := GetSKR04Catalog()
	if err != nil || cat == nil {
		return nil
	}

	result := make([]domain.Account, 0, len(cat.Accounts))
	for _, entry := range cat.Accounts {
		var hf, hfDesc, zf, zfDesc, az string
		if entry.SteuerFunktion.Hauptfunktion != nil {
			hf = *entry.SteuerFunktion.Hauptfunktion
		}
		if entry.SteuerFunktion.HauptfunktionDescription != nil {
			hfDesc = *entry.SteuerFunktion.HauptfunktionDescription
		}
		if entry.SteuerFunktion.Zusatzfunktion != nil {
			zf = *entry.SteuerFunktion.Zusatzfunktion
		}
		if entry.SteuerFunktion.ZusatzfunktionDescription != nil {
			zfDesc = *entry.SteuerFunktion.ZusatzfunktionDescription
		}
		if entry.SteuerFunktion.Abschlusszweck != nil {
			az = *entry.SteuerFunktion.Abschlusszweck
		}

		// Calculate tax rate from name/key
		taxRate := 0.0
		if strings.Contains(entry.Name, "19 %") || strings.Contains(entry.Name, "19%") {
			taxRate = 0.19
		} else if strings.Contains(entry.Name, "7 %") || strings.Contains(entry.Name, "7%") {
			taxRate = 0.07
		} else if strings.Contains(entry.Name, "16 %") || strings.Contains(entry.Name, "16%") {
			taxRate = 0.16
		}

		desc := entry.Description
		if desc == "" {
			var parts []string
			if entry.Bilanzierung.Posten != "" && entry.Bilanzierung.Posten != entry.Name {
				parts = append(parts, "Posten: "+entry.Bilanzierung.Posten)
			}
			if hfDesc != "" {
				parts = append(parts, hfDesc)
			}
			if zfDesc != "" {
				parts = append(parts, zfDesc)
			}
			desc = strings.Join(parts, " • ")
		}

		accType := domain.AccountType(entry.Bilanzierung.AccountType)
		if accType == "" {
			accType = domain.AccountTypeAsset
		}

		result = append(result, domain.Account{
			Number:             entry.Number,
			Name:               entry.Name,
			Type:               accType,
			Category:           entry.Category,
			Subcategory:        entry.Subcategory,
			Kontenklasse:       entry.Kontenklasse.Number,
			KontenklasseName:   entry.Kontenklasse.Name,
			PositionID:         entry.PositionID,
			Posten:             entry.Bilanzierung.Posten,
			BalanceSide:        entry.Bilanzierung.BalanceSide,
			HGBCode:            entry.Bilanzierung.HGBCode,
			StatementType:      entry.Bilanzierung.StatementType,
			TaxRate:            taxRate,
			Hauptfunktion:      hf,
			HauptfunktionDesc:  hfDesc,
			Zusatzfunktion:     zf,
			ZusatzfunktionDesc: zfDesc,
			Abschlusszweck:     az,
			IsRange:            entry.IsRange,
			RangeStart:         entry.RangeStart,
			RangeEnd:           entry.RangeEnd,
			IsReserved:         entry.IsReserved,
			Description:        desc,
			IsActive:           !entry.IsReserved,
		})
	}

	return result
}
