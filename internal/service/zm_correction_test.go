package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestZMCorrectionChainAmounts(t *testing.T) {
	first := domain.ZMReturn{ID: 1, Status: domain.VatReturnSubmitted, Lines: []domain.ZMLine{
		{VatID: "FR12345678901", Kind: "L", Amount: 100000},
		{VatID: "FR12345678901", Kind: "S", Amount: 200000},
		{VatID: "FR12345678901", Kind: "D", Amount: 300000},
	}}
	tests := []struct {
		name    string
		current []domain.ZMLine
		amounts map[string]domain.Cents
	}{
		{"Storno Lieferung", first.Lines[1:], map[string]domain.Cents{"FR12345678901:L": 0}},
		{"Nachmeldung", append(append([]domain.ZMLine{}, first.Lines...), domain.ZMLine{VatID: "IT00743110157", Kind: "L", Amount: 400000}), map[string]domain.Cents{"IT00743110157:L": 400000}},
		{"USt-ID geändert", []domain.ZMLine{{VatID: "IT00743110157", Kind: "L", Amount: 100000}, first.Lines[1], first.Lines[2]}, map[string]domain.Cents{"FR12345678901:L": 0, "IT00743110157:L": 100000}},
		{"Dreieck und Leistung", []domain.ZMLine{first.Lines[0], {VatID: "FR12345678901", Kind: "S", Amount: 250000}, {VatID: "FR12345678901", Kind: "D", Amount: 350000}}, map[string]domain.Cents{"FR12345678901:S": 250000, "FR12345678901:D": 350000}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := zmChanges(zmReported([]domain.ZMReturn{first}), tt.current)
			if len(changes) != len(tt.amounts) {
				t.Fatalf("Zeilen: %+v", changes)
			}
			for _, l := range changes {
				if want, ok := tt.amounts[zmKey(l)]; !ok || want != l.Amount {
					t.Errorf("unerwartete Zeile: %+v", l)
				}
			}
			correction := domain.ZMReturn{ID: 2, Status: domain.VatReturnSubmitted, IsCorrection: true, Lines: changes}
			state := zmReported([]domain.ZMReturn{correction, first})
			if got := zmChanges(state, tt.current); len(got) != 0 {
				t.Fatalf("unveränderte zweite Berichtigung: %+v", got)
			}
			next := append([]domain.ZMLine{}, tt.current...)
			next[0].Amount += 10000
			got := zmChanges(state, next)
			if len(got) != 1 || got[0].Amount != next[0].Amount {
				t.Fatalf("zweite Berichtigung: %+v", got)
			}
		})
	}
}

func TestZMCorrectionSubmissionAndContactChange(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client", "FR", "FR12345678901")
	other := env.customer(t, "Altro", "IT", "IT00743110157")
	env.euService(t, customer.ID, "2026-02-10", 100000)
	env.euService(t, other.ID, "2026-02-10", 200000)
	svc := env.zmReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatal(err)
	}
	env.commitUntil(t, "2026-03-31")
	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-24", "first", ""); err != nil {
		t.Fatal(err)
	}
	customer.VatID = "FR12345678902"
	if err := env.contactRepo.Save(ctx, customer); err != nil {
		t.Fatal(err)
	}
	correction, err := svc.CreateCorrection(ctx, "2026-Q1", "2026-04-30")
	if err != nil {
		t.Fatal(err)
	}
	if len(correction.Lines) != 2 || correction.Lines[0].Amount != 0 || correction.Lines[1].Amount != 100000 {
		t.Fatalf("Berichtigung: %+v", correction.Lines)
	}
	obligations := listObligations(t, env.obligations(t), ObligationQuery{FiscalYear: 2026})
	obligation, ok := obligationByKey(obligations, "zm.2026-Q1")
	if !ok || obligation.Status == domain.ObligationDone || obligation.DueDate != "2026-06-01" {
		t.Fatalf("Berichtigungspflicht: %+v", obligation)
	}
	if correction.DueDate != "2026-06-01" {
		t.Fatalf("Frist: %s", correction.DueDate)
	}
	again, err := svc.CreateCorrection(ctx, "2026-Q1", "2026-05-10")
	if err != nil || again.ID != correction.ID || again.DueDate != correction.DueDate {
		t.Fatalf("Fortschreibung verlängert Frist: %+v, %v", again, err)
	}
	if _, err := svc.ConfirmSubmitted(ctx, correction.ID, "2026-05-01", "correction", ""); err != nil {
		t.Fatal(err)
	}
	preview, err := svc.Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Lines) != 0 {
		t.Fatalf("erneute Änderung: %+v", preview.Lines)
	}
	late := &domain.Invoice{ContactID: other.ID, Date: "2026-04-20", ServiceDateFrom: "2026-03-20", ServiceDateTo: "2026-03-20", TaxTreatment: domain.TaxTreatmentReverseChargeSupply,
		Items: []domain.InvoiceItem{{Description: "Nachmeldung", QuantityMilli: 1000, UnitPrice: 50000, TaxRate: domain.TaxRateNone}}}
	if err := env.invoices(t).Issue(ctx, late); err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateCorrection(ctx, "2026-Q1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Lines) != 1 || second.Lines[0].Amount != 250000 {
		t.Fatalf("zweite Berichtigung: %+v", second.Lines)
	}
	csv, err := svc.ExportCSV(ctx, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csv, "IT00743110157,2500,S\n") || strings.Contains(csv, "FR123") {
		t.Fatal(csv)
	}
	other.VatID = "IT00743110158"
	if err := env.contactRepo.Save(ctx, other); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ExportCSV(ctx, second.ID); err == nil {
		t.Fatal("überholter Entwurf exportiert")
	}
	if _, err := svc.ConfirmSubmitted(ctx, second.ID, "2026-05-02", "stale", ""); err == nil {
		t.Fatal("überholter Entwurf bestätigt")
	}
	other.VatID = "IT00743110157"
	if err := env.contactRepo.Save(ctx, other); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmSubmitted(ctx, second.ID, "2026-05-02", "second", ""); err != nil {
		t.Fatal(err)
	}
}

func TestZMBOPCSVKindsAndZero(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	rec := &domain.ZMReturn{FiscalYear: 2026, PeriodKey: "2026-Q1", PeriodType: domain.VatPeriodQuarter, Status: domain.VatReturnSubmitted, IsCorrection: true,
		Lines: []domain.ZMLine{{VatID: "FR12345678901", Kind: "L", Amount: 0}, {VatID: "FR12345678901", Kind: "D", Amount: -12349}, {VatID: "FR12345678901", Kind: "S", Amount: 45699}}}
	if err := repository.NewZMReturnRepository(env.db).Create(ctx, rec); err != nil {
		t.Fatal(err)
	}
	got, err := env.zmReturns(t).ExportCSV(ctx, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "#v3.0\nUmsatzsteuer-Identifikationsnummer (USt-IdNr.),Summe (Euro),Art der Leistung\nFR12345678901,0,L\nFR12345678901,-123,D\nFR12345678901,456,S\n"
	if got != want {
		t.Fatalf("CSV:\n%s", got)
	}
}
