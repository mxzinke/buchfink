package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
)

// Übermittlungsbestätigung, Abschlussstatus, Stammdaten und Bankkonten weist
// der Dienst selbst ab, wenn ein Agent sie auslöst — gleich, ob der Aufrufer
// den Vorgang als App-vorbehalten gekennzeichnet hat.
func TestAgentsCannotTriggerAppOnlyOperations(t *testing.T) {
	channel, err := actor.Agent("claude-desktop")
	if err != nil {
		t.Fatal(err)
	}
	ctx := actor.WithChannel(context.Background(), channel)

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
	} {
		err := call()
		if err == nil || !strings.Contains(err.Error(), "App vorbehalten") {
			t.Errorf("%s: ein Agent wurde nicht abgewiesen: %v", name, err)
		}
	}

	if err := refuseAgent(actor.WithChannel(context.Background(), actor.ChannelUI), "Test"); err != nil {
		t.Errorf("die App wird abgewiesen: %v", err)
	}
	if err := refuseAgent(context.Background(), "Test"); err != nil {
		t.Errorf("ein Programmlauf wird abgewiesen: %v", err)
	}
}
