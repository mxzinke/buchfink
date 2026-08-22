// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"gorm.io/gorm"
)

type testEnv struct {
	db          *gorm.DB
	journal     *JournalService
	posting     *PostingService
	accounting  *AccountingService
	contacts    *ContactService
	journalRepo domain.JournalRepository
	contactRepo domain.ContactRepository
	numberRepo  domain.NumberRangeRepository
	fiscalYear  int
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	db, err := repository.InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank konnte nicht angelegt werden: %v", err)
	}
	if err := repository.SeedDefaultsIfEmpty(context.Background(), db, 2026); err != nil {
		t.Fatalf("SKR04-Kontenplan konnte nicht geladen werden: %v", err)
	}

	accountRepo := repository.NewAccountRepository(db)
	journalRepo := repository.NewJournalRepository(db)
	contactRepo := repository.NewContactRepository(db)
	numberRepo := repository.NewNumberRangeRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	journal := NewJournalService(journalRepo, accountRepo, contactRepo, auditRepo, settingsRepo, 2026)
	posting := NewPostingService(journal, contactRepo)
	acc := NewAccountingService(accountRepo, journalRepo, contactRepo, settingsRepo, journal, 2026)
	contacts := NewContactService(contactRepo, journalRepo, numberRepo, auditRepo, 2026)

	return &testEnv{
		db: db, journal: journal, posting: posting, accounting: acc, contacts: contacts,
		journalRepo: journalRepo, contactRepo: contactRepo, numberRepo: numberRepo,
		fiscalYear: 2026,
	}
}

func (e *testEnv) vendor(t *testing.T, name, country, vatID string) *domain.Contact {
	t.Helper()
	c := &domain.Contact{Type: domain.ContactTypeVendor, Name: name, CountryCode: country, VatID: vatID}
	if err := e.contacts.SaveContact(context.Background(), c); err != nil {
		t.Fatalf("Lieferant %s konnte nicht angelegt werden: %v", name, err)
	}
	return c
}

func (e *testEnv) customer(t *testing.T, name, country, vatID string) *domain.Contact {
	t.Helper()
	c := &domain.Contact{Type: domain.ContactTypeCustomer, Name: name, CountryCode: country, VatID: vatID}
	if err := e.contacts.SaveContact(context.Background(), c); err != nil {
		t.Fatalf("Kunde %s konnte nicht angelegt werden: %v", name, err)
	}
	return c
}

// simpleEntry is a minimal balanced booking used to exercise the journal rules.
func simpleEntry(debit, credit string, amount domain.Cents) *domain.JournalEntry {
	return &domain.JournalEntry{
		BookingDate:     "2026-03-01",
		DocumentDate:    "2026-03-01",
		ServiceDateFrom: "2026-03-01",
		ServiceDateTo:   "2026-03-01",
		Description:     "Testbuchung",
		Source:          domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: debit, Amount: amount},
			{Side: domain.SideCredit, Account: credit, Amount: amount},
		},
	}
}

// --- Grundinvarianten -----------------------------------------------------

func TestPostRejectsUnbalancedEntry(t *testing.T) {
	env := newTestEnv(t)

	entry := simpleEntry("6815", "1800", 10000)
	entry.Lines[1].Amount = 9900

	_, err := env.journal.Post(context.Background(), entry)
	if err == nil {
		t.Fatal("eine unausgeglichene Buchung darf nicht ins Journal gelangen")
	}
	if !strings.Contains(err.Error(), "nicht ausgeglichen") {
		t.Errorf("die Fehlermeldung sollte die Unausgeglichenheit benennen, lautet aber: %v", err)
	}
}

func TestPostRejectsUnknownAndUnpostableAccounts(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	cases := []struct {
		account string
		reason  string
	}{
		{"9999999", "eine Nummer außerhalb des Kontenrahmens"},
		{"4400-4409", "ein Bereichskonto"},
		{"8400", "die SKR03-Erlösnummer in der reservierten Klasse 8"},
		{"0055", "ein reserviertes Konto"},
	}

	for _, c := range cases {
		_, err := env.journal.Post(ctx, simpleEntry(c.account, "1800", 10000))
		if err == nil {
			t.Errorf("auf %s (%s) darf nicht gebucht werden", c.account, c.reason)
		}
	}
}

// Tax accounts hold the figures of the Umsatzsteuer-Voranmeldung. A hand-written
// line on one of them would make the return diverge from the journal.
func TestPostRejectsManualBookingOnTaxAccount(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.journal.Post(context.Background(), simpleEntry(domain.AccountVorsteuer19, "1800", 1900))
	if err == nil {
		t.Fatal("Steuerkonten dürfen nicht von Hand bebucht werden")
	}
	if !strings.Contains(err.Error(), "Steuerautomatik") {
		t.Errorf("die Fehlermeldung sollte auf die Steuerautomatik verweisen, lautet aber: %v", err)
	}
}

func TestPostRejectsIncompleteDates(t *testing.T) {
	env := newTestEnv(t)

	entry := simpleEntry("6815", "1800", 10000)
	entry.ServiceDateTo = "2026-02-01" // liegt vor dem Leistungsbeginn

	if _, err := env.journal.Post(context.Background(), entry); err == nil {
		t.Fatal("ein Leistungsende vor dem Leistungsbeginn muss abgewiesen werden")
	}
}

// --- Nummernkreis und Hash-Chain -----------------------------------------

func TestEntryNumbersAreGaplessAndChained(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	var numbers []string
	for i := 0; i < 5; i++ {
		entry, err := env.journal.Post(ctx, simpleEntry("6815", "1800", domain.Cents(1000*(i+1))))
		if err != nil {
			t.Fatalf("Buchung %d: %v", i+1, err)
		}
		numbers = append(numbers, entry.EntryNumber)
	}

	want := []string{"2026-000001", "2026-000002", "2026-000003", "2026-000004", "2026-000005"}
	for i := range want {
		if numbers[i] != want[i] {
			t.Errorf("Buchungsnummer %d = %q, erwartet %q", i+1, numbers[i], want[i])
		}
	}

	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if !result.IsValid || result.CheckedEntries != 5 {
		t.Errorf("die Hash-Chain sollte 5 gültige Buchungen melden, meldet aber: %+v", result)
	}
}

// A failed booking must not consume a number — otherwise the series has a gap
// the company cannot explain to an auditor.
func TestFailedPostDoesNotConsumeNumber(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000)); err != nil {
		t.Fatalf("erste Buchung: %v", err)
	}

	// This one fails inside the transaction: the account does not exist.
	if _, err := env.journal.Post(ctx, simpleEntry("6815", "9999999", 10000)); err == nil {
		t.Fatal("die Buchung auf ein unbekanntes Konto hätte scheitern müssen")
	}

	entry, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 20000))
	if err != nil {
		t.Fatalf("dritte Buchung: %v", err)
	}
	if entry.EntryNumber != "2026-000002" {
		t.Errorf("nach einer gescheiterten Buchung muss die nächste Nummer 2026-000002 sein, ist aber %q", entry.EntryNumber)
	}
}

// The hash has to cover the fields that carry accounting meaning. The previous
// implementation left the Buchungstext and the tax amount out, so both could be
// changed without breaking the chain.
func TestHashCoversDescriptionAndTaxFields(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	entry, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}

	// Tamper with the stored row the way a manual database edit would.
	if err := env.db.Model(&domain.JournalEntry{}).Where("id = ?", entry.ID).
		Update("description", "Manipulierter Buchungstext").Error; err != nil {
		t.Fatalf("Testmanipulation fehlgeschlagen: %v", err)
	}

	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if result.IsValid {
		t.Error("ein geänderter Buchungstext muss die Hash-Chain brechen")
	}
}

func TestHashCoversLineAmounts(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	entry, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}

	if err := env.db.Model(&domain.JournalLine{}).
		Where("entry_id = ? AND side = ?", entry.ID, domain.SideDebit).
		Update("amount", 50000).Error; err != nil {
		t.Fatalf("Testmanipulation fehlgeschlagen: %v", err)
	}

	result, _ := env.journal.VerifyIntegrity(ctx)
	if result.IsValid {
		t.Error("ein geänderter Zeilenbetrag muss die Hash-Chain brechen")
	}
}

// --- Generalumkehr --------------------------------------------------------

// The point of a Generalumkehr is that it returns the turnover of the affected
// accounts to zero. A side-swapped Storno would leave 1.000 € on both sides and
// inflate the Summen- und Saldenliste.
func TestReverseIsGeneralumkehrAndClearsTurnover(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	original, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatalf("Ursprungsbuchung: %v", err)
	}

	reversal, err := env.journal.Reverse(ctx, original.ID, "Beleg doppelt erfasst")
	if err != nil {
		t.Fatalf("Storno: %v", err)
	}

	if reversal.Kind != domain.EntryKindReversal {
		t.Errorf("die Stornobuchung muss als Generalumkehr gekennzeichnet sein, ist aber %q", reversal.Kind)
	}

	for i, line := range reversal.Lines {
		orig := original.Lines[i]
		if line.Side != orig.Side {
			t.Errorf("Zeile %d: die Buchungsseite muss erhalten bleiben (%s), ist aber %s – das wäre ein Seitentausch, keine Generalumkehr",
				i+1, orig.Side, line.Side)
		}
		if line.Amount != -orig.Amount {
			t.Errorf("Zeile %d: der Betrag muss negiert sein (%s), ist aber %s", i+1, -orig.Amount, line.Amount)
		}
	}

	turnovers, err := env.journalRepo.AccountTurnovers(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Verkehrszahlen: %v", err)
	}
	expense := turnovers["6815"]
	if expense.Debit != 0 || expense.Credit != 0 {
		t.Errorf("nach der Generalumkehr müssen die Verkehrszahlen von 6815 auf null stehen, sind aber Soll %s / Haben %s",
			expense.Debit, expense.Credit)
	}

	accounts, err := env.accounting.GetAccounts(ctx)
	if err != nil {
		t.Fatalf("Konten: %v", err)
	}
	for _, a := range accounts {
		if a.Number == "6815" && a.Balance != 0 {
			t.Errorf("der Saldo von 6815 muss null sein, ist aber %s", a.Balance)
		}
	}
}

func TestReverseRefusesDoubleAndChainedStorno(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	original, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatalf("Ursprungsbuchung: %v", err)
	}

	reversal, err := env.journal.Reverse(ctx, original.ID, "Grund")
	if err != nil {
		t.Fatalf("Storno: %v", err)
	}

	if _, err := env.journal.Reverse(ctx, original.ID, "nochmal"); err == nil {
		t.Error("eine bereits stornierte Buchung darf nicht erneut storniert werden")
	}
	if _, err := env.journal.Reverse(ctx, reversal.ID, "Storno des Stornos"); err == nil {
		t.Error("eine Generalumkehr darf nicht selbst storniert werden")
	}
	if _, err := env.journal.Reverse(ctx, original.ID, ""); err == nil {
		t.Error("eine Stornierung ohne Grund darf nicht möglich sein")
	}
}

// --- Festschreibung -------------------------------------------------------

func TestCommittedPeriodBlocksBackdatedBooking(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	festRepo := repository.NewFestschreibungRepository(env.db)
	env.journal.SetFestschreibungRepo(festRepo)

	if err := festRepo.Create(ctx, &domain.Festschreibung{
		FiscalYear: 2026, PeriodType: "month", PeriodLabel: "März 2026",
		CutoffDate: "2026-03-31", ChainHead: domain.GenesisHash,
	}); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}

	entry := simpleEntry("6815", "1800", 10000)
	entry.BookingDate = "2026-03-15"
	if _, err := env.journal.Post(ctx, entry); err == nil {
		t.Error("eine Buchung in einen festgeschriebenen Zeitraum muss abgewiesen werden")
	}

	later := simpleEntry("6815", "1800", 10000)
	later.BookingDate = "2026-04-02"
	if _, err := env.journal.Post(ctx, later); err != nil {
		t.Errorf("eine Buchung nach dem Festschreibungsstichtag muss möglich bleiben: %v", err)
	}
}

// --- Personenkonten -------------------------------------------------------

func TestLedgerAccountsComeFromDATEVRanges(t *testing.T) {
	env := newTestEnv(t)

	customer := env.customer(t, "Kunde Alpha", "DE", "")
	vendor := env.vendor(t, "Lieferant Beta", "DE", "")

	if customer.LedgerAccount != "10000" {
		t.Errorf("das erste Debitorenkonto muss 10000 sein, ist aber %q", customer.LedgerAccount)
	}
	if vendor.LedgerAccount != "70000" {
		t.Errorf("das erste Kreditorenkonto muss 70000 sein, ist aber %q", vendor.LedgerAccount)
	}

	second := env.customer(t, "Kunde Gamma", "DE", "")
	if second.LedgerAccount != "10001" {
		t.Errorf("das zweite Debitorenkonto muss 10001 sein, ist aber %q", second.LedgerAccount)
	}

	if kind, ok := domain.LedgerAccountKind("10000"); !ok || kind != domain.ContactTypeCustomer {
		t.Error("10000 muss als Debitorenkonto erkannt werden")
	}
	if kind, ok := domain.LedgerAccountKind("70000"); !ok || kind != domain.ContactTypeVendor {
		t.Error("70000 muss als Kreditorenkonto erkannt werden")
	}
	if domain.IsLedgerAccount("6815") {
		t.Error("ein vierstelliges Sachkonto ist kein Personenkonto")
	}
}

func TestPostRejectsUnknownLedgerAccount(t *testing.T) {
	env := newTestEnv(t)

	entry := simpleEntry("6815", "70000", 10000)
	if _, err := env.journal.Post(context.Background(), entry); err == nil {
		t.Fatal("eine Buchung auf ein Personenkonto ohne Geschäftspartner muss abgewiesen werden")
	}
}

func TestPostingRuleVersionIsRecorded(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	vendor := env.vendor(t, "Lieferant", "DE", "")
	entry, err := env.posting.PostIncomingReceipt(ctx, ReceiptRequest{
		ContactID:      vendor.ID,
		DocumentDate:   "2026-03-01",
		BookingDate:    "2026-03-01",
		DocumentNumber: "RE-4711",
		TaxTreatment:   domain.TaxTreatmentDomestic,
		Positions:      []ReceiptPosition{{PostingGroup: "buerobedarf", Net: 5000, TaxRate: domain.TaxRateStandard}},
		Settlement:     SettlementOpen,
	})
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}
	if entry.PostingRuleVersion != accounting.PostingRuleVersion {
		t.Errorf("die Buchung muss die Version des Kontierungsregelwerks tragen, hat aber %q", entry.PostingRuleVersion)
	}
}

// invoices wires the invoice service on demand; it needs the posting service,
// which the base environment already holds.
func (e *testEnv) invoices(t *testing.T) *InvoiceService {
	t.Helper()
	return NewInvoiceService(
		repository.NewInvoiceRepository(e.db),
		e.contactRepo,
		repository.NewSettingsRepository(e.db),
		e.numberRepo,
		e.posting,
		repository.NewAuditRepository(e.db),
	)
}

// Offene Posten gehören auf das Personenkonto. Eine Buchung direkt auf 1200
// oder 3300 stünde in der Bilanz, aber in keiner OPOS-Liste — zwei Wahrheiten
// für dieselbe Zahl.
func TestCollectiveAccountsAreNotPostable(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	for _, account := range []string{domain.AccountForderungenLuL, domain.AccountVerbindlichkeitenLuL} {
		_, err := env.journal.Post(ctx, simpleEntry(account, "1800", 10000))
		if err == nil {
			t.Errorf("auf das Sammelkonto %s darf nicht direkt gebucht werden", account)
			continue
		}
		if !strings.Contains(err.Error(), "Personenkonto") {
			t.Errorf("die Fehlermeldung zu %s sollte auf das Personenkonto verweisen, lautet aber: %v", account, err)
		}
	}
}

// Die Bilanzposition entsteht durch Verdichtung der Personenkonten, und die
// Oberfläche kann zeigen, aus wie vielen.
func TestCollectiveAccountReportsItsSources(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	first := env.vendor(t, "Lieferant A", "DE", "")
	second := env.vendor(t, "Lieferant B", "DE", "")
	for _, vendor := range []*domain.Contact{first, second} {
		if _, err := env.posting.PostIncomingReceipt(ctx,
			receipt(vendor.ID, "buerobedarf", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)); err != nil {
			t.Fatalf("Beleg für %s: %v", vendor.Name, err)
		}
	}

	accounts, err := env.accounting.GetAccounts(ctx)
	if err != nil {
		t.Fatalf("Konten: %v", err)
	}
	for _, a := range accounts {
		if a.Number != domain.AccountVerbindlichkeitenLuL {
			continue
		}
		if a.Balance != 238000 {
			t.Errorf("Verbindlichkeiten aus LuL = %s, erwartet 2.380,00", a.Balance)
		}
		if a.AggregatedAccounts != 2 {
			t.Errorf("die Bilanzposition muss aus 2 Kreditorenkonten verdichtet sein, gemeldet werden %d", a.AggregatedAccounts)
		}
		return
	}
	t.Error("Konto 3300 wurde in der Kontenliste nicht gefunden")
}
