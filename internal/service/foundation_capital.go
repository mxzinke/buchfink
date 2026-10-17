package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Der Stand der Einlagen auf das Stammkapital und ihre Einforderung.
//
// Gezeichnet wird bei der Beurkundung (PreviewPostings), eingezahlt über den
// Bankumsatz auf 1298, eingefordert durch Beschluss der Gesellschafter
// (§ 46 Nr. 2 GmbHG). Alles Weitere steht im Journal; der Stand wird daraus
// gerechnet.

// CapitalStatus ist der Stand der Einlagen an einem Tag.
type CapitalStatus struct {
	AsOf string `json:"asOf"`
	// Subscribed ist das gezeichnete Kapital (2900), Uncalled der nicht
	// eingeforderte Teil (2910), Called der Rest.
	Subscribed domain.Cents `json:"subscribed"`
	Uncalled   domain.Cents `json:"uncalled"`
	Called     domain.Cents `json:"called"`
	// CalledUnpaid ist eingefordert und noch nicht eingezahlt (1298), PaidIn
	// eingezahlt.
	CalledUnpaid domain.Cents `json:"calledUnpaid"`
	PaidIn       domain.Cents `json:"paidIn"`
	Findings     []string     `json:"findings"`
}

// SetReceiptService gibt der Gründung die Belegablage für den Beschluss, der die
// Einforderung belegt.
func (s *FoundationService) SetReceiptService(r closingReceiptFiler) { s.receipts = r }

// CapitalStatus rechnet den Stand der Einlagen an einem Tag aus den Buchungen
// seines Geschäftsjahres.
func (s *FoundationService) CapitalStatus(ctx context.Context, asOf string) (*CapitalStatus, error) {
	if err := domain.CheckDate("Stichtag", asOf); err != nil {
		return nil, err
	}
	year := s.journalSvc.FiscalYearOf(ctx, asOf)
	turnovers, err := s.journalRepo.AccountTurnoversUntil(ctx, year, asOf)
	if err != nil {
		return nil, err
	}
	balance := func(account string) domain.Cents {
		t := turnovers[account]
		return t.Debit - t.Credit
	}
	out := &CapitalStatus{
		AsOf:         asOf,
		Subscribed:   -balance(domain.AccountGezeichnetesKapital),
		Uncalled:     balance(domain.AccountAusstehendeEinlagenOffen),
		CalledUnpaid: balance(domain.AccountAusstehendeEinlagenGeford),
		Findings:     make([]string, 0),
	}
	out.Called = out.Subscribed - out.Uncalled
	out.PaidIn = out.Called - out.CalledUnpaid
	if out.Subscribed == 0 {
		out.Findings = append(out.Findings, fmt.Sprintf(
			"Im Geschäftsjahr %d steht bis zum %s kein gezeichnetes Kapital: die Zeichnung oder der "+
				"Saldenvortrag fehlt.", year, germanDate(asOf)))
	}
	if out.CalledUnpaid < 0 {
		out.Findings = append(out.Findings, fmt.Sprintf(
			"Auf %s steht ein Habensaldo von %s €: eingezahlt wurde mehr als eingefordert. Haben die "+
				"Gesellschafter den Rest eingefordert (§ 46 Nr. 2 GmbHG), ist die Einforderung zu buchen.",
			domain.AccountAusstehendeEinlagenGeford, -out.CalledUnpaid))
	}
	return out, nil
}

// CapitalCallRequest ist die Einforderung ausstehender Einlagen nach
// Beschluss der Gesellschafter.
type CapitalCallRequest struct {
	Date   string       `json:"date"`
	Amount domain.Cents `json:"amount"`
	// ResolutionReceiptID ist der abgelegte Gesellschafterbeschluss; er belegt
	// die Buchung.
	ResolutionReceiptID uint   `json:"resolutionReceiptId"`
	Reason              string `json:"reason"`
}

// CallInCapital bucht die Einforderung: der eingeforderte Betrag wird zur
// Forderung (1298) und kommt aus der offenen Absetzung (2910) heraus (§ 272
// Abs. 1 Satz 2 HGB). Die Einzahlung folgt über den Bankumsatz.
func (s *FoundationService) CallInCapital(ctx context.Context, req CapitalCallRequest) (*domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		return s.callInCapital(ctx, req)
	})
}

func (s *FoundationService) callInCapital(ctx context.Context, req CapitalCallRequest) (*domain.JournalEntry, error) {
	if err := domain.CheckDate("Tag des Beschlusses", req.Date); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("der eingeforderte Betrag muss größer als null sein")
	}
	if req.ResolutionReceiptID == 0 {
		return nil, fmt.Errorf(
			"die Einforderung beruht auf einem Beschluss der Gesellschafter (§ 46 Nr. 2 GmbHG); " +
				"er ist als Beleg abzulegen und anzugeben")
	}
	status, err := s.CapitalStatus(ctx, req.Date)
	if err != nil {
		return nil, err
	}
	if req.Amount > status.Uncalled {
		return nil, fmt.Errorf(
			"am %s sind nur %s € nicht eingefordert; einfordern lassen sich nicht %s €",
			germanDate(req.Date), status.Uncalled, req.Amount)
	}
	if s.receipts != nil {
		if _, err := s.receipts.Get(ctx, req.ResolutionReceiptID); err != nil {
			return nil, fmt.Errorf("der Beschluss %d ist nicht abgelegt: %w", req.ResolutionReceiptID, err)
		}
	}
	receiptID := req.ResolutionReceiptID
	description := "Einforderung ausstehender Einlagen laut Gesellschafterbeschluss"
	if req.Reason != "" {
		description += ": " + req.Reason
	}
	created, err := s.journalSvc.Post(ctx, &domain.JournalEntry{
		BookingDate: req.Date, DocumentDate: req.Date, ServiceDateFrom: req.Date, ServiceDateTo: req.Date,
		Description: description, Source: domain.EntrySourceManual,
		TaxTreatment: domain.TaxTreatmentNotTaxable, ReceiptID: &receiptID,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountAusstehendeEinlagenGeford, Amount: req.Amount},
			{Side: domain.SideCredit, Account: domain.AccountAusstehendeEinlagenOffen, Amount: req.Amount},
		},
	})
	if err != nil {
		return nil, err
	}
	if s.receipts != nil {
		if err := s.receipts.Seal(ctx, receiptID, created.ID); err != nil {
			return nil, fmt.Errorf("der Beschluss ließ sich nicht an die Buchung binden: %w", err)
		}
	}
	if err := s.audit(ctx, domain.AuditActionCreate, 0, fmt.Sprintf(
		"Einforderung ausstehender Einlagen über %s € zum %s gebucht (%s)",
		req.Amount, req.Date, created.EntryNumber)); err != nil {
		return nil, err
	}
	return created, nil
}
