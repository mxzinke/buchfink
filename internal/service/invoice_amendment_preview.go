package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// PreviewInvoiceAmendment zeigt die Buchungswirkung von Storno und Ersatzrechnung.
func (s *InvoiceService) PreviewInvoiceAmendment(
	ctx context.Context, invoiceID uint, replacement *domain.Invoice,
) (*PostingPreview, error) {
	original, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if err := ensureNotCancellation(original); err != nil {
		return nil, err
	}
	if replacement != nil && original.Kind != domain.InvoiceKindAdvance {
		draft := *replacement
		draft.Items = append([]domain.InvoiceItem(nil), replacement.Items...)
		_, lines, err := s.prepareAmendment(ctx, original, &draft)
		if err != nil {
			return nil, err
		}
		result := s.postingSvc.preview(ctx, lines)
		result.BookingDate = draft.Date
		result.Net = draft.NetAmount - original.NetAmount
		result.Tax = draft.TaxAmount - original.TaxAmount
		result.Gross = draft.GrossAmount - original.GrossAmount
		return result, nil
	}
	if !original.Status.IsIssued() || original.Status == domain.InvoiceStatusCancelled {
		return nil, fmt.Errorf("Rechnung %s ist nicht ausgestellt", original.InvoiceNumber)
	}
	if err := s.ensureAdvanceCancellable(ctx, original); err != nil {
		return nil, err
	}
	if err := s.ensureFinalCancellable(ctx, original); err != nil {
		return nil, err
	}
	state, err := s.postingSvc.journalSvc.periods.State(ctx, s.today())
	if err != nil {
		return nil, err
	}
	if !state.Open {
		return nil, fmt.Errorf("der Stornotag %s ist festgeschrieben", s.today())
	}
	if err := s.checkInvoiceWriteoffs(ctx, original); err != nil {
		return nil, err
	}
	if replacement != nil {
		group, err := s.replacementGroupOf(ctx, original)
		if err != nil {
			return nil, err
		}
		draft := *replacement
		draft.ID, draft.InvoiceNumber = 0, ""
		draft.FiscalYear = 0
		draft.JournalEntryID, draft.ReceiptID = nil, nil
		draft.Items = append([]domain.InvoiceItem(nil), replacement.Items...)
		dropForeignKindFields(&draft)
		draft.Kind, draft.GroupID = domain.InvoiceKindAdvance, &group.ID
		draft.ContactID, draft.TaxTreatment = original.ContactID, domain.TaxTreatmentDomestic
		draft.CorrectsInvoiceID = &original.ID
		draft.CorrectsInvoiceNumber, draft.CorrectsInvoiceDate = original.InvoiceNumber, original.Date
		if draft.EInvoiceProfile == "" {
			draft.EInvoiceProfile = original.EInvoiceProfile
		}
		for _, item := range draft.Items {
			if item.TaxRate != group.TaxRate {
				return nil, fmt.Errorf("der Steuersatz muss dem Rechnungsverbund entsprechen")
			}
		}
		if _, err := s.prepareForIssue(ctx, &draft, domain.InvoiceKindAdvance); err != nil {
			return nil, err
		}
		state, err := s.postingSvc.journalSvc.periods.State(ctx, draft.Date)
		if err != nil {
			return nil, err
		}
		if !state.Open {
			return nil, fmt.Errorf("der Berichtigungstag %s ist festgeschrieben", draft.Date)
		}
		for i := range group.Advances {
			if group.Advances[i].InvoiceID == original.ID {
				group.Advances[i].Cancelled = true
			}
		}
		if err := group.EnsureAdvanceFits(draft.NetAmount); err != nil {
			return nil, err
		}
		result := s.postingSvc.preview(ctx, []domain.JournalLine{})
		result.BookingDate = draft.Date
		return result, nil
	}
	entries, err := s.invoiceBookingChain(ctx, original)
	if err != nil {
		return nil, err
	}
	var lines []domain.JournalLine
	for _, entry := range entries {
		for _, line := range entry.Lines {
			line.Amount = -line.Amount
			line.TaxBase = -line.TaxBase
			lines = append(lines, line)
		}
	}
	result := s.postingSvc.preview(ctx, lines)
	result.BookingDate = s.today()
	result.Net, result.Tax, result.Gross = -original.NetAmount, -original.TaxAmount, -original.GrossAmount

	return result, nil
}
