package core

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ops"
	"github.com/buchfink/buchfink/internal/service"
)

// Jede kanalgeschützte AppOnly-Operation hat ausführbare Proben für sämtliche
// Schutzstellen. Dienste werden ohne core.AppOnly aufgerufen, damit auch ihre
// eigene Prüfung wirkt. Die private App-Verwaltung wird strukturell geprüft.
func TestAgentsAreRefusedForHumanOnlyOperations(t *testing.T) {
	h := newHost(t)
	tc := newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	agent := agentCtx(t)
	until := coreTestNow().AddDate(0, 0, 10).Format("2006-01-02")
	inService := func(fn func(ctx context.Context, s *Services) error) func(context.Context) error {
		return func(ctx context.Context) error { return tenant.Write(ctx, fn) }
	}

	cases := map[string]func(context.Context) error{
		"core.Tenant.EnableExaminer":  func(ctx context.Context) error { _, err := tenant.EnableExaminer(ctx, until, "Prüfung"); return err },
		"core.Tenant.DisableExaminer": func(ctx context.Context) error { _, err := tenant.DisableExaminer(ctx, "Ende"); return err },
		"core.Tenant.CommitPeriod/Monat": func(ctx context.Context) error {
			_, err := tenant.CommitPeriod(ctx, CommitRequest{PeriodType: "month", PeriodLabel: "Jan", CutoffDate: "2026-01-31"})
			return err
		},
		"core.Tenant.CommitPeriod/Jahr": func(ctx context.Context) error {
			_, err := tenant.CommitPeriod(ctx, CommitRequest{PeriodType: domain.PeriodTypeYear, PeriodLabel: "2026", CutoffDate: "2026-12-31"})
			return err
		},
		"service.PeriodService.Commit": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Periods.Commit(ctx, service.PeriodCommitRequest{PeriodType: "month", CutoffDate: "2026-01-31"})
			return err
		}),
		"service.PeriodService.Adopt/Aufstellen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Periods.Adopt(ctx, 2026, domain.FiscalYearPrepared, "2027-03-01", "")
			return err
		}),
		"service.PeriodService.Adopt/Feststellen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Periods.Adopt(ctx, 2026, domain.FiscalYearAdopted, "2027-06-01", "Beschluss")
			return err
		}),
		"service.PeriodService.Adopt/Offenlegen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Periods.Adopt(ctx, 2026, domain.FiscalYearDisclosed, "2027-07-01", "Auftrag")
			return err
		}),
		"service.PeriodService.Reopen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Periods.Reopen(ctx, 2026, "Korrektur")
			return err
		}),
		"service.ClosingService.SetFiscalYearStatus/Aufstellen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-01", "")
			return err
		}),
		"service.ClosingService.SetFiscalYearStatus/Feststellen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearAdopted, "2027-06-01", "")
			return err
		}),
		"service.ClosingService.SetFiscalYearStatus/Offenlegen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearDisclosed, "2027-12-01", "Auftrag 1")
			return err
		}),
		"service.ClosingService.ReopenFiscalYear": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Closing.ReopenFiscalYear(ctx, 2026, "Korrektur")
			return err
		}),
		"service.VatReturnService.ConfirmSubmitted": inService(func(ctx context.Context, s *Services) error {
			_, err := s.VatReturns.ConfirmSubmitted(ctx, 1, "2026-02-10", "et-123", "")
			return err
		}),
		"service.ZMService.ConfirmSubmitted": inService(func(ctx context.Context, s *Services) error {
			_, err := s.ZM.ConfirmSubmitted(ctx, 1, "2026-02-25", "et-123", "")
			return err
		}),
		"service.SettingsService.UpdateCompanySettings": inService(func(ctx context.Context, s *Services) error {
			return s.Settings.UpdateCompanySettings(ctx, &domain.CompanySettings{CompanyName: "Andere GmbH"})
		}),
		"service.BankService.ConfigureAccounts": inService(func(ctx context.Context, s *Services) error {
			return s.Bank.ConfigureAccounts(ctx, nil)
		}),
		"service.BankService.SetInvoiceAccount": inService(func(ctx context.Context, s *Services) error {
			return s.Bank.SetInvoiceAccount(ctx, "DE02120300000000202051")
		}),
		"service.FoundationService.Save": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Foundation.Save(ctx, &domain.Foundation{NotarizedOn: "2026-01-10", ShareCapital: 2_500_000})
			return err
		}),
		"service.FoundationService.Register": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Foundation.Register(ctx, "2026-02-01", "AG Köln", "HRB 4711")
			return err
		}),
		"service.ClosingSettingsService.Save": inService(func(ctx context.Context, s *Services) error {
			_, err := s.ClosingSettings.Save(ctx, service.ClosingSettings{
				TradeTaxRatePercent: 250, AccrualMethod: "monthly", AccrualRelease: "yearly"})
			return err
		}),
		"service.SettingsService.SaveAgentSettings": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Settings.SaveAgentSettings(ctx, domain.AgentSettings{DirectMaxGross: 99_999_999})
			return err
		}),
		"service.SettingsService.SetInboxFolder": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Settings.SetInboxFolder(ctx, t.TempDir())
			return err
		}),
		"service.ContactService.ConfirmDefaultPostingGroup": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Contacts.ConfirmDefaultPostingGroup(ctx, 1, "software")
			return err
		}),
		"service.ObligationService.Confirm/USt": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Obligations.Confirm(ctx, "ust-erklaerung.2025", service.ConfirmInput{Date: "2026-03-01"})
			return err
		}),
		"service.ObligationService.Confirm/Sondervorauszahlung": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Obligations.Confirm(ctx, "sondervorauszahlung.2026", service.ConfirmInput{Date: "2026-02-10"})
			return err
		}),
		"service.ObligationService.Waive": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Obligations.Waive(ctx, "gewerbeanmeldung.gruendung", "freiberuflich")
			return err
		}),
		"service.ObligationService.Reopen": inService(func(ctx context.Context, s *Services) error {
			_, err := s.Obligations.Reopen(ctx, "kst-erklaerung.2025", "Irrtum")
			return err
		}),
		"core.Host.SetBackupDir": func(ctx context.Context) error { _, err := h.SetBackupDir(ctx, t.TempDir()); return err },
		"core.Host.Switch":       func(ctx context.Context) error { _, err := h.Switch(ctx, tc.ID); return err },
		"core.Host.CreateTenant": func(ctx context.Context) error {
			_, err := h.CreateTenant(ctx, "Zweite GmbH", filepath.Join(t.TempDir(), "neu"), domain.CompanySettings{})
			return err
		},
		"core.Host.RemoveTenant": func(ctx context.Context) error { return h.RemoveTenant(ctx, tc.ID) },
		"core.Host.RegisterTenant": func(ctx context.Context) error {
			_, err := h.RegisterTenant(ctx, filepath.Join(tc.DataDir, "buchfink.sqlite"), "", domain.MigrationKindOpen)
			return err
		},
		"core.Host.RestoreBackup": func(ctx context.Context) error {
			_, err := h.RestoreBackup(ctx, filepath.Join(t.TempDir(), "x.zip"), t.TempDir())
			return err
		},
	}
	confirmed := service.InputRequest{Kind: declaration.KindCorporateTax, Year: 2026,
		Key: "kst.v.z14", Value: declaration.Cents(500000)}
	if _, err := tenant.SetDeclarationInput(uiCtx, confirmed); err != nil {
		t.Fatal(err)
	}
	cases["core.Host.SetMCPPort"] = func(ctx context.Context) error { return h.SetMCPPort(ctx, 7466) }
	cases["core.Tenant.changeExaminer"] = func(ctx context.Context) error {
		_, err := tenant.changeExaminer(ctx, func(context.Context, *Services, *domain.ExaminerPeriod) (*domain.ExaminerPeriod, string, error) {
			t.Fatal("der geschützte Prüfermodus-Callback wurde erreicht")
			return nil, "", nil
		})
		return err
	}
	cases["core.Tenant.ConfirmDeclarationInput"] = func(ctx context.Context) error {
		_, err := tenant.ConfirmDeclarationInput(ctx, confirmed.Kind, confirmed.Year, confirmed.Key)
		return err
	}
	cases["core.Tenant.SubmitDeclaration"] = func(ctx context.Context) error {
		_, err := tenant.SubmitDeclaration(ctx, service.SubmissionRequest{})
		return err
	}
	cases["service.DeclarationService.ConfirmInput"] = inService(func(ctx context.Context, s *Services) error {
		_, err := s.Declarations.ConfirmInput(ctx, confirmed.Kind, confirmed.Year, confirmed.Key)
		return err
	})
	cases["service.DeclarationService.SetInput"] = inService(func(ctx context.Context, s *Services) error {
		_, err := s.Declarations.SetInput(ctx, confirmed)
		return err
	})
	cases["service.DeclarationService.DeleteInput"] = inService(func(ctx context.Context, s *Services) error {
		return s.Declarations.DeleteInput(ctx, confirmed.Kind, confirmed.Year, confirmed.Key)
	})
	cases["service.DeclarationService.MarkSubmitted"] = inService(func(ctx context.Context, s *Services) error {
		_, err := s.Declarations.MarkSubmitted(ctx, service.SubmissionRequest{})
		return err
	})
	checks := map[string]func(*testing.T, context.Context){}
	for name, run := range cases {
		checks[name] = func(t *testing.T, ctx context.Context) {
			err := run(ctx)
			switch _, ok := actor.ChannelOf(ctx); {
			case !ok:
				if !errors.Is(err, ErrNoChannel) {
					t.Errorf("ohne Erfassungsweg: %v", err)
				}
			case actor.RecordedChannel(ctx) == "unbekannt":
				if !errors.Is(err, ErrUnknownChannel) {
					t.Errorf("mit unbekanntem Erfassungsweg: %v", err)
				}
			default:
				if !errors.Is(err, ErrAppOnly) {
					t.Errorf("Vorbehalt der App fehlt: %v", err)
				}
			}
		}
	}
	checks["service.keepPostingGroup"] = func(t *testing.T, ctx context.Context) {
		contact := domain.Contact{Type: domain.ContactTypeVendor, Name: "Gruppe bleibt Vorschlag", DefaultPostingGroup: "software"}
		err := tenant.Write(ctx, func(ctx context.Context, s *Services) error { return s.Contacts.SaveContact(ctx, &contact) })
		if _, ok := actor.ChannelOf(ctx); !ok {
			if !errors.Is(err, ErrNoChannel) {
				t.Errorf("ohne Erfassungsweg: %v", err)
			}
			return
		}
		if actor.RecordedChannel(ctx) == "unbekannt" {
			if !errors.Is(err, ErrUnknownChannel) {
				t.Errorf("unbekannter Erfassungsweg: %v", err)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if contact.DefaultPostingGroup != "" || contact.DefaultPostingGroupConfirmedAt != "" || contact.DefaultPostingGroupConfirmedBy != "" {
			t.Errorf("Standardgruppe über SaveContact bestätigt: %+v", contact)
		}
	}
	covered := map[string]bool{}
	channels := map[string]context.Context{
		"agent": agent, "anderer_agent": actor.WithChannel(context.Background(), "agent:anderer-client"),
		"system": systemCtx, "ohne": context.Background(),
		"unbekannt": actor.WithChannel(context.Background(), "unbekannt"),
	}
	for _, op := range ops.All() {
		if op.Tier != ops.AppOnly || op.Enforcement != ops.Channel {
			continue
		}
		t.Run(op.Name, func(t *testing.T) {
			for _, guard := range op.Guards {
				found := false
				for name, check := range checks {
					site, _, _ := strings.Cut(name, "/")
					if site != guard {
						continue
					}
					found, covered[name] = true, true
					for channel, ctx := range channels {
						t.Run(name+"/"+channel, func(t *testing.T) { check(t, ctx) })
					}
				}
				if !found {
					t.Errorf("keine ausführbare Probe für %s", guard)
				}
			}
		})
	}
	for name := range checks {
		if !covered[name] {
			t.Errorf("Probe %s gehört zu keiner AppOnly-Operation", name)
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
