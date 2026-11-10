package core

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

// addBankTx legt einen offenen Bankumsatz an, wie ihn der Import ablegt.
func addBankTx(t *testing.T, tenant *Tenant, date string, amount domain.Cents, ref string) uint {
	t.Helper()
	var id uint
	if err := tenant.Write(systemCtx, func(ctx context.Context, s *Services) error {
		repo := repository.NewBankRepository(tenant.db)
		if _, err := repo.CreateBatch(ctx, []domain.BankTransaction{{
			FiscalYear: 2026, AccountIBAN: "DE02120300000000202051", BookingDate: date, ValueDate: date,
			Amount: amount, Currency: "EUR", CounterpartyName: "Hausbank", EndToEndID: ref,
			MatchStatus: domain.MatchStatusUnmatched, LedgerAccount: domain.AccountBank,
		}}); err != nil {
			return err
		}
		txs, err := repo.FindAll(ctx, 2026)
		for i := range txs {
			if txs[i].EndToEndID == ref {
				id = txs[i].ID
			}
		}
		return err
	}); err != nil {
		t.Fatalf("Bankumsatz: %v", err)
	}
	return id
}

// Ein Agent teilt einen Bankumsatz auf; eine Handbuchung mit Bezug auf
// denselben Umsatz kommt danach nicht mehr durch, auch nicht über die
// Bedienung in der App.
func TestAgentSplitsABankTransactionOnce(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	id := addBankTx(t, tenant, "2026-09-30", 7363, "zinsen-q3")

	var entry *domain.JournalEntry
	if err := tenant.Write(agentCtx(t), func(ctx context.Context, s *Services) error {
		var err error
		entry, err = s.Payments.BookBankSplit(ctx, service.BankSplitRequest{BankTxID: id, Lines: []service.BankSplitLine{
			{Account: domain.AccountSonstigeZinsertraege, Amount: 10000},
			{Account: "7630", Amount: -2500},
			{Account: "7633", Amount: -137},
		}})
		return err
	}); err != nil {
		t.Fatalf("Aufteilung durch den Agenten: %v", err)
	}
	if entry.Channel != "agent:claude-desktop" {
		t.Errorf("Erfassungsweg %q", entry.Channel)
	}

	err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		req, err := receiptAndEntry(ctx, s)
		if err != nil {
			return err
		}
		req.Entry.BankTxID = &id
		req.Entry.Lines = []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 7363},
			{Side: domain.SideCredit, Account: domain.AccountSonstigeZinsertraege, Amount: 7363},
		}
		_, err = s.Posting.PostManualEntry(ctx, req)
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "bereits mit Buchung") {
		t.Errorf("erwartet Abweisung der zweiten Buchung, erhalten %v", err)
	}
}

// Die Dienste der Gesellschafter und des Lohns sind verdrahtet; der Prüflauf
// nimmt ihre Befunde auf.
func TestShareholderAndPayrollServicesAreWired(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		if s.Shareholders == nil || s.Payroll == nil {
			t.Fatal("Gesellschafter- oder Lohndienst fehlt")
		}
		_, err := s.Checks.Preview(ctx, service.CheckRequest{CutoffDate: "2026-06-30"})
		return err
	}); err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
}
