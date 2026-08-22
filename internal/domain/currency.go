// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package domain

import "context"

// ExchangeRate represents a foreign currency conversion relative to EUR.
type ExchangeRate struct {
	Currency string  `gorm:"primaryKey;size:3" json:"currency"` // "USD", "GBP", "CHF"
	Rate     float64 `gorm:"not null" json:"rate"`              // 1 EUR = rate * Currency
	Date     string  `gorm:"size:10;not null" json:"date"`      // YYYY-MM-DD
	Source   string  `gorm:"size:100" json:"source"`            // e.g. "EZB via Frankfurter.app"
}

// CurrencyFetcher defines the contract for querying external FX rate providers.
type CurrencyFetcher interface {
	GetRate(ctx context.Context, currency string) (float64, string, error)
	// TODO: Add offline exchange rate caching in SQLite database table
}
