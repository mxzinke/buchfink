package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/repository"
)

// --- Keine `null`-Listen in den Nachweisen --------------------------------

// Die Nachweisseite liest die Listen ohne Umweg. Ein nicht belegter Slice wird
// zu `null`, und `null.map` nimmt im Render den ganzen Baum mit — und zwar im
// Regelfall: die frische Buchhaltung ohne Bruch und ohne offenen Posten.
func TestEvidenceAnswersMarshalEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	chain, err := NewAuditService(repository.NewAuditRepository(env.db)).VerifyChain(ctx)
	if err != nil {
		t.Fatalf("die Kettenprüfung ist fehlgeschlagen: %v", err)
	}
	assertNoNullLists(t, "Kette des Änderungsprotokolls", chain, "breaks")

	aging := accounting.AgeOpenItems(nil, "2026-06-30")
	assertNoNullLists(t, "Altersstruktur (ohne offene Posten)", aging, "sides")

	preview := &OpeningBalancePreview{}
	preview.EnsureLists()
	assertNoNullLists(t, "Eröffnungsbilanz (leere Vorschau)", preview,
		"entries", "messages")
}
