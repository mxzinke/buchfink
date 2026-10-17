package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// shareholders ist der Gesellschafterdienst über der Testumgebung. Die
// Zinsbuchung erzeugt einen Eigenbeleg; der Setzer liefert feste Bytes.
func (e *testEnv) shareholders(t *testing.T) *ShareholderService {
	t.Helper()
	e.receipts.SetRenderer(&stubRenderer{})
	e.receipts.SetNumberRepo(e.numberRepo)
	svc := NewShareholderService(
		repository.NewShareholderLoanRepository(e.db), e.journalRepo, e.contactRepo,
		repository.NewSettingsRepository(e.db), repository.NewAuditRepository(e.db),
		e.journal, e.posting, e.receipts)
	return svc
}

func findingWith(findings []domain.CheckFinding, text string) *domain.CheckFinding {
	for i := range findings {
		if strings.Contains(findings[i].Message, text) {
			return &findings[i]
		}
	}
	return nil
}

// Die Gesellschafterin leiht der GmbH 50.000 € zu 3,5 % bis Ende 2029. Das
// Darlehen gehört auf 3642 (Restlaufzeit 1 bis 5 Jahre), die Zinsen auf 7317
// (Beteiligung 60 %). Ausgezahlt am 02.01.2026; Zinsen 2026 nach act/365:
// 50.000 × 3,5 % × 364/365 = 1.745,21 €.
func TestShareholderLoanToCompanyWithInterest(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withShareholders(t)
	mara := env.person(t, "Mara Beispiel")
	svc := env.shareholders(t)
	payments := env.paymentsWithBank(t)
	payments.SetLoanSource(svc)
	contract := env.fileIncoming(t, "darlehensvertrag.pdf")

	view, err := svc.RegisterLoan(ctx, LoanRequest{
		ContactID: mara.ID, Direction: domain.LoanToCompany, Principal: 5_000_000,
		InterestRateBP: 350, ContractDate: "2025-12-15", StartDate: "2026-01-02", MaturityDate: "2029-12-31",
		ContractReceiptID: contract.ID,
	})
	if err != nil {
		t.Fatalf("Darlehen erfassen: %v", err)
	}
	if view.Loan.Account != domain.AccountVerbGesellschafter1Bis5 ||
		view.Loan.InterestAccount != domain.AccountZinsenGesellschafterUeber25 {
		t.Errorf("Konten %s/%s, erwartet 3642/7317", view.Loan.Account, view.Loan.InterestAccount)
	}
	if findingWith(view.Findings, "kein Vertrag") != nil {
		t.Error("der Vertrag ist abgelegt")
	}
	if findingWith(view.Findings, "kein Vergleichszins") == nil {
		t.Error("ohne belegten Vergleichszins fehlt der Hinweis auf den Fremdvergleich")
	}

	id := env.bankLine(t, "2026-01-02", 5_000_000, "Mara Beispiel", "Darlehen lt. Vertrag")
	disbursement, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: id,
		Lines: []BankSplitLine{{LoanID: view.Loan.ID, Amount: 5_000_000}}})
	if err != nil {
		t.Fatalf("Auszahlung: %v", err)
	}
	assertLines(t, disbursement, []bookedLine{
		{domain.SideDebit, domain.AccountBank, 5_000_000},
		{domain.SideCredit, domain.AccountVerbGesellschafter1Bis5, 5_000_000},
	})

	// Vor der Zinsbuchung meldet der Jahresabschluss die offenen Zinsen.
	findings, err := svc.Findings(ctx, "2026-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if f := findingWith(findings, "nicht gebucht (1.745,21 €"); f == nil {
		t.Errorf("erwartet Befund zu den offenen Zinsen 2026, erhalten %+v", findings)
	}

	entry, err := svc.BookInterest(ctx, InterestRequest{LoanID: view.Loan.ID, To: "2026-12-31"})
	if err != nil {
		t.Fatalf("Zinsen buchen: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, domain.AccountZinsenGesellschafterUeber25, 174521},
		{domain.SideCredit, domain.AccountVerbindlichkeitenGesellschafter, 174521},
	})
	if entry.ReceiptID == nil {
		t.Error("die Zinsbuchung braucht einen Beleg")
	}
	if entry.ServiceDateFrom != "2026-01-02" {
		t.Errorf("Zinsbeginn %s, erwartet der Vertragsbeginn", entry.ServiceDateFrom)
	}
	if _, err := svc.BookInterest(ctx, InterestRequest{LoanID: view.Loan.ID, From: "2026-06-01", To: "2026-12-31"}); err == nil {
		t.Error("ein schon gebuchter Zeitraum wurde ein zweites Mal verzinst")
	}

	loans, err := svc.Loans(ctx, "2026-12-31")
	if err != nil || len(loans) != 1 {
		t.Fatalf("Darlehen: %v", err)
	}
	if loans[0].Outstanding != 5_000_000 || loans[0].InterestBookedThrough != "2026-12-31" {
		t.Errorf("Stand %s €, Zinsen bis %s", loans[0].Outstanding, loans[0].InterestBookedThrough)
	}
	if findingWith(loans[0].Findings, "nicht gebucht") != nil {
		t.Error("nach der Zinsbuchung bleibt der Befund zu den offenen Zinsen stehen")
	}

	// Je Gesellschafterin: Darlehen 50.000,00 € und Zinsen 1.745,21 € als
	// Verbindlichkeiten, nicht mit Forderungen verrechnet.
	balances, err := svc.Balances(ctx, "2026-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if len(balances) != 1 || balances[0].ContactID != mara.ID || balances[0].Liabilities != 5_174_521 ||
		balances[0].Receivables != 0 || len(balances[0].Accounts) != 2 {
		t.Errorf("Salden je Gesellschafter: %+v", balances)
	}
}

// Den Darlehensvertrag erfasst und berichtigt der Mensch in der App; die
// Berichtigung steht mit Grund, vorher und nachher im Protokoll. Sind Zinsen
// gebucht, bleibt der Zinssatz bis zu deren Storno.
func TestLoanContractIsCorrectedByTheHuman(t *testing.T) {
	env := newTestEnv(t)
	env.withShareholders(t)
	mara := env.person(t, "Mara Beispiel")
	svc := env.shareholders(t)
	req := LoanRequest{ContactID: mara.ID, Direction: domain.LoanToCompany, Principal: 2_000_000,
		InterestRateBP: 400, StartDate: "2026-01-01", MaturityDate: "2027-12-31"}

	if _, err := svc.RegisterLoan(agentCtx, req); !errors.Is(err, actor.ErrAppOnly) {
		t.Fatalf("ein Agent erfasst ein Gesellschafterdarlehen: %v", err)
	}
	view, err := svc.RegisterLoan(appCtx, req)
	if err != nil {
		t.Fatalf("Darlehen erfassen: %v", err)
	}
	id := view.Loan.ID

	fix := req
	fix.Account = view.Loan.Account
	fix.ComparisonRateBP, fix.ComparisonSource = intPtr(450), "Angebot der Hausbank vom 10.12.2025"
	if _, err := svc.CorrectLoan(agentCtx, id, fix, "Vergleichszins nachgetragen"); !errors.Is(err, actor.ErrAppOnly) {
		t.Errorf("ein Agent berichtigt den Vertrag: %v", err)
	}
	if _, err := svc.CorrectLoan(appCtx, id, fix, " "); err == nil {
		t.Error("eine Berichtigung ohne Grund wurde angenommen")
	}
	corrected, err := svc.CorrectLoan(appCtx, id, fix, "Vergleichszins nachgetragen")
	if err != nil {
		t.Fatalf("Berichtigung: %v", err)
	}
	if corrected.Loan.ID != id || corrected.Loan.ComparisonRateBP == nil || *corrected.Loan.ComparisonRateBP != 450 {
		t.Errorf("berichtigter Vertrag: %+v", corrected.Loan)
	}

	payments := env.paymentsWithBank(t)
	payments.SetLoanSource(svc)
	tx := env.bankLine(t, "2026-01-02", 2_000_000, "Mara Beispiel", "Darlehen")
	if _, err := payments.BookBankSplit(context.Background(), BankSplitRequest{BankTxID: tx,
		Lines: []BankSplitLine{{LoanID: id, Amount: 2_000_000}}}); err != nil {
		t.Fatalf("Auszahlung: %v", err)
	}
	interest, err := svc.BookInterest(context.Background(), InterestRequest{LoanID: id, To: "2026-06-30"})
	if err != nil {
		t.Fatalf("Zinsen buchen: %v", err)
	}
	fix.InterestRateBP = 300
	if _, err := svc.CorrectLoan(appCtx, id, fix, "Zinssatz falsch erfasst"); err == nil ||
		!strings.Contains(err.Error(), "der Zinssatz") {
		t.Errorf("Zinssatz trotz gebuchter Zinsen geändert: %v", err)
	}
	if _, err := env.journal.Reverse(context.Background(), interest.ID, "Zinssatz falsch erfasst"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CorrectLoan(appCtx, id, fix, "Zinssatz falsch erfasst"); err != nil {
		t.Errorf("nach dem Storno der Zinsen bleibt der Zinssatz gesperrt: %v", err)
	}

	log, err := repository.NewAuditRepository(env.db).FindFiltered(context.Background(), 20,
		domain.AuditFilter{EntityType: "SHAREHOLDER_LOAN"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range log {
		found = found || strings.Contains(entry.Details, "berichtigt: Zinssatz falsch erfasst")
	}
	if !found {
		t.Errorf("die Berichtigung fehlt im Protokoll: %+v", log)
	}
}

func intPtr(v int) *int { return &v }

// Ein zinsloses Darlehen der GmbH an ihre geschäftsführende Gesellschafterin
// ohne Vertrag und Sicherheit: vier Hinweise, keine Sperre.
func TestInterestFreeLoanToShareholderRaisesFindings(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withShareholders(t)
	mara := env.person(t, "Mara Beispiel")
	svc := env.shareholders(t)

	view, err := svc.RegisterLoan(ctx, LoanRequest{
		ContactID: mara.ID, Direction: domain.LoanToShareholder, Principal: 2_000_000,
		StartDate: "2026-02-01", MaturityDate: "2026-12-31",
	})
	if err != nil {
		t.Fatalf("Darlehen erfassen: %v", err)
	}
	if view.Loan.Account != domain.AccountForderungenGesellschafterBis1 ||
		view.Loan.InterestAccount != domain.AccountSonstigeZinsertraege {
		t.Errorf("Konten %s/%s, erwartet 1308/7110", view.Loan.Account, view.Loan.InterestAccount)
	}
	for _, want := range []struct{ text, reference string }{
		{"kein Vertrag", "R 8.5 Abs. 2 KStR 2022"},
		{"zinslos", "R 8.5 Abs. 1 KStR 2022, § 8 Abs. 3 Satz 2 KStG"},
		{"keine Sicherheit", "§ 30 Abs. 1 Satz 2 GmbHG"},
		{"Geschäftsführer", "§ 43a GmbHG"},
	} {
		f := findingWith(view.Findings, want.text)
		if f == nil || f.Reference != want.reference || f.Severity != domain.CheckWarning {
			t.Errorf("Befund %q: %+v", want.text, f)
		}
	}
}

// Ein Zins unter dem belegten Vergleichszins ist ein Hinweis, darüber bei
// einem Darlehen an die GmbH ebenso.
func TestComparisonRateFindings(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withShareholders(t)
	holding := env.person(t, "Muster Holding GmbH")
	svc := env.shareholders(t)
	comparison := 400
	view, err := svc.RegisterLoan(ctx, LoanRequest{
		ContactID: holding.ID, Direction: domain.LoanToCompany, Principal: 1_000_000, InterestRateBP: 650,
		StartDate: "2026-03-01", ComparisonRateBP: &comparison, ComparisonSource: "Angebot Hausbank vom 20.02.2026",
	})
	if err != nil {
		t.Fatalf("Darlehen erfassen: %v", err)
	}
	if view.Loan.InterestAccount != domain.AccountZinsenGesellschafterUeber25 {
		t.Errorf("40 %% Beteiligung: Zinskonto %s, erwartet 7317", view.Loan.InterestAccount)
	}
	if view.Loan.Account != domain.AccountVerbGesellschafterBis1 {
		t.Errorf("ohne bestimmte Laufzeit: Konto %s, erwartet 3641", view.Loan.Account)
	}
	if findingWith(view.Findings, "über dem belegten Vergleichszins von 4,00 %") == nil {
		t.Errorf("erwartet Hinweis auf den überhöhten Zins, erhalten %+v", view.Findings)
	}

	stranger := env.person(t, "Erika Mustermann")
	if _, err := svc.RegisterLoan(ctx, LoanRequest{ContactID: stranger.ID, Direction: domain.LoanToCompany,
		Principal: 100_000, StartDate: "2026-03-01"}); err == nil {
		t.Error("ein Darlehen eines Dritten wurde als Gesellschafterdarlehen erfasst")
	}
}

// Ein Sollsaldo auf dem Verrechnungskonto über drei Monate und ein
// Darlehenssaldo ohne Vertrag erscheinen im Prüflauf.
func TestShareholderFindingsInTheCheckRun(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withShareholders(t)
	mara := env.person(t, "Mara Beispiel")
	svc := env.shareholders(t)
	payments := env.paymentsWithBank(t)

	private := env.bankLine(t, "2026-02-01", -80000, "Möbelhaus", "Sofa privat")
	if _, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: private, Lines: []BankSplitLine{
		{Account: domain.AccountForderungenGesellschafter, ContactID: mara.ID, Amount: 80000, Text: "Privat, Rückzahlung vereinbart"},
	}}); err != nil {
		t.Fatalf("Verrechnung: %v", err)
	}
	loan := env.bankLine(t, "2026-03-01", -300000, "Mara Beispiel", "Darlehen")
	if _, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: loan, Lines: []BankSplitLine{
		{Account: domain.AccountForderungenGesellschafterUeber1, ContactID: mara.ID, Amount: 300000},
	}}); err != nil {
		t.Fatalf("Darlehen ohne Vertrag: %v", err)
	}

	early, err := svc.Findings(ctx, "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	if findingWith(early, "Verrechnungskonto") != nil {
		t.Error("nach zwei Monaten ist der Sollsaldo noch kein stehendes Darlehen")
	}
	findings, err := svc.Findings(ctx, "2026-06-30")
	if err != nil {
		t.Fatal(err)
	}
	if f := findingWith(findings, "seit 01.02.2026 eine Forderung"); f == nil || f.Rule != domain.CheckRuleShareholderClearing {
		t.Errorf("erwartet Befund zum Verrechnungskonto, erhalten %+v", findings)
	}
	if f := findingWith(findings, "Darlehenskonto 1309"); f == nil {
		t.Errorf("erwartet Befund zum Darlehen ohne Vertrag, erhalten %+v", findings)
	}

	checks := env.checksOn(t, "2026-07-15")
	checks.AddFindingSource(svc)
	run, err := checks.Preview(ctx, CheckRequest{CutoffDate: "2026-06-30"})
	if err != nil {
		t.Fatal(err)
	}
	if findingWith(run.Findings, "Verrechnungskonto") == nil {
		t.Error("der Prüflauf übernimmt die Befunde der Gesellschafter nicht")
	}
}
