package ebilanz

import (
	"github.com/buchfink/buchfink/internal/domain"
)

// Steuerlicher Betriebsvermögensvergleich (Leitfaden B.14.3.1.10,
// B.14.3.2.12): Pflicht, sobald die GuV übermittelt wird. Maßgebend sind die
// steuerlichen Werte, bei einer Handelsbilanz also die übergeleiteten.
//
//	Gewinn = BV Ende − BV Vorjahr + Entnahmen/Ausschüttungen − Einlagen
//
// Bei einer GmbH ist das Betriebsvermögen das Eigenkapital. Einlagen sind der
// Zuwachs von gezeichnetem Kapital und Kapitalrücklage; was danach zwischen
// Gewinn und Vermögenszuwachs fehlt, ist ausgeschüttet worden.

const (
	bvvProfit       = "BVV.profitLoss"
	bvvCurrent      = "BVV.profitLoss.assetsCurrentYear"
	bvvPrevious     = "BVV.profitLoss.assetsPreviousYear"
	bvvPreviousBase = "BVV.profitLoss.assetsPreviousYear.assets"
	bvvContribution = "BVV.profitLoss.contribution"
	bvvWithdrawal   = "BVV.profitLoss.withdrawalDistrib"
	equityTotal     = "bs.eqLiab.equity"
	equityDeficit   = "bs.ass.deficitNotCoveredByCapital"
)

// assetComparison rechnet den BVV aus dem steuerlichen Datensatz und setzt
// seine Werte in den übermittelten.
func (g *generator) assetComparison(st *dataset, netIncome domain.Cents) {
	equity := st.value(equityTotal) - st.value(equityDeficit)
	paidIn := st.value("bs.eqLiab.equity.subscribed") + st.value("bs.eqLiab.equity.capRes")

	var prior domain.Cents
	contributions := domain.Cents(0)
	switch {
	case g.in.PriorEquity != nil:
		prior = *g.in.PriorEquity
		priorPaidIn := domain.Cents(0)
		if g.in.PriorPaidIn != nil {
			priorPaidIn = *g.in.PriorPaidIn
		}
		if paidIn > priorPaidIn {
			contributions = paidIn - priorPaidIn
		}
	default:
		// Ohne Vorjahreswert bleibt nur die Annahme, dass im Jahr weder
		// eingelegt noch ausgeschüttet wurde.
		prior = equity - netIncome
		g.warn("bvv-vorjahr", "Das Eigenkapital zum Ende des Vorjahres ist nicht bekannt; der "+
			"Betriebsvermögensvergleich setzt es als Eigenkapital abzüglich Jahresergebnis an und "+
			"nimmt keine Einlagen und Ausschüttungen an.", bvvPreviousBase)
	}
	residual := equity - prior - netIncome
	distributions := contributions - residual
	if distributions < 0 {
		contributions -= distributions
		distributions = 0
	}

	ds := g.hb
	ds.set(bvvCurrent, equity)
	ds.set(bvvPreviousBase, prior)
	if contributions != 0 {
		ds.set(bvvContribution, contributions)
	}
	if distributions != 0 {
		ds.set(bvvWithdrawal, distributions)
	}
	ds.rollUp(RoleTaxAssets, bvvProfit)
}
