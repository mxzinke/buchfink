package core

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Was nur ein Mensch in der App auslöst, weist der Kern jedem Agenten ab, über
// welchen Einstieg auch immer: Prüfermodus, Festschreibung,
// Übermittlungsbestätigung, Aufstellen/Feststellen/Offenlegen, Stammdaten,
// Bankkonten und die Verwaltung der Mandanten. Die Vorgänge der Dienste laufen
// hier ohne core.AppOnly(): sie weisen den Agenten selbst ab, auch wenn ein
// Einstieg die Kennzeichnung vergisst.
func TestAgentsAreRefusedForHumanOnlyOperations(t *testing.T) {
	h := newHost(t)
	tc := newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	agent := agentCtx(t)
	until := time.Now().AddDate(0, 0, 10).Format("2006-01-02")
	inService := func(fn func(ctx context.Context, s *Services) error) func() error {
		return func() error { return tenant.Write(agent, fn) }
	}

	cases := map[string]func() error{
		"Prüfermodus einschalten": func() error { _, err := tenant.EnableExaminer(agent, until, "Prüfung"); return err },
		"Prüfermodus beenden":     func() error { _, err := tenant.DisableExaminer(agent, "Ende"); return err },
		"Monat festschreiben": func() error {
			_, err := tenant.CommitPeriod(agent, CommitRequest{PeriodType: "month", PeriodLabel: "Jan", CutoffDate: "2026-01-31"})
			return err
		},
		"Jahr festschreiben": func() error {
			_, err := tenant.CommitPeriod(agent, CommitRequest{PeriodType: domain.PeriodTypeYear, PeriodLabel: "2026", CutoffDate: "2026-12-31"})
			return err
		},
		"Aufstellen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-01", "")
			return err
		}),
		"Feststellung zurücknehmen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Closing.ReopenFiscalYear(ctx, 2026, "Korrektur")
			return err
		}),
		"Voranmeldung als übermittelt bestätigen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.VatReturns.ConfirmSubmitted(ctx, 1, "2026-02-10", "et-123", "")
			return err
		}),
		"ZM als übermittelt bestätigen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.ZM.ConfirmSubmitted(ctx, 1, "2026-02-25", "et-123", "")
			return err
		}),
		"Unternehmensdaten ändern": inService(func(ctx context.Context, s *Services) error {
			return s.Settings.UpdateCompanySettings(ctx, &domain.CompanySettings{CompanyName: "Andere GmbH"})
		}),
		"Bankkonten einrichten": inService(func(ctx context.Context, s *Services) error {
			return s.Bank.ConfigureAccounts(ctx, nil)
		}),
		"Rechnungskonto wählen": inService(func(ctx context.Context, s *Services) error {
			return s.Bank.SetInvoiceAccount(ctx, "DE02120300000000202051")
		}),
		"Gründung erfassen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Foundation.Save(ctx, &domain.Foundation{NotarizedOn: "2026-01-10", ShareCapital: 2_500_000})
			return err
		}),
		"Eintragung ins Handelsregister": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Foundation.Register(ctx, "2026-02-01", "AG Köln", "HRB 4711")
			return err
		}),
		"Hebesatz ändern": inService(func(ctx context.Context, s *Services) error {
			_, err := s.ClosingSettings.Save(ctx, service.ClosingSettings{
				TradeTaxRatePercent: 250, AccrualMethod: "monthly", AccrualRelease: "yearly"})
			return err
		}),
		"Organisationsanweisung ändern": inService(func(ctx context.Context, s *Services) error {
			return s.ProcDoc.SaveOrganisationTexts(ctx, domain.OrganisationTexts{Responsibilities: "Agent"})
		}),
		"USt-Jahreserklärung als erledigt bestätigen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Obligations.Confirm(ctx, "ust-erklaerung.2025", service.ConfirmInput{Date: "2026-03-01"})
			return err
		}),
		"Sondervorauszahlung als erledigt bestätigen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Obligations.Confirm(ctx, "sondervorauszahlung.2026", service.ConfirmInput{Date: "2026-02-10"})
			return err
		}),
		"Pflicht als entfallen kennzeichnen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Obligations.Waive(ctx, "gewerbeanmeldung.gruendung", "freiberuflich")
			return err
		}),
		"Bestätigung einer Pflicht zurücknehmen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Obligations.Reopen(ctx, "kst-erklaerung.2025", "Irrtum")
			return err
		}),
		"Ergebnisverwendung beschließen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Appropriation.BookAppropriation(ctx, 2025, service.AppropriationRequest{DecisionDate: "2026-06-01"})
			return err
		}),
		"Gesellschafterdarlehen erfassen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Shareholders.RegisterLoan(ctx, service.LoanRequest{ContactID: 1, Direction: domain.LoanToCompany,
				Principal: 1_000_000, InterestRateBP: 300, StartDate: "2026-01-01"})
			return err
		}),
		"Gesellschafterdarlehen berichtigen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Shareholders.CorrectLoan(ctx, 1, service.LoanRequest{ContactID: 1, Direction: domain.LoanToCompany,
				Principal: 1_000_000, InterestRateBP: 300, StartDate: "2026-01-01"}, "Zinssatz")
			return err
		}),
		"Sicherungsordner setzen": func() error { _, err := h.SetBackupDir(agent, t.TempDir()); return err },
		"Mandant wechseln":        func() error { _, err := h.Switch(agent, tc.ID); return err },
		"Mandant anlegen": func() error {
			_, err := h.CreateTenant(agent, "Zweite GmbH", filepath.Join(t.TempDir(), "neu"), domain.CompanySettings{})
			return err
		},
		"Mandant entfernen": func() error { return h.RemoveTenant(agent, tc.ID) },
		"Datenbank anmelden": func() error {
			_, err := h.RegisterTenant(agent, filepath.Join(tc.DataDir, "buchfink.sqlite"), "", domain.MigrationKindOpen)
			return err
		},
		"Sicherung wiederherstellen": func() error {
			_, err := h.RestoreBackup(agent, filepath.Join(t.TempDir(), "x.zip"), t.TempDir())
			return err
		},
	}
	for name, run := range cases {
		err := run()
		if err == nil || !(errors.Is(err, ErrAppOnly) || strings.Contains(err.Error(), "vorbehalten")) {
			t.Errorf("%s durch einen Agenten: %v", name, err)
		}
	}

	// Die Unternehmensdaten gehen nur über UpdateCompanySettings; der einzelne
	// Schlüssel nimmt sie nicht an, auch nicht von der App.
	for _, key := range []string{"legal_form", "fiscal_year_start_month", "company_name", "vat_id", "invoice_number_format"} {
		for _, ctx := range []context.Context{agent, uiCtx} {
			if err := tenant.Write(ctx, func(ctx context.Context, s *Services) error {
				return s.Settings.SetValue(ctx, key, "x", "")
			}); err == nil {
				t.Errorf("SetValue setzt %s", key)
			}
		}
	}
	var company *domain.CompanySettings
	if err := tenant.Read(agent, func(ctx context.Context, s *Services) (err error) {
		company, err = s.Settings.GetCompanySettings(ctx)
		return err
	}); err != nil || company.LegalForm != domain.LegalFormGmbH || company.CompanyName != "Pfennig Ventures GmbH" {
		t.Errorf("Unternehmensdaten nach den Versuchen: %+v, %v", company, err)
	}

	// Gegenprobe: dieselben Wege stehen der App offen, und im Prüfermodus
	// liest der Agent weiter, schreibt aber nicht.
	if _, err := tenant.EnableExaminer(uiCtx, until, "Betriebsprüfung"); err != nil {
		t.Fatalf("die App schaltet den Prüfermodus ein: %v", err)
	}
	if err := tenant.Read(agent, func(context.Context, *Services) error { return nil }); err != nil {
		t.Errorf("ein Agent liest im Prüfermodus nicht: %v", err)
	}
	if err := tenant.Write(agent, func(ctx context.Context, s *Services) error {
		_, err := post(ctx, s)
		return err
	}); err == nil || !strings.Contains(err.Error(), "Prüfermodus") {
		t.Errorf("ein Agent bucht im Prüfermodus: %v", err)
	}
	if _, err := tenant.CheckVatID(agent, 1); err == nil || !strings.Contains(err.Error(), "Prüfermodus") {
		t.Errorf("ein Agent fragt im Prüfermodus beim Bundeszentralamt an: %v", err)
	}
}
