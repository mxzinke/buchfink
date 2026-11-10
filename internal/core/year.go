package core

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
)

// FiscalYearOf ist das Geschäftsjahr, in das ein Tag fällt.
func (t *Tenant) FiscalYearOf(ctx context.Context, date string) (int, error) {
	if err := domain.CheckDate("Datum", date); err != nil {
		return 0, err
	}
	year := 0
	err := t.Read(systemContext(ctx), func(ctx context.Context, s *Services) error {
		year = s.Journal.FiscalYearOf(ctx, date)
		return nil
	})
	return year, err
}

// CurrentFiscalYear ist das Geschäftsjahr des heutigen Tages nach Beginnmonat.
// Es ist die Vorgabe für Agenten, nie das Jahr, das gerade jemand ansieht.
func (t *Tenant) CurrentFiscalYear(ctx context.Context) (int, error) {
	return t.FiscalYearOf(ctx, t.Today())
}
