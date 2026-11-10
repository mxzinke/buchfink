package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// InTx führt fn in einer Transaktion aus. Läuft schon eine, gehört fn zu ihr.
//
// Jeder Vorgang, der mehr als einen Datensatz schreibt — Buchung, Vermerk am
// Bankumsatz, Siegel am Beleg, Bewegung in einer Kartei, Protokolleintrag —,
// läuft so ganz oder gar nicht.
func (s *JournalService) InTx(ctx context.Context, fn func(context.Context) error) error {
	return s.tx.RunInTx(ctx, fn)
}

// atomically ist InTx mit Ergebnis. Scheitert fn, ist das Ergebnis der
// Nullwert: was fn bis dahin geschrieben hat, ist zurückgerollt.
func atomically[T any](ctx context.Context, j *JournalService, fn func(context.Context) (T, error)) (T, error) {
	var out T
	err := j.InTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = fn(ctx)
		return err
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return out, nil
}

// logAudit schreibt einen Protokolleintrag. Scheitert er, scheitert der
// Vorgang: ein Änderungsprotokoll mit Lücken belegt nichts.
func logAudit(ctx context.Context, repo domain.AuditRepository, action domain.AuditAction, entityType, entityID, details string) error {
	if repo == nil {
		return nil
	}
	if err := repo.Log(ctx, action, entityType, entityID, details); err != nil {
		return fmt.Errorf("der Protokolleintrag ließ sich nicht schreiben: %w", err)
	}
	return nil
}

// logAuditChange ist logAudit mit Vorher und Nachher.
func logAuditChange(
	ctx context.Context, repo domain.AuditRepository, action domain.AuditAction,
	entityType, entityID, details string, before, after any,
) error {
	if repo == nil {
		return nil
	}
	if err := repo.LogChange(ctx, action, entityType, entityID, details, before, after); err != nil {
		return fmt.Errorf("der Protokolleintrag ließ sich nicht schreiben: %w", err)
	}
	return nil
}
