package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

func missingResult[T any](value *T, err error) (bool, error) { return value == nil, err }

func TestRepositoryMissingRecords(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = CloseDB(db) })
	cases := []struct {
		name      string
		query     func(context.Context) (bool, error)
		wantError bool
	}{
		{"Account.FindByNumber", func(ctx context.Context) (bool, error) {
			return missingResult(NewAccountRepository(db).FindByNumber(ctx, "9999"))
		}, true},
		{"Contact.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewContactRepository(db).FindByID(ctx, 999))
		}, true},
		{"Bank.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewBankRepository(db).FindByID(ctx, 999))
		}, true},
		{"Invoice.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewInvoiceRepository(db).FindByID(ctx, 999))
		}, true},
		{"Invoice.FindByNumber", func(ctx context.Context) (bool, error) {
			return missingResult(NewInvoiceRepository(db).FindByNumber(ctx, "fehlt"))
		}, true},
		{"Journal.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewJournalRepository(db).FindByID(ctx, 999))
		}, true},
		{"Receipt.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewReceiptRepository(db).FindByID(ctx, 999))
		}, true},
		{"Receipt.FindByJournalEntry", func(ctx context.Context) (bool, error) {
			return missingResult(NewReceiptRepository(db).FindByJournalEntry(ctx, 999))
		}, true},
		{"Asset.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewAssetRepository(db).FindByID(ctx, 999))
		}, true},
		{"CheckRun.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewCheckRunRepository(db).FindByID(ctx, 999))
		}, true},
		{"Festschreibung.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewFestschreibungRepository(db).FindByID(ctx, 999))
		}, true},
		{"InputTaxCorrection.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewInputTaxCorrectionRepository(db).FindByID(ctx, 999))
		}, true},
		{"InvoiceGroup.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewInvoiceGroupRepository(db).FindByID(ctx, 999))
		}, true},
		{"ShareholderLoan.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewShareholderLoanRepository(db).FindByID(ctx, 999))
		}, true},
		{"VatReturn.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewVatReturnRepository(db).FindByID(ctx, 999))
		}, true},
		{"ZMReturn.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewZMReturnRepository(db).FindByID(ctx, 999))
		}, true},
		{"VendorAdvance.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewVendorAdvanceRepository(db).FindByID(ctx, 999))
		}, true},
		{"Accrual.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewAccrualRepository(db).FindByID(ctx, 999))
		}, false},
		{"Provision.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewProvisionRepository(db).FindByID(ctx, 999))
		}, false},
		{"Appropriation.FindByYear", func(ctx context.Context) (bool, error) {
			return missingResult(NewAppropriationRepository(db).FindByYear(ctx, 2099))
		}, false},
		{"FiscalYear.FindByYear", func(ctx context.Context) (bool, error) {
			return missingResult(NewFiscalYearRepository(db).FindByYear(ctx, 2099))
		}, false},
		{"Contact.FindByLedgerAccount", func(ctx context.Context) (bool, error) {
			return missingResult(NewContactRepository(db).FindByLedgerAccount(ctx, "99999"))
		}, false},
		{"Asset.FindPool", func(ctx context.Context) (bool, error) {
			return missingResult(NewAssetRepository(db).FindPool(ctx, 2099))
		}, false},
		{"Asset.FindByAcquisitionEntry", func(ctx context.Context) (bool, error) {
			return missingResult(NewAssetRepository(db).FindByAcquisitionEntry(ctx, 999))
		}, false},
		{"CheckRun.Latest", func(ctx context.Context) (bool, error) {
			return missingResult(NewCheckRunRepository(db).Latest(ctx, 2099))
		}, false},
		{"Foundation.Get", func(ctx context.Context) (bool, error) { return missingResult(NewFoundationRepository(db).Get(ctx)) }, false},
		{"BackupRun.LatestSuccessful", func(ctx context.Context) (bool, error) {
			return missingResult(NewBackupRunRepository(db).LatestSuccessful(ctx))
		}, false},
		{"ProcedureDocumentation.FindByID", func(ctx context.Context) (bool, error) {
			return missingResult(NewProcedureDocumentationRepository(db).FindByID(ctx, 999))
		}, false},
		{"BankRule.FindByPattern", func(ctx context.Context) (bool, error) {
			return missingResult(NewBankRuleRepository(db).FindByPattern(ctx, "fehlt"))
		}, false},
		{"Obligation.FindRecord", func(ctx context.Context) (bool, error) {
			return missingResult(NewObligationRepository(db).FindRecord(ctx, "fehlt"))
		}, false},
		{"Declaration.FindInput", func(ctx context.Context) (bool, error) {
			return missingResult(NewDeclarationRepository(db).FindInput(ctx, "KST", 2099, "fehlt"))
		}, false},
		{"Examiner.Latest", func(ctx context.Context) (bool, error) { return missingResult(NewExaminerRepository(db).Latest(ctx)) }, false},
		{"Idempotency.Find", func(ctx context.Context) (bool, error) {
			return missingResult(NewIdempotencyRepository(db).Find(ctx, "client", "fehlt"))
		}, false},
		{"InputTaxCorrection.FindByAsset", func(ctx context.Context) (bool, error) {
			return missingResult(NewInputTaxCorrectionRepository(db).FindByAsset(ctx, 999))
		}, false},
		{"InvoiceGroup.FindAdvanceByInvoice", func(ctx context.Context) (bool, error) {
			return missingResult(NewInvoiceGroupRepository(db).FindAdvanceByInvoice(ctx, 999))
		}, false},
		{"Journal.FindReversalOf", func(ctx context.Context) (bool, error) {
			return missingResult(NewJournalRepository(db).FindReversalOf(ctx, 999))
		}, false},
		{"Journal.FindCorrectionOf", func(ctx context.Context) (bool, error) {
			return missingResult(NewJournalRepository(db).FindCorrectionOf(ctx, 999))
		}, false},
		{"Journal.GetLastEntry", func(ctx context.Context) (bool, error) {
			return missingResult(NewJournalRepository(db).GetLastEntry(ctx, 2099))
		}, false},
		{"Receipt.FindByOriginalHash", func(ctx context.Context) (bool, error) {
			return missingResult(NewReceiptRepository(db).FindByOriginalHash(ctx, "fehlt"))
		}, false},
		{"ExchangeRate.FindRate", func(ctx context.Context) (bool, error) {
			return missingResult(NewExchangeRateRepository(db).FindRate(ctx, "USD", "2099-01-01"))
		}, false},
		{"ExchangeRate.FindRateOnOrBefore", func(ctx context.Context) (bool, error) {
			return missingResult(NewExchangeRateRepository(db).FindRateOnOrBefore(ctx, "USD", "2099-01-01"))
		}, false},
		{"ExchangeRate.FindVatRate", func(ctx context.Context) (bool, error) {
			return missingResult(NewExchangeRateRepository(db).FindVatRate(ctx, "USD", "2099-01"))
		}, false},
		{"VatIDCheck.FindLatestValid", func(ctx context.Context) (bool, error) {
			return missingResult(NewVatIDCheckRepository(db).FindLatestValid(ctx, 999, "DE123456789"))
		}, false},
		{"Settings.Get", func(ctx context.Context) (bool, error) {
			value, err := NewSettingsRepository(db).Get(ctx, "fehlt")
			return value == "", err
		}, true},
		{"Festschreibung.LatestCutoff", func(ctx context.Context) (bool, error) {
			value, err := NewFestschreibungRepository(db).LatestCutoff(ctx, 2099)
			return value == "", err
		}, false},
		{"NumberRange.Peek", func(ctx context.Context) (bool, error) {
			value, err := NewNumberRangeRepository(db).Peek(ctx, domain.NumberRangeReceipt, 2099)
			return value == 1, err
		}, false},
		{"Receipt.SaveHeader", func(ctx context.Context) (bool, error) {
			return missingResult(NewReceiptRepository(db).SaveHeader(ctx, 999, domain.ReceiptHeader{}, nil))
		}, true},
		{"Receipt.ReplaceFiles", func(ctx context.Context) (bool, error) {
			return missingResult(NewReceiptRepository(db).ReplaceFiles(ctx, 999, nil, nil))
		}, true},
		{"Receipt.Seal", func(ctx context.Context) (bool, error) {
			return true, NewReceiptRepository(db).Seal(ctx, 999, 1)
		}, true},
		{"Receipt.Discard", func(ctx context.Context) (bool, error) {
			return true, NewReceiptRepository(db).Discard(ctx, 999, "fehlt")
		}, true},
	}
	for _, inTx := range []bool{false, true} {
		name := "direkt"
		if inTx {
			name = "transaktion"
		}
		t.Run(name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					check := func(ctx context.Context) error {
						empty, err := tc.query(ctx)
						if !empty {
							t.Error("fehlender Datensatz liefert keinen leeren Wert")
						}
						if tc.wantError {
							if !errors.Is(err, gorm.ErrRecordNotFound) {
								t.Fatalf("Fehler = %v, erwartet: nicht gefunden", err)
							}
							if err.Error() != "record not found" {
								t.Errorf("Fehlertext = %q", err.Error())
							}
						} else if err != nil {
							t.Errorf("optionale Abfrage: %v", err)
						}
						return nil
					}
					if inTx {
						err = NewTxRunner(db).RunInTx(t.Context(), check)
					} else {
						err = check(t.Context())
					}
					if err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}
