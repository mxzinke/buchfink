package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Einzelzahlung und Bankaufteilung verwenden dieselbe Vorbereitung.
type advancePayment struct {
	advance    *domain.AdvanceItem
	invoice    *domain.Invoice
	entry      *domain.JournalEntry
	refund     bool
	gross, tax domain.Cents
}

func (s *InvoiceService) prepareAdvancePayment(ctx context.Context, req SettleAdvanceRequest, refund bool, reason string) (*advancePayment, error) {
	if s.groupRepo == nil {
		return nil, fmt.Errorf("der Rechnungsverbund ist nicht eingerichtet")
	}
	inv, err := s.invoiceRepo.FindByID(ctx, req.AdvanceID)
	if err != nil {
		return nil, err
	}
	a, err := s.groupRepo.FindAdvanceByInvoice(ctx, inv.ID)
	if err != nil {
		return nil, err
	}
	if a == nil || a.Cancelled {
		return nil, fmt.Errorf("die Rechnung hat keinen offenen Abschlag")
	}
	group, err := s.groupRepo.FindByID(ctx, a.GroupID)
	if err != nil {
		return nil, err
	}
	// Nach der Schlussrechnung würde eine weitere Anzahlung weder verrechnet
	// noch aufgelöst; die Zahlung gehört dann zur Schlussrechnung.
	if group.Closed || a.SettledInFinal {
		return nil, fmt.Errorf("der Abschlag ist mit der Schlussrechnung abgeschlossen")
	}
	received, receivedTax := a.Received()
	limit := a.GrossAmount - received
	if refund {
		limit = received
		if strings.TrimSpace(reason) == "" {
			return nil, fmt.Errorf("zu jeder Rückzahlung gehört eine Begründung")
		}
	}
	if limit <= 0 {
		return nil, fmt.Errorf("die Abschlagsrechnung ist bereits vereinnahmt oder es gibt nichts zurückzuzahlen")
	}
	if req.BankTxID != nil {
		if s.bankRepo == nil {
			return nil, fmt.Errorf("der Bankimport ist nicht eingerichtet")
		}
		tx, err := s.bankRepo.FindByID(ctx, *req.BankTxID)
		if err != nil {
			return nil, err
		}
		if tx.MatchStatus == domain.MatchStatusMatched || tx.MatchStatus == domain.MatchStatusIgnored {
			return nil, fmt.Errorf("Bankumsatz %d ist bereits zugeordnet oder ignoriert", tx.ID)
		}
		if tx.Amount == 0 || (tx.Amount < 0) != refund {
			return nil, fmt.Errorf("die Richtung des Bankumsatzes passt nicht zum Abschlag")
		}
		if tx.Currency != "" && tx.Currency != "EUR" {
			return nil, fmt.Errorf("Abschläge benötigen einen Bankumsatz in Euro")
		}
		if req.Amount != 0 && req.Amount != tx.Amount.Abs() {
			return nil, fmt.Errorf("der Ausgleich muss dem Bankumsatz entsprechen")
		}
		req.Amount, req.PaymentAccount, req.PaymentDate = tx.Amount.Abs(), tx.LedgerAccount, tx.BookingDate
	}
	if req.Amount == 0 {
		req.Amount = limit
	}
	if req.Amount <= 0 || req.Amount > limit {
		return nil, fmt.Errorf("der Zahlbetrag übersteigt den offenen Abschlag oder den vereinnahmten Betrag")
	}
	if req.PaymentDate == "" {
		return nil, fmt.Errorf("ohne Zahlungsdatum lässt sich die Anzahlung keinem Voranmeldungszeitraum zuordnen")
	}
	if refund && req.PaymentDate < a.SettledAt {
		return nil, fmt.Errorf("die Rückzahlung liegt vor der Vereinnahmung")
	}
	if req.PaymentAccount == "" {
		req.PaymentAccount = domain.AccountBank
	}
	if !isLiquidAccount(req.PaymentAccount) {
		return nil, fmt.Errorf("das Zahlungskonto ist kein Bank- oder Geldtransitkonto")
	}
	contact, err := s.contactRepo.FindByID(ctx, a.ContactID)
	if err != nil {
		return nil, err
	}
	part := *a
	part.GrossAmount = req.Amount
	// Kumulierte Aufteilung hält die Steuer bei Teilzahlungen auf dem Cent zusammen.
	after := received + req.Amount
	if refund {
		after = received - req.Amount
	}
	afterTax := domain.MulRound(a.TaxAmount, int64(after), int64(a.GrossAmount))
	part.TaxAmount = afterTax - receivedTax
	if refund {
		part.TaxAmount = receivedTax - afterTax
	}
	part.NetAmount = part.GrossAmount - part.TaxAmount
	var entry *domain.JournalEntry
	if refund {
		entry, err = s.postingSvc.advanceRefundEntry(ctx, a, contact, req.PaymentDate, req.PaymentAccount, reason)
	} else {
		entry, err = s.postingSvc.advanceSettlementEntry(ctx, a, contact, req.PaymentDate, req.PaymentAccount, req.BankTxID, inv.ReceiptID)
	}
	if err != nil {
		return nil, err
	}
	lines := entry.Lines[:0]
	for _, line := range entry.Lines {
		switch {
		case line.Account == req.PaymentAccount:
			line.Amount = part.GrossAmount
		case line.TaxKey != "":
			line.Amount, line.TaxBase = part.TaxAmount, part.NetAmount
		default:
			line.Amount = part.NetAmount
			if refund {
				line.Text += ": " + reason
			}
		}
		if line.Amount != 0 {
			lines = append(lines, line)
		}
	}
	entry.Lines = lines

	entry.BankTxID = req.BankTxID
	return &advancePayment{advance: a, invoice: inv, entry: entry, refund: refund, gross: part.GrossAmount, tax: part.TaxAmount}, nil
}

func (s *InvoiceService) finishAdvancePayment(ctx context.Context, p *advancePayment, entry *domain.JournalEntry) error {
	a := p.advance
	gross, tax := a.Received()
	payments, err := s.groupRepo.AdvancePayments(ctx, a.InvoiceID)
	if err != nil {
		return err
	}
	if len(payments) == 0 && gross > 0 && a.SettlementEntryID != nil {
		if err := s.groupRepo.SaveAdvancePayment(ctx, &domain.AdvancePayment{InvoiceID: a.InvoiceID, EntryID: *a.SettlementEntryID, Date: a.SettledAt, Gross: gross, Tax: tax}); err != nil {
			return err
		}
	}
	movement := domain.AdvancePayment{InvoiceID: a.InvoiceID, EntryID: entry.ID, Date: entry.BookingDate, Gross: p.gross, Tax: p.tax}
	if p.refund {
		movement.Gross = -movement.Gross
		movement.Tax = -movement.Tax
	}
	if err := s.groupRepo.SaveAdvancePayment(ctx, &movement); err != nil {
		return err
	}
	if p.refund {
		a.ReceivedGross, a.ReceivedTax = gross-p.gross, tax-p.tax
	} else {
		a.ReceivedGross, a.ReceivedTax = gross+p.gross, tax+p.tax
	}
	a.SettledAt, a.SettlementEntryID = entry.BookingDate, &entry.ID
	if a.ReceivedGross == 0 {
		a.SettledAt, a.SettlementEntryID = "", nil
	}
	if err := s.groupRepo.SaveAdvance(ctx, a); err != nil {
		return err
	}
	if !p.refund && len(payments) == 0 && gross == 0 {
		if err := s.sealInvoiceReceipt(ctx, p.invoice, entry.ID); err != nil {
			return err
		}
	}
	if entry.BankTxID != nil {
		if err := s.bankRepo.SetMatchStatus(ctx, *entry.BankTxID, domain.MatchStatusMatched); err != nil {
			return err
		}
	}
	return s.log(ctx, domain.AuditActionUpdate, p.invoice, p.entry.Description)
}

// AdvancePaymentEffect schützt die Verrechnung und nachfolgende Teilzahlungen
// vor einem isolierten Storno ihrer vorausgehenden Vereinnahmung.
func (s *InvoiceService) AdvancePaymentEffect() domain.EntryEffect {
	return domain.EntryEffect{Name: "advance_payment", AfterReverse: func(ctx context.Context, original, reversal *domain.JournalEntry) error {
		if s.groupRepo == nil {
			return nil
		}
		groups, err := s.groupRepo.FindAll(ctx, 0)
		if err != nil {
			return err
		}
		for _, g := range groups {
			for _, a := range g.Advances {
				payments, err := s.groupRepo.AdvancePayments(ctx, a.InvoiceID)
				if err != nil {
					return err
				}
				var target uint
				for _, p := range payments {
					if p.EntryID == original.ID {
						target = p.ID
					}
				}
				legacy := len(payments) == 0 && a.SettlementEntryID != nil && *a.SettlementEntryID == original.ID
				if target == 0 && !legacy {
					continue
				}
				if g.Closed || a.SettledInFinal || a.Cancelled {
					return fmt.Errorf("die Zahlung zu %s gehört zu einer verrechneten oder stornierten Abschlagsrechnung; zuerst die Rechnung berichtigen", a.InvoiceNumber)
				}
				for _, p := range payments {
					if p.ID > target && p.ReversedAt == "" {
						return fmt.Errorf("zu %s gibt es spätere Zahlungen oder Rückzahlungen; zuerst diese Zuordnungen korrigieren", a.InvoiceNumber)
					}
				}
			}
		}
		return nil
	}}
}
