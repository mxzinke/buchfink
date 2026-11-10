package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// AllocationRequest ordnet einen Teil der Zahlung einem offenen Posten zu.
type AllocationRequest struct {
	OpenItemEntryID  uint `json:"openItemEntryId"`
	AdvanceInvoiceID uint `json:"advanceInvoiceId,omitempty"`
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
func (s *PaymentService) SetReceipts(receipts evidenceReceipts)    { s.settlement.receipts = receipts }

func (s *PaymentService) SetInvoiceService(invoices *InvoiceService) {
	s.settlement.invoices = invoices
}

// OpenItems liefert die aktuell offenen Forderungen und Verbindlichkeiten aller Geschäftsjahre.
func (s *PaymentService) OpenItems(ctx context.Context) ([]domain.OpenItem, error) {
	items, err := s.settlement.OpenItems(ctx, "")
	if err != nil || s.settlement.invoices == nil {
		return items, err
	}
	advances, err := s.settlement.invoices.OpenAdvanceItems(ctx)
	return append(items, advances...), err
}

// OpenItemsAt berücksichtigt nur Buchungen und Ausgleiche bis zum Stichtag.
func (s *PaymentService) OpenItemsAt(ctx context.Context, cutoff string) ([]domain.OpenItem, error) {
	items, err := s.settlement.OpenItems(ctx, cutoff)
	if err != nil || s.settlement.invoices == nil {
		return items, err
	}
	advances, err := s.settlement.invoices.OpenAdvanceItems(ctx, cutoff)
	for _, item := range advances {
		if item.DocumentDate <= cutoff {
			items = append(items, item)
		}
	}
	return items, err
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
	for _, a := range req.Allocations {
		if a.AdvanceInvoiceID == 0 {
			continue
		}
		if s.settlement.invoices == nil {
			return nil, fmt.Errorf("Abschläge sind nicht eingerichtet")
		}
		if len(req.Allocations) != 1 || a.OpenItemEntryID != 0 || a.DifferenceAmount != 0 || (a.DifferenceKind != "" && a.DifferenceKind != domain.DifferenceNone) {
			return nil, fmt.Errorf("der Abschlag braucht eine einzelne Zuordnung ohne Differenz")
		}
		return atomically(ctx, s.settlement.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
			if a.SettledAmount <= 0 {
				return nil, fmt.Errorf("der Ausgleich muss positiv sein")
			}

			advance, err := s.settlement.invoices.SettleAdvance(ctx, SettleAdvanceRequest{AdvanceID: a.AdvanceInvoiceID, Amount: a.SettledAmount, BankTxID: req.BankTxID, PaymentDate: req.PaymentDate, PaymentAccount: req.PaymentAccount})
			if err != nil {
				return nil, err
			}
			return s.settlement.journalRepo.FindByID(ctx, *advance.SettlementEntryID)
		})
	}
	return s.settlement.Settle(ctx, req)
}

func (s *PaymentService) BookBankSplit(ctx context.Context, req BankSplitRequest) (*domain.JournalEntry, error) {
	return s.settlement.Split(ctx, req)
}

func (s *PaymentService) WriteOffOpenItem(ctx context.Context, req WriteOffRequest) (*domain.JournalEntry, error) {
	return s.settlement.WriteOff(ctx, req)
}

func (s *PaymentService) SettleCreditCard(ctx context.Context, req CreditCardSettlementRequest) (*domain.JournalEntry, error) {
	return s.settlement.settleCreditCard(ctx, req)
}

func (s *PaymentService) CreditCardStatementDate(ctx context.Context, debit domain.Cents, debitDate string) (string, error) {
	return s.settlement.creditCardStatementDate(ctx, debit, debitDate)
}

func (s *PaymentService) SetClock(now func() time.Time) { s.settlement.openItemReader.SetClock(now) }

func (s *PaymentService) PreviewWriteOff(ctx context.Context, req WriteOffRequest) (*domain.JournalEntry, error) {
	return s.settlement.previewWriteOff(ctx, req)
}

// RefundItems liefert mögliche Rückzahlungen für die Bankvorschläge.
func (s *PaymentService) RefundItems(ctx context.Context) ([]domain.OpenItem, error) {
	out := []domain.OpenItem{}
	if s.settlement.invoices == nil {
		return out, nil
	}
	groups, err := s.settlement.invoices.GetInvoiceGroups(ctx, 0)
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		if group.Closed {
			continue
		}
		for _, a := range group.Advances {
			gross, _ := a.Received()
			if gross <= 0 || a.Cancelled || a.SettledInFinal {
				continue
			}
			contact, err := s.settlement.contactRepo.FindByID(ctx, a.ContactID)
			if err != nil {
				return nil, err
			}
			out = append(out, domain.OpenItem{Source: domain.OpenItemSourceAdvanceRefund, AdvanceInvoiceID: a.InvoiceID, ContactID: a.ContactID, ContactName: contact.Name, ContactType: contact.Type, DocumentNumber: a.InvoiceNumber, DocumentDate: a.InvoiceDate, GrossAmount: gross, OpenAmount: gross, Side: domain.SideCredit, TaxRate: a.TaxRate})
		}
	}
	return out, nil
}
