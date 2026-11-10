package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestJournalCoreRulesWithoutExtensions(t *testing.T) {
	cases := []struct {
		name   string
		change func(*testing.T, *testEnv, *domain.JournalEntry)
		want   string
	}{
		{"gültig", nil, ""},
		{"Pflichtfeld", func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Description = "" }, "Buchungstext fehlt"},
		{"Ausgleich", func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Lines[1].Amount = 1800 }, "Buchung ist nicht ausgeglichen: Soll 19,00 €, Haben 18,00 € (Differenz 1,00 €)"},
		{"Konto", func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Lines[0].Account = "9999999" }, "Zeile 1: Konto 9999999 ist im SKR04 nicht vorhanden"},
		{"Personenkonto", func(t *testing.T, env *testEnv, e *domain.JournalEntry) {
			customer := env.customer(t, "Kunde", "DE", "")
			other := env.customer(t, "Anderer", "DE", "")
			e.Lines[0].Account, e.Lines[0].ContactID = customer.LedgerAccount, &other.ID
		}, "Zeile 1: Personenkonto 10000 gehört zu Kunde, die Buchung verweist aber auf einen anderen Geschäftspartner"},
		{"Steuerkonto", func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Lines[0].Account = "1406" }, "Zeile 1: Konto 1406 ist ein Steuerkonto und darf nur über die Steuerautomatik bebucht werden"},
		{"USTV-ohne-registrierte-Ausnahme", func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) {
			e.Lines[0].Account, e.Source, e.DocumentNumber = "1406", domain.EntrySourceClosing, "USTV 2026"
		}, "Zeile 1: Konto 1406 ist ein Steuerkonto und darf nur über die Steuerautomatik bebucht werden"},
		{"Feststellung", func(t *testing.T, env *testEnv, _ *domain.JournalEntry) {
			if err := repository.NewFiscalYearRepository(env.db).Save(context.Background(), &domain.FiscalYear{
				Year: 2026, StartDate: "2026-01-01", EndDate: "2026-12-31", Status: domain.FiscalYearAdopted, AdoptedOn: "2026-02-01",
			}); err != nil {
				t.Fatal(err)
			}
		}, "Der Jahresabschluss 2026 ist am 2026-02-01 festgestellt. Eine Buchung zum 2026-03-01 ist deshalb nicht mehr möglich. Muss der Abschluss geändert werden, setzen Sie die Feststellung unter Abschlüsse › Jahr mit Angabe des Grundes zurück"},
		{"Festschreibung", func(t *testing.T, env *testEnv, _ *domain.JournalEntry) { env.commitUntil(t, "2026-03-31") }, "Der Zeitraum bis 2026-03-31 ist festgeschrieben. Eine Buchung zum 2026-03-01 ist nicht mehr möglich – bitte über eine Stornierung korrigieren"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.journal = NewJournalService(env.journalRepo, repository.NewAccountRepository(env.db), env.contactRepo,
				repository.NewAuditRepository(env.db), repository.NewSettingsRepository(env.db), repository.NewTxRunner(env.db))
			env.journal.SetFiscalYearRepo(repository.NewFiscalYearRepository(env.db))
			env.journal.SetFestschreibungRepo(repository.NewFestschreibungRepository(env.db))
			e := simpleEntry("6815", "1800", 1900)
			if tc.change != nil {
				tc.change(t, env, e)
			}
			assertJournalRuleError(t, "ValidatePostable", env.journal.ValidatePostable(context.Background(), e), tc.want)
			_, err := env.journal.Post(context.Background(), e)
			assertJournalRuleError(t, "Post", err, tc.want)
		})
	}
}

func TestManualTaxRulePrecedesOtherManualChecks(t *testing.T) {
	env := newTestEnv(t)
	e := simpleEntry("6815", "1800", 1900)
	e.TaxTreatment, e.Description = "", ""
	_, err := env.posting.PostManualEntry(context.Background(), ManualEntryRequest{Entry: *e, ReceiptID: 42})
	const want = "der Buchung fehlt der Steuerfall. Er sagt, warum Steuer entsteht oder nicht — beim Inlandsumsatz, beim innergemeinschaftlichen Erwerb, bei § 13b UStG oder bei einem steuerfreien Vorgang. Ohne ihn läuft die Buchung an der Voranmeldung vorbei"
	assertJournalRuleError(t, "PostManualEntry", err, want)
}
