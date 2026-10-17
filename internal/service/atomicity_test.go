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

// journalWithAudit baut einen Journaldienst auf derselben Datenbank, dessen
// Protokoll beim Auslöser scheitert.
func (e *testEnv) journalWithAudit(t *testing.T, trigger string) *JournalService {
	t.Helper()
	audit := failingAudit{AuditRepository: repository.NewAuditRepository(e.db), trigger: trigger}
	j := NewJournalService(e.journalRepo, repository.NewAccountRepository(e.db), e.contactRepo,
		audit, repository.NewSettingsRepository(e.db), repository.NewTxRunner(e.db), e.fiscalYear)
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
		env.contactRepo, repository.NewBankRepository(env.db), env.fiscalYear)
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
