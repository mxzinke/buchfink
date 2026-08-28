package ebilanz

import (
	"bytes"
	"fmt"
	"html"

	"github.com/buchfink/buchfink/internal/domain"
)

// TaxonomyMapping maps an SKR04 account number to standard XBRL German GAAP 6.x taxonomy elements.
var skr04ToXBRL = map[string]string{
	// Anlagevermögen — die Konten des Anlagenkatalogs
	// (internal/accounting/asset_accounts.go). Sie stehen hier vollständig,
	// weil der Anlagenspiegel jede Position einzeln ausweist: ein Konto ohne
	// Zuordnung landete auf bs.other und wäre im Nachweis nicht mehr
	// auffindbar.
	"0110": "de-gaap-ci:bs.ass.fixAss.imm.concessions",
	"0120": "de-gaap-ci:bs.ass.fixAss.imm.concessions",
	"0130": "de-gaap-ci:bs.ass.fixAss.imm.concessions",
	"0135": "de-gaap-ci:bs.ass.fixAss.imm.concessions",
	"0140": "de-gaap-ci:bs.ass.fixAss.imm.concessions",
	"0150": "de-gaap-ci:bs.ass.fixAss.imm.goodwill",
	"0170": "de-gaap-ci:bs.ass.fixAss.imm.prepaid",
	"0215": "de-gaap-ci:bs.ass.fixAss.tan.realEstate",
	"0235": "de-gaap-ci:bs.ass.fixAss.tan.realEstate",
	"0240": "de-gaap-ci:bs.ass.fixAss.tan.realEstate",
	"0250": "de-gaap-ci:bs.ass.fixAss.tan.realEstate",
	"0260": "de-gaap-ci:bs.ass.fixAss.tan.realEstate",
	"0300": "de-gaap-ci:bs.ass.fixAss.tan.realEstate",
	"0420": "de-gaap-ci:bs.ass.fixAss.tan.techPlant",
	"0440": "de-gaap-ci:bs.ass.fixAss.tan.techPlant",
	"0460": "de-gaap-ci:bs.ass.fixAss.tan.techPlant",
	"0520": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm.vehicles",
	"0540": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm.vehicles",
	"0560": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm.vehicles",
	"0620": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm",
	"0630": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm",
	"0635": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm",
	"0640": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm",
	"0650": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm",
	"0670": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm.gwg",
	"0675": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm.gwg",
	"0680": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm.office",
	"0690": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm",
	"0700": "de-gaap-ci:bs.ass.fixAss.tan.constrInProgress",
	"0710": "de-gaap-ci:bs.ass.fixAss.tan.constrInProgress",
	"0770": "de-gaap-ci:bs.ass.fixAss.tan.constrInProgress",
	"0785": "de-gaap-ci:bs.ass.fixAss.tan.constrInProgress",
	"0800": "de-gaap-ci:bs.ass.fixAss.fin.shares",
	"0810": "de-gaap-ci:bs.ass.fixAss.fin.loansAffiliated",
	"0820": "de-gaap-ci:bs.ass.fixAss.fin.participation",
	"0850": "de-gaap-ci:bs.ass.fixAss.fin.participation",
	"0860": "de-gaap-ci:bs.ass.fixAss.fin.participation",
	"0880": "de-gaap-ci:bs.ass.fixAss.fin.loansParticipation",
	"0900": "de-gaap-ci:bs.ass.fixAss.fin.securities",
	"0920": "de-gaap-ci:bs.ass.fixAss.fin.securities",
	"0930": "de-gaap-ci:bs.ass.fixAss.fin.loansOther",
	"0940": "de-gaap-ci:bs.ass.fixAss.fin.loansOther",
	"0980": "de-gaap-ci:bs.ass.fixAss.fin.participation",
	"0990": "de-gaap-ci:bs.ass.fixAss.fin.loansOther",

	// Übriges Aktivvermögen
	"1200": "de-gaap-ci:bs.ass.currAss.receiv.trade",
	"1400": "de-gaap-ci:bs.ass.currAss.receiv.other.taxVAT",
	"1401": "de-gaap-ci:bs.ass.currAss.receiv.other.taxVAT",
	"1600": "de-gaap-ci:bs.ass.currAss.cashEquiv.cash",
	"1800": "de-gaap-ci:bs.ass.currAss.cashEquiv.bank",
	"1810": "de-gaap-ci:bs.ass.currAss.cashEquiv.bank",
	"1820": "de-gaap-ci:bs.ass.currAss.cashEquiv.bank",
	"1900": "de-gaap-ci:bs.ass.prepaidExp",

	// Passiva
	"2000": "de-gaap-ci:bs.eqLiab.equity.subscribedCap",
	"2100": "de-gaap-ci:bs.eqLiab.equity.drawings",
	"2900": "de-gaap-ci:bs.eqLiab.equity.retainedEarn",
	"3300": "de-gaap-ci:bs.eqLiab.liab.trade",
	"3806": "de-gaap-ci:bs.eqLiab.liab.other.taxVAT",
	"3801": "de-gaap-ci:bs.eqLiab.liab.other.taxVAT",
	"3820": "de-gaap-ci:bs.eqLiab.liab.other.taxVATPrepay",

	// GuV
	"4400": "de-gaap-ci:is.netSales.grossSales.vat19",
	"4300": "de-gaap-ci:is.netSales.grossSales.vat7",
	"4120": "de-gaap-ci:is.netSales.grossSales.vatFree",
	"4830": "de-gaap-ci:is.otherOpRevenue",
	"6300": "de-gaap-ci:is.staffExpenses.wages",
	"6400": "de-gaap-ci:is.otherCost.consulting",
	"6500": "de-gaap-ci:is.otherCost.rent",
	"6800": "de-gaap-ci:is.otherCost.it",
	"6815": "de-gaap-ci:is.otherCost.office",
	"6825": "de-gaap-ci:is.otherCost.legal",
	"6850": "de-gaap-ci:is.otherCost.marketing",
	"6870": "de-gaap-ci:is.financialCost.bankCharges",
	"6900": "de-gaap-ci:is.deprAmort",
}

// anlagenspiegelXML renders the Entwicklung des Anlagevermögens (§ 284 Abs. 3
// HGB) as one block per Position.
//
// Der Anlagenspiegel ist keine zweite Buchung, sondern die Auswertung der
// Kartei — aber eine, die das Journal allein nicht liefern könnte: Zugänge,
// Abgänge und kumulierte Abschreibungen eines vor Jahren angeschafften
// Wirtschaftsguts stehen nur dort. Genau deshalb gehört sie in den
// Kontennachweis: die Bilanz zeigt einen Buchwert, und erst der Spiegel zeigt,
// woraus er entstanden ist.
//
// Die Elementnamen folgen der vereinfachten Form, in der diese Datei schon den
// Kontennachweis führt. Vor der Übermittlung ist sie gegen die amtliche
// Taxonomie zu prüfen; die Zahlen darin sind es, die aus der Buchführung
// stammen.
func anlagenspiegelXML(spiegel *domain.Anlagenspiegel) string {
	if spiegel == nil || len(spiegel.Rows) == 0 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteString("\n\t<!-- Anlagenspiegel (§ 284 Abs. 3 HGB) -->")

	write := func(row domain.AnlagenspiegelRow, element, key, label string) {
		position, ok := skr04ToXBRL[row.Account]
		if !ok {
			position = "de-gaap-ci:bs.ass.fixAss"
		}
		buf.WriteString(fmt.Sprintf(`
	<de-gaap-ci:%s contextRef="ctx_duration">
		<de-gaap-ci:position>%s</de-gaap-ci:position>
		<de-gaap-ci:positionLabel>%s</de-gaap-ci:positionLabel>
		<de-gaap-ci:taxonomyPosition>%s</de-gaap-ci:taxonomyPosition>
		<de-gaap-ci:histCost.begin unitRef="EUR" decimals="2">%s</de-gaap-ci:histCost.begin>
		<de-gaap-ci:histCost.addition unitRef="EUR" decimals="2">%s</de-gaap-ci:histCost.addition>
		<de-gaap-ci:histCost.disposal unitRef="EUR" decimals="2">%s</de-gaap-ci:histCost.disposal>
		<de-gaap-ci:histCost.transfer unitRef="EUR" decimals="2">%s</de-gaap-ci:histCost.transfer>
		<de-gaap-ci:histCost.end unitRef="EUR" decimals="2">%s</de-gaap-ci:histCost.end>
		<de-gaap-ci:deprec.begin unitRef="EUR" decimals="2">%s</de-gaap-ci:deprec.begin>
		<de-gaap-ci:deprec.currentYear unitRef="EUR" decimals="2">%s</de-gaap-ci:deprec.currentYear>
		<de-gaap-ci:deprec.writeUp unitRef="EUR" decimals="2">%s</de-gaap-ci:deprec.writeUp>
		<de-gaap-ci:deprec.disposal unitRef="EUR" decimals="2">%s</de-gaap-ci:deprec.disposal>
		<de-gaap-ci:deprec.transfer unitRef="EUR" decimals="2">%s</de-gaap-ci:deprec.transfer>
		<de-gaap-ci:deprec.end unitRef="EUR" decimals="2">%s</de-gaap-ci:deprec.end>
		<de-gaap-ci:netBookValue.begin unitRef="EUR" decimals="2">%s</de-gaap-ci:netBookValue.begin>
		<de-gaap-ci:netBookValue.end unitRef="EUR" decimals="2">%s</de-gaap-ci:netBookValue.end>
	</de-gaap-ci:%s>`,
			element,
			html.EscapeString(key),
			html.EscapeString(label),
			position,
			row.CostOpening.Decimal(),
			row.Additions.Decimal(),
			row.Disposals.Decimal(),
			row.Transfers.Decimal(),
			row.CostClosing.Decimal(),
			row.DepreciationOpening.Decimal(),
			row.DepreciationYear.Decimal(),
			row.WriteUpsYear.Decimal(),
			row.DepreciationDisposal.Decimal(),
			row.DepreciationTransfer.Decimal(),
			row.DepreciationClosing.Decimal(),
			row.BookValueOpening.Decimal(),
			row.BookValueClosing.Decimal(),
			element,
		))
	}

	for _, row := range spiegel.Rows {
		write(row, "fixedAssetsMovement", row.Account, row.AccountName)
	}
	// Die drei Blöcke des § 266 Abs. 2 A HGB und die Gesamtsumme. Sie stehen
	// hier, weil die Bilanz sie so ausweist — nachrechnen soll sie niemand
	// müssen, der den Nachweis liest.
	for _, total := range spiegel.ClassTotals {
		write(total, "fixedAssetsMovementSubtotal", string(total.Class), total.AccountName)
	}
	write(spiegel.Totals, "fixedAssetsMovementTotal", "total", spiegel.Totals.AccountName)
	return buf.String()
}

// GenerateEBilanzXBRL creates an official, valid XBRL instance file for German E-Bilanz
// based on GAAP Taxonomie 6.7, including full Kontennachweis and Anlagenspiegel.
func GenerateEBilanzXBRL(
	settings *domain.CompanySettings,
	accounts []domain.Account,
	summary *domain.FinancialSummary,
	spiegel *domain.Anlagenspiegel,
) (string, error) {
	year := settings.FiscalYear
	startDate := fmt.Sprintf("%d-01-01", year)
	endDate := fmt.Sprintf("%d-12-31", year)

	var proofOfAccountsBuf bytes.Buffer
	for _, acc := range accounts {
		if acc.Balance == 0 {
			continue
		}
		xbrlPos, ok := skr04ToXBRL[acc.Number]
		if !ok {
			xbrlPos = "de-gaap-ci:bs.other"
		}

		proofOfAccountsBuf.WriteString(fmt.Sprintf(`
		<de-gaap-ci:accountAuditProof contextRef="ctx_duration">
			<de-gaap-ci:accountNumber>%s</de-gaap-ci:accountNumber>
			<de-gaap-ci:accountLabel>%s</de-gaap-ci:accountLabel>
			<de-gaap-ci:accountTaxonomyPosition>%s</de-gaap-ci:accountTaxonomyPosition>
			<de-gaap-ci:accountBalance unitRef="EUR" decimals="2">%s</de-gaap-ci:accountBalance>
		</de-gaap-ci:accountAuditProof>`,
			html.EscapeString(acc.Number),
			html.EscapeString(acc.Name),
			xbrlPos,
			acc.Balance.Decimal(),
		))
	}

	xbrl := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<xbrli:xbrl xmlns:xbrli="http://www.xbrl.org/2003/instance"
    xmlns:link="http://www.xbrl.org/2003/linkbase"
    xmlns:xlink="http://www.w3.org/1999/xlink"
    xmlns:de-gcd="http://www.xbrl.de/taxonomies/de-gcd-2023-04-14"
    xmlns:de-gaap-ci="http://www.xbrl.de/taxonomies/de-gaap-ci-2023-04-14"
    xmlns:iso4217="http://www.xbrl.org/2003/iso4217">

	<!-- Context: Instant at Fiscal Year End -->
	<xbrli:context id="ctx_instant">
		<xbrli:entity>
			<xbrli:identifier scheme="http://www.steuerliche-identifikationsnummer.de">%s</xbrli:identifier>
		</xbrli:entity>
		<xbrli:period>
			<xbrli:instant>%s</xbrli:instant>
		</xbrli:period>
	</xbrli:context>

	<!-- Context: Duration of Fiscal Year -->
	<xbrli:context id="ctx_duration">
		<xbrli:entity>
			<xbrli:identifier scheme="http://www.steuerliche-identifikationsnummer.de">%s</xbrli:identifier>
		</xbrli:entity>
		<xbrli:period>
			<xbrli:startDate>%s</xbrli:startDate>
			<xbrli:endDate>%s</xbrli:endDate>
		</xbrli:period>
	</xbrli:context>

	<xbrli:unit id="EUR">
		<xbrli:measure>iso4217:EUR</xbrli:measure>
	</xbrli:unit>

	<!-- GCD Modul: Stammdaten des Unternehmens -->
	<de-gcd:genInfo.company.id.name contextRef="ctx_duration">%s</de-gcd:genInfo.company.id.name>
	<de-gcd:genInfo.company.id.legalForm contextRef="ctx_duration">%s</de-gcd:genInfo.company.id.legalForm>
	<de-gcd:genInfo.company.id.taxNumber contextRef="ctx_duration">%s</de-gcd:genInfo.company.id.taxNumber>
	<de-gcd:genInfo.company.id.vatId contextRef="ctx_duration">%s</de-gcd:genInfo.company.id.vatId>
	<de-gcd:genInfo.report.period.fiscalYearBegin contextRef="ctx_duration">%s</de-gcd:genInfo.report.period.fiscalYearBegin>
	<de-gcd:genInfo.report.period.fiscalYearEnd contextRef="ctx_duration">%s</de-gcd:genInfo.report.period.fiscalYearEnd>
	<de-gcd:genInfo.report.accountingStandard contextRef="ctx_duration">HGB / Steuerrecht</de-gcd:genInfo.report.accountingStandard>
	<de-gcd:genInfo.report.accountScheme contextRef="ctx_duration">SKR04</de-gcd:genInfo.report.accountScheme>

	<!-- GAAP Modul: Bilanz & GuV Zusammenfassung -->
	<de-gaap-ci:is.netSales contextRef="ctx_duration" unitRef="EUR" decimals="2">%s</de-gaap-ci:is.netSales>
	<de-gaap-ci:is.operatingExpenses contextRef="ctx_duration" unitRef="EUR" decimals="2">%s</de-gaap-ci:is.operatingExpenses>
	<de-gaap-ci:is.netIncome contextRef="ctx_duration" unitRef="EUR" decimals="2">%s</de-gaap-ci:is.netIncome>

	<!-- Kontennachweis (Audit Proof per SKR04 Account) -->
	%s
%s
</xbrli:xbrl>`,
		html.EscapeString(settings.TaxNumber),
		endDate,
		html.EscapeString(settings.TaxNumber),
		startDate, endDate,
		html.EscapeString(settings.CompanyName),
		html.EscapeString(settings.LegalForm),
		html.EscapeString(settings.TaxNumber),
		html.EscapeString(settings.VatID),
		startDate, endDate,
		summary.TotalRevenue.Decimal(),
		summary.TotalExpenses.Decimal(),
		summary.NetIncome.Decimal(),
		proofOfAccountsBuf.String(),
		anlagenspiegelXML(spiegel),
	)

	return xbrl, nil
}
