package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// halfPaidFoundation ist die Musterprotokoll-Gründung mit Halbeinzahlung:
// 25.000 € Stammkapital, eine Gesellschafterin, 50 % sofort, der Rest nach
// Beschluss.
func halfPaidFoundation() *domain.Foundation {
	return &domain.Foundation{
		ShareCapital: 2_500_000,
		Shareholders: []domain.Shareholder{
			{Name: "Anna Bauer", ShareCapital: 2_500_000, PaidIn: 1_250_000, Uncalled: 1_250_000,
				Kind: domain.ContributionCash},
		},
	}
}

// Zeichnung mit Halbeinzahlung: 1298 und 2910 an 2900; die Einzahlung kommt
// über den Bankumsatz auf 1298, die Einforderung nach Beschluss von 2910 auf
// 1298 (§ 272 Abs. 1 Satz 2 HGB, § 46 Nr. 2 GmbHG).
func TestHalfPaymentIsSubscribedCalledAndPaidThroughTheBank(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	svc.SetReceiptService(env.receipts)
	ctx := context.Background()
	env.saveFoundation(t, svc, halfPaidFoundation())

	preview, err := svc.PreviewPostings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Postings) != 1 {
		t.Fatalf("Buchungsvorschläge: %+v", preview.Postings)
	}
	lines := preview.Postings[0].Lines
	if len(lines) != 3 ||
		lines[0].Account != domain.AccountAusstehendeEinlagenGeford || lines[0].Amount != 1_250_000 ||
		lines[1].Account != domain.AccountAusstehendeEinlagenOffen || lines[1].Amount != 1_250_000 ||
		lines[2].Account != domain.AccountGezeichnetesKapital || lines[2].Amount != 2_500_000 {
		t.Fatalf("Zeichnung: %+v", lines)
	}
	if _, err := svc.BookPostings(ctx); err != nil {
		t.Fatal(err)
	}

	bank := NewBankService(repository.NewBankRepository(env.db), env.journal, repository.NewAuditRepository(env.db))
	txID := env.bankCredit(t, "2026-01-22", 1_250_000, "STAMMKAPITAL")
	if _, err := bank.BookDirect(ctx, txID, domain.AccountAusstehendeEinlagenGeford, "Einzahlung Stammkapital"); err != nil {
		t.Fatalf("Einzahlung über den Bankumsatz: %v", err)
	}
	status, err := svc.CapitalStatus(ctx, "2026-01-31")
	if err != nil {
		t.Fatal(err)
	}
	if status.Subscribed != 2_500_000 || status.Uncalled != 1_250_000 || status.Called != 1_250_000 ||
		status.CalledUnpaid != 0 || status.PaidIn != 1_250_000 || len(status.Findings) != 0 {
		t.Fatalf("Stand der Einlagen: %+v", status)
	}

	// Einforderung ohne Beschluss geht nicht; mehr als offen ist, auch nicht.
	if _, err := svc.CallInCapital(ctx, CapitalCallRequest{Date: "2026-06-30", Amount: 1_250_000}); err == nil {
		t.Error("eine Einforderung ohne Beschluss wurde gebucht")
	}
	resolution := env.fileIncoming(t, "beschluss.pdf")
	if _, err := svc.CallInCapital(ctx, CapitalCallRequest{
		Date: "2026-06-30", Amount: 1_300_000, ResolutionReceiptID: resolution.ID,
	}); err == nil {
		t.Error("eingefordert wurde mehr als nicht eingefordert war")
	}
	if _, err := svc.CallInCapital(ctx, CapitalCallRequest{
		Date: "2026-06-30", Amount: 1_250_000, ResolutionReceiptID: resolution.ID,
	}); err != nil {
		t.Fatalf("Einforderung: %v", err)
	}
	status, err = svc.CapitalStatus(ctx, "2026-06-30")
	if err != nil {
		t.Fatal(err)
	}
	if status.Uncalled != 0 || status.Called != 2_500_000 || status.CalledUnpaid != 1_250_000 {
		t.Errorf("nach der Einforderung: %+v", status)
	}
}

// Was eingefordert sein muss, folgt aus der Anmeldung: ein Viertel je Anteil,
// 12.500 € zusammen; Sacheinlage und UG kennen keinen nicht eingeforderten Teil.
func TestUncalledCapitalIsLimitedByWhatTheRegistrationNeeds(t *testing.T) {
	cases := map[string]func(*domain.Foundation){
		"negativ": func(f *domain.Foundation) { f.Shareholders[0].Uncalled = -1 },
		"unter der Hälfte": func(f *domain.Foundation) {
			f.Shareholders[0].Uncalled = 1_300_000
			f.Shareholders[0].PaidIn = 1_200_000
		},
		"mehr als gefordert": func(f *domain.Foundation) { f.Shareholders[0].PaidIn = 1_300_000 },
		"Sacheinlage":        func(f *domain.Foundation) { f.Shareholders[0].Kind = domain.ContributionInKind },
	}
	for name, change := range cases {
		env := newTestEnv(t)
		svc := env.foundations(t)
		f := halfPaidFoundation()
		change(f)
		if _, err := svc.Save(context.Background(), notarizedOn, f); err == nil {
			t.Errorf("%s: angenommen", name)
		}
	}

	env := newTestEnv(t)
	env.setSetting(t, "legal_form", "UG (haftungsbeschränkt)")
	svc := env.foundations(t)
	_, err := svc.Save(context.Background(), notarizedOn, &domain.Foundation{
		ShareCapital: 100_000,
		Shareholders: []domain.Shareholder{{Name: "Anna Bauer", ShareCapital: 100_000, PaidIn: 50_000,
			Uncalled: 50_000, Kind: domain.ContributionCash}},
	})
	if err == nil || !strings.Contains(err.Error(), "voll einzuzahlen") {
		t.Errorf("UG mit nicht eingefordertem Teil: %v", err)
	}
}
