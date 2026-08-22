// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package service

import (
	"context"

	"github.com/buchfink/buchfink/internal/currency"
)

// CurrencyService fetches and caches ECB foreign exchange reference rates.
type CurrencyService struct {
	ecbClient *currency.CurrencyService
}

func NewCurrencyService() *CurrencyService {
	return &CurrencyService{
		ecbClient: currency.NewCurrencyService(),
	}
}

func (s *CurrencyService) GetExchangeRate(_ context.Context, curr string) (float64, string, error) {
	return s.ecbClient.GetExchangeRate(curr)
}
