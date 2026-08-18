package service

import (
	"context"
	"fmt"
	"io"
	"math"

	"github.com/buchfink/buchfink/internal/bank"
	"github.com/buchfink/buchfink/internal/domain"
)

// BankService manages bank statement parsing and automated journal reconciliation.
type BankService struct {
	bankRepo      domain.BankRepository
	accountingSvc *AccountingService
	auditRepo     domain.AuditRepository
}

func NewBankService(
	bankRepo domain.BankRepository,
	accountingSvc *AccountingService,
	auditRepo domain.AuditRepository,
) *BankService {
	return &BankService{
		bankRepo:      bankRepo,
		accountingSvc: accountingSvc,
		auditRepo:     auditRepo,
	}
}

// GetTransactions returns all imported bank transactions.
func (s *BankService) GetTransactions(ctx context.Context) ([]domain.BankTransaction, error) {
	return s.bankRepo.FindAll(ctx)
}

// ImportCAMT053 parses ISO 20022 CAMT.053 XML data and stores new transactions.
func (s *BankService) ImportCAMT053(ctx context.Context, r io.Reader) (int, error) {
	parsedModels, err := bank.ParseCAMT053(r)
	if err != nil {
		return 0, fmt.Errorf("CAMT.053 parse error: %w", err)
	}

	var domainTxs []domain.BankTransaction
	for _, p := range parsedModels {
		domainTxs = append(domainTxs, domain.BankTransaction{
			AccountIBAN:      p.AccountIBAN,
			BookingDate:      p.BookingDate,
			ValueDate:        p.ValueDate,
			Amount:           p.Amount,
			Currency:         p.Currency,
			CounterpartyName: p.CounterpartyName,
			CounterpartyIBAN: p.CounterpartyIBAN,
			RemittanceInfo:   p.RemittanceInfo,
			EndToEndID:       p.EndToEndID,
			MatchStatus:      domain.MatchStatusUnmatched,
			SuggestedAccount: p.SuggestedAccount,
			SuggestedContact: p.SuggestedContact,
		})
	}

	inserted, err := s.bankRepo.CreateBatch(ctx, domainTxs)
	if err != nil {
		return 0, err
	}

	_ = s.auditRepo.Log(
		ctx,
		domain.AuditActionImport,
		"BANK_TX",
		fmt.Sprintf("%d", inserted),
		fmt.Sprintf("%d Transaktionen aus CAMT.053 Bankauszug importiert", inserted),
	)

	return inserted, nil
}

// MatchAndBook confirms a bank transaction and triggers an automated double-entry journal booking.
func (s *BankService) MatchAndBook(
	ctx context.Context,
	bankTxID uint,
	debitAcc, creditAcc, receiptNr, desc string,
) (*domain.BookingEntry, error) {
	tx, err := s.bankRepo.FindByID(ctx, bankTxID)
	if err != nil {
		return nil, fmt.Errorf("bank transaction ID %d not found: %w", bankTxID, err)
	}

	if tx.MatchStatus == domain.MatchStatusMatched {
		return nil, fmt.Errorf("bank transaction ID %d is already matched", bankTxID)
	}

	amt := math.Abs(tx.Amount)
	if desc == "" {
		desc = fmt.Sprintf("%s (%s)", tx.RemittanceInfo, tx.CounterpartyName)
	}

	booking := &domain.BookingEntry{
		Date:          tx.BookingDate,
		ValueDate:     tx.ValueDate,
		Description:   desc,
		DebitAccount:  debitAcc,
		CreditAccount: creditAcc,
		Amount:        amt,
		Currency:      tx.Currency,
		ExchangeRate:  1.0,
		ReceiptNumber: receiptNr,
		BankTxID:      &bankTxID,
	}

	created, err := s.accountingSvc.CreateBooking(ctx, booking)
	if err != nil {
		return nil, fmt.Errorf("failed to create automated booking: %w", err)
	}

	if err := s.bankRepo.MarkMatched(ctx, bankTxID, created.ID); err != nil {
		return nil, fmt.Errorf("failed to mark bank transaction as matched: %w", err)
	}

	return created, nil
}
