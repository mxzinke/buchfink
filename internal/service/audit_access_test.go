package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// „Zugriffe" ist eine Abfrage und keine zwei: die Herausgabe von Daten steht als
// EXPORT, der Prüfermodus als Änderung an der Entität READ_ONLY (QUE-02 K2).
func TestAuditAccessFilterFindsBothKindsOfAccess(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	auditRepo := repository.NewAuditRepository(env.db)

	if err := auditRepo.Log(ctx, domain.AuditActionExport, "RECEIPT", "1",
		"Belegdatei herausgegeben"); err != nil {
		t.Fatalf("Protokolleintrag Export: %v", err)
	}
	if err := auditRepo.Log(ctx, domain.AuditActionUpdate, domain.AuditEntityReadOnly,
		"tenant", "Prüfermodus eingeschaltet"); err != nil {
		t.Fatalf("Protokolleintrag Prüfermodus: %v", err)
	}
	if err := auditRepo.Log(ctx, domain.AuditActionUpdate, "CONTACT", "7",
		"Kontakt geändert"); err != nil {
		t.Fatalf("Protokolleintrag Kontakt: %v", err)
	}

	entries, err := auditRepo.FindFiltered(ctx, 0, domain.AuditFilter{Access: true})
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("%d Einträge unter den Zugriffen — erwartet die Herausgabe und den "+
			"Prüfermodus: %+v", len(entries), entries)
	}
	for _, e := range entries {
		if e.EntityType == "CONTACT" {
			t.Error("eine Stammdatenänderung ist kein Zugriff auf personenbezogene Daten")
		}
	}
}
