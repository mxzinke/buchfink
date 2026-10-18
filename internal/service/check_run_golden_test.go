package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestGoldenCheckRun(t *testing.T) {
	for _, tc := range []struct {
		name, cutoff, period, today string
		populated                   bool
	}{
		{"leer", "2026-01-01", "month", "2026-01-01", false},
		{"monat", "2026-03-31", "month", "2026-03-31", true},
		{"jahr", "2026-12-31", "year", "2026-12-31", true},
		{"spaeter_lauf", "2026-03-06", "month", "2026-05-15", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			svc := env.checksOn(t, tc.today)
			if tc.populated {
				populateCheckGolden(t, env, svc)
			}
			run, err := svc.Run(context.Background(), CheckRequest{
				CutoffDate: tc.cutoff, PeriodType: tc.period, OverrideReason: "  Begründung für den Prüflauf  ",
			})
			if err != nil {
				t.Fatal(err)
			}
			stored, err := svc.Runs(context.Background(), run.FiscalYear)
			if err != nil || len(stored) != 1 {
				t.Fatalf("Gespeicherter Lauf: %v, %v", stored, err)
			}
			assertGolden(t, "check_run_"+tc.name, struct {
				Run     *domain.CheckRun
				Stored  domain.CheckRun
				Summary string
			}{run, stored[0], run.BlockingSummary()})
		})
	}
}

func populateCheckGolden(t *testing.T, env *testEnv, svc *CheckService) {
	t.Helper()
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")
	for _, date := range []string{"2026-03-01", "2026-03-02", "2026-04-01"} {
		entry := simpleEntry(domain.AccountGeldtransit, vendor.LedgerAccount, 100000)
		entry.BookingDate, entry.DocumentNumber, entry.ContactID = date, "RG-4711", &vendor.ID
		entry.Lines[1].ContactID = &vendor.ID
		if _, err := env.journal.Post(ctx, entry); err != nil {
			t.Fatal(err)
		}
	}
	// Die Erfassungsabstände gehören zum Ergebnis und brauchen feste Zeitpunkte.
	if err := env.db.Model(&domain.JournalEntry{}).Where("1 = 1").Update("created_at", time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)).Error; err != nil {
		t.Fatal(err)
	}
	for _, date := range []string{"2026-03-01", "2026-04-01"} {
		env.setReceivedAt(t, env.fileIncoming(t, date+".pdf").ID, date)
	}
	if _, err := env.numberRepo.Allocate(ctx, domain.NumberRangeReceipt, 2026); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.NewBankRepository(env.db).CreateBatch(ctx, []domain.BankTransaction{
		{FiscalYear: 2026, BookingDate: "2026-03-02", Amount: 50000, CounterpartyName: "Kunde", MatchStatus: domain.MatchStatusUnmatched},
		{FiscalYear: 2026, BookingDate: "2026-04-02", Amount: 60000, CounterpartyName: "Späterer Kunde", MatchStatus: domain.MatchStatusUnmatched},
	}); err != nil {
		t.Fatal(err)
	}
	svc.SetOpenItemSource(openItemStub{items: []domain.OpenItem{{EntryID: 1, EntryNumber: "2026-000001", ContactName: "Lieferant", GrossAmount: 119000, SettledAmount: 238000}}})
	svc.SetAccountSource(accountStub{accounts: []domain.Account{{Number: "1234", Name: "Ohne Position", Type: domain.AccountTypeAsset, StatementType: "Bilanz", PositionID: "fehlt", DebitSum: 10000}}})
	svc.SetDepreciationSource(depreciationStub{due: []DepreciationDue{{AssetID: 1, InventoryNumber: "AV-2026-0001", Name: "Maschine", Due: 120000}}})
	svc.SetProvisionSource(goldenProvisionSource{})
	svc.SetClosingStepSource(goldenClosingSource{})
	for _, message := range []string{"Erster Fachbereich", "Zweiter Fachbereich"} {
		svc.AddFindingSource(goldenFindingSource(func(context.Context, string) ([]domain.CheckFinding, error) {
			return []domain.CheckFinding{{Rule: domain.CheckRuleProvisionDiscount, Severity: domain.CheckWarning, Message: message}}, nil
		}))
	}
}

type goldenFindingSource func(context.Context, string) ([]domain.CheckFinding, error)

func (s goldenFindingSource) Findings(ctx context.Context, cutoff string) ([]domain.CheckFinding, error) {
	return s(ctx, cutoff)
}

type goldenProvisionSource struct{}

func (goldenProvisionSource) DiscountFindings(context.Context, int) ([]domain.CheckFinding, error) {
	return nil, errors.New("Abzinsung nicht lesbar")
}

type goldenClosingSource struct{}

func (goldenClosingSource) SkippedSteps(context.Context, int) ([]SkippedClosingStep, error) {
	return []SkippedClosingStep{{Key: domain.ClosingStepKey("provisions"), Label: "Rückstellungen", Reason: "Keine offenen Verpflichtungen", ChangedOn: "2026-12-31"}}, nil
}

type goldenFailingJournal struct{ domain.JournalRepository }

func (goldenFailingJournal) FindAll(context.Context, int) ([]domain.JournalEntry, error) {
	return nil, errors.New("Journal nicht lesbar")
}

type goldenFailingReceipts struct{ domain.ReceiptRepository }

func (goldenFailingReceipts) FindAll(context.Context, int) ([]domain.Receipt, error) {
	return nil, errors.New("Belege nicht lesbar")
}

type goldenFailingAccounts struct{ calls *[]string }

func (s goldenFailingAccounts) AccountsForYear(context.Context, int) ([]domain.Account, error) {
	*s.calls = append(*s.calls, "Konten")
	return nil, errors.New("Konten nicht lesbar")
}

func TestGoldenCheckRunErrors(t *testing.T) {
	for _, failure := range []string{"journal", "receipts", "source", "accounts", "cutoff"} {
		t.Run(failure, func(t *testing.T) {
			env := newTestEnv(t)
			svc := env.checks(t)
			calls := []string{}
			svc.SetAccountSource(goldenFailingAccounts{calls: &calls})
			svc.SetProvisionSource(goldenProvisionSource{})
			for _, name := range []string{"Quelle 1", "Quelle 2"} {
				svc.AddFindingSource(goldenFindingSource(func(context.Context, string) ([]domain.CheckFinding, error) {
					calls = append(calls, name)
					if failure == "source" {
						return nil, errors.New("Fachbereich nicht lesbar")
					}
					return nil, nil
				}))
			}
			req := CheckRequest{CutoffDate: "2026-12-31", PeriodType: "year"}
			switch failure {
			case "journal":
				svc.journalRepo = goldenFailingJournal{}
			case "receipts":
				svc.receiptRepo = goldenFailingReceipts{}
			case "cutoff":
				req.CutoffDate = ""
			}
			run, err := svc.Run(context.Background(), req)
			errText := ""
			if err != nil {
				errText = err.Error()
			}
			stored, readErr := svc.Runs(context.Background(), 2026)
			if readErr != nil {
				t.Fatal(readErr)
			}
			assertGolden(t, "check_error_"+failure, struct {
				Run    *domain.CheckRun
				Error  string
				Calls  []string
				Stored int
			}{run, errText, calls, len(stored)})
		})
	}
}
