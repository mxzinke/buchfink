package core

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/service"
)

// bookOnAccount bucht 100 € von einem Aufwandskonto gegen die Bank.
func bookOnAccount(ctx context.Context, tenant *Tenant, account string) error {
	return tenant.Write(ctx, func(ctx context.Context, s *Services) error {
		req, err := receiptAndEntry(ctx, s)
		if err != nil {
			return err
		}
		req.Entry.Lines[0].Account = account
		_, err = s.Posting.PostManualEntry(ctx, req)
		return err
	})
}

// Ein Lesevorgang, dessen Stand älter ist als eine Sperre, füllt den
// Kontenplan des Buchungswegs nicht mit diesem Stand: das gesperrte Konto
// bleibt gesperrt.
func TestReadDoesNotRefillTheChartWithAnOldSnapshot(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.Accounts.CreateCustom(ctx, service.CustomAccountRequest{
			Number: "6011", Name: "Fremdleistungen Fotografie",
			HGBPosition: "guv.guv_8.sonstige_betriebliche_aufwendungen",
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := bookOnAccount(uiCtx, tenant, "6011"); err != nil {
		t.Fatalf("Buchung auf das neue Konto: %v", err)
	}

	snapshot, blocked, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- tenant.Read(agentCtx(t), func(ctx context.Context, s *Services) error {
			// Die erste Abfrage legt den Stand des Lesevorgangs fest.
			if _, err := s.Repos.Journal.Count(ctx, 2026); err != nil {
				return err
			}
			close(snapshot)
			<-blocked
			_, err := s.Journal.Chart(ctx)
			return err
		})
	}()
	<-snapshot
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.Accounts.SetBlocked(ctx, "6011", true, "falsch angelegt")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	close(blocked)
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	if err := bookOnAccount(agentCtx(t), tenant, "6011"); err == nil {
		t.Error("Buchung auf das gesperrte Konto 6011 angenommen")
	}
}

// Ein angelegtes Konto zählt im Kontenplan erst nach dem Commit; rollt der
// Vorgang zurück, kennt der Buchungsweg es nicht.
func TestChartFollowsTheCommit(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	failure := context.Canceled
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		if _, err := s.Accounts.CreateCustom(ctx, service.CustomAccountRequest{
			Number: "6011", Name: "Fremdleistungen Fotografie",
			HGBPosition: "guv.guv_8.sonstige_betriebliche_aufwendungen",
		}); err != nil {
			return err
		}
		// Im selben Vorgang ist das Konto bebuchbar.
		if _, err := s.Journal.Chart(ctx); err != nil {
			return err
		}
		return failure
	}); err != failure {
		t.Fatalf("Vorgang: %v", err)
	}
	if err := bookOnAccount(uiCtx, tenant, "6011"); err == nil {
		t.Error("Buchung auf ein zurückgerolltes Konto angenommen")
	}
}
