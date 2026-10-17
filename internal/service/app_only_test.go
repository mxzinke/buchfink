package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
)

// Übermittlungsbestätigung, Abschlussstatus, Stammdaten samt Gründung,
// Abschluss-Einstellungen, Organisationsanweisung und Bankkonten weist der
// Dienst selbst jedem Erfassungsweg außer der App ab — gleich, ob der Aufrufer
// den Vorgang als App-vorbehalten gekennzeichnet hat.
func TestAgentsCannotTriggerAppOnlyOperations(t *testing.T) {
	channel, err := actor.Agent("claude-desktop")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []context.Context{
		actor.WithChannel(context.Background(), channel),
		actor.WithChannel(context.Background(), actor.ChannelSystem),
		actor.WithChannel(context.Background(), actor.Channel("mcp:claude")),
	} {
		refusesAppOnlyOperations(t, ctx)
	}

	if err := requireHuman(actor.WithChannel(context.Background(), actor.ChannelUI), "Test"); err != nil {
		t.Errorf("die App wird abgewiesen: %v", err)
	}
	if err := requireHuman(context.Background(), "Test"); err != nil {
		t.Errorf("ein Aufruf unterhalb des Kerns wird abgewiesen: %v", err)
	}
}

func refusesAppOnlyOperations(t *testing.T, ctx context.Context) {
	t.Helper()
	channel, _ := actor.ChannelOf(ctx)
	for name, call := range map[string]func() error{
		"Unternehmensdaten": func() error {
			return (&SettingsService{}).UpdateCompanySettings(ctx, &domain.CompanySettings{})
		},
		"Übermittlung UStVA": func() error {
			_, err := (&VatReturnService{}).ConfirmSubmitted(ctx, 1, "2026-02-10", "T", "")
			return err
		},
		"Übermittlung ZM": func() error {
			_, err := (&ZMService{}).ConfirmSubmitted(ctx, 1, "2026-02-10", "T", "")
			return err
		},
		"Feststellen": func() error {
			_, err := (&ClosingService{}).SetFiscalYearStatus(ctx, 2025, domain.FiscalYearStatus("adopted"), "2026-06-30", "")
			return err
		},
		"Feststellung zurücknehmen": func() error {
			_, err := (&ClosingService{}).ReopenFiscalYear(ctx, 2025, "Fehler")
			return err
		},
		"Bankkonten": func() error { return (&BankService{}).ConfigureAccounts(ctx, nil) },
		"Rechnungskonto": func() error {
			return (&BankService{}).SetInvoiceAccount(ctx, "DE02120300000000202051")
		},
		"Gründung erfassen": func() error {
			_, err := (&FoundationService{}).Save(ctx, &domain.Foundation{})
			return err
		},
		"Eintragung ins Handelsregister": func() error {
			_, err := (&FoundationService{}).Register(ctx, "2026-03-01", "AG Köln", "HRB 1")
			return err
		},
		"Abschluss-Einstellungen": func() error {
			_, err := (&ClosingSettingsService{}).Save(ctx, ClosingSettings{TradeTaxRatePercent: 300})
			return err
		},
		"Organisationsanweisung": func() error {
			return (&ProcDocService{}).SaveOrganisationTexts(ctx, domain.OrganisationTexts{})
		},
		"USt-Jahreserklärung abhaken": func() error {
			return (&DeadlineService{}).MarkDone(ctx, DeadlineKeyAnnualVat+".2025", "")
		},
		"Sondervorauszahlung abhaken": func() error {
			return (&DeadlineService{}).MarkDone(ctx, DeadlineKeyPrepayment+".2026", "")
		},
	} {
		err := call()
		if err == nil || !strings.Contains(err.Error(), "App vorbehalten") || !errors.Is(err, actor.ErrAppOnly) {
			t.Errorf("%s über %q wurde nicht als App-vorbehalten abgewiesen: %v", name, channel, err)
		}
	}
}
