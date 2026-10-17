package service

import (
	"context"
	"sync"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Mit `go test -race` aussagekräftig: Lesen und Verwerfen des Kontenplans
// laufen gleichzeitig, sobald Oberfläche und Agent denselben Dienst benutzen.
func TestChartCacheIsSafeForConcurrentUse(t *testing.T) {
	db, err := repository.InitTenantDB(t.TempDir())
	if err != nil {
		t.Fatalf("Mandantendatenbank: %v", err)
	}
	t.Cleanup(func() { _ = repository.CloseDB(db) })
	accounts := repository.NewAccountRepository(db)
	journal := NewJournalService(repository.NewJournalRepository(db), accounts,
		repository.NewContactRepository(db), repository.NewAuditRepository(db),
		repository.NewSettingsRepository(db), repository.NewTxRunner(db))
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				chart, err := journal.Chart(ctx)
				if err != nil {
					t.Error(err)
					return
				}
				if err := chart.EnsurePostable("1800"); err != nil {
					t.Error(err)
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				journal.InvalidateChart()
			}
		}()
	}
	wg.Wait()

	custom := &domain.Account{Number: "6309", Name: "Eigenes Aufwandskonto", Type: domain.AccountTypeExpense,
		Category: "Aufwand", Kontenklasse: 6, IsActive: true, IsCustom: true}
	if _, err := journal.Chart(ctx); err != nil {
		t.Fatal(err)
	}
	if err := accounts.Create(ctx, custom); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	journal.InvalidateChart()
	chart, err := journal.Chart(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := chart.Lookup("6309"); !ok {
		t.Error("nach dem Verwerfen muss der Kontenplan das neue Konto kennen")
	}
}
