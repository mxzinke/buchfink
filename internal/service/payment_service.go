package service

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
)

// AllocationRequest ordnet einen Teil der Zahlung einem offenen Posten zu.
type AllocationRequest struct {
	OpenItemEntryID uint `json:"openItemEntryId"`
	// SettledAmount mindert den offenen Posten einschließlich Skonto.
	SettledAmount    domain.Cents          `json:"settledAmount"`
	DifferenceKind   domain.DifferenceKind `json:"differenceKind"`
	DifferenceAmount domain.Cents          `json:"differenceAmount"`
}

// PaymentRequest gleicht mit einer Zahlung einen oder mehrere offene Posten aus.
type PaymentRequest struct {
	BankTxID       *uint               `json:"bankTxId,omitempty"`
	PaymentAccount string              `json:"paymentAccount"`
	PaymentDate    string              `json:"paymentDate"`
	ValueDate      string              `json:"valueDate,omitempty"`
	Description    string              `json:"description,omitempty"`
	Allocations    []AllocationRequest `json:"allocations"`
}

// AssetRegister ordnet Rechnungen einem Anlagegut zu und erfasst die
// Anschaffungskostenminderung durch Skonto.
type AssetRegister interface {
	AssetForEntry(ctx context.Context, entryID uint) (*domain.FixedAsset, error)
	RecordCostAdjustment(ctx context.Context, req CostAdjustmentRequest) (*domain.FixedAsset, error)
}

// PaymentService stellt den Ausgleich für Dienste, Bridge und MCP bereit.
type PaymentService struct {
	settlement *settlement
}

func NewPaymentService(
	journalSvc *JournalService,
	journalRepo domain.JournalRepository,
	allocationRepo domain.PaymentAllocationRepository,
	contactRepo domain.ContactRepository,
	bankRepo domain.BankRepository,
) *PaymentService {
	return &PaymentService{settlement: &settlement{
		journalSvc: journalSvc, bankRepo: bankRepo,
		openItemReader: &openItemReader{journalRepo: journalRepo, allocationRepo: allocationRepo, contactRepo: contactRepo},
	}}
}

// OpenItemSource liefert die Stichtagssicht ohne Abhängigkeit vom Buchungsweg.
func (s *PaymentService) OpenItemSource() OpenItemSource { return s.settlement.openItemReader }

func (s *PaymentService) SetAuditRepo(repo domain.AuditRepository) { s.settlement.auditRepo = repo }
func (s *PaymentService) SetAssetRegister(register AssetRegister)  { s.settlement.assets = register }
func (s *PaymentService) SetLoanSource(source LoanLineSource)      { s.settlement.loans = source }

// OpenItems liefert die aktuell offenen Forderungen und Verbindlichkeiten aller Geschäftsjahre.
func (s *PaymentService) OpenItems(ctx context.Context) ([]domain.OpenItem, error) {
	return s.settlement.OpenItems(ctx, "")
}

// OpenItemsAt berücksichtigt nur Buchungen und Ausgleiche bis zum Stichtag.
func (s *PaymentService) OpenItemsAt(ctx context.Context, cutoff string) ([]domain.OpenItem, error) {
	return s.settlement.OpenItems(ctx, cutoff)
}

func (s *PaymentService) Allocations(ctx context.Context, paymentEntryID uint) ([]domain.PaymentAllocationDetail, error) {
	return s.settlement.allocations(ctx, paymentEntryID)
}

func (s *PaymentService) AgeOpenItems(items []domain.OpenItem, cutoff string) *domain.OpenItemsAging {
	return ageOpenItems(items, cutoff)
}

func (s *PaymentService) OpenItemsAging(ctx context.Context, cutoff string) (*domain.OpenItemsAging, error) {
	return s.settlement.openItemsAging(ctx, cutoff)
}

func (s *PaymentService) Settle(ctx context.Context, req PaymentRequest) (*domain.JournalEntry, error) {
	return s.settlement.Settle(ctx, req)
}

func (s *PaymentService) BookBankSplit(ctx context.Context, req BankSplitRequest) (*domain.JournalEntry, error) {
	return s.settlement.Split(ctx, req)
}

func (s *PaymentService) WriteOffOpenItem(ctx context.Context, req WriteOffRequest) (*domain.JournalEntry, error) {
	return s.settlement.WriteOff(ctx, req)
}

func (s *PaymentService) CreditCardBalance(ctx context.Context, date string) (domain.Cents, error) {
	return s.settlement.creditCardBalance(ctx, date)
}

func (s *PaymentService) SettleCreditCard(ctx context.Context, req CreditCardSettlementRequest) (*domain.JournalEntry, error) {
	return s.settlement.settleCreditCard(ctx, req)
}

func (s *PaymentService) CreditCardStatementDate(ctx context.Context, debit domain.Cents, debitDate string) (string, error) {
	return s.settlement.creditCardStatementDate(ctx, debit, debitDate)
}
