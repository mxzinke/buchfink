package service

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Jede Pflicht führt in einen Reiter, den die Oberfläche kennt, und nimmt ihren
// Schlüssel als Kennung mit.
func TestObligationTargetsAreValidRoutes(t *testing.T) {
	cases := map[domain.ObligationKind]domain.TaskTarget{
		domain.ObligationVatReturn:          {Area: domain.AreaClosings, Tab: "month"},
		domain.ObligationSpecialPrepayment:  {Area: domain.AreaClosings, Tab: "month"},
		domain.ObligationECSalesList:        {Area: domain.AreaClosings, Tab: "month"},
		domain.ObligationPayrollTax:         {Area: domain.AreaClosings, Tab: "month"},
		domain.ObligationMonthCommit:        {Area: domain.AreaClosings, Tab: "month"},
		domain.ObligationCorporatePrepay:    {Area: domain.AreaClosings, Tab: "declarations"},
		domain.ObligationTradePrepay:        {Area: domain.AreaClosings, Tab: "declarations"},
		domain.ObligationEBilanz:            {Area: domain.AreaClosings, Tab: "declarations"},
		domain.ObligationCorporateTaxReturn: {Area: domain.AreaClosings, Tab: "declarations"},
		domain.ObligationTradeTaxReturn:     {Area: domain.AreaClosings, Tab: "declarations"},
		domain.ObligationAnnualVatReturn:    {Area: domain.AreaClosings, Tab: "declarations"},
		domain.ObligationClosingBookings:    {Area: domain.AreaClosings, Tab: "year"},
		domain.ObligationStatementPrep:      {Area: domain.AreaClosings, Tab: "year"},
		domain.ObligationAdoption:           {Area: domain.AreaClosings, Tab: "year"},
		domain.ObligationAppropriation:      {Area: domain.AreaClosings, Tab: "year"},
		domain.ObligationDisclosure:         {Area: domain.AreaClosings, Tab: "year"},
		domain.ObligationWithholdingTax:     {Area: domain.AreaClosings, Tab: "year"},
		domain.ObligationTransparency:       {Area: domain.AreaCompany, Tab: "profile"},
		domain.ObligationShareholderList:    {Area: domain.AreaCompany, Tab: "profile"},
		domain.ObligationTaxRegistration:    {Area: domain.AreaCompany, Tab: "profile"},
		domain.ObligationTradeRegistration:  {Area: domain.AreaCompany, Tab: "profile"},
	}
	for kind, want := range cases {
		key := string(kind) + ".2026"
		got := obligationTarget(domain.Obligation{Kind: kind, Key: key})
		want.ID = key
		if got != want {
			t.Errorf("%s: Ziel %+v, erwartet %+v", kind, got, want)
		}
		if !got.Valid() {
			t.Errorf("%s: Ziel %+v kennt die Oberfläche nicht", kind, got)
		}
	}
}

func TestTaskTargetValid(t *testing.T) {
	valid := []domain.TaskTarget{
		{Area: domain.AreaOverview},
		{Area: domain.AreaBooks},
		{Area: domain.AreaBooks, Tab: "journal", ID: "B-2026-0001"},
		{Area: domain.AreaSettings, Tab: "audit", ID: "receipt_unbooked"},
	}
	for _, target := range valid {
		if !target.Valid() {
			t.Errorf("%+v sollte gültig sein", target)
		}
	}
	invalid := []domain.TaskTarget{
		{},
		{Area: "receipts"},
		{Area: domain.AreaBooks, Tab: "month"},
		{Area: domain.AreaOverview, Tab: "receipts"},
	}
	for _, target := range invalid {
		if target.Valid() {
			t.Errorf("%+v sollte ungültig sein", target)
		}
	}
}
