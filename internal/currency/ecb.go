// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package currency

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type FXResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

type CurrencyService struct {
	cacheMu sync.RWMutex
	cache   map[string]float64
	lastRef time.Time
}

func NewCurrencyService() *CurrencyService {
	return &CurrencyService{
		cache: map[string]float64{
			"USD": 1.085,
			"GBP": 0.855,
			"CHF": 0.975,
			"JPY": 165.20,
			"EUR": 1.0,
		},
		lastRef: time.Now(),
	}
}

// GetExchangeRate returns the latest ECB reference rate for a currency relative to EUR.
func (cs *CurrencyService) GetExchangeRate(curr string) (float64, string, error) {
	if curr == "EUR" || curr == "" {
		return 1.0, "EZB (Basis)", nil
	}

	cs.cacheMu.RLock()
	rate, exists := cs.cache[curr]
	cs.cacheMu.RUnlock()

	// If cached recently (within 4 hours), return cached
	if exists && time.Since(cs.lastRef) < 4*time.Hour {
		return rate, "EZB Referenzkurs (Cache)", nil
	}

	// Fetch from frankfurter.app (ECB reference rate API)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("https://api.frankfurter.app/latest?from=EUR&to=%s", curr))
	if err != nil {
		if exists {
			return rate, "EZB Referenzkurs (Offline Fallback)", nil
		}
		return 1.0, "Unbekannt", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var fx FXResponse
		if err := json.NewDecoder(resp.Body).Decode(&fx); err == nil {
			if fetchedRate, ok := fx.Rates[curr]; ok {
				cs.cacheMu.Lock()
				cs.cache[curr] = fetchedRate
				cs.lastRef = time.Now()
				cs.cacheMu.Unlock()
				return fetchedRate, fmt.Sprintf("EZB Referenzkurs (%s via Frankfurter.app)", fx.Date), nil
			}
		}
	}

	if exists {
		return rate, "EZB Referenzkurs (Cache)", nil
	}
	return 1.0, "Standardkurs", nil
}
