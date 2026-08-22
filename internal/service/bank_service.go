// SPDX-License-Identifier: EUPL-1.2

package service

import (
	"context"
	"fmt"
	"io"

	"github.com/buchfink/buchfink/internal/bank"
	"github.com/buchfink/buchfink/internal/domain"
)

// BankService imports bank statements and books transactions that carry no
// document of their own (fees, interest, transfers between own accounts).
//
// Payments that settle an open item do not run through here — they go through
// the payment matching, which needs the open item and the difference handling.
type BankService struct {
	bankRepo   domain.BankRepository
	journalSvc *JournalService
	auditRepo  domain.AuditRepository
}

// NewBankService creates the bank import service.
func NewBankService(
	bankRepo domain.BankRepository,
	journalSvc *JournalService,
	auditRepo domain.AuditRepository,
) *BankService {
	return &BankService{bankRepo: bankRepo, journalSvc: journalSvc, auditRepo: auditRepo}
}

// GetTransactions returns imported bank transactions of a fiscal year.
func (s *BankService) GetTransactions(ctx context.Context, fiscalYear int) ([]domain.BankTransaction, error) {
	return s.bankRepo.FindAll(ctx, fiscalYear)
}

// ImportCAMT053 parses an ISO 20022 CAMT.053 statement and stores its lines.
func (s *BankService) ImportCAMT053(ctx context.Context, r io.Reader, ledgerAccount string) (int, error) {
	if ledgerAccount == "" {
		ledgerAccount = domain.AccountBank
	}

	parsed, err := bank.ParseCAMT053(r)
	if err != nil {
		return 0, err
	}

	for i := range parsed {
		parsed[i].FiscalYear = domain.GetFiscalYearForDate(parsed[i].BookingDate, 1)
		parsed[i].LedgerAccount = ledgerAccount
	}

	inserted, err := s.bankRepo.CreateBatch(ctx, parsed)
	if err != nil {
		return 0, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionImport, "BANK_TX", fmt.Sprintf("%d", inserted),
			fmt.Sprintf("%d Umsätze aus CAMT.053 für Konto %s importiert", inserted, ledgerAccount))
	}
	return inserted, nil
}

// BookDirect books a bank transaction that has no document behind it against a
// single counter account.
//
// The bank side is taken from the statement, so the direction cannot be entered
// wrongly: money in is a debit on the liquid account, money out a credit.
func (s *BankService) BookDirect(ctx context.Context, bankTxID uint, counterAccount, description string) (*domain.JournalEntry, error) {
	tx, err := s.bankRepo.FindByID(ctx, bankTxID)
	if err != nil {
		return nil, fmt.Errorf("Bankumsatz %d wurde nicht gefunden: %w", bankTxID, err)
	}
	if tx.MatchStatus == domain.MatchStatusMatched {
		return nil, fmt.Errorf("Bankumsatz %d ist bereits zugeordnet", bankTxID)
	}
	if counterAccount == "" {
		return nil, fmt.Errorf("Gegenkonto fehlt")
	}
	if tx.Amount == 0 {
		return nil, fmt.Errorf("Bankumsatz %d hat den Betrag null", bankTxID)
	}

	if description == "" {
		description = fmt.Sprintf("%s – %s", tx.CounterpartyName, tx.RemittanceInfo)
	}

	amount := tx.Amount.Abs()
	bankSide, counterSide := domain.SideDebit, domain.SideCredit
	if tx.Amount < 0 {
		bankSide, counterSide = domain.SideCredit, domain.SideDebit
	}

	entry := &domain.JournalEntry{
		FiscalYear:      tx.FiscalYear,
		BookingDate:     tx.BookingDate,
		DocumentDate:    tx.BookingDate,
		ServiceDateFrom: tx.BookingDate,
		ServiceDateTo:   tx.BookingDate,
		ValueDate:       tx.ValueDate,
		Description:     description,
		Source:          domain.EntrySourcePayment,
		BankTxID:        &tx.ID,
		Currency:        tx.Currency,
		Lines: []domain.JournalLine{
			{Position: 1, Side: bankSide, Account: tx.LedgerAccount, Amount: amount},
			{Position: 2, Side: counterSide, Account: counterAccount, Amount: amount},
		},
	}

	created, err := s.journalSvc.Post(ctx, entry)
	if err != nil {
		return nil, err
	}

	if err := s.bankRepo.SetMatchStatus(ctx, bankTxID, domain.MatchStatusMatched); err != nil {
		return nil, fmt.Errorf("Bankumsatz konnte nicht als zugeordnet markiert werden: %w", err)
	}

	return created, nil
}

// Ignore marks a bank transaction as deliberately not booked.
func (s *BankService) Ignore(ctx context.Context, bankTxID uint) error {
	return s.bankRepo.SetMatchStatus(ctx, bankTxID, domain.MatchStatusIgnored)
}
