package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Was einen gescheiterten Vorgang überdauern soll, steht in einem eigenen
// Vorgang oder schließt ihn ab, statt mit ihm zurückzurollen. Die Tests gehen
// über den Kern und nicht direkt an den Dienst: erst die Transaktion je Write
// macht den Unterschied.

// Ein unlesbarer Kontoauszug bleibt als Beleg abgelegt.
func TestUnreadableStatementStaysFiled(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	path := filepath.Join(t.TempDir(), "auszug.xml")
	if err := os.WriteFile(path, []byte("<Document>kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tenant.ImportBankStatement(uiCtx, path, domain.AccountBank); err == nil {
		t.Fatal("ein unlesbarer Auszug wurde importiert")
	}
	// Der Auszug trägt den Tag des Imports.
	year, err := tenant.CurrentFiscalYear(uiCtx)
	if err != nil {
		t.Fatal(err)
	}
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		receipts, err := s.Receipts.List(ctx, year, "")
		if err != nil {
			return err
		}
		for _, r := range receipts {
			if r.Kind == domain.ReceiptKindStatement {
				return nil
			}
		}
		return errors.New("der Kontoauszug ist nicht als Beleg abgelegt")
	}); err != nil {
		t.Error(err)
	}
}

// Eine Bestätigungsanfrage ohne Antwort steht im Protokoll, und ihr Fehler
// kommt beim Aufrufer an.
func TestUnansweredVatIDCheckIsLogged(t *testing.T) {
	h := newHost(t)
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
	if _, err := tenant.CheckVatID(uiCtx, contact.ID); !errors.Is(err, ErrOffline) {
		t.Fatalf("Anfrage ohne Netz: %v", err)
	}
	if !auditContains(t, tenant, "ohne Ergebnis") {
		t.Error("die ausgebliebene Antwort steht nicht im Protokoll")
	}
}

// Blockierende Befunde verhindern die Festschreibung; der Prüflauf, der sie
// fand, bleibt gespeichert und protokolliert.
func TestBlockedCommitKeepsItsCheckRun(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.Journal.Post(ctx, &domain.JournalEntry{
			BookingDate: "2026-01-15", Description: "Buchung ohne Beleg", Source: domain.EntrySourceManual,
			TaxTreatment: domain.TaxTreatmentNotTaxable,
			Lines: []domain.JournalLine{
				{Side: domain.SideDebit, Account: "6815", Amount: 10000},
				{Side: domain.SideCredit, Account: "1800", Amount: 10000},
			},
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	_, err := tenant.CommitPeriod(uiCtx, CommitRequest{PeriodType: "month", PeriodLabel: "Januar 2026", CutoffDate: "2026-01-31"})
	var blocked *service.CommitBlocked
	if !errors.As(err, &blocked) {
		t.Fatalf("Festschreibung trotz blockierender Befunde: %v", err)
	}
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		runs, err := s.Checks.Runs(ctx, 2026)
		if err == nil && (len(runs) != 1 || !runs[0].HasBlocking()) {
			err = errors.New("der Prüflauf mit den blockierenden Befunden fehlt")
		}
		if err == nil {
			commitments, ferr := s.Repos.Festschreibung.FindByFiscalYear(ctx, 2026)
			if ferr == nil && len(commitments) != 0 {
				ferr = errors.New("es ist trotzdem festgeschrieben")
			}
			err = ferr
		}
		return err
	}); err != nil {
		t.Error(err)
	}
	if !auditContains(t, tenant, "Prüflauf bis 2026-01-31") {
		t.Error("der Prüflauf steht nicht im Protokoll")
	}
}

func auditContains(t *testing.T, tenant *Tenant, text string) bool {
	t.Helper()
	found := false
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		entries, err := s.Repos.Audit.FindFiltered(ctx, 0, domain.AuditFilter{})
		for _, e := range entries {
			found = found || strings.Contains(e.Details, text)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return found
}

// Scheitert das Übernahmeprotokoll, bleibt der Mandant angemeldet, und der
// Aufrufer erfährt es.
func TestFailedMigrationRecordIsReported(t *testing.T) {
	h := newHost(t)
	tc := newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	if err := tenant.db.Exec(`CREATE TRIGGER kein_uebernahmeprotokoll BEFORE INSERT ON migration_records
		BEGIN SELECT RAISE(ABORT, 'Protokoll nicht verfügbar'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if err := h.RemoveTenant(uiCtx, tc.ID); err != nil {
		t.Fatal(err)
	}
	registered, err := h.RegisterTenant(uiCtx, filepath.Join(tc.DataDir, "buchfink.sqlite"), "", domain.MigrationKindOpen)
	if err == nil || !strings.Contains(err.Error(), "nicht protokollieren") {
		t.Fatalf("gescheitertes Übernahmeprotokoll: %v", err)
	}
	if registered == nil || h.ActiveTenant() == nil || h.ActiveTenant().ID != registered.ID {
		t.Errorf("der Mandant ist nicht angemeldet: %+v", registered)
	}
}
