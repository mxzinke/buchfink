package service

import "github.com/buchfink/buchfink/internal/domain"

// requireYear weist einen Aufruf ohne Geschäftsjahr ab. Die Dienste kennen
// kein Jahr der Ansicht; welches Jahr gemeint ist, sagt der Aufrufer.
func requireYear(year int) error {
	if year <= 0 {
		return domain.ErrFiscalYearRequired
	}
	return nil
}
