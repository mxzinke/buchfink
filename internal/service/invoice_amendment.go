package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// invoiceBookingChain umfasst die Rechnung und ihre Differenzbuchungen.
func (s *InvoiceService) invoiceBookingChain(ctx context.Context, inv *domain.Invoice) ([]domain.JournalEntry, error) {
	var entries []domain.JournalEntry
	seen := map[uint]bool{}
	for inv != nil {
		if seen[inv.ID] {
			return nil, fmt.Errorf("zyklischer Rechnungsbezug")
		}
		seen[inv.ID] = true
		if inv.JournalEntryID != nil {
			entry, err := s.postingSvc.journalSvc.journalRepo.FindByID(ctx, *inv.JournalEntryID)
			if err != nil {
				return nil, err
			}
			entries = append(entries, *entry)
		}
		if inv.Kind != domain.InvoiceKindCorrection || inv.CorrectsInvoiceID == nil {
			break
		}
		var err error
		inv, err = s.invoiceRepo.FindByID(ctx, *inv.CorrectsInvoiceID)
		if err != nil {
			return nil, err
		}
		// Ein vorhandenes Stornodokument kennzeichnet den Weg über Storno und Ersatz.
		if inv.CancelledByInvoiceID != nil {
			next, err := s.invoiceRepo.FindByID(ctx, *inv.CancelledByInvoiceID)
			if err != nil {
				return nil, err
			}
			if next.Kind == domain.InvoiceKindCancellation {
				break
			}
		}
	}
	return entries, nil
}

func (s *InvoiceService) checkInvoiceWriteoffs(ctx context.Context, inv *domain.Invoice) error {
	entries, err := s.invoiceBookingChain(ctx, inv)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		reversal, err := s.postingSvc.journalSvc.journalRepo.FindReversalOf(ctx, entry.ID)
		if err != nil {
			return err
		}
		if reversal != nil {
			return fmt.Errorf("die Rechnungsbuchung %s wurde bereits storniert", entry.EntryNumber)
		}
		if s.allocationRepo == nil {
			continue
		}
		allocations, err := s.allocationRepo.FindByOpenItem(ctx, entry.ID)
		if err != nil {
			return err
		}
		for _, allocation := range allocations {
			if allocation.DifferenceKind != domain.DifferenceWriteoff {
				continue
			}
			reversed, err := s.postingSvc.journalSvc.journalRepo.FindReversalOf(ctx, allocation.PaymentEntryID)
			if err != nil {
				return err
			}
			if reversed == nil {
				return fmt.Errorf("die Forderung wurde ausgebucht; zuerst die Ausbuchung stornieren")
			}
		}
	}
	return nil
}

func (s *InvoiceService) prepareAmendment(
	ctx context.Context, original, replacement *domain.Invoice,
) (*domain.Contact, []domain.JournalLine, error) {
	if err := ensureNotCancellation(original); err != nil {
		return nil, nil, err
	}
	if !original.Status.IsIssued() || original.Status == domain.InvoiceStatusCancelled {
		return nil, nil, fmt.Errorf("Rechnung %s ist nicht mehr gültig", original.InvoiceNumber)
	}
	if err := s.checkInvoiceWriteoffs(ctx, original); err != nil {
		return nil, nil, err
	}
	replacement.ID = 0
	replacement.InvoiceNumber = ""
	replacement.JournalEntryID, replacement.ReceiptID = nil, nil
	replacement.Status = domain.InvoiceStatusDraft
	replacement.FiscalYear = 0
	replacement.ZUGFeRDXML = ""
	replacement.SentAt, replacement.SentVia, replacement.SentNote = "", "", ""
	for i := range replacement.Items {
		replacement.Items[i].ID, replacement.Items[i].InvoiceID = 0, 0
	}
	dropForeignKindFields(replacement)
	replacement.Kind = domain.InvoiceKindCorrection
	if original.GroupID != nil {
		if err := s.ensureFinalCancellable(ctx, original); err != nil {
			return nil, nil, err
		}
		replacement.GroupID = original.GroupID
		replacement.PrepaidAmount = original.PrepaidAmount
		replacement.PrecedingRefs = append([]domain.InvoiceReference(nil), original.PrecedingRefs...)
		for i := range replacement.PrecedingRefs {
			replacement.PrecedingRefs[i].ID, replacement.PrecedingRefs[i].InvoiceID = 0, 0
		}
	}
	replacement.CorrectsInvoiceID = &original.ID
	replacement.CorrectsInvoiceNumber, replacement.CorrectsInvoiceDate = original.InvoiceNumber, original.Date
	if replacement.ContactID == 0 {
		replacement.ContactID = original.ContactID
	}
	if replacement.ContactID != original.ContactID {
		return nil, nil, fmt.Errorf("bei einem anderen Kunden die Rechnung stornieren und neu ausstellen")
	}
	if replacement.EInvoiceProfile == "" {
		replacement.EInvoiceProfile = original.EInvoiceProfile
	}
	contact, err := s.prepareForIssue(ctx, replacement, domain.InvoiceKindCorrection)
	if err != nil {
		return nil, nil, err
	}
	if replacement.PrepaidAmount > replacement.GrossAmount {
		return nil, nil, fmt.Errorf("die berichtigte Schlussrechnung unterschreitet die verrechneten Anzahlungen")
	}
	if original.Currency != replacement.Currency || replacement.Currency != "EUR" {
		return nil, nil, fmt.Errorf("Rechnungsberichtigungen unterstützen nur Euro; eine Fremdwährungsrechnung stornieren und neu ausstellen")
	}
	state, err := s.postingSvc.journalSvc.periods.State(ctx, replacement.Date)
	if err != nil {
		return nil, nil, err
	}
	if !state.Open {
		return nil, nil, fmt.Errorf("der Berichtigungstag %s ist festgeschrieben", replacement.Date)
	}
	before, _, err := s.postingSvc.buildOutgoingLines(ctx, original, contact)
	if err != nil {
		return nil, nil, err
	}
	after, _, err := s.postingSvc.buildOutgoingLines(ctx, replacement, contact)
	if err != nil {
		return nil, nil, err
	}
	return contact, invoiceDifference(before, after), nil
}

func invoiceDifference(before, after []domain.JournalLine) []domain.JournalLine {
	type key struct {
		account string
		side    domain.Side
		tax     string
	}
	positions := map[key]int{}
	lines := []domain.JournalLine{}
	for pass, source := range [][]domain.JournalLine{before, after} {
		for _, line := range source {
			if pass == 0 {
				line.Amount = -line.Amount
				line.TaxBase = -line.TaxBase
			}
			k := key{line.Account, line.Side, line.TaxKey}
			if i, ok := positions[k]; ok {
				lines[i].Amount += line.Amount
				lines[i].TaxBase += line.TaxBase
			} else {
				line.ID, line.EntryID = 0, 0
				positions[k] = len(lines)
				lines = append(lines, line)
			}
		}
	}
	out := []domain.JournalLine{}
	for _, line := range lines {
		if line.Amount != 0 {
			if line.Amount < 0 {
				line.Amount = -line.Amount
				line.TaxBase = -line.TaxBase
				line.Side = line.Side.Opposite()
			}
			line.Position = len(out) + 1
			out = append(out, line)
		}
	}
	return out
}

func (s *InvoiceService) correctBookedInvoice(
	ctx context.Context, original *domain.Invoice, reason string, replacement *domain.Invoice,
) (*domain.Invoice, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("der Grund der Berichtigung fehlt")
	}
	contact, lines, err := s.prepareAmendment(ctx, original, replacement)
	if err != nil {
		return nil, err
	}
	err = s.issueDocument(ctx, replacement, contact, func(ctx context.Context) (*domain.JournalEntry, error) {
		if len(lines) == 0 {
			return nil, nil
		}
		// Die Differenz gehört zum Änderungstag (§ 17 Abs. 1 UStG). Der ursprüngliche
		// Leistungszeitraum bleibt auf dem unveränderten Rechnungsdokument erhalten.
		entry := &domain.JournalEntry{
			BookingDate: replacement.Date, DocumentDate: replacement.Date,
			ServiceDateFrom: replacement.Date, ServiceDateTo: replacement.Date,
			DueDate: replacement.DueDate, Description: "Berichtigung " + replacement.InvoiceNumber + ": " + reason,
			Source: domain.EntrySourceInvoice, DocumentNumber: replacement.InvoiceNumber,
			ReceiptID: replacement.ReceiptID, ContactID: &contact.ID,
			TaxTreatment: replacement.TaxTreatment, Currency: replacement.Currency, Lines: lines,
		}
		return s.postingSvc.journalSvc.Post(ctx, entry)
	}, func(ctx context.Context) error {
		if err := s.offsetInvoiceCredit(ctx, original, replacement); err != nil {
			return err
		}
		if original.GroupID != nil {
			group, err := s.groupRepo.FindByID(ctx, *original.GroupID)
			if err != nil {
				return err
			}
			group.FinalInvoiceID = &replacement.ID
			if err := s.groupRepo.Save(ctx, group); err != nil {
				return err
			}
		}
		original.Status = domain.InvoiceStatusCancelled
		original.CancelledByInvoiceID = &replacement.ID
		if err := s.invoiceRepo.Save(ctx, original); err != nil {
			return err
		}
		return s.log(ctx, domain.AuditActionUpdate, original, "Berichtigt durch "+replacement.InvoiceNumber+": "+reason)
	})
	if err != nil {
		return nil, err
	}
	return replacement, nil
}

// offsetInvoiceCredit verrechnet eine Minderung mit offenen Teilen derselben Rechnung.
func (s *InvoiceService) offsetInvoiceCredit(ctx context.Context, original, correction *domain.Invoice) error {
	if s.allocationRepo == nil || correction.JournalEntryID == nil {
		return nil
	}
	reader := &openItemReader{
		journalRepo:    s.postingSvc.journalSvc.journalRepo,
		allocationRepo: s.allocationRepo, contactRepo: s.contactRepo,
	}
	items, err := reader.OpenItems(ctx, "")
	if err != nil {
		return err
	}
	var credit domain.Cents
	for _, item := range items {
		if item.EntryID == *correction.JournalEntryID && item.IsCredit() {
			credit = item.OpenAmount
		}
	}
	if credit == 0 {
		return nil
	}
	chain, err := s.invoiceBookingChain(ctx, original)
	if err != nil {
		return err
	}
	for _, prior := range chain {
		for _, item := range items {
			if item.EntryID != prior.ID || item.IsCredit() || item.OpenAmount <= 0 {
				continue
			}
			amount := min(credit, item.OpenAmount)
			if amount == 0 {
				return nil
			}
			err := s.allocationRepo.Create(ctx, []domain.PaymentAllocation{
				{
					OpenItemEntryID: prior.ID, PaymentEntryID: *correction.JournalEntryID,
					ContactID: original.ContactID, SettledAmount: amount,
				},
				{
					OpenItemEntryID: *correction.JournalEntryID, PaymentEntryID: *correction.JournalEntryID,
					ContactID: original.ContactID, SettledAmount: amount,
				},
			})
			if err != nil {
				return err
			}
			credit -= amount
		}
	}
	return nil
}
