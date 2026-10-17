package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Regression coverage for the bank import findings.
func TestBugHuntBankKeepsDifferentNOTPROVIDEDPayments(t *testing.T) {
	db, err := InitTenantDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewBankRepository(db)
	count, err := repo.CreateBatch(context.Background(), []domain.BankTransaction{
		{AccountIBAN: "DE89370400440532013000", BookingDate: "2026-09-01", ValueDate: "2026-09-01", Amount: 50000, EndToEndID: "NOTPROVIDED", RemittanceInfo: "Rechnung A"},
		{AccountIBAN: "DE89370400440532013000", BookingDate: "2026-09-02", ValueDate: "2026-09-02", Amount: 70000, EndToEndID: "NOTPROVIDED", RemittanceInfo: "Rechnung B"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("two different bank payments must survive; imported %d", count)
	}
}

func TestBugHuntBankDeduplicatesReimport(t *testing.T) {
	db, err := InitTenantDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewBankRepository(db)
	tx := domain.BankTransaction{AccountIBAN: "DE89370400440532013000", BookingDate: "2026-09-01", ValueDate: "2026-09-01", Amount: 50000, CounterpartyIBAN: "DE02120300000000202051"}
	if _, err := repo.CreateBatch(context.Background(), []domain.BankTransaction{tx}); err != nil {
		t.Fatal(err)
	}
	count, err := repo.CreateBatch(context.Background(), []domain.BankTransaction{tx})
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("reimport of a bank transaction created %d duplicate(s)", count)
	}
}

func TestBugHuntBankReportsPersistenceFailure(t *testing.T) {
	db, err := InitTenantDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropTable(&domain.BankTransaction{}); err != nil {
		t.Fatal(err)
	}
	_, err = NewBankRepository(db).CreateBatch(context.Background(), []domain.BankTransaction{{Amount: 50000}})
	if err == nil {
		t.Error("database write failure was reported as a successful import")
	}
}
