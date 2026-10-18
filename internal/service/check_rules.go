package service

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
)

type checkScope struct {
	year          int
	cutoff        string
	reference     string
	cfg           *domain.CompanySettings
	entries       []domain.JournalEntry
	receipts      []domain.Receipt
	withReceipt   map[uint]bool
	checkedBankTx int
}

type checkRule struct {
	name     string
	yearOnly bool
	check    func(context.Context, *checkScope) ([]domain.CheckFinding, error)
}

func (s *CheckService) checkRules() []checkRule {
	return []checkRule{
		{name: "entries_without_receipt", check: func(_ context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkEntriesWithoutReceipt(scope.entries, scope.withReceipt), nil
		}},
		{name: "duplicate_documents", check: func(_ context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkDuplicateDocuments(scope.entries), nil
		}},
		{name: "interim_balances", check: func(_ context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkInterimBalances(scope.entries, scope.cutoff), nil
		}},
		{name: "receipts", check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkReceipts(scope.receipts, scope.cutoff, scope.reference, scope.cfg, s.pendingByDesign(ctx, scope.year)), nil
		}},
		{name: "bank", check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			findings, count := s.checkBank(ctx, scope.year, scope.cutoff)
			scope.checkedBankTx = count
			return findings, nil
		}},
		{name: "number_gaps", check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkNumberGaps(ctx, scope.year), nil
		}},
		{name: "overpaid_items", check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkOverpaidItems(ctx, scope.cutoff), nil
		}},
		{name: "vat_returns", check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkVatReturns(ctx, scope.year, scope.cutoff, scope.reference, scope.cfg), nil
		}},
		{name: "commit_overdue", check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkCommitOverdue(ctx, scope.year, scope.cutoff, scope.cfg), nil
		}},
		{name: "periods_not_committed", check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkPeriodsNotCommitted(ctx, scope.year, scope.cutoff, scope.cfg), nil
		}},
		{name: "unconfirmed_vat_ids", check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkUnconfirmedVatIDs(ctx, scope.year), nil
		}},
		{name: "service_proof", check: func(_ context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkServiceProof(scope.receipts, scope.cutoff, scope.cfg), nil
		}},
		{name: "finding_sources", check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			var findings []domain.CheckFinding
			for _, src := range s.domains {
				extra, err := src.Findings(ctx, scope.cutoff)
				if err != nil {
					return nil, err
				}
				findings = append(findings, extra...)
			}
			return findings, nil
		}},
		{name: "account_mapping", yearOnly: true, check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkAccountMapping(ctx, scope.year), nil
		}},
		{name: "depreciation", yearOnly: true, check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkDepreciation(ctx, scope.year), nil
		}},
		{name: "provision_discounting", yearOnly: true, check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkProvisionDiscounting(ctx, scope.year), nil
		}},
		{name: "skipped_closing_steps", yearOnly: true, check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkSkippedClosingSteps(ctx, scope.year), nil
		}},
		{name: "size_class_change", yearOnly: true, check: func(ctx context.Context, scope *checkScope) ([]domain.CheckFinding, error) {
			return s.checkSizeClassChange(ctx, scope.year), nil
		}},
	}
}
