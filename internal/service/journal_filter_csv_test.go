package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Herausgabe der gefilterten Journalmenge steht als Zugriff im Protokoll
// (QUE-02 K2).
func TestFilteredJournalCSVIsLogged(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	auditRepo := repository.NewAuditRepository(env.db)
	env.accounting.SetAuditRepo(auditRepo)

	entry := simpleEntry("6815", domain.AccountBank, 10000)
	if _, err := env.journal.Post(ctx, entry); err != nil {
		t.Fatalf("Buchung: %v", err)
	}

	csv, err := env.accounting.FilterEntriesCSV(ctx, accounting.JournalFilter{Account: "6815"})
	if err != nil {
		t.Fatalf("gefiltertes Journal als CSV: %v", err)
	}
	if !strings.Contains(csv, "6815") {
		t.Fatal("die Ausgabe enthält die gefilterte Zeile nicht")
	}

	entries, err := auditRepo.FindAll(ctx, 100)
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Action == domain.AuditActionExport && e.EntityType == "JOURNAL_FILTER" {
			found = true
			if !strings.Contains(e.Details, "6815") {
				t.Errorf("der Protokolleintrag nennt den Filter nicht: %s", e.Details)
			}
		}
	}
	if !found {
		t.Error("die Herausgabe der Journalzeilen steht nicht im Protokoll")
	}
}
