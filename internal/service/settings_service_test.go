package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Buchfink führt GmbH und UG. Eine andere Rechtsform wird abgewiesen, ein
// Formular ohne Rechtsform lässt die gespeicherte stehen.
func TestSettingsAcceptOnlyGmbHAndUG(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	repo := repository.NewSettingsRepository(env.db)
	svc := NewSettingsService(repo, repository.NewAuditRepository(env.db))

	current, err := svc.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	for _, form := range []string{"GbR", "AG", "Einzelunternehmen"} {
		next := *current
		next.LegalForm = form
		if err := svc.UpdateCompanySettings(ctx, &next); err == nil {
			t.Errorf("die Rechtsform %q muss abgewiesen werden", form)
		}
	}

	next := *current
	next.LegalForm = domain.LegalFormUG
	if err := svc.UpdateCompanySettings(ctx, &next); err != nil {
		t.Fatalf("UG speichern: %v", err)
	}
	next.LegalForm = ""
	if err := svc.UpdateCompanySettings(ctx, &next); err != nil {
		t.Fatalf("ohne Rechtsform speichern: %v", err)
	}
	saved, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	if saved.LegalForm != domain.LegalFormUG {
		t.Errorf("Rechtsform = %q, erwartet die gespeicherte UG", saved.LegalForm)
	}
}
