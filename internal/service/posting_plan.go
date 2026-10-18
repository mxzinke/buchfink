package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// incomingPlan hält Anfrage, Zeilen und Befunde zusammen. Er bleibt intern,
// damit die Buchung keinen vom Aufrufer veränderten Vorschauzustand übernimmt.
type incomingPlan struct {
	request  ReceiptRequest
	specials []string
	preview  *PostingPreview
	lines    []domain.JournalLine
	contact  *domain.Contact
	receipt  *domain.Receipt
	// fx ist die Umrechnung eines Fremdwährungsbelegs, nil bei einem in Euro.
	fx                  *fxContext
	gifts               []domain.GiftRecord
	warnings            []PostingWarning
	findings            []InputTaxFinding
	serviceProofWarning *PostingWarning
}

// planIncomingReceipt ist der gemeinsame Rechenweg für Vorschau und Buchung.
func (s *PostingService) planIncomingReceipt(ctx context.Context, req ReceiptRequest) (*incomingPlan, error) {
	if len(req.Positions) == 0 {
		return nil, fmt.Errorf("der Beleg hat keine Positionen")
	}
	// Kein stiller Vorgabewert: ein steuerfreier, ein nullbesteuerter und ein
	// dem Reverse-Charge unterliegender Einkauf sehen im Betrag gleich aus und
	// werden verschieden gebucht. Wo der Steuerfall fehlt — etwa weil der
	// Rechnungsdatensatz Kategorien mischt —, ist er zu wählen, nicht zu raten.
	if req.TaxTreatment == "" {
		return nil, fmt.Errorf(
			"der Steuerfall fehlt. Er ist anzugeben und lässt sich nicht aus den Beträgen erschließen")
	}

	contact, err := s.contactRepo.FindByID(ctx, req.ContactID)
	if err != nil {
		return nil, fmt.Errorf("Lieferant konnte nicht geladen werden: %w", err)
	}
	if contact.Type != domain.ContactTypeVendor {
		return nil, fmt.Errorf("%s ist als Kunde angelegt und kann keinen Eingangsbeleg stellen", contact.Name)
	}
	if err := ensureNotBlocked(contact); err != nil {
		return nil, err
	}
	if err := validateIncomingTreatment(req.TaxTreatment, contact); err != nil {
		return nil, err
	}

	receipt, err := s.loadBookableReceipt(ctx, req.ReceiptID)
	if err != nil {
		return nil, err
	}

	if err := s.validateAdvanceRequest(req); err != nil {
		return nil, err
	}

	// Die Geschenke des Wirtschaftsjahres, einmal gelesen: die Freigrenze des
	// § 4 Abs. 5 Satz 1 Nr. 1 EStG läuft je Empfänger über das Jahr, und ohne
	// den bisherigen Stand ließe sich nicht sagen, ob dieses Geschenk sie reißt.
	giftTotals, err := s.giftTotals(ctx, req)
	if err != nil {
		return nil, err
	}

	// Die Umrechnung steht vor allem anderen: ab hier rechnet der ganze
	// Belegweg in Euro. Die Freigrenze für Geschenke, die 70/30-Aufteilung der
	// Bewirtung und jede Wertgrenze sind Eurobeträge, und sie gegen einen
	// Dollarbetrag zu halten wäre die stillste Art, sie falsch anzuwenden.
	fx, err := s.prepareCurrency(ctx, req)
	if err != nil {
		return nil, err
	}

	out := &incomingPlan{
		request:  req,
		specials: req.Specials(),
		contact:  contact,
		receipt:  receipt,
		fx:       fx,
		gifts:    make([]domain.GiftRecord, 0, 1),
		warnings: make([]PostingWarning, 0, 2),
	}
	if w := reverseChargeParagraphWarning(req.TaxTreatment, contact); w != nil {
		out.warnings = append(out.warnings, *w)
	}

	w := &incomingWork{
		incomingPlan: out, giftTotals: giftTotals,
		netByRate: map[domain.TaxRate]domain.Cents{}, owedByRate: map[domain.TaxRate]domain.Cents{},
		prepared: make([]preparedPosition, 0, len(req.Positions)),
	}
	w.euroNet = make([]domain.Cents, len(req.Positions))
	for i := range req.Positions {
		w.euroNet[i] = req.Positions[i].Net
	}
	if fx != nil {
		foreign := make([]domain.Cents, len(req.Positions))
		for i := range req.Positions {
			foreign[i] = req.Positions[i].Net
		}
		w.euroNet = fx.splitToEuro(foreign)
	}

	for _, rule := range incomingReceiptRules {
		if err := rule.apply(ctx, s, w); err != nil {
			return nil, err
		}
	}

	s.completeIncomingPreview(ctx, out)
	return out, nil
}

func (s *PostingService) completeIncomingPreview(ctx context.Context, built *incomingPlan) {
	req := built.request
	preview := s.preview(ctx, built.lines)
	if notice := eInvoiceNotice(
		built.contact, built.receipt, req.TaxTreatment, req.DocumentDate, preview.Gross); notice != nil {
		preview.Warnings = append(preview.Warnings, *notice)
	}
	preview.Warnings = append(preview.Warnings, built.warnings...)
	// Der fehlende Leistungsnachweis steht schon in der Vorschau: die Maske soll
	// das Feld verlangen, bevor jemand auf „Buchen" drückt.
	if notice := built.serviceProofWarning; notice != nil {
		preview.Warnings = append(preview.Warnings, *notice)
	}
	preview.InputTaxFindings = built.findings
	preview.specials = built.specials
	if built.fx != nil {
		preview.Conversion = built.fx.conv
	}
	built.preview = preview
}
