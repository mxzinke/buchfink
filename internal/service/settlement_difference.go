package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// assetCostReduction verknüpft eine Anschaffungskostenminderung mit der Zahlung.
type assetCostReduction struct {
	assetID uint
	amount  domain.Cents
	note    string
}

func (s *settlement) differenceLines(
	ctx context.Context,
	alloc AllocationRequest,
	item domain.OpenItem,
) ([]domain.JournalLine, *assetCostReduction, error) {
	if alloc.DifferenceKind == domain.DifferenceNone {
		return nil, nil, nil
	}

	// Abzüge stehen dem Ausgleich gegenüber.
	opposite := item.SettleSide().Opposite()

	switch alloc.DifferenceKind {
	case domain.DifferenceSkonto:
		return s.skontoLines(ctx, alloc, item, opposite)

	case domain.DifferenceBankFee:
		// Die Gebühr ist Aufwand, beim Zahlungsausgang wie beim -eingang. Den
		// Unterschied macht der Zahlbetrag (cashAmount).
		return []domain.JournalLine{{
			Side: domain.SideDebit, Account: domain.AccountNebenkostenGeld,
			Amount: alloc.DifferenceAmount, Text: "Bankgebühr",
		}}, nil, nil

	case domain.DifferenceRounding, domain.DifferenceCurrency:
		// Ein positiver Betrag steht dem Ausgleich gegenüber, ein negativer
		// (es floss mehr) auf seiner Seite. Im Haben entsteht ein Ertrag, im
		// Soll ein Aufwand.
		side, amount := opposite, alloc.DifferenceAmount
		if amount < 0 {
			side, amount = opposite.Opposite(), -amount
		}
		// Auch die realisierte Kursdifferenz ist Ergebnis der Währungsumrechnung
		// und gesondert auszuweisen (§ 277 Abs. 5 Satz 2 HGB).
		loss, gain, text := accounting.CurrencyLossAccount, accounting.CurrencyGainAccount, "Kursdifferenz"
		if alloc.DifferenceKind == domain.DifferenceRounding {
			loss, gain, text = "6300", "4830", "Rundungsdifferenz"
		}
		account := loss
		if side == domain.SideCredit {
			account = gain
		}
		return []domain.JournalLine{{Side: side, Account: account, Amount: amount, Text: text}}, nil, nil
	}

	return nil, nil, fmt.Errorf("unbekannte Differenzart %q", alloc.DifferenceKind)
}

// skontoLines ordnet die Entgeltminderung aus dem Steuerfall-Katalog dem
// offenen Posten und gegebenenfalls seinem Anlagegut zu.
func (s *settlement) skontoLines(
	ctx context.Context,
	alloc AllocationRequest,
	item domain.OpenItem,
	opposite domain.Side,
) ([]domain.JournalLine, *assetCostReduction, error) {
	if item.TaxTreatment == "" {
		return nil, nil, fmt.Errorf(
			"der Steuerfall von %s lässt sich nicht bestimmen; ein Skonto darauf ließe sich nach § 17 Abs. 1 UStG nicht zutreffend berichtigen",
			item.DocumentNumber)
	}
	direction := item.Direction()

	skonto, err := accounting.TaxCaseFor(item.TaxTreatment).Skonto(direction, item.TaxRate, alloc.DifferenceAmount)
	if err != nil {
		return nil, nil, fmt.Errorf("die Steuerkorrektur des Skontos ließ sich nicht auflösen: %w", err)
	}

	// Anschaffungspreisminderungen senken die Anschaffungskosten (§ 255 Abs. 1
	// Satz 3 HGB). Die Steuerkorrektur richtet sich weiterhin nach dem Umsatz.
	account := skonto.Account
	text := "Skonto"
	var reduction *assetCostReduction
	if asset := s.assetForItem(ctx, item, direction); asset != nil {
		account = asset.Account
		text = "Skonto: Minderung der Anschaffungskosten " + asset.InventoryNumber
		reduction = &assetCostReduction{
			assetID: asset.ID,
			amount:  skonto.Net,
			note: fmt.Sprintf("Skonto auf %s: Minderung der Anschaffungskosten "+
				"(§ 255 Abs. 1 Satz 3 HGB)", item.DocumentNumber),
		}
	}

	var lines []domain.JournalLine
	if reduction == nil && direction == domain.DirectionIncoming {
		lines = s.expenseReductionLines(ctx, item, opposite, skonto.Account, skonto.Net)
	}
	if lines == nil {
		lines = []domain.JournalLine{{Side: opposite, Account: account, Amount: skonto.Net, Text: text}}
	}
	for _, leg := range skonto.Legs {
		line := taxLegLine(leg)
		line.Text = "Steuerkorrektur Skonto (§ 17 Abs. 1 UStG)"
		lines = append(lines, line)
	}
	return lines, reduction, nil
}

// expenseReductionLines verteilt ein erhaltenes Skonto auf die Aufwandskonten
// der Eingangsrechnung: es mindert den Aufwand, auf dem die Rechnung steht.
// Materialaufwand behält das Skontokonto, das im SKR04 dort steht. Ohne
// Aufwandszeile, etwa bei einem Bestandskonto, kommt nil zurück, und es bleibt
// beim Skontokonto.
func (s *settlement) expenseReductionLines(
	ctx context.Context, item domain.OpenItem, side domain.Side, skontoAccount string, net domain.Cents,
) []domain.JournalLine {
	entry, err := s.journalRepo.FindByID(ctx, item.EntryID)
	if err != nil || entry == nil {
		return nil
	}
	weights := map[string]domain.Cents{}
	order := []string{}
	var total domain.Cents
	for _, l := range entry.Lines {
		if l.Side != domain.SideDebit || l.TaxKey != "" || !isExpenseAccount(l.Account) {
			continue
		}
		account := l.Account
		if account[0] == '5' {
			account = skontoAccount
		}
		if _, seen := weights[account]; !seen {
			order = append(order, account)
		}
		weights[account] += l.Amount
		total += l.Amount
	}
	if total <= 0 {
		return nil
	}
	lines := make([]domain.JournalLine, 0, len(order))
	var assigned domain.Cents
	for i, account := range order {
		amount := domain.MulRound(net, int64(weights[account]), int64(total))
		if i == len(order)-1 {
			amount = net - assigned
		}
		assigned += amount
		if amount == 0 {
			continue
		}
		text := "Skonto: Minderung des Aufwands"
		if account == skontoAccount {
			text = "Skonto"
		}
		lines = append(lines, domain.JournalLine{Side: side, Account: account, Amount: amount, Text: text})
	}
	return lines
}

// isExpenseAccount erkennt die Aufwandskonten des SKR04 (Klassen 5 bis 7).
func isExpenseAccount(account string) bool {
	return len(account) == 4 && account[0] >= '5' && account[0] <= '7'
}

// assetForItem berücksichtigt aktive Anlagen auf der Eingangsseite. Ohne Kartei
// oder bei einem Lesefehler bleibt es beim gewöhnlichen Skonto.
func (s *settlement) assetForItem(
	ctx context.Context, item domain.OpenItem, direction domain.Direction,
) *domain.FixedAsset {
	if s.assets == nil || direction != domain.DirectionIncoming {
		return nil
	}
	asset, err := s.assets.AssetForEntry(ctx, item.EntryID)
	if err != nil || asset == nil || asset.IsDisposed() {
		return nil
	}
	return asset
}

// cashAmount ist der Zahlbetrag nach Abzügen und Gebühren. settle ist die
// Seite, auf der der Posten ausgeglichen wird: Soll beim Zahlungsausgang, Haben
// beim Zahlungseingang.
//
// Eine Bankgebühr belastet beim Ausgang zusätzlich zum Rechnungsbetrag; beim
// Eingang hat eine Bank in der Kette sie vom Betrag abgezogen.
func (a AllocationRequest) cashAmount(settle domain.Side) domain.Cents {
	switch a.DifferenceKind {
	case domain.DifferenceSkonto, domain.DifferenceRounding, domain.DifferenceCurrency:
		return a.SettledAmount - a.DifferenceAmount
	case domain.DifferenceBankFee:
		if settle == domain.SideCredit {
			return a.SettledAmount - a.DifferenceAmount
		}
		return a.SettledAmount + a.DifferenceAmount
	default:
		return a.SettledAmount
	}
}
