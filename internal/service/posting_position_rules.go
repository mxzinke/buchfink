package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// positionRule arbeitet auf einer Position. Die Reihenfolge bestimmt auch,
// welcher Fehler zuerst gemeldet wird; eine Regel schreibt keine Buchung.
type positionRule struct {
	name  string
	apply func(context.Context, *PostingService, *positionWork) error
}

type positionWork struct {
	request    ReceiptRequest
	resolved   resolvedPosition
	giftTotals map[string]domain.Cents
	fx         *fxContext
	euroNet    domain.Cents
	foreignNet domain.Cents
	advance    bool
}

var incomingPositionRules = []positionRule{
	{"fremdwaehrung", applyPositionCurrency},
	{"anzahlung", applyAdvancePosition},
	{"vorsteueranteil", applyInputTaxShare},
	{"buchungsgruppe", applyPostingGroup},
	{"vorsteuerausschluss", applyInputTaxExclusion},
	{"geschenk_freigrenze", applyGiftLimit},
}

func (s *PostingService) prepareIncomingPosition(ctx context.Context, req ReceiptRequest, p ReceiptPosition, fx *fxContext, euroNet domain.Cents, giftTotals map[string]domain.Cents) (preparedPosition, error) {
	w := positionWork{request: req, giftTotals: giftTotals, fx: fx, euroNet: euroNet,
		resolved: resolvedPosition{position: p, permille: 1000, warnings: make([]PostingWarning, 0, 1)}}
	for _, rule := range incomingPositionRules {
		if err := rule.apply(ctx, s, &w); err != nil {
			return preparedPosition{}, err
		}
		if w.advance {
			break
		}
	}
	return preparedPosition{position: w.resolved.position, foreignNet: w.foreignNet, advance: w.advance, resolved: w.resolved}, nil
}

// applyPositionCurrency rechnet vor jeder Euro-Freigrenze um (§ 256a HGB).
// Die Kurse sind vorbereitet; diese Regel fragt keinen Netzdienst.
func applyPositionCurrency(_ context.Context, _ *PostingService, w *positionWork) error {
	p := &w.resolved.position
	if p.Net <= 0 {
		return fmt.Errorf("der Nettobetrag muss größer als null sein")
	}
	if w.fx != nil {
		w.foreignNet, p.Net = p.Net, w.euroNet
		if p.Net <= 0 {
			return fmt.Errorf("%s %s ergeben zum Kurs %s keinen Betrag über null", w.foreignNet, w.fx.currency, w.fx.conv.Rate.Source)
		}
	}
	return nil
}

// applyAdvancePosition führt die gezahlte Anzahlung als Vermögen (§ 266 HGB).
// Der Vorsteuerabzug setzt ihre Zahlung voraus (§ 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG).
func applyAdvancePosition(_ context.Context, _ *PostingService, w *positionWork) error {
	w.advance = w.request.AdvanceTarget != ""
	return nil
}

// applyInputTaxShare begrenzt den Abzug nach § 15 Abs. 4 UStG und hält
// den Maßstab der Schätzung fest.
func applyInputTaxShare(_ context.Context, _ *PostingService, w *positionWork) error {
	p := w.resolved.position
	if p.InputTaxShare != 0 {
		if p.InputTaxShare < 0 || p.InputTaxShare > 1000 {
			return fmt.Errorf("der abziehbare Vorsteueranteil liegt zwischen 0 und 100 %%")
		}
		if p.InputTaxShare < 1000 && strings.TrimSpace(p.InputTaxShareReason) == "" {
			return fmt.Errorf(
				"zu einem geteilten Vorsteuerabzug gehört sein Maßstab. § 15 Abs. 4 Satz 2 UStG lässt " +
					"die Aufteilung nach einer sachgerechten Schätzung zu — halte fest, worauf sie " +
					"beruht, etwa „Kfz zu 60 %% betrieblich genutzt, Fahrtenbuch 2026\"")
		}
		// Der geteilte Abzug gilt auch beim § 13b-Umsatz und beim
		// innergemeinschaftlichen Erwerb (UST-07 K2). Dort entstehen zwei
		// Steuerzeilen aus derselben Bemessungsgrundlage, und der Schlüssel
		// wirkt nur auf die Vorsteuerzeile — die geschuldete Steuer bleibt voll
		// (siehe taxLinesForShare).
		w.resolved.permille = int64(p.InputTaxShare)
		if w.resolved.permille < 1000 {
			w.resolved.note = fmt.Sprintf("Vorsteuer zu %s abziehbar: %s",
				accounting.PermilleLabel(w.resolved.permille), strings.TrimSpace(p.InputTaxShareReason))
		}
	}

	return nil
}

// applyPostingGroup ordnet die Position dem SKR04 zu. Konten mit
// Aufzeichnungspflicht (§ 4 Abs. 7 EStG) verlangen ihre Buchungsgruppe.
func applyPostingGroup(_ context.Context, _ *PostingService, w *positionWork) error {
	p := w.resolved.position
	if p.Account != "" {
		if label, blocked := accounting.AccountsRequiringGroup()[p.Account]; blocked {
			return fmt.Errorf(
				"das Konto %s gehört zur Gruppe %q. Diese Konten sind nur über ihre Buchungsgruppe "+
					"erreichbar: an ihnen hängen die Aufzeichnungspflicht des § 4 Abs. 7 EStG, die "+
					"Freigrenze je Empfänger und der Ausschluss der Vorsteuer — von Hand gebucht wäre "+
					"davon nichts geprüft", p.Account, label)
		}
		w.resolved.account = p.Account
		return nil
	}
	if p.PostingGroup == "" {
		return fmt.Errorf("weder eine Buchungsgruppe noch ein Konto angegeben")
	}
	group, err := accounting.LookupPostingGroup(p.PostingGroup)
	if err != nil {
		return err
	}
	if group.Direction != domain.DirectionIncoming {
		return fmt.Errorf("%q passt nicht zur Belegrichtung", group.Label)
	}
	w.resolved.group, w.resolved.hasGroup = group, true

	return nil
}

// applyInputTaxExclusion rechnet die Vorsteuer nach § 15 Abs. 1a UStG
// dem Aufwand zu (§ 9b Abs. 1 EStG).
func applyInputTaxExclusion(_ context.Context, _ *PostingService, w *positionWork) error {
	if !w.resolved.hasGroup {
		return nil
	}
	group := w.resolved.group
	// § 15 Abs. 1a UStG: für die Aufwendungen des § 4 Abs. 5 Satz 1 Nr. 1 bis 4
	// und 7 EStG gibt es keinen Vorsteuerabzug. Die Steuer gehört dann zum
	// Aufwand.
	if group.InputTaxExcluded {
		w.resolved.permille = 0
		w.resolved.note = appendText(w.resolved.note,
			"Kein Vorsteuerabzug (§ 15 Abs. 1a UStG); die Umsatzsteuer gehört zum Aufwand")
	}

	return nil
}

// applyGiftLimit prüft die Freigrenze je Empfänger (§ 4 Abs. 5 Satz 1
// Nr. 1 EStG) und zeichnet das Geschenk nach § 4 Abs. 7 EStG auf.
func applyGiftLimit(ctx context.Context, s *PostingService, w *positionWork) error {
	if !w.resolved.hasGroup {
		return nil
	}
	p, group := w.resolved.position, w.resolved.group
	if group.RecipientRequired || group.Limit == accounting.LimitGiftPerRecipient {
		gift, warning, err := s.resolveGift(ctx, w.request, p, group, w.giftTotals)
		if err != nil {
			return err
		}
		w.resolved.gift = gift
		if warning != nil {
			w.resolved.warnings = append(w.resolved.warnings, *warning)
		}
		if gift != nil && gift.NonDeductible {
			w.resolved.account = group.NonDeductibleAccount
			w.resolved.permille = 0
			w.resolved.hasGroup = false
			w.resolved.note = appendText(w.resolved.note, fmt.Sprintf(
				"Freigrenze des § 4 Abs. 5 Satz 1 Nr. 1 EStG für %s überschritten: nicht abziehbar, "+
					"und ohne Vorsteuerabzug (§ 15 Abs. 1a UStG)", gift.RecipientName))
		}
	}
	return nil
}
