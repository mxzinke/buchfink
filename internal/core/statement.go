package core

import (
	"context"

	"github.com/buchfink/buchfink/internal/service"
)

// ImportBankStatement legt einen Kontoauszug (CAMT.053) als Beleg ab und
// importiert danach seine Umsätze, in zwei Vorgängen: scheitert der Import,
// bleibt die empfangene Datei als Beleg abgelegt, wie es die Belegsicherung
// verlangt.
func (t *Tenant) ImportBankStatement(ctx context.Context, path, ledgerAccount string) (int, error) {
	data, filename, err := service.ReadStatementFile(path)
	if err != nil {
		return 0, err
	}
	var statementID *uint
	if err := t.Write(ctx, func(ctx context.Context, s *Services) (err error) {
		statementID, err = s.Bank.FileStatement(ctx, data, filename)
		return err
	}); err != nil {
		return 0, err
	}
	imported := 0
	err = t.Write(ctx, func(ctx context.Context, s *Services) (err error) {
		imported, err = s.Bank.ImportStatement(ctx, data, ledgerAccount, statementID)
		return err
	})
	return imported, err
}
