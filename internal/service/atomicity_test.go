package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// failingAudit lässt jeden Protokolleintrag scheitern, dessen Text den
// Auslöser enthält. Damit lässt sich der letzte Schritt eines Vorgangs zum
// Scheitern bringen, nachdem alles davor geschrieben ist.
type failingAudit struct {
	domain.AuditRepository
	trigger string
}

func (f failingAudit) Log(ctx context.Context, action domain.AuditAction, entityType, entityID, details string) error {
	if strings.Contains(entityType+" "+details, f.trigger) {
		return errors.New("Protokoll nicht beschreibbar")
	}
	return f.AuditRepository.Log(ctx, action, entityType, entityID, details)
}

func (f failingAudit) LogChange(
	ctx context.Context, action domain.AuditAction, entityType, entityID, details string, before, after any,
) error {
	if strings.Contains(entityType+" "+details, f.trigger) {
		return errors.New("Protokoll nicht beschreibbar")
	}
	return f.AuditRepository.LogChange(ctx, action, entityType, entityID, details, before, after)
}

// journalWithAudit baut einen Journaldienst auf derselben Datenbank, dessen
// Protokoll beim Auslöser scheitert.
func (e *testEnv) journalWithAudit(t *testing.T, trigger string) *JournalService {
	t.Helper()
	audit := failingAudit{AuditRepository: repository.NewAuditRepository(e.db), trigger: trigger}
	j := newJournalWithTaxRules(e.journalRepo, repository.NewAccountRepository(e.db), e.contactRepo,
		audit, repository.NewSettingsRepository(e.db), repository.NewTxRunner(e.db))
	j.SetReceiptRepo(e.receiptRepo)
	return j
}

func (e *testEnv) entryCount(t *testing.T) int {
	t.Helper()
	entries, err := e.journalRepo.FindAll(context.Background(), e.fiscalYear)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func TestPostFailsWhenItsAuditEntryFails(t *testing.T) {
	env := newTestEnv(t)
	journal := env.journalWithAudit(t, "Buchung")

	if _, err := journal.Post(context.Background(), simpleEntry("6815", "1800", 10000)); err == nil {
		t.Fatal("ohne Protokolleintrag darf die Buchung nicht stehen")
	}
	if n := env.entryCount(t); n != 0 {
		t.Errorf("%d Buchungen stehen ohne Protokolleintrag im Journal", n)
	}
}

func TestCorrectEntryLeavesNoReversalWhenTheReplacementFails(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	original, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatal(err)
	}

	// Vorprüfung und Storno gehen durch, der Protokolleintrag der Ersetzung
	// scheitert.
	journal := env.journalWithAudit(t, "ersetzt die stornierte Buchung")
	if _, err := journal.CorrectEntry(ctx, original.ID, "falscher Betrag", simpleEntry("6815", "1800", 9000)); err == nil {
		t.Fatal("die Korrektur muss am Protokoll scheitern")
	}
	if reversal, err := env.journalRepo.FindReversalOf(ctx, original.ID); err != nil || reversal != nil {
		t.Errorf("der Storno muss mit der gescheiterten Neubuchung zurückgerollt sein: %v, %v", reversal, err)
	}
	if n := env.entryCount(t); n != 1 {
		t.Errorf("%d Buchungen, erwartet nur die ursprüngliche", n)
	}
}

func TestBookDirectIsAtomic(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	txID := env.bankCredit(t, "2026-03-05", 25000, "E2E-ATOM")

	journal := env.journalWithAudit(t, "BANK_RULE")
	bank := NewBankService(repository.NewBankRepository(env.db), journal,
		failingAudit{AuditRepository: repository.NewAuditRepository(env.db), trigger: "BANK_RULE"})
	rules := repository.NewBankRuleRepository(env.db)
	bank.SetRuleRepo(rules)

	if _, err := bank.BookDirect(ctx, txID, "4830", "Erstattung"); err == nil {
		t.Fatal("der Vorgang muss am Protokoll der gelernten Regel scheitern")
	}
	if n := env.entryCount(t); n != 0 {
		t.Errorf("%d Buchungen stehen nach dem gescheiterten Vorgang", n)
	}
	if tx := env.bankTx(t, txID); tx.MatchStatus != domain.MatchStatusUnmatched {
		t.Errorf("der Umsatz steht auf %q, erwartet offen", tx.MatchStatus)
	}
	if all, err := rules.FindAll(ctx); err != nil || len(all) != 0 {
		t.Errorf("die Regel darf ohne Buchung nicht stehen bleiben: %v, %v", all, err)
	}

	ok := NewBankService(repository.NewBankRepository(env.db), env.journal, repository.NewAuditRepository(env.db))
	ok.SetRuleRepo(rules)
	if _, err := ok.BookDirect(ctx, txID, "4830", "Erstattung"); err != nil {
		t.Fatalf("der zweite Versuch muss gelingen: %v", err)
	}
	if _, err := ok.BookDirect(ctx, txID, "4830", "Erstattung"); err == nil {
		t.Error("ein zugeordneter Umsatz darf nicht ein zweites Mal gebucht werden")
	}
	if err := ok.Ignore(ctx, txID); err == nil {
		t.Error("ein gebuchter Umsatz darf nicht nachträglich als ignoriert gelten")
	}
}

func TestSettleIsAtomic(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant GmbH", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 10000, domain.TaxRateStandard)
	txID := env.bankCredit(t, "2026-03-20", -11900, "E2E-SETTLE")

	payments := NewPaymentService(env.journal, env.journalRepo,
		failingAllocations{repository.NewPaymentAllocationRepository(env.db)},
		env.contactRepo, repository.NewBankRepository(env.db))
	before := env.entryCount(t)

	_, err := payments.Settle(ctx, PaymentRequest{
		BankTxID:    &txID,
		Allocations: []AllocationRequest{{OpenItemEntryID: invoice.ID, SettledAmount: 11900}},
	})
	if err == nil {
		t.Fatal("die Zahlung muss an der Zuordnung scheitern")
	}
	if n := env.entryCount(t); n != before {
		t.Errorf("%d Buchungen nach dem gescheiterten Ausgleich, erwartet %d", n, before)
	}
	if tx := env.bankTx(t, txID); tx.MatchStatus != domain.MatchStatusUnmatched {
		t.Errorf("der Umsatz steht auf %q, erwartet offen", tx.MatchStatus)
	}
}

type failingAllocations struct {
	domain.PaymentAllocationRepository
}

func (failingAllocations) Create(context.Context, []domain.PaymentAllocation) error {
	return errors.New("Zuordnung nicht speicherbar")
}

func (e *testEnv) assetsWithAudit(t *testing.T, trigger string) *AssetService {
	t.Helper()
	return NewAssetService(repository.NewAssetRepository(e.db), e.journalRepo, e.journal, e.numberRepo,
		e.contactRepo, repository.NewSettingsRepository(e.db),
		failingAudit{AuditRepository: repository.NewAuditRepository(e.db), trigger: trigger})
}

func TestDisposalAndDepreciationAreAtomic(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	asset := env.machine(t, env.assets(t))
	before := env.entryCount(t)

	// Der Abgang bucht die nachgeholte AfA und den Abgang; erst sein
	// Protokolleintrag scheitert.
	_, err := env.assetsWithAudit(t, "Abgang").Dispose(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-06-30", Kind: domain.DisposalSale,
		Proceeds: 1_100_000, TaxTreatment: domain.TaxTreatmentDomestic,
		TaxRate: domain.TaxRateStandard, Settlement: SettlementPaid, PaymentAccount: "1800",
	})
	if err == nil {
		t.Fatal("der Abgang muss am Protokoll scheitern")
	}
	if n := env.entryCount(t); n != before {
		t.Errorf("%d Buchungen nach dem gescheiterten Abgang, erwartet %d", n, before)
	}
	fresh, err := repository.NewAssetRepository(env.db).FindByID(ctx, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Status == domain.AssetStatusDisposed || len(fresh.Movements) != len(asset.Movements) {
		t.Errorf("das Anlagegut wurde verändert: Status %q, %d Bewegungen", fresh.Status, len(fresh.Movements))
	}

	if _, err := env.assetsWithAudit(t, "Abschreibungslauf").BookDepreciation(ctx,
		BookDepreciationRequest{FiscalYear: 2026}); err == nil {
		t.Fatal("der AfA-Lauf muss am Protokoll scheitern")
	}
	if n := env.entryCount(t); n != before {
		t.Errorf("%d Buchungen nach dem gescheiterten AfA-Lauf, erwartet %d", n, before)
	}
	run, err := env.assets(t).Run(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Due) != 1 || run.Due[0].Booked != 0 {
		t.Errorf("nach dem gescheiterten Lauf darf keine AfA als gebucht gelten: %+v", run.Due)
	}
}

func TestProvisionConsumptionFromAReceiptIsAtomic(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	m.rate(t, "2026-12", 3, 15_000)

	provision, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability,
		Text: "Abschlussprüfung", Amount: 300_000, ExpectedOn: "2027-06-30",
		Reason: "Prüfungsauftrag vom 1.10.2026",
	})
	if err != nil {
		t.Fatalf("Rückstellung bilden: %v", err)
	}

	failing := NewProvisionService(m.provisionRepo, repository.NewDiscountRateRepository(env.db),
		env.journalRepo, env.journal, repository.NewSettingsRepository(env.db),
		failingAudit{AuditRepository: repository.NewAuditRepository(env.db), trigger: "Verbrauch"},
		m.closing)
	env.posting.SetProvisionConsumer(failing)

	vendor := env.vendor(t, "Prüfer GmbH", "DE", "")
	filed := env.fileIncoming(t, "pruefung.pdf")
	before := env.entryCount(t)
	req := ReceiptRequest{
		ContactID: vendor.ID, ReceiptID: filed.ID,
		BookingDate: "2027-03-01", DocumentDate: "2027-03-01",
		ServiceDateFrom: "2027-03-01", ServiceDateTo: "2027-03-01",
		Description: "Abschlussprüfung 2026", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions:  []ReceiptPosition{{PostingGroup: "fremdleistungen", Net: 250_000, TaxRate: domain.TaxRateStandard}},
		Settlement: SettlementOpen, ProvisionID: provision.ID,
	}
	_, err = env.posting.PostIncomingReceipt(ctx, req)
	if err == nil {
		t.Fatal("die Buchung muss am Protokoll des Verbrauchs scheitern")
	}
	if n := env.entryCount(t); n != before {
		t.Errorf("%d Buchungen nach dem gescheiterten Verbrauch, erwartet %d", n, before)
	}
	stored, err := m.provisionRepo.FindByID(ctx, provision.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Balance() != 300_000 {
		t.Errorf("die Rückstellung steht auf %s €, erwartet unverändert 3.000,00 €", stored.Balance())
	}
	receipt, err := env.receiptRepo.FindByID(ctx, filed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != domain.ReceiptStatusFiled {
		t.Errorf("der Beleg steht auf %q, erwartet offen", receipt.Status)
	}

	env.posting.SetProvisionConsumer(m.provisions)
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("mit funktionierendem Protokoll muss derselbe Beleg buchbar sein: %v", err)
	}
}

// Auch außerhalb des Journals wird ein gescheiterter Protokolleintrag nicht mehr
// verschluckt.
func TestServicesReportAFailedAuditEntry(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	audit := failingAudit{AuditRepository: repository.NewAuditRepository(env.db), trigger: "CONTACT"}
	contacts := NewContactService(env.contactRepo, env.journalRepo, env.numberRepo, audit)
	err := contacts.SaveContact(ctx, &domain.Contact{
		Type: domain.ContactTypeCustomer, Name: "Kunde GmbH", CountryCode: "DE",
		Street: "Kundenweg 2", PostalCode: "10115", City: "Berlin",
	})
	if err == nil || !strings.Contains(err.Error(), "Protokolleintrag") {
		t.Errorf("der gescheiterte Protokolleintrag muss gemeldet werden, Fehler: %v", err)
	}
}
