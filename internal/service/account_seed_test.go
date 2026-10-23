package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/repository"
)

// --- Fehlende SKR04-Konten werden auch in einer vollen Datei ergänzt -------

// Der frühere Schwellwert von hundert Konten war die Bedingung, unter der
// ein neuer SKR04-Stand nicht mehr in eine bestehende Datei kam.
func TestSeedAddsMissingAccountsEvenInAFullChart(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	accountRepo := repository.NewAccountRepository(env.db)
	all, err := accountRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("Kontenplan lesen: %v", err)
	}
	if len(all) < 100 {
		t.Fatalf("der Test braucht einen vollen Kontenplan, gefunden %d Konten", len(all))
	}

	// Ein einzelnes Konto verschwindet — so, wie es fehlte, wenn der
	// Kontenrahmen später um es erweitert wird.
	victim := all[len(all)/2].Number
	if err := env.db.Exec("DELETE FROM accounts WHERE number = ?", victim).Error; err != nil {
		t.Fatalf("Konto entfernen: %v", err)
	}

	if err := repository.SeedDefaultsIfEmpty(ctx, env.db, 2026); err != nil {
		t.Fatalf("Konten ergänzen: %v", err)
	}
	if _, err := accountRepo.FindByNumber(ctx, victim); err != nil {
		t.Errorf("das fehlende Konto %s wurde nicht ergänzt: %v", victim, err)
	}

	after, err := accountRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("Kontenplan lesen: %v", err)
	}
	if len(after) != len(all) {
		t.Errorf("nach dem Ergänzen stehen %d Konten da, vorher waren es %d", len(after), len(all))
	}
}
