package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// incomingPlan hält Anfrage, Zeilen und Befunde zusammen. Er bleibt intern,
// damit die Buchung keinen vom Aufrufer veränderten Vorschauzustand übernimmt.
type incomingPlan struct {
	request ReceiptRequest
	preview *PostingPreview
	lines   []domain.JournalLine
	contact *domain.Contact
	receipt *domain.Receipt
	// fx ist die Umrechnung eines Fremdwährungsbelegs, nil bei einem in Euro.
	fx       *fxContext
	gifts    []domain.GiftRecord
	warnings []PostingWarning
	findings []InputTaxFinding
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
	euroNet := make([]domain.Cents, len(req.Positions))
	for i := range req.Positions {
		euroNet[i] = req.Positions[i].Net
	}
	if fx != nil {
		foreign := make([]domain.Cents, len(req.Positions))
		for i := range req.Positions {
			foreign[i] = req.Positions[i].Net
		}
		euroNet = fx.splitToEuro(foreign)
	}

	out := &incomingPlan{
		request:  req,
		contact:  contact,
		receipt:  receipt,
		fx:       fx,
		gifts:    make([]domain.GiftRecord, 0, 1),
		warnings: make([]PostingWarning, 0, 2),
	}
	if w := reverseChargeParagraphWarning(req.TaxTreatment, contact); w != nil {
		out.warnings = append(out.warnings, *w)
	}
	var lines []domain.JournalLine

	// 1. Aufwands- bzw. Anschaffungszeilen aus den fachlichen Gruppen.
	//
	// In zwei Durchgängen: erst stehen Konto, abziehbarer Vorsteueranteil und
	// die Aufzeichnung zum Geschenk je Position fest, dann entstehen die Zeilen.
	// Der Umweg ist die nicht abziehbare Vorsteuer eines geteilten Abzugs — sie
	// wird je Steuersatzgruppe einmal gerundet wie die Steuerzeile selbst, und
	// dafür müssen alle Positionen der Gruppe bekannt sein
	// (siehe nonDeductibleTaxByPosition).
	netByRate := map[domain.TaxRate]domain.Cents{}
	// owedByRate ist die volle Bemessungsgrundlage. Sie geht auseinander mit
	// netByRate, sobald ein Vorsteuerschlüssel greift: beim § 13b-Umsatz und
	// beim innergemeinschaftlichen Erwerb kürzt der abziehbare Anteil nur die
	// Vorsteuerzeile, während die geschuldete Steuer voll entsteht (UST-07 K2).
	owedByRate := map[domain.TaxRate]domain.Cents{}
	needsEntertainmentRecord := false
	prepared := make([]preparedPosition, 0, len(req.Positions))
	for i, p := range req.Positions {
		if p.Net <= 0 {
			return nil, fmt.Errorf("Position %d: der Nettobetrag muss größer als null sein", i+1)
		}
		foreignNet := domain.Cents(0)
		if fx != nil {
			foreignNet, p.Net = p.Net, euroNet[i]
			if p.Net <= 0 {
				return nil, fmt.Errorf(
					"Position %d: %s %s ergeben zum Kurs %s keinen Betrag über null",
					i+1, foreignNet, fx.currency, fx.conv.Rate.Source)
			}
		}
		// Die geleistete Anzahlung geht nicht durch die fachlichen Gruppen: sie
		// ist ein Posten des Vermögens und kein Aufwand, und welcher, entscheidet
		// die Verwendung und nicht die Art der Leistung.
		if req.AdvanceTarget != "" {
			prepared = append(prepared, preparedPosition{
				position: p, foreignNet: foreignNet, advance: true})
			continue
		}
		resolved, err := s.resolvePosition(ctx, req, p, giftTotals)
		if err != nil {
			return nil, fmt.Errorf("Position %d: %w", i+1, err)
		}
		if resolved.gift != nil {
			out.gifts = append(out.gifts, *resolved.gift)
			giftTotals[resolved.gift.RecipientKey()] += resolved.gift.NetAmount
		}
		out.warnings = append(out.warnings, resolved.warnings...)
		prepared = append(prepared, preparedPosition{
			position: p, foreignNet: foreignNet, resolved: resolved})
	}

	nonDeductibleTax := nonDeductibleTaxByPosition(prepared)
	for i, pp := range prepared {
		if pp.advance {
			account, err := accounting.VendorAdvanceAccountFor(req.AdvanceTarget)
			if err != nil {
				return nil, err
			}
			text := pp.position.Text
			if text == "" {
				text = "Geleistete Anzahlung"
			}
			lines = append(lines, domain.JournalLine{
				Side: domain.SideDebit, Account: account, Amount: pp.position.Net,
				ForeignAmount: pp.foreignNet, Text: text,
			})
			netByRate[pp.position.TaxRate] += pp.position.Net
			owedByRate[pp.position.TaxRate] += pp.position.Net
			continue
		}
		positionLines, quota, taxableNet, err := s.expenseLines(
			pp.resolved, nonDeductibleTax[i], req.TaxTreatment, req.DocumentDate)
		if err != nil {
			return nil, fmt.Errorf("Position %d: %w", i+1, err)
		}
		spreadForeign(positionLines, pp.foreignNet)
		lines = append(lines, positionLines...)
		if quota == accounting.QuotaEntertainment {
			needsEntertainmentRecord = true
		}
		// In die Bemessungsgrundlage geht der abziehbare Teil. Beim ungeteilten
		// Vorsteuerabzug ist das der volle Nettobetrag — auch dort, wo der
		// *Aufwand* geteilt wird: § 15 Abs. 1a Satz 2 UStG nimmt
		// Bewirtungsaufwendungen vom Vorsteuerausschluss ausdrücklich aus. Nur
		// ein Vorsteuerschlüssel oder ein Ausschluss nach § 15 Abs. 1a UStG
		// mindert sie.
		netByRate[pp.position.TaxRate] += taxableNet
		owedByRate[pp.position.TaxRate] += pp.position.Net
	}

	if needsEntertainmentRecord {
		if req.Entertainment == nil {
			return nil, fmt.Errorf(
				"zu einer Bewirtung gehören Ort, Tag, Teilnehmer und Anlass (§ 4 Abs. 5 Satz 1 Nr. 2 EStG). Ohne diese Aufzeichnung ist der Abzug auch für die abziehbaren 70 %% verloren")
		}
		if err := req.Entertainment.Validate(); err != nil {
			return nil, err
		}
	}

	// 2. Die abgesetzten Anzahlungen. Sie stehen vor den Steuerzeilen, weil sie
	// deren Bemessungsgrundlage mindern: die Vorsteuer auf den angezahlten Teil
	// ist mit der Zahlung schon gezogen worden, und ein zweites Mal gäbe es sie
	// nicht.
	// Die abgesetzte Anzahlung mindert beide Bemessungsgrundlagen: die
	// geschuldete Steuer entsteht auf den angezahlten Teil schon mit der
	// Zahlung, nicht ein zweites Mal mit der Schlussrechnung. Abgezogen wird
	// deshalb dieselbe Minderung, die advanceDeductionLines an netByRate
	// vornimmt — sonst stünde die volle Bemessungsgrundlage weiter in der
	// geschuldeten Steuer.
	beforeDeduction := make(map[domain.TaxRate]domain.Cents, len(netByRate))
	for rate, value := range netByRate {
		beforeDeduction[rate] = value
	}
	deductions, err := s.advanceDeductionLines(ctx, req, netByRate)
	if err != nil {
		return nil, err
	}
	lines = append(lines, deductions...)
	for rate, before := range beforeDeduction {
		owedByRate[rate] -= before - netByRate[rate]
	}

	// 3. Steuerzeilen, einmal je Steuersatzgruppe gerundet. In Fremdwährung
	// kommt die Kursdifferenz zwischen Tages- und Umsatzsteuerkurs hinzu.
	taxLines, err := s.taxLinesForShare(
		domain.DirectionIncoming, req.TaxTreatment, netByRate, owedByRate, fx)
	if err != nil {
		return nil, err
	}
	lines = append(lines, taxLines...)

	// 4. Ist der Beleg einer Rückstellung zugeordnet, tritt das
	// Rückstellungskonto an die Stelle des Aufwands — bis zur Höhe ihres
	// Bestands.
	if req.ProvisionID != 0 {
		lines, err = s.applyProvision(ctx, req.ProvisionID, lines)
		if err != nil {
			return nil, err
		}
	}

	// 5. Gegenzeile: was tatsächlich an den Lieferanten zu zahlen ist.
	settlementLine, err := s.settlementLine(lines, req.Settlement, req.PaymentAccount, contact)
	if err != nil {
		return nil, err
	}
	if req.Settlement == SettlementShareholderExpense {
		if err := s.expenseCreditor(ctx, req, contact, &settlementLine); err != nil {
			return nil, err
		}
	}
	if fx != nil {
		settlementLine.ForeignAmount = fx.toForeign(settlementLine.Amount)
	}
	lines = append(lines, settlementLine)

	// Zuletzt die Rechnungsprüfung: sie braucht die fertigen Zeilen, weil erst
	// aus ihnen hervorgeht, ob überhaupt Vorsteuer gezogen wird. Ein Beleg ohne
	// Vorsteuerabzug hat keine Pflichtangaben zu erfüllen, für die er gesperrt
	// werden müsste.
	findings, err := s.inputTaxFindings(ctx, req, contact, receipt, lines)
	if err != nil {
		return nil, err
	}
	out.findings = findings
	out.lines = lines
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
	if notice := s.serviceProofNotice(ctx, built.receipt); notice != nil {
		preview.Warnings = append(preview.Warnings, *notice)
	}
	preview.InputTaxFindings = built.findings
	if built.fx != nil {
		preview.Conversion = built.fx.conv
	}
	built.preview = preview
}
