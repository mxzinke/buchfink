package service

import (
	"context"
	"fmt"
	"regexp"

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
	header   *domain.ReceiptHeader
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
	if err := validateDueDate(req); err != nil {
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
	if w := feeGroupWarning(req, contact); w != nil {
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
	out.header = incomingReceiptHeader(out)
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
	if req.Settlement == SettlementOpen {
		preview.DueDate = req.DueDate
		if preview.DueDate == "" {
			preview.DueDate = dueDate(&domain.JournalEntry{DocumentDate: req.DocumentDate}, *built.contact)
			preview.Warnings = append(preview.Warnings, PostingWarning{
				Code: warningDueFromTerms, Severity: "info",
				Title: fmt.Sprintf("Fällig am %s nach dem Zahlungsziel des Lieferanten", domain.GermanDate(preview.DueDate)),
				Detail: "Nennt der Beleg eine andere Fälligkeit, gehört sie in die Buchung; sonst stehen offener " +
					"Posten und Zahlungsvorschläge auf dem falschen Tag.",
			})
		}
	}
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

// incomingReceiptHeader ergänzt nur fehlende Angaben ohne strukturierten Datensatz.
func incomingReceiptHeader(plan *incomingPlan) *domain.ReceiptHeader {
	r, req := plan.receipt, plan.request
	if _, structured := r.FileByRole(domain.ReceiptRoleStructured); structured ||
		(r.DocumentDate != "" && r.IssuerName != "" && r.GrossAmount != 0) {
		return nil
	}
	h := domain.ReceiptHeader{Kind: r.Kind, DocumentDate: r.DocumentDate, IssuerName: r.IssuerName, GrossAmount: r.GrossAmount, TaxAmount: r.TaxAmount, Currency: r.Currency, Subject: r.Subject}
	if h.DocumentDate == "" {
		h.DocumentDate = req.DocumentDate
	}
	if h.IssuerName == "" {
		h.IssuerName = plan.contact.LegalName()
	}
	if h.GrossAmount == 0 {
		if req.Currency != "" && req.Currency != "EUR" {
			h.GrossAmount, h.Currency = req.ForeignAmount, req.Currency
		} else {
			h.GrossAmount, h.TaxAmount = plan.preview.Gross, plan.preview.Tax
		}
	}
	return &h
}

const warningDueFromTerms = "faelligkeit_aus_zahlungsziel"

// validateDueDate weist eine Fälligkeit zurück, die kein Datum ist oder vor dem
// Belegdatum liegt.
func validateDueDate(req ReceiptRequest) error {
	if req.DueDate == "" || req.Settlement != SettlementOpen {
		return nil
	}
	if err := domain.CheckDate("Fälligkeit", req.DueDate); err != nil {
		return err
	}
	if req.DocumentDate != "" && req.DueDate < req.DocumentDate {
		return fmt.Errorf("die Fälligkeit %s liegt vor dem Belegdatum %s",
			domain.GermanDate(req.DueDate), domain.GermanDate(req.DocumentDate))
	}
	return nil
}

const warningFeeGroup = "gruppe_passt_nicht_zum_aussteller"

var (
	authorityName = regexp.MustCompile(`(?i)gericht|behörde|landeshauptkasse|gerichtskasse|(^|[\s-])(bezirks|ordnungs|gewerbe|grundbuch|stadt|kreis|landrats)?amt\b|gemeinde|stadtkasse`)
	chamberName   = regexp.MustCompile(`(?i)kammer|verband|\bihk\b|\bhwk\b|verein`)
)

// feeGroupWarning meldet eine Buchungsgruppe, die nicht zum Aussteller passt:
// die Gebühr eines Gerichts oder einer Behörde unter den Beiträgen oder den
// Beitrag einer Kammer unter den Gebühren. Erkannt wird am Namen; es ist ein
// Hinweis, keine Sperre.
func feeGroupWarning(req ReceiptRequest, contact *domain.Contact) *PostingWarning {
	name := contact.Name + " " + contact.Company
	for _, p := range req.Positions {
		switch {
		case p.PostingGroup == "beitraege" && authorityName.MatchString(name) && !chamberName.MatchString(name):
			return &PostingWarning{Code: warningFeeGroup, Severity: "warning",
				Title:  "Gebühr unter Beiträgen",
				Detail: contact.Name + " ist ein Gericht oder eine Behörde. Deren Gebühren gehören in die Gruppe „Gebühren & Abgaben“ (6430), Beiträge (6420) sind Kammer- und Verbandsbeiträge; Gebühren der Gründung sind Gründungskosten."}
		case p.PostingGroup == "gebuehren" && chamberName.MatchString(name):
			return &PostingWarning{Code: warningFeeGroup, Severity: "warning",
				Title:  "Beitrag unter Gebühren",
				Detail: contact.Name + " ist eine Kammer oder ein Verband. Deren Beiträge gehören in die Gruppe „Beiträge“ (6420)."}
		}
	}
	return nil
}
