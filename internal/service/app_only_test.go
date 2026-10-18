package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
)

// Übermittlungsbestätigung, Pflicht als erledigt, Aufstellen, Feststellen und
// Offenlegen, Stammdaten samt Gesellschaftern und Gründung, Bankkonten,
// Abschluss-Einstellungen, Eingang, Regeln für Apps und das Bestätigen von
// Standardgruppe und Erklärungsangaben weist der Dienst selbst jedem
// Erfassungsweg außer der App ab — gleich, ob der Aufrufer den Vorgang als
// App-vorbehalten gekennzeichnet hat.
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
	status := func(s domain.FiscalYearStatus) func() error {
		return func() error {
			_, err := (&ClosingService{}).SetFiscalYearStatus(ctx, 2025, s, "2026-06-30", "")
			return err
		}
	}
	for name, call := range map[string]func() error{
		"Perioden festschreiben": func() error {
			_, err := (&PeriodService{}).Commit(ctx, PeriodCommitRequest{})
			return err
		},
		"Perioden aufstellen": func() error {
			_, err := (&PeriodService{}).Adopt(ctx, 2025, domain.FiscalYearPrepared, "2026-03-01", "")
			return err
		},
		"Perioden feststellen": func() error {
			_, err := (&PeriodService{}).Adopt(ctx, 2025, domain.FiscalYearAdopted, "2026-06-01", "Beschluss")
			return err
		},
		"Perioden offenlegen": func() error {
			_, err := (&PeriodService{}).Adopt(ctx, 2025, domain.FiscalYearDisclosed, "2026-07-01", "Auftrag")
			return err
		},
		"Periodenfeststellung zurücknehmen": func() error {
			_, err := (&PeriodService{}).Reopen(ctx, 2025, "Korrektur")
			return err
		},
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
		"Übermittlung einer Jahreserklärung": func() error {
			_, err := (&DeclarationService{}).MarkSubmitted(ctx, SubmissionRequest{})
			return err
		},
		"Angabe zur Erklärung bestätigen": func() error {
			_, err := (&DeclarationService{}).ConfirmInput(ctx, declaration.KindCorporateTax, 2025, "kst.v.z14")
			return err
		},
		"Aufstellen":  status(domain.FiscalYearPrepared),
		"Feststellen": status(domain.FiscalYearAdopted),
		"Offenlegen":  status(domain.FiscalYearDisclosed),
		"Feststellung zurücknehmen": func() error {
			_, err := (&ClosingService{}).ReopenFiscalYear(ctx, 2025, "Fehler")
			return err
		},
		"Bankkonten": func() error { return (&BankService{}).ConfigureAccounts(ctx, nil) },
		"Rechnungskonto": func() error {
			return (&BankService{}).SetInvoiceAccount(ctx, "DE02120300000000202051")
		},
		"Gründung samt Gesellschaftern erfassen": func() error {
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
		"Regeln für Apps": func() error {
			_, err := (&SettingsService{}).SaveAgentSettings(ctx, domain.AgentSettings{})
			return err
		},
		"Eingang wählen": func() error {
			_, err := (&SettingsService{}).SetInboxFolder(ctx, "/tmp/eingang")
			return err
		},
		"Standardgruppe bestätigen": func() error {
			_, err := (&ContactService{}).ConfirmDefaultPostingGroup(ctx, 1, "software")
			return err
		},
		"USt-Jahreserklärung bestätigen": func() error {
			_, err := (&ObligationService{}).Confirm(ctx, "ust-erklaerung.2025", ConfirmInput{Date: "2026-03-01"})
			return err
		},
		"Sondervorauszahlung bestätigen": func() error {
			_, err := (&ObligationService{}).Confirm(ctx, "sondervorauszahlung.2026", ConfirmInput{Date: "2026-02-10"})
			return err
		},
		"Pflicht als entfallen kennzeichnen": func() error {
			_, err := (&ObligationService{}).Waive(ctx, "gewerbeanmeldung.gruendung", "freiberuflich")
			return err
		},
		"Bestätigung zurücknehmen": func() error {
			_, err := (&ObligationService{}).Reopen(ctx, "kst-erklaerung.2025", "Irrtum")
			return err
		},
	} {
		err := call()
		if err == nil || !strings.Contains(err.Error(), "App vorbehalten") || !errors.Is(err, actor.ErrAppOnly) {
			t.Errorf("%s über %q wurde nicht als App-vorbehalten abgewiesen: %v", name, channel, err)
		}
	}
}

// Was sich per Storno korrigieren lässt, bucht ein Agent selbst: Kapital,
// Rücklagen, Ausschüttung und Gesellschafterkonten von Hand, als Neubuchung
// beim Storno und in der Aufteilung eines Bankumsatzes; dazu die
// Organisationstexte der Verfahrensdokumentation.
func TestAgentsBookWhatAStornoCorrects(t *testing.T) {
	env := newTestEnv(t)
	env.withShareholders(t)
	mara := env.person(t, "Mara Beispiel")
	payments := env.paymentsWithBank(t)
	manual := func(receipt *domain.Receipt, debit, credit string, amount domain.Cents) domain.JournalEntry {
		return domain.JournalEntry{BookingDate: "2026-03-01", DocumentDate: "2026-03-01",
			ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01", Description: "Gesellschafterbeschluss",
			TaxTreatment: domain.TaxTreatmentNotTaxable, ReceiptID: &receipt.ID,
			Lines: []domain.JournalLine{{Side: domain.SideDebit, Account: debit, Amount: amount},
				{Side: domain.SideCredit, Account: credit, Amount: amount}}}
	}

	contract := env.fileIncoming(t, "gesellschaftsvertrag.pdf")
	subscription, err := env.posting.PostManualEntry(agentCtx, ManualEntryRequest{ReceiptID: contract.ID,
		Entry: manual(contract, domain.AccountAusstehendeEinlagenGeford, domain.AccountGezeichnetesKapital, 2_500_000)})
	if err != nil {
		t.Fatalf("Zeichnung des Stammkapitals durch den Agenten: %v", err)
	}
	if subscription.Channel != agentChannel(t) {
		t.Errorf("die Zeichnung trägt den Erfassungsweg %q", subscription.Channel)
	}

	resolution := env.fileIncoming(t, "beschluss.pdf")
	reserve, err := env.posting.PostManualEntry(agentCtx, ManualEntryRequest{ReceiptID: resolution.ID,
		Entry: manual(resolution, domain.AccountGewinnvortrag, domain.AccountAndereGewinnruecklagen, 100_000)})
	if err != nil {
		t.Fatalf("Einstellung in die Rücklagen durch den Agenten: %v", err)
	}
	fix := manual(resolution, domain.AccountGewinnvortrag, domain.AccountAusschuettung, 100_000)
	if _, err := env.journal.CorrectEntry(agentCtx, reserve.ID, "Ausschüttung statt Rücklage beschlossen", &fix); err != nil {
		t.Fatalf("Storno mit Neubuchung auf 3519 durch den Agenten: %v", err)
	}

	tx := env.bankLine(t, "2026-03-10", -5_000, "Mara Beispiel", "Erstattung Auslagen")
	if _, err := payments.BookBankSplit(agentCtx, BankSplitRequest{BankTxID: tx,
		Lines: []BankSplitLine{{Account: domain.AccountVerbindlichkeitenGesellschafter, ContactID: mara.ID, Amount: 5_000}}}); err != nil {
		t.Fatalf("Aufteilung auf das Gesellschafterkonto durch den Agenten: %v", err)
	}

	procdoc := newProcDocService(t, env)
	texts := procdoc.OrganisationTexts(agentCtx)
	texts.Scanning = "Papierbelege scannt die Geschäftsführerin am Tag des Eingangs."
	if err := procdoc.SaveOrganisationTexts(agentCtx, texts); err != nil {
		t.Fatalf("Organisationstexte durch den Agenten: %v", err)
	}
	if got := procdoc.OrganisationTexts(appCtx).Scanning; got != texts.Scanning {
		t.Errorf("Scannen: %q", got)
	}
}

func agentChannel(t *testing.T) string {
	t.Helper()
	channel, ok := actor.ChannelOf(agentCtx)
	if !ok {
		t.Fatal("agentCtx trägt keinen Erfassungsweg")
	}
	return string(channel)
}
