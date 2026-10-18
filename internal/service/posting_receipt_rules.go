package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// receiptRule verarbeitet die Positionen oder die daraus gebildeten Zeilen.
// Rundung und Steuerbasis benötigen alle Positionen; deshalb folgen die
// Zeilenregeln erst nach der vollständigen Positionskette.
type receiptRule struct {
	name  string
	apply func(context.Context, *PostingService, *incomingWork) error
}

type incomingWork struct {
	*incomingPlan
	euroNet    []domain.Cents
	giftTotals map[string]domain.Cents
	prepared   []preparedPosition
	netByRate  map[domain.TaxRate]domain.Cents
	// Bei § 13b und ig. Erwerb entsteht die geschuldete Steuer auf die volle
	// Basis; ein geteilter Vorsteuerabzug mindert nur netByRate (UST-07 K2).
	owedByRate               map[domain.TaxRate]domain.Cents
	needsEntertainmentRecord bool
}

var incomingReceiptRules = []receiptRule{
	{"positionen", applyIncomingPositions},
	{"aufwand_und_bewirtung", applyExpenseLines},
	{"bewirtungsnachweis", applyEntertainmentRecord},
	{"anzahlungen_verrechnen", applyAdvanceDeductions},
	{"steuerzeilen", applyIncomingTaxLines},
	{"rueckstellung", applyIncomingProvision},
	{"gegenzeile_und_auslage", applyIncomingSettlement},
	{"rechnungspruefung", applyIncomingFindings},
	{"leistungsnachweis", applyIncomingServiceProof},
}

// Die Positionsregeln bestimmen Konto, Abzug und Geschenkaufzeichnung
// (§ 15 UStG, § 4 Abs. 5 und 7 EStG).
func applyIncomingPositions(ctx context.Context, s *PostingService, w *incomingWork) error {
	req := w.request
	for i, p := range req.Positions {
		pp, err := s.prepareIncomingPosition(ctx, req, p, w.fx, w.euroNet[i], w.giftTotals)
		if err != nil {
			return fmt.Errorf("Position %d: %w", i+1, err)
		}
		resolved := pp.resolved

		if resolved.gift != nil {
			w.gifts = append(w.gifts, *resolved.gift)
			w.giftTotals[resolved.gift.RecipientKey()] += resolved.gift.NetAmount
		}
		w.warnings = append(w.warnings, resolved.warnings...)
		w.prepared = append(w.prepared, pp)
	}

	return nil
}

// Die Aufwandszeilen teilen Bewirtung nach § 4 Abs. 5 Satz 1 Nr. 2 EStG.
// Nicht abziehbare Vorsteuer wird je Steuersatz gerundet (§ 9b Abs. 1 EStG).
func applyExpenseLines(ctx context.Context, s *PostingService, w *incomingWork) error {
	req := w.request
	nonDeductibleTax := nonDeductibleTaxByPosition(w.prepared)
	for i, pp := range w.prepared {
		if pp.advance {
			account, err := accounting.VendorAdvanceAccountFor(req.AdvanceTarget)
			if err != nil {
				return err
			}
			text := pp.position.Text
			if text == "" {
				text = "Geleistete Anzahlung"
			}
			w.lines = append(w.lines, domain.JournalLine{
				Side: domain.SideDebit, Account: account, Amount: pp.position.Net,
				ForeignAmount: pp.foreignNet, Text: text,
			})
			w.netByRate[pp.position.TaxRate] += pp.position.Net
			w.owedByRate[pp.position.TaxRate] += pp.position.Net
			continue
		}
		positionLines, quota, taxableNet, err := s.expenseLines(
			pp.resolved, nonDeductibleTax[i], req.TaxTreatment, req.DocumentDate)
		if err != nil {
			return fmt.Errorf("Position %d: %w", i+1, err)
		}
		spreadForeign(positionLines, pp.foreignNet)
		w.lines = append(w.lines, positionLines...)
		if quota == accounting.QuotaEntertainment {
			w.needsEntertainmentRecord = true
		}
		// In die Bemessungsgrundlage geht der abziehbare Teil. Beim ungeteilten
		// Vorsteuerabzug ist das der volle Nettobetrag — auch dort, wo der
		// *Aufwand* geteilt wird: § 15 Abs. 1a Satz 2 UStG nimmt
		// Bewirtungsaufwendungen vom Vorsteuerausschluss ausdrücklich aus. Nur
		// ein Vorsteuerschlüssel oder ein Ausschluss nach § 15 Abs. 1a UStG
		// mindert sie.
		w.netByRate[pp.position.TaxRate] += taxableNet
		w.owedByRate[pp.position.TaxRate] += pp.position.Net
	}

	return nil
}

// Die Bewirtungsaufzeichnung ist Voraussetzung des Abzugs
// (§ 4 Abs. 5 Satz 1 Nr. 2 EStG).
func applyEntertainmentRecord(ctx context.Context, s *PostingService, w *incomingWork) error {
	req := w.request
	if w.needsEntertainmentRecord {
		if req.Entertainment == nil {
			return fmt.Errorf(
				"zu einer Bewirtung gehören Ort, Tag, Teilnehmer und Anlass (§ 4 Abs. 5 Satz 1 Nr. 2 EStG). Ohne diese Aufzeichnung ist der Abzug auch für die abziehbaren 70 %% verloren")
		}
		if err := req.Entertainment.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Die Schlussrechnung setzt bereits versteuerte Anzahlungen ab
// (§ 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG).
func applyAdvanceDeductions(ctx context.Context, s *PostingService, w *incomingWork) error {
	req := w.request
	// Die abgesetzten Anzahlungen. Sie stehen vor den Steuerzeilen, weil sie
	// deren Bemessungsgrundlage mindern: die Vorsteuer auf den angezahlten Teil
	// ist mit der Zahlung schon gezogen worden, und ein zweites Mal gäbe es sie
	// nicht.
	// Die abgesetzte Anzahlung mindert beide Bemessungsgrundlagen: die
	// geschuldete Steuer entsteht auf den angezahlten Teil schon mit der
	// Zahlung, nicht ein zweites Mal mit der Schlussrechnung. Abgezogen wird
	// deshalb dieselbe Minderung, die advanceDeductionLines an w.netByRate
	// vornimmt — sonst stünde die volle Bemessungsgrundlage weiter in der
	// geschuldeten Steuer.
	beforeDeduction := make(map[domain.TaxRate]domain.Cents, len(w.netByRate))
	for rate, value := range w.netByRate {
		beforeDeduction[rate] = value
	}
	deductions, err := s.advanceDeductionLines(ctx, req, w.netByRate)
	if err != nil {
		return err
	}
	w.lines = append(w.lines, deductions...)
	for rate, before := range beforeDeduction {
		w.owedByRate[rate] -= before - w.netByRate[rate]
	}

	return nil
}

// Steuerzeilen entstehen je Steuersatz, bei Fremdwährung zum Monatskurs
// nach § 16 Abs. 6 UStG. Die Steuerfall-Verzweigung bleibt im Resolver.
func applyIncomingTaxLines(ctx context.Context, s *PostingService, w *incomingWork) error {
	req := w.request
	// Steuerzeilen, einmal je Steuersatzgruppe gerundet. In Fremdwährung
	// kommt die Kursdifferenz zwischen Tages- und Umsatzsteuerkurs hinzu.
	taxLines, err := s.taxLinesForShare(
		domain.DirectionIncoming, req.TaxTreatment, w.netByRate, w.owedByRate, w.fx)
	if err != nil {
		return err
	}
	w.lines = append(w.lines, taxLines...)

	return nil
}

// Die Erfüllung einer zurückgestellten Verpflichtung verbraucht den
// Bestand (§ 249 HGB); nur ein Mehrbetrag wird laufender Aufwand.
func applyIncomingProvision(ctx context.Context, s *PostingService, w *incomingWork) error {
	req := w.request
	// Ist der Beleg einer Rückstellung zugeordnet, tritt das
	// Rückstellungskonto an die Stelle des Aufwands — bis zur Höhe ihres
	// Bestands.
	var err error
	if req.ProvisionID != 0 {
		w.lines, err = s.applyProvision(ctx, req.ProvisionID, w.lines)
		if err != nil {
			return err
		}
	}

	return nil
}

// Die Gegenzeile ordnet die Schuld dem Lieferanten oder dem
// Erstattungsberechtigten einer Auslage zu (§ 246 Abs. 1 HGB).
func applyIncomingSettlement(ctx context.Context, s *PostingService, w *incomingWork) error {
	req := w.request
	// Gegenzeile: was tatsächlich an den Lieferanten zu zahlen ist.
	settlementLine, err := s.settlementLine(w.lines, req.Settlement, req.PaymentAccount, w.contact)
	if err != nil {
		return err
	}
	if req.Settlement == SettlementShareholderExpense {
		if err := s.expenseCreditor(ctx, req, w.contact, &settlementLine); err != nil {
			return err
		}
	}
	if w.fx != nil {
		settlementLine.ForeignAmount = w.fx.toForeign(settlementLine.Amount)
	}
	w.lines = append(w.lines, settlementLine)

	return nil
}

// Der Vorsteuerabzug benötigt die Pflichtangaben der Rechnung
// (§ 15 Abs. 1 Satz 1 Nr. 1 i. V. m. §§ 14, 14a UStG).
func applyIncomingFindings(ctx context.Context, s *PostingService, w *incomingWork) error {
	req := w.request
	// Zuletzt die Rechnungsprüfung: sie braucht die fertigen Zeilen, weil erst
	// aus ihnen hervorgeht, ob überhaupt Vorsteuer gezogen wird. Ein Beleg ohne
	// Vorsteuerabzug hat keine Pflichtangaben zu erfüllen, für die er gesperrt
	// werden müsste.
	findings, err := s.inputTaxFindings(ctx, req, w.contact, w.receipt, w.lines)
	if err != nil {
		return err
	}
	w.findings = findings
	return nil
}

// applyIncomingServiceProof zeigt den fehlenden Kontrollvermerk nach RECH-08.
// Die Grenze legt das Unternehmen selbst fest; sie ist keine Gesetzesgrenze.
func applyIncomingServiceProof(ctx context.Context, s *PostingService, w *incomingWork) error {
	w.serviceProofWarning = s.serviceProofNotice(ctx, w.receipt)
	return nil
}
