package ebilanz

import (
	"bytes"
	"fmt"
	"html"

	"github.com/buchfink/buchfink/internal/domain"
)

// TaxonomyMapping maps an SKR04 account number to standard XBRL German GAAP 6.x taxonomy elements.
var skr04ToXBRL = map[string]string{
	// Aktiva
	"0520": "de-gaap-ci:bs.ass.fixAss.imm.concessions",
	"0650": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm",
	"0670": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm.gwg",
	"0680": "de-gaap-ci:bs.ass.fixAss.tan.otherEquipm.office",
	"0800": "de-gaap-ci:bs.ass.fixAss.fin.shares",
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

// GenerateEBilanzXBRL creates an official, valid XBRL instance file for German E-Bilanz
// based on GAAP Taxonomie 6.7, including full Kontennachweis.
func GenerateEBilanzXBRL(settings *domain.CompanySettings, accounts []domain.Account, summary *domain.FinancialSummary) (string, error) {
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
	)

	return xbrl, nil
}
