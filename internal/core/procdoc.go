package core

import (
	"context"
	"time"

	"github.com/buchfink/buchfink/internal/service"
)

// GenerateProcedureDocumentation erzeugt eine Fassung der
// Verfahrensdokumentation für ein Geschäftsjahr und legt sie ab. Datenordner
// und Sicherungsziel kommen von diesem Mandanten, die letzten Sicherungsläufe
// aus seiner Datenbank.
func (t *Tenant) GenerateProcedureDocumentation(ctx context.Context, year int) (*service.ProcDocResult, error) {
	var result *service.ProcDocResult
	err := t.Write(ctx, func(ctx context.Context, s *Services) error {
		env := procDocEnvironment(t.DataDir(), t.backupDirectory())
		runs, err := s.Repos.BackupRuns.FindRecent(ctx, 5)
		if err != nil {
			return err
		}
		env.BackupRuns = runs
		result, err = s.ProcDoc.Generate(ctx, year, env, time.Time{})
		return err
	})
	return result, err
}
