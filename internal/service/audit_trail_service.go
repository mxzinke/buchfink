package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Prüfpfad eines Belegs (RECH-08).
//
// Eine Betriebsprüfung fragt, ob der Beleg zur Buchung führt, die Buchung zur
// Zahlung und die Zahlung zum Kontoauszug — nicht bloß, ob es den Beleg gibt.
// Jede dieser Verbindungen steht in Buchfink schon in den Daten — sie stand
// nur nirgends zusammen. Der Prüfpfad ist diese Kette, in einer Ansicht und in
// einer Datei (GoBD Rz. 36: progressive und retrograde Prüfbarkeit).

// AuditTrailStage benennt die Stufe der Kette.
type AuditTrailStage string

const (
	TrailStageReceipt AuditTrailStage = "receipt"
	TrailStageBooking AuditTrailStage = "booking"
	TrailStagePayment AuditTrailStage = "payment"
	TrailStageBank    AuditTrailStage = "bank"
)

// AuditTrailStep ist eine Stufe des Prüfpfads.
type AuditTrailStep struct {
	Stage     AuditTrailStage `json:"stage"`
	Title     string          `json:"title"`
	Date      string          `json:"date"`
	Reference string          `json:"reference"`
	Amount    domain.Cents    `json:"amount"`
	Detail    string          `json:"detail"`
}

// AuditTrail ist der Prüfpfad eines Belegs.
type AuditTrail struct {
	ReceiptID      uint         `json:"receiptId"`
	ReceiptNumber  string       `json:"receiptNumber"`
	Direction      string       `json:"direction"`
	DocumentDate   string       `json:"documentDate"`
	IssuerName     string       `json:"issuerName"`
	GrossAmount    domain.Cents `json:"grossAmount"`
	OrderReference string       `json:"orderReference,omitempty"`
	ServiceProof   string       `json:"serviceProof,omitempty"`
	ServiceProofAt string       `json:"serviceProofAt,omitempty"`

	Steps []AuditTrailStep `json:"steps"`
	// Note sagt, wo die Kette endet: ein Beleg ohne Buchung oder eine Rechnung
	// ohne Zahlung ist kein Fehler, aber eine Auskunft.
	Note string `json:"note"`
}

// EnsureLists ersetzt eine nicht belegte Schrittliste durch eine leere.
func (t *AuditTrail) EnsureLists() {
	if t.Steps == nil {
		t.Steps = make([]AuditTrailStep, 0)
	}
}

// AuditTrailService baut den Prüfpfad und gibt ihn aus.
type AuditTrailService struct {
	receiptRepo    domain.ReceiptRepository
	journalRepo    domain.JournalRepository
	allocationRepo domain.PaymentAllocationRepository
	bankRepo       domain.BankRepository
	auditRepo      domain.AuditRepository
}

// NewAuditTrailService wires the Prüfpfad.
func NewAuditTrailService(
	receiptRepo domain.ReceiptRepository,
	journalRepo domain.JournalRepository,
	allocationRepo domain.PaymentAllocationRepository,
	bankRepo domain.BankRepository,
	auditRepo domain.AuditRepository,
) *AuditTrailService {
	return &AuditTrailService{
		receiptRepo: receiptRepo, journalRepo: journalRepo,
		allocationRepo: allocationRepo, bankRepo: bankRepo, auditRepo: auditRepo,
	}
}

// Trail baut den Prüfpfad eines Belegs.
func (s *AuditTrailService) Trail(ctx context.Context, receiptID uint) (*AuditTrail, error) {
	receipt, err := s.receiptRepo.FindByID(ctx, receiptID)
	if err != nil {
		return nil, fmt.Errorf("Beleg %d wurde nicht gefunden: %w", receiptID, err)
	}

	trail := &AuditTrail{
		ReceiptID: receipt.ID, ReceiptNumber: receipt.ReceiptNumber,
		Direction: string(receipt.Direction), DocumentDate: receipt.DocumentDate,
		IssuerName: receipt.IssuerName, GrossAmount: receipt.GrossAmount,
		OrderReference: receipt.OrderReference,
		ServiceProof:   receipt.ServiceProof, ServiceProofAt: receipt.ServiceProofAt,
	}
	trail.EnsureLists()

	detail := receipt.Subject
	if receipt.OrderReference != "" {
		detail = strings.TrimSpace(detail + " (Bestellung " + receipt.OrderReference + ")")
	}
	trail.Steps = append(trail.Steps, AuditTrailStep{
		Stage: TrailStageReceipt, Title: "Beleg", Date: receipt.DocumentDate,
		Reference: receipt.ReceiptNumber, Amount: receipt.GrossAmount,
		Detail: strings.TrimSpace(strings.Join([]string{receipt.IssuerName, detail}, " – ")),
	})

	if receipt.JournalEntryID == nil {
		trail.Note = "Der Beleg ist noch nicht gebucht; der Pfad endet hier."
		return trail, nil
	}

	entry, err := s.journalRepo.FindByID(ctx, *receipt.JournalEntryID)
	if err != nil || entry == nil {
		trail.Note = "Die Buchung zu diesem Beleg ließ sich nicht lesen."
		return trail, nil
	}
	trail.Steps = append(trail.Steps, AuditTrailStep{
		Stage: TrailStageBooking, Title: "Buchung", Date: entry.BookingDate,
		Reference: entry.EntryNumber, Amount: entryAmount(entry),
		Detail: entry.Description,
	})

	if s.allocationRepo == nil {
		return trail, nil
	}
	allocations, err := s.allocationRepo.FindByOpenItem(ctx, entry.ID)
	if err != nil {
		trail.Note = "Die Zahlungszuordnungen ließen sich nicht lesen."
		return trail, nil
	}
	if len(allocations) == 0 {
		trail.Note = "Zu der Buchung ist noch keine Zahlung zugeordnet; der Pfad endet hier."
		return trail, nil
	}

	for _, allocation := range allocations {
		payment, err := s.journalRepo.FindByID(ctx, allocation.PaymentEntryID)
		date, reference, description := "", "", ""
		if err == nil && payment != nil {
			date, reference, description = payment.BookingDate, payment.EntryNumber, payment.Description
		}
		difference := ""
		if allocation.DifferenceKind != "" && allocation.DifferenceKind != domain.DifferenceNone {
			difference = fmt.Sprintf(" (%s %s €)", allocation.DifferenceKind, allocation.DifferenceAmount)
		}
		trail.Steps = append(trail.Steps, AuditTrailStep{
			Stage: TrailStagePayment, Title: "Zahlung", Date: date,
			Reference: reference, Amount: allocation.SettledAmount,
			Detail: strings.TrimSpace(description + difference),
		})

		if allocation.BankTxID == nil || s.bankRepo == nil {
			continue
		}
		tx, err := s.bankRepo.FindByID(ctx, *allocation.BankTxID)
		if err != nil || tx == nil {
			continue
		}
		trail.Steps = append(trail.Steps, AuditTrailStep{
			Stage: TrailStageBank, Title: "Bankumsatz", Date: tx.BookingDate,
			Reference: tx.EndToEndID, Amount: tx.Amount,
			Detail: strings.TrimSpace(tx.CounterpartyName + " – " + tx.RemittanceInfo),
		})
	}
	if trail.Note == "" {
		trail.Note = "Der Pfad ist vollständig: Beleg, Buchung und Zahlung sind verbunden."
	}
	return trail, nil
}

// entryAmount ist die Summe der Sollseite einer Buchung — der Betrag, in dem
// sie im Journal steht.
func entryAmount(entry *domain.JournalEntry) domain.Cents {
	var sum domain.Cents
	for i := range entry.Lines {
		if entry.Lines[i].Side == domain.SideDebit {
			sum += entry.Lines[i].Amount
		}
	}
	return sum
}
