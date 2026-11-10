package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// AssetParametersForYear liefert die Wertgrenzen zum ersten Januar des Anzeigejahres.
// Die Anzeige steht auch ohne geöffneten Mandanten zur Verfügung.
func AssetParametersForYear(year int) (accounting.AfAParameters, error) {
	return accounting.AfAParametersFor(fmt.Sprintf("%d-01-01", year))
}

// InvestmentTaxRules beschreibt die Anlegerstellung und die Freistellung je Fondsart.
type InvestmentTaxRules struct {
	InvestorType domain.InvestorType
	Reason       string
	LegalForm    string
	Exemptions   []FundExemption
}

type FundExemption struct {
	Class                        accounting.FundClass
	Permille                     int
	Source, Explanation, Problem string
}

func (s *AssetService) InvestmentTaxRules(ctx context.Context) (*InvestmentTaxRules, error) {
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil {
		return nil, err
	}
	investor, reason := cfg.InvestorTypeOrDerived()
	rules := &InvestmentTaxRules{InvestorType: investor, Reason: reason, LegalForm: cfg.LegalForm, Exemptions: []FundExemption{}}
	for _, class := range accounting.AllFundClasses() {
		if !class.IsFund() {
			continue
		}
		exemption, err := accounting.PartialExemptionFor(class, investor)
		info := FundExemption{Class: class}
		if err != nil {
			info.Problem = err.Error()
		} else {
			info.Permille, info.Source, info.Explanation = exemption.Permille, exemption.Source, exemption.Explanation
		}
		rules.Exemptions = append(rules.Exemptions, info)
	}
	return rules, nil
}
