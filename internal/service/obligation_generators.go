package service

import "github.com/buchfink/buchfink/internal/domain"

type obligationAxis uint8

const (
	obligationCalendarYear obligationAxis = iota
	obligationFiscalYear
	obligationOnce
)

type obligationPeriod struct {
	year int
	fy   domain.FiscalYear
}

type obligationGenerator struct {
	kinds    []domain.ObligationKind
	axis     obligationAxis
	generate func(obligationPeriod) error
}

func (g obligationGenerator) selected(want func(domain.ObligationKind) bool) bool {
	for _, kind := range g.kinds {
		if want(kind) {
			return true
		}
	}
	return false
}

func (r *obligationRun) generators(
	want func(domain.ObligationKind) bool, add func(domain.Obligation, *domain.ObligationEvidence),
) []obligationGenerator {
	calendarYear := func(kind domain.ObligationKind, generate func(int, func(domain.Obligation, *domain.ObligationEvidence)) error) obligationGenerator {
		return obligationGenerator{
			kinds: []domain.ObligationKind{kind}, axis: obligationCalendarYear,
			generate: func(period obligationPeriod) error { return generate(period.year, add) },
		}
	}
	return []obligationGenerator{
		calendarYear(domain.ObligationVatReturn, r.vatReturns),
		calendarYear(domain.ObligationSpecialPrepayment, r.specialPrepayment),
		calendarYear(domain.ObligationECSalesList, r.ecSalesLists),
		calendarYear(domain.ObligationCorporatePrepay, r.corporatePrepayments),
		calendarYear(domain.ObligationTradePrepay, r.tradePrepayments),
		calendarYear(domain.ObligationPayrollTax, r.payrollTax),
		calendarYear(domain.ObligationCorporateTaxReturn, r.annualReturn(domain.ObligationCorporateTaxReturn, domain.DeclarationCorporateTax)),
		calendarYear(domain.ObligationTradeTaxReturn, r.annualReturn(domain.ObligationTradeTaxReturn, domain.DeclarationTradeTax)),
		calendarYear(domain.ObligationAnnualVatReturn, r.annualReturn(domain.ObligationAnnualVatReturn, domain.DeclarationAnnualVat)),
		{
			kinds: []domain.ObligationKind{domain.ObligationMonthCommit}, axis: obligationFiscalYear,
			generate: func(period obligationPeriod) error { return r.monthCommits(period.fy, add) },
		},
		{
			kinds: []domain.ObligationKind{
				domain.ObligationClosingBookings, domain.ObligationStatementPrep, domain.ObligationAdoption,
				domain.ObligationAppropriation, domain.ObligationDisclosure, domain.ObligationEBilanz,
				domain.ObligationWithholdingTax,
			},
			axis:     obligationFiscalYear,
			generate: func(period obligationPeriod) error { return r.annualStatement(period.fy, want, add) },
		},
		{
			kinds: []domain.ObligationKind{domain.ObligationTransparency, domain.ObligationShareholderList}, axis: obligationOnce,
			generate: func(obligationPeriod) error {
				r.eventObligations(want, add)
				return nil
			},
		},
		{
			kinds: []domain.ObligationKind{
				domain.ObligationTaxRegistration, domain.ObligationTradeRegistration, domain.ObligationTransparency,
				domain.ObligationOpeningBalance, domain.ObligationOpeningEBilanz,
			},
			axis: obligationOnce,
			generate: func(obligationPeriod) error {
				r.foundingObligations(want, add)
				return nil
			},
		},
	}
}

func (r *obligationRun) generatorPeriods(axis obligationAxis, from, to int) []obligationPeriod {
	var periods []obligationPeriod
	switch axis {
	case obligationCalendarYear:
		for year := from; year <= to; year++ {
			periods = append(periods, obligationPeriod{year: year})
		}
	case obligationFiscalYear:
		for _, fy := range r.fiscalYearsIn(from, to) {
			periods = append(periods, obligationPeriod{fy: fy})
		}
	case obligationOnce:
		periods = []obligationPeriod{{}}
	}
	return periods
}
