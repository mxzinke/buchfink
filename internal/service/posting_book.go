package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// bookIncomingReceipt schreibt Nachweise, Buchung, Nebenbücher und Siegel
// in derselben Transaktion. Das Journal prüft darin auch die Periodensperren.
func (s *PostingService) bookIncomingReceipt(ctx context.Context, built *incomingPlan) (*domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		return s.postIncomingPlan(ctx, built)
	})
}

func (s *PostingService) postIncomingPlan(ctx context.Context, built *incomingPlan) (*domain.JournalEntry, error) {
	req := built.request
	if err := built.blockingError(req); err != nil {
		return nil, err
	}
	// Der Leistungsnachweis ist ab der eingestellten Grenze Pflicht (RECH-08):
	// geprüft wird vor der Buchung, weil ein gebuchter Beleg den Vermerk zwar
	// noch aufnimmt, die Prüfung gegen die Bestellung dann aber niemand mehr
	// vornimmt. Geprüft wird gegen den Beleg *und* gegen den mitgeschickten
	// Vermerk; geschrieben wird der mitgeschickte erst in der Transaktion.
	if err := s.requireServiceProof(ctx, built.receipt, req); err != nil {
		return nil, err
	}
	lines, contact, receipt := built.lines, built.contact, built.receipt

	description := req.Description
	if description == "" {
		description = fmt.Sprintf("Eingangsbeleg %s, %s", receipt.ReceiptNumber, contact.Name)
	}
	if req.CreditNoteFor != "" {
		description = fmt.Sprintf("Gutschrift zu %s: %s", req.CreditNoteFor, description)
	}

	// Die Quelle „advance" ist die Bedingung der Periodenzuordnung und keine
	// bloße Etikettierung: accounting.VatPeriodFor legt die Steuer einer
	// Anzahlung in den Zeitraum der Zahlung. Ohne sie entschiede der
	// Leistungszeitraum — und geleistet ist bei einer Anzahlung noch nichts.
	source := domain.EntrySourceReceipt
	if req.AdvanceTarget != "" {
		source = domain.EntrySourceAdvance
		if req.Description == "" {
			description = fmt.Sprintf("Geleistete Anzahlung %s an %s", receipt.ReceiptNumber, contact.Name)
		}
	}

	entry := &domain.JournalEntry{
		BookingDate:        req.BookingDate,
		DocumentDate:       req.DocumentDate,
		ServiceDateFrom:    req.ServiceDateFrom,
		ServiceDateTo:      req.ServiceDateTo,
		DueDate:            openDueDate(req),
		Description:        description,
		Source:             source,
		DocumentNumber:     receipt.ReceiptNumber,
		TaxTreatment:       req.TaxTreatment,
		ReceiptID:          &receipt.ID,
		ReceiptHash:        receipt.ReceiptHash,
		ContactID:          &contact.ID,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
		Entertainment:      req.Entertainment,
		Gifts:              built.gifts,
	}
	// Währung, Kurs, Quelle und Kurstag erklären gemeinsam die Eurobeträge.
	if built.fx != nil {
		built.fx.head(entry)
	}

	if err := s.saveServiceProof(ctx, built, req); err != nil {
		return nil, err
	}
	created, err := s.journalSvc.Post(ctx, entry)
	if err != nil {
		return nil, err
	}

	// Die geleistete Anzahlung wird vermerkt: die Schlussrechnung des
	// Lieferanten muss sie absetzen können, und ohne diesen Vermerk stünde sie
	// nur als Saldo auf 1180.
	if req.AdvanceTarget != "" {
		if err := s.recordVendorAdvance(ctx, req, receipt, created); err != nil {
			return nil, fmt.Errorf("die Anzahlung ließ sich nicht vermerken: %w", err)
		}
	}
	for _, id := range req.SettledAdvanceIDs {
		advance, err := s.vendorAdvances.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		advance.SettledByEntryID = &created.ID
		if err := s.vendorAdvances.Save(ctx, advance); err != nil {
			return nil, fmt.Errorf("die Anzahlung %s ließ sich nicht als verrechnet vermerken: %w", advance.DocumentNumber, err)
		}
	}

	// Der Verbrauch wird in der Rückstellungskartei vermerkt, sobald die
	// Buchung steht: der Rückstellungsspiegel liest ihn dort und nicht aus dem
	// Journal, weil er die Bewegungsart braucht und nicht nur das Konto.
	if req.ProvisionID != 0 && s.provisions != nil {
		var consumed domain.Cents
		for _, l := range created.Lines {
			if l.Side == domain.SideDebit && l.Text == "Verbrauch der Rückstellung" {
				consumed += l.Amount
			}
		}
		if consumed > 0 {
			if err := s.provisions.RecordConsumption(
				ctx, req.ProvisionID, consumed, req.BookingDate, created.ID,
				fmt.Sprintf("Rechnung %s", receipt.ReceiptNumber),
			); err != nil {
				return nil, fmt.Errorf("der Verbrauch der Rückstellung ließ sich nicht vermerken: %w", err)
			}
		}
	}

	// Der Grund der Übersteuerung gehört an den Beleg und ins Protokoll — vor
	// der Versiegelung, weil ein versiegelter Beleg nichts mehr aufnimmt.
	if strings.TrimSpace(req.OverrideReason) != "" && len(built.findings) > 0 {
		if err := s.receiptSvc.SaveInputTaxOverride(ctx, receipt.ID, req.OverrideReason); err != nil {
			return nil, fmt.Errorf("der Grund der Übersteuerung ließ sich nicht am Beleg vermerken: %w", err)
		}
	}

	if err := s.receiptSvc.Seal(ctx, receipt.ID, created.ID); err != nil {
		return nil, fmt.Errorf("der Beleg %s ließ sich nicht versiegeln: %w", receipt.ReceiptNumber, err)
	}
	return created, nil
}

// openDueDate ist die Fälligkeit, die an der Buchung steht: nur bei offener
// Zahlung, sonst gibt es keinen offenen Posten.
func openDueDate(req ReceiptRequest) string {
	if req.Settlement != SettlementOpen {
		return ""
	}
	return req.DueDate
}
