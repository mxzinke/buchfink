package core

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/buchfink/buchfink/internal/timestamp"
	"github.com/buchfink/buchfink/internal/vatid"
)

// countingRates liefert einen festen Kurs und zählt die Aufrufe.
type countingRates struct{ calls atomic.Int32 }

func (r *countingRates) RateAt(_ context.Context, _, date string) (int64, string, string, error) {
	r.calls.Add(1)
	return 1_085_000, date, "Testkurs", nil
}

// Jeder Weg ins Netz weist einen Aufruf aus einem Schreibvorgang ab; außerhalb
// geht er durch.
func TestNetworkIsRefusedInsideAWrite(t *testing.T) {
	rates := &countingRates{}
	stamps := 0
	vatIDCalls := 0
	h := newHostWith(t, Network{
		Timestamp: func(context.Context, string, string) (*timestamp.Result, error) {
			stamps++
			return &timestamp.Result{GenTime: time.Now()}, nil
		},
		Rates: rates,
		VatID: func() service.VatIDClient { return &stubVatID{calls: &vatIDCalls} },
	})
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)

	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		if _, err := tenant.network.Timestamp(ctx, "tsa", "hash"); !errors.Is(err, ErrNetworkInWrite) {
			t.Errorf("Zeitstempel im Schreibvorgang: %v", err)
		}
		if _, _, _, err := tenant.network.Rates.RateAt(ctx, "USD", "2026-03-02"); !errors.Is(err, ErrNetworkInWrite) {
			t.Errorf("Kurs im Schreibvorgang: %v", err)
		}
		if _, err := tenant.network.VatID().Check(ctx, vatid.Request{}); !errors.Is(err, ErrNetworkInWrite) {
			t.Errorf("Bestätigungsanfrage im Schreibvorgang: %v", err)
		}
		// Auch der Dienst darf im Schreibvorgang nicht an das Netz: ohne
		// abgelegten Kurs gibt es keinen.
		if _, err := s.Currency.RateAt(ctx, "USD", "2026-03-02"); !errors.Is(err, ErrNetworkInWrite) {
			t.Errorf("Kursdienst im Schreibvorgang: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if stamps != 0 || rates.calls.Load() != 0 || vatIDCalls != 0 {
		t.Fatalf("ein Netzaufruf ging aus dem Schreibvorgang hinaus: %d/%d/%d", stamps, rates.calls.Load(), vatIDCalls)
	}

	// Vorher geholt, findet der Schreibvorgang den Kurs in der Historie.
	if err := tenant.PrefetchRate(uiCtx, "usd", "2026-03-02"); err != nil {
		t.Fatalf("Kurs vorab holen: %v", err)
	}
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		rate, err := s.Currency.RateAt(ctx, "USD", "2026-03-02")
		if err == nil && rate.RateMicros != 1_085_000 {
			t.Errorf("Kurs %d, erwartet den geholten", rate.RateMicros)
		}
		return err
	}); err != nil {
		t.Fatalf("der vorab geholte Kurs fehlt im Schreibvorgang: %v", err)
	}
	if rates.calls.Load() != 1 {
		t.Errorf("%d Kursabrufe, erwartet einen", rates.calls.Load())
	}
	if err := tenant.PrefetchRate(uiCtx, "EUR", "2026-03-02"); err != nil || rates.calls.Load() != 1 {
		t.Errorf("für Euro wird kein Kurs geholt: %v, %d", err, rates.calls.Load())
	}
}

// stubVatID ist ein Bundeszentralamt, das jede Nummer bestätigt.
type stubVatID struct{ calls *int }

func (s *stubVatID) Check(context.Context, vatid.Request) (*vatid.Result, error) {
	*s.calls++
	return &vatid.Result{Status: domain.VatIDValid, Code: "evatr-0000", CheckedAt: time.Now(), RequestID: "1"}, nil
}

func (s *stubVatID) Endpoint() string { return "test" }

// Die Bestätigungsanfrage fragt ohne Sperre und legt das Ergebnis ab; das
// Ausstellen danach liest nur noch.
func TestCheckVatIDStoresTheAnswerOutsideTheWrite(t *testing.T) {
	calls := 0
	h := newHostWith(t, Network{VatID: func() service.VatIDClient { return &stubVatID{calls: &calls} }})
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	var contact domain.Contact
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		contact = domain.Contact{Type: domain.ContactTypeCustomer, Name: "Client SARL", CountryCode: "FR",
			VatID: "FR12345678901", Street: "Rue 1", PostalCode: "75001", City: "Paris"}
		return s.Contacts.SaveContact(ctx, &contact)
	}); err != nil {
		t.Fatal(err)
	}
	check, err := tenant.CheckVatID(uiCtx, contact.ID)
	if err != nil || check.Status != domain.VatIDValid || calls != 1 {
		t.Fatalf("Bestätigung: %+v, %v, %d Aufrufe", check, err, calls)
	}
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		status, err := s.VatIDs.StatusForContact(ctx, contact.ID)
		if err == nil && !status.Confirmed {
			return errors.New("die Bestätigung ist nicht abgelegt")
		}
		return err
	}); err != nil {
		t.Error(err)
	}
}

// Die Zeitstempelstelle hält keine Sperre: hängt sie, laufen Lesen und
// Schreiben weiter, und die Festschreibung steht schon, bevor der Stempel kommt.
func TestHangingTimestampAuthorityBlocksNobody(t *testing.T) {
	inside, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	h := newHostWith(t, Network{Timestamp: func(ctx context.Context, _, _ string) (*timestamp.Result, error) {
		once.Do(func() { close(inside) })
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return &timestamp.Result{Token: []byte("token"), GenTime: time.Now().UTC(), TSAName: "Test-TSA"}, nil
	}})
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := post(ctx, s)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	type result struct {
		rec *domain.Festschreibung
		err error
	}
	done := make(chan result, 1)
	go func() {
		rec, err := tenant.CommitPeriod(uiCtx, CommitRequest{
			PeriodType: "month", PeriodLabel: "März 2026", CutoffDate: "2026-03-31",
			OverrideReason: "Testlauf ohne Belegprüfung",
		})
		done <- result{rec, err}
	}()
	<-inside

	start := time.Now()
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		return s.Repos.Settings.Set(ctx, "notiz", "während des Zeitstempels")
	}); err != nil {
		t.Fatalf("Schreiben während des Zeitstempels: %v", err)
	}
	var pending []domain.Festschreibung
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		var err error
		pending, err = s.Repos.Festschreibung.FindPendingTimestamp(ctx)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if waited := time.Since(start); waited > 2*time.Second {
		t.Errorf("Lesen und Schreiben warteten %s auf die Zeitstempelstelle", waited)
	}
	if len(pending) != 1 {
		t.Errorf("die Festschreibung steht nicht schon vor dem Stempel: %+v", pending)
	}

	close(release)
	got := <-done
	if got.err != nil {
		t.Fatalf("Festschreibung: %v", got.err)
	}
	if got.rec.TimestampStatus != "confirmed" || got.rec.TSAName != "Test-TSA" {
		t.Errorf("Stempel nicht eingetragen: %+v", got.rec)
	}
}

// Die Festschreibung nimmt ihr Jahr aus dem Stichtag, protokolliert sich als
// eigene Aktion, holt einen ausgebliebenen Stempel nach und bleibt der App
// vorbehalten.
func TestCommitPeriodInTheCore(t *testing.T) {
	var online atomic.Bool
	h := newHostWith(t, Network{Timestamp: func(context.Context, string, string) (*timestamp.Result, error) {
		if !online.Load() {
			return nil, ErrOffline
		}
		return &timestamp.Result{Token: []byte("token"), GenTime: time.Now().UTC(), TSAName: "Test-TSA"}, nil
	}})
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	req := CommitRequest{PeriodType: "month", PeriodLabel: "Dezember 2025", CutoffDate: "2025-12-31",
		OverrideReason: "Testlauf ohne Belegprüfung"}

	if _, err := tenant.CommitPeriod(agentCtx(t), req); !errors.Is(err, ErrAppOnly) {
		t.Errorf("ein Agent schreibt fest: %v", err)
	}
	if _, err := tenant.CommitPeriod(systemCtx, req); !errors.Is(err, ErrAppOnly) {
		t.Errorf("ein Programmlauf schreibt fest: %v", err)
	}

	rec, err := tenant.CommitPeriod(uiCtx, req)
	if err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	if rec.FiscalYear != 2025 || rec.TimestampStatus != "pending" {
		t.Errorf("Festschreibung %+v: erwartet Jahr 2025 aus dem Stichtag, Stempel ausstehend", rec)
	}
	if _, err := tenant.CommitPeriod(uiCtx, req); err == nil || !strings.Contains(err.Error(), "bereits festgeschrieben") {
		t.Errorf("doppelte Festschreibung: %v", err)
	}

	var logged []domain.AuditLogEntry
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		var err error
		logged, err = s.Repos.Audit.FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "FESTSCHREIBUNG"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(logged) != 1 || logged[0].Action != domain.AuditActionCommit {
		t.Errorf("Protokoll der Festschreibung: %+v", logged)
	}

	if n, err := tenant.RetryTimestamps(context.Background()); err != nil || n != 0 {
		t.Errorf("ohne Netz: %d nachgeholt, %v", n, err)
	}
	online.Store(true)
	if n, err := tenant.RetryTimestamps(context.Background()); err != nil || n != 1 {
		t.Fatalf("mit Netz: %d nachgeholt, %v", n, err)
	}
	verification, err := tenant.VerifyCommitment(uiCtx, rec.ID)
	if err != nil || !verification.HasTimestamp {
		t.Errorf("Prüfung nach dem Nachholen: %+v, %v", verification, err)
	}
}
