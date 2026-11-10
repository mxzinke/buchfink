package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func (e *testEnv) payroll(t *testing.T) *PayrollService {
	t.Helper()
	return NewPayrollService(repository.NewPayrollRepository(e.db), e.journalRepo, e.journal, e.posting,
		repository.NewAuditRepository(e.db))
}

// Lohnjournal Oktober 2026: Gehälter 8.000,00 €, Geschäftsführergehalt der
// Gesellschafterin 5.000,00 € (nicht sozialversicherungspflichtig), SV
// Arbeitnehmer 1.560,00 €, Lohnsteuer 2.100,00 €, Netto 9.340,00 €;
// Arbeitgeberanteil 1.680,00 €, Umlagen 160,00 €.
// Probe: 13.000,00 − 1.560,00 − 2.100,00 = 9.340,00.
func octoberPayroll(receiptID uint) PayrollRequest {
	return PayrollRequest{
		Month: "2026-10", ReceiptID: receiptID,
		Salaries: 800_000, ShareholderDirectorSalaries: 500_000,
		EmployeeSocialSecurity: 156_000, WageTax: 210_000, NetPay: 934_000,
		EmployerSocialSecurity: 168_000, Levies: 16_000,
	}
}

func TestPayrollJournalIsBookedAsMonthlySum(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	journal := env.fileIncoming(t, "lohnjournal-2026-10.pdf")

	entry, err := env.payroll(t).BookPayroll(ctx, octoberPayroll(journal.ID))
	if err != nil {
		t.Fatalf("Lohnjournal buchen: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, domain.AccountGehaelter, 800_000},
		{domain.SideDebit, domain.AccountGFGehaelterGesellschafter, 500_000},
		{domain.SideDebit, domain.AccountGesetzlicheSozialeAufw, 168_000},
		{domain.SideDebit, domain.AccountGesetzlicheSozialeAufw, 16_000},
		{domain.SideCredit, domain.AccountVerbLohnGehalt, 934_000},
		{domain.SideCredit, domain.AccountVerbLohnKirchensteuer, 210_000},
		{domain.SideCredit, domain.AccountVerbSozialeSicherheit, 156_000},
		{domain.SideCredit, domain.AccountVerbSozialeSicherheit, 168_000},
		{domain.SideCredit, domain.AccountVerbSozialeSicherheit, 16_000},
	})
	if entry.BookingDate != "2026-10-31" || entry.ReceiptID == nil || *entry.ReceiptID != journal.ID {
		t.Errorf("Buchungstag %s, Beleg %v", entry.BookingDate, entry.ReceiptID)
	}
	if entry.TaxTreatment != domain.TaxTreatmentNotTaxable {
		t.Errorf("Steuerfall %q, Arbeitslohn ist nicht steuerbar", entry.TaxTreatment)
	}
	if !strings.Contains(entry.Description, "10/2026") {
		t.Errorf("Buchungstext %q nennt den Monat nicht", entry.Description)
	}

	// Derselbe Monat ein zweites Mal ist eine Doppelbuchung.
	again := env.fileIncoming(t, "lohnjournal-2026-10-kopie.pdf")
	if _, err := env.payroll(t).BookPayroll(ctx, octoberPayroll(again.ID)); err == nil ||
		!strings.Contains(err.Error(), "bereits mit Buchung") {
		t.Errorf("erwartet Abweisung der zweiten Buchung, erhalten %v", err)
	}

	// Nach dem Storno ist der Monat wieder buchbar.
	if _, err := env.journal.Reverse(ctx, entry.ID, "berichtigtes Lohnjournal"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.payroll(t).BookPayroll(ctx, octoberPayroll(again.ID)); err != nil {
		t.Errorf("nach dem Storno muss das berichtigte Journal buchbar sein: %v", err)
	}
}

// Minijob an der Geringfügigkeitsgrenze 2026 (603,00 €, § 8 Abs. 1a SGB IV)
// mit Pauschalabgaben an die Minijob-Zentrale: 13 % KV und 15 % RV pauschal
// (78,39 € + 90,45 € = 168,84 €), 2 % Pauschsteuer (12,06 €, § 40a Abs. 2
// EStG). Die Abgaben gehen samt Pauschsteuer auf 3740.
func TestPayrollMinijobGoesToTheMinijobCentre(t *testing.T) {
	env := newTestEnv(t)
	journal := env.fileIncoming(t, "lohnjournal-minijob.pdf")
	entry, err := env.payroll(t).BookPayroll(context.Background(), PayrollRequest{
		Month: "2026-03", ReceiptID: journal.ID,
		MinijobWages: 60_300, NetPay: 60_300,
		MinijobContributions: 16_884, MinijobFlatTax: 1_206,
	})
	if err != nil {
		t.Fatalf("Minijob buchen: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, domain.AccountLoehneMinijobs, 60_300},
		{domain.SideDebit, domain.AccountSozialeAbgabenMinijobs, 16_884},
		{domain.SideDebit, domain.AccountPauschsteuerMinijobs, 1_206},
		{domain.SideCredit, domain.AccountVerbLohnGehalt, 60_300},
		{domain.SideCredit, domain.AccountVerbSozialeSicherheit, 18_090},
	})
}

// Die Netto-Probe hält Sonderzeilen auf, die in den Summen fehlen.
func TestPayrollNetCheck(t *testing.T) {
	env := newTestEnv(t)
	journal := env.fileIncoming(t, "lohnjournal.pdf")
	req := octoberPayroll(journal.ID)
	req.NetPay = 924_000
	_, err := env.payroll(t).BookPayroll(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "Netto-Probe") || !strings.Contains(err.Error(), "-100,00") {
		t.Errorf("erwartet Abweisung mit Differenz, erhalten %v", err)
	}
	req = octoberPayroll(0)
	if _, err := env.payroll(t).BookPayroll(context.Background(), req); err == nil {
		t.Error("ein Lohnjournal ohne Beleg wurde gebucht")
	}
}

// Abstimmung zum 31.10.: Nettolöhne und Beiträge sind bezahlt, die Lohnsteuer
// steht bis zum 10.11. offen. Fehlt die Zahlung der Beiträge, meldet die
// Abstimmung sie (§ 23 Abs. 1 Satz 2 SGB IV).
func TestPayrollReconciliation(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.paymentsWithBank(t)
	journal := env.fileIncoming(t, "lohnjournal-2026-10.pdf")
	svc := env.payroll(t)
	if _, err := svc.BookPayroll(ctx, octoberPayroll(journal.ID)); err != nil {
		t.Fatal(err)
	}

	before, err := svc.Reconcile(ctx, "2026-10-31")
	if err != nil {
		t.Fatal(err)
	}
	if findingWith(before.Findings, "Sozialversicherungsbeiträge offen") == nil {
		t.Errorf("erwartet Befund zu den offenen Beiträgen, erhalten %+v", before.Findings)
	}

	sv := env.bankLine(t, "2026-10-28", -340_000, "AOK", "Beitrag 10/2026")
	net := env.bankLine(t, "2026-10-30", -934_000, "Sammelauftrag", "Gehälter 10/2026")
	for id, line := range map[uint]BankSplitLine{
		sv:  {Account: domain.AccountVerbSozialeSicherheit, Amount: 340_000},
		net: {Account: domain.AccountVerbLohnGehalt, Amount: 934_000},
	} {
		if _, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: id, Lines: []BankSplitLine{line}}); err != nil {
			t.Fatalf("Zahlung: %v", err)
		}
	}

	after, err := svc.Reconcile(ctx, "2026-10-31")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Findings) != 0 {
		t.Errorf("nach den Zahlungen bleiben Befunde: %+v", after.Findings)
	}
	for _, st := range after.Accounts {
		if st.Account == domain.AccountVerbLohnKirchensteuer && (st.Balance != 210_000 || st.Expected != 210_000) {
			t.Errorf("Lohnsteuer offen %s €, erwartet %s €: 2.100,00 bis zum 10.11.", st.Balance, st.Expected)
		}
	}

	// Bis zum 10.11. ist die Lohnsteuer nicht fällig, danach ohne Zahlung an
	// das Finanzamt ein Befund.
	due, err := svc.Reconcile(ctx, "2026-11-10")
	if err != nil {
		t.Fatal(err)
	}
	if findingWith(due.Findings, "Lohnsteuer früherer Zeiträume") != nil {
		t.Errorf("am Fälligkeitstag ist die Lohnsteuer noch nicht rückständig: %+v", due.Findings)
	}
	late, err := svc.Reconcile(ctx, "2026-11-11")
	if err != nil {
		t.Fatal(err)
	}
	if f := findingWith(late.Findings, "Lohnsteuer früherer Zeiträume"); f == nil || f.Reference != "§ 41a Abs. 1 EStG, § 42d Abs. 1 EStG" {
		t.Errorf("erwartet Befund zur nicht abgeführten Lohnsteuer, erhalten %+v", late.Findings)
	}
}

// Lohn als Lieferantenrechnung gibt es nicht mehr: er läuft über das
// Lohnjournal.
func TestSalaryIsNoPostingGroup(t *testing.T) {
	env := newTestEnv(t)
	if _, err := env.postingGroup("gehaelter"); err == nil {
		t.Error("die Buchungsgruppe gehaelter besteht noch")
	}
}
