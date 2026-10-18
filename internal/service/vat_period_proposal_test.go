package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Der Steuersatz wird gegen das Leistungsdatum gehalten (UNV-03 K2).
//
// Eine Rechnung über eine Leistung vom August 2020 mit 19 % passt nicht
// zusammen: damals galten 16 und 5 % (§ 12 Abs. 1 und 2 UStG in der Fassung des
// Zweiten Corona-Steuerhilfegesetzes). Der Vorschlag sagt es und bucht nicht
// stillschweigend.
func TestProposalWarnsAboutARateThatDoesNotFitTheServiceDate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	receipt := env.filedReceipt(t)

	read := receivedInvoice()
	read.IssueDate = "2020-09-01"
	read.DeliveryDate = "2020-08-15"
	proposal, err := env.einvoicesWith(fakeReader{invoice: read}).Propose(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	note := strings.Join(proposal.Notes, " | ")
	if !strings.Contains(note, "16 %") || !strings.Contains(note, "2020-08-15") {
		t.Errorf("der Vorschlag nennt den Satz des Leistungstages nicht: %s", note)
	}

	// Derselbe Beleg mit einem Leistungsdatum von heute: kein Vermerk.
	today := receivedInvoice()
	proposal, err = env.einvoicesWith(fakeReader{invoice: today}).Propose(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	for _, n := range proposal.Notes {
		if strings.Contains(n, "passt nicht zum Leistungsdatum") {
			t.Errorf("19 %% passen zum Jahr 2026: %s", n)
		}
	}
}

// Der Voranmeldungszeitraum folgt der Steuer des Vorjahres (§ 18 Abs. 2 UStG).
func TestVatPeriodProposalFollowsThePriorYearTax(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.vatReturns(t)
	repo := repository.NewVatReturnRepository(env.db)

	// Vier Quartale des Vorjahres mit zusammen 12.000 € Steuer.
	for _, key := range []string{"2025-Q1", "2025-Q2", "2025-Q3", "2025-Q4"} {
		rec := &domain.VatReturn{
			FiscalYear: 2025, PeriodKey: key, PeriodType: domain.VatPeriodQuarter,
			Status: domain.VatReturnSubmitted, Payable: 300_000,
		}
		if err := repo.Create(ctx, rec); err != nil {
			t.Fatalf("Voranmeldung %s: %v", key, err)
		}
	}

	proposal, err := svc.SuggestPeriodType(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if proposal.PriorYearTax != 1_200_000 {
		t.Errorf("Vorjahressteuer %s € — erwartet 12.000,00 €", proposal.PriorYearTax)
	}
	if proposal.Proposed != domain.VatPeriodMonth {
		t.Errorf("Vorschlag %q — über 9.000 € ist monatlich anzumelden", proposal.Proposed)
	}
	if !proposal.Changes {
		t.Error("der Mandant steht auf Quartal; der Vorschlag weicht ab und muss das sagen")
	}
	if !proposal.Complete {
		t.Errorf("alle vier Quartale liegen vor, es fehlen angeblich %d", proposal.MissingPeriods)
	}
	if !strings.Contains(proposal.Note, "§ 18 Abs. 2") &&
		!strings.Contains(proposal.Reference, "§ 18 Abs. 2") {
		t.Error("der Vorschlag nennt seine Vorschrift nicht")
	}
}

// Unter der Grenze bleibt es beim Vierteljahr — die Befreiung spricht das
// Finanzamt aus und nicht das Programm.
func TestVatPeriodProposalStaysQuarterlyBelowTheThreshold(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.vatReturns(t)
	repo := repository.NewVatReturnRepository(env.db)

	for _, key := range []string{"2025-Q1", "2025-Q2", "2025-Q3", "2025-Q4"} {
		rec := &domain.VatReturn{
			FiscalYear: 2025, PeriodKey: key, PeriodType: domain.VatPeriodQuarter,
			Status: domain.VatReturnSubmitted, Payable: 25_000,
		}
		if err := repo.Create(ctx, rec); err != nil {
			t.Fatalf("Voranmeldung %s: %v", key, err)
		}
	}

	proposal, err := svc.SuggestPeriodType(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if proposal.Proposed != domain.VatPeriodQuarter {
		t.Errorf("Vorschlag %q — unter 2.000 € bleibt es beim Vierteljahr, bis das "+
			"Finanzamt befreit", proposal.Proposed)
	}
	if !strings.Contains(proposal.Note, "Finanzamt") {
		t.Error("der Hinweis muss sagen, dass die Befreiung das Finanzamt ausspricht")
	}
}
