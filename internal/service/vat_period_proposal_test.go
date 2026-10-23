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

// Gründung im Oktober 2025: die Steuer des Gründungsjahres wird für 2026 in
// eine Jahressteuer umgerechnet (§ 18 Abs. 2 Satz 6 UStG). 3.000 € aus drei
// Monaten sind 12.000 € im Jahr und damit monatlich. Die Quartale vor der
// Gründung fehlen nicht.
func TestVatPeriodProposalAnnualisesTheFoundingYear(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) { c.FoundedOn = "2025-10-15" })
	svc := env.vatReturns(t)
	repo := repository.NewVatReturnRepository(env.db)
	if err := repo.Create(ctx, &domain.VatReturn{
		FiscalYear: 2025, PeriodKey: "2025-Q4", PeriodType: domain.VatPeriodQuarter,
		Status: domain.VatReturnSubmitted, Payable: 300_000,
	}); err != nil {
		t.Fatalf("Voranmeldung: %v", err)
	}

	proposal, err := svc.SuggestPeriodType(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if proposal.Proposed != domain.VatPeriodMonth {
		t.Errorf("Vorschlag %q — umgerechnet 12.000 € sind monatlich: %s", proposal.Proposed, proposal.Note)
	}
	if !proposal.Complete || proposal.MissingPeriods != 0 {
		t.Errorf("vor der Gründung fehlt keine Anmeldung, gemeldet sind %d", proposal.MissingPeriods)
	}
	if !strings.Contains(proposal.Note, "aus 3 Monaten") || strings.Contains(proposal.Note, "befreien") {
		t.Errorf("Begründung: %s", proposal.Note)
	}
}

// Im Gründungsjahr 2026 entscheidet die voraussichtliche Steuer, die Buchfink
// nicht kennt; für ein Jahr nach 2026 gilt wieder die Monatspflicht des
// § 18 Abs. 2 Satz 4 UStG.
func TestVatPeriodProposalInTheFoundingYear(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.FoundedOn, c.VatPeriod = "2026-03-01", string(domain.VatPeriodUnknown)
	})
	svc := env.vatReturns(t)

	proposal, err := svc.SuggestPeriodType(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if proposal == nil {
		t.Fatal("eine Gründung ohne festgelegten Zeitraum braucht den Vorschlag")
	}
	if proposal.Changes || !strings.Contains(proposal.Note, "voraussichtliche Steuer") {
		t.Errorf("ohne Schätzung kein Wechsel, aber die Regel: %+v", proposal)
	}

	next, err := svc.SuggestPeriodType(ctx, 2027)
	if err != nil {
		t.Fatalf("Vorschlag 2027: %v", err)
	}
	if next.Proposed != domain.VatPeriodMonth || !strings.Contains(next.Note, "Satz 4") {
		t.Errorf("2027 ist das Folgejahr der Gründung: %+v", next)
	}
}
