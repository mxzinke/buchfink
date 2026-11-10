package service

import (
	"context"
	"errors"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

type auditContractRepo struct {
	domain.AuditRepository
	ctx                 context.Context
	action              domain.AuditAction
	entity, id, details string
	before, after       any
	err                 error
	calls               int
}

func (r *auditContractRepo) Log(ctx context.Context, action domain.AuditAction, entity, id, details string) error {
	r.ctx, r.action, r.entity, r.id, r.details = ctx, action, entity, id, details
	r.calls++
	return r.err
}
func (r *auditContractRepo) LogChange(ctx context.Context, action domain.AuditAction, entity, id, details string, before, after any) error {
	r.before, r.after = before, after
	return r.Log(ctx, action, entity, id, details)
}

func TestServiceAuditContracts(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{}{}, "Transaktion")
	const details = "Änderung: 42 → 43\nGrund: Prüfung"
	for _, tc := range []struct {
		name, entity, id string
		action           domain.AuditAction
		write            func(domain.AuditRepository) error
	}{
		{"Accrual", "ACCRUAL", "42", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewAccrualService(nil, nil, nil, nil, repo, nil).audit(ctx, domain.AuditActionUpdate, 42, details)
		}},
		{"Asset", "ANLAGE", "42", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewAssetService(nil, nil, NewJournalService(nil, nil, nil, nil, nil, nil), nil, nil, nil, repo).audit(ctx, domain.AuditActionUpdate, 42, details)
		}},
		{"Closing", "FISCAL_YEAR", "2024", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewClosingService(nil, nil, nil, nil, nil, nil, nil, repo, nil).audit(ctx, domain.AuditActionUpdate, 2024, details)
		}},
		{"Journal", "JOURNAL", "42", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewJournalService(nil, nil, nil, repo, nil, nil).audit(ctx, domain.AuditActionUpdate, 42, details)
		}},
		{"Provision", "PROVISION", "42", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewProvisionService(nil, nil, nil, nil, nil, repo, nil).audit(ctx, domain.AuditActionUpdate, 42, details)
		}},
		{"ZM", "ZM_RETURN", "42", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewZMService(nil, nil, nil, nil, nil, nil, repo).audit(ctx, domain.AuditActionUpdate, 42, details)
		}},
		{"VatReturn", "VAT_RETURN", "42", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewVatReturnService(nil, nil, nil, nil, nil, nil, nil, repo).audit(ctx, domain.AuditActionUpdate, 42, details)
		}},
		{"Foundation", "GRUENDUNG", "42", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewFoundationService(nil, nil, nil, nil, nil, repo).audit(ctx, domain.AuditActionUpdate, 42, details)
		}},
		{"ClosingSteps", "CLOSING_STEP", "2024", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewClosingStepsService(nil, nil, nil, nil, nil, nil, repo, nil).audit(ctx, domain.AuditActionUpdate, 2024, details)
		}},
		{"VatID", "VAT_ID_CHECK", "42", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewVatIDService(nil, nil, nil, repo).audit(ctx, domain.AuditActionUpdate, 42, details)
		}},
		{"InputTax", "INPUT_TAX_CORRECTION", "42", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewInputTaxService(nil, nil, nil, nil, nil, repo).audit(ctx, domain.AuditActionUpdate, 42, details)
		}},
		{"Gift", "GIFT", "", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewGiftService(nil, nil, nil, repo).audit(ctx, domain.AuditActionUpdate, "", details)
		}},
		{"Currency", "CURRENCY", "", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewCurrencyService(nil, nil, repo).audit(ctx, domain.AuditActionUpdate, "", details)
		}},
		{"ClosingBooking", "VAT_SETTLEMENT", "2024", domain.AuditActionCreate, func(repo domain.AuditRepository) error {
			return NewClosingBookingService(nil, nil, nil, nil, repo, nil).auditEntity(ctx, domain.AuditActionCreate, "VAT_SETTLEMENT", 2024, details)
		}},
		{"Appropriation", "APPROPRIATION", "2024", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewAppropriationService(nil, nil, nil, nil, nil, repo, nil).auditEntity(ctx, domain.AuditActionUpdate, "APPROPRIATION", 2024, details)
		}},
		{"Anhang", "NOTES", "2024", domain.AuditActionUpdate, func(repo domain.AuditRepository) error {
			return NewAppropriationService(nil, nil, nil, nil, nil, repo, nil).auditEntity(ctx, domain.AuditActionUpdate, "NOTES", 2024, details)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, fail := range []bool{false, true} {
				repo := &auditContractRepo{}
				if fail {
					repo.err = errors.New("Protokoll gesperrt")
				}
				err := tc.write(repo)
				if !errors.Is(err, repo.err) {
					t.Fatalf("Fehler: %v, erwartet %v", err, repo.err)
				}
				if repo.calls != 1 || repo.ctx != ctx || repo.action != tc.action || repo.entity != tc.entity || repo.id != tc.id || repo.details != details {
					t.Fatalf("Protokolleintrag weicht ab: %+v", repo)
				}
			}
			if err := tc.write(nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestServiceAuditChangeContracts(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, entity, id string
		write            func(domain.AuditRepository, bool) error
	}{
		{"Asset", "ANLAGE", "42", func(repo domain.AuditRepository, withBefore bool) error {
			var before *domain.FixedAsset
			if withBefore {
				before = &domain.FixedAsset{}
			}
			return NewAssetService(nil, nil, NewJournalService(nil, nil, nil, nil, nil, nil), nil, nil, nil, repo).auditChange(ctx, domain.AuditActionCreate, 42, "angelegt", before, &domain.FixedAsset{})
		}},
		{"Closing", "FISCAL_YEAR", "2024", func(repo domain.AuditRepository, withBefore bool) error {
			var before *domain.FiscalYear
			if withBefore {
				before = &domain.FiscalYear{}
			}
			return NewClosingService(nil, nil, nil, nil, nil, nil, nil, repo, nil).auditChange(ctx, domain.AuditActionCreate, 2024, "angelegt", before, &domain.FiscalYear{})
		}},
		{"VatReturn", "VAT_RETURN", "42", func(repo domain.AuditRepository, withBefore bool) error {
			var before *domain.VatReturn
			if withBefore {
				before = &domain.VatReturn{}
			}
			return NewVatReturnService(nil, nil, nil, nil, nil, nil, nil, repo).auditChange(ctx, domain.AuditActionCreate, 42, "angelegt", before, &domain.VatReturn{})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, withBefore := range []bool{false, true} {
				repo := &auditContractRepo{}
				if err := tc.write(repo, withBefore); err != nil {
					t.Fatal(err)
				}
				if repo.calls != 1 || repo.ctx != ctx || repo.action != domain.AuditActionCreate || repo.entity != tc.entity || repo.id != tc.id || repo.details != "angelegt" || (repo.before != nil) != withBefore || repo.after == nil {
					t.Fatalf("Änderungsprotokoll weicht ab: %+v", repo)
				}
				repo.err = errors.New("Protokoll gesperrt")
				if err := tc.write(repo, withBefore); !errors.Is(err, repo.err) {
					t.Fatalf("Fehler nicht weitergereicht: %v", err)
				}
			}
			if err := tc.write(nil, false); err != nil {
				t.Fatal(err)
			}
		})
	}
}
