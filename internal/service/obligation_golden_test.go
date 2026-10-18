package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/goldentest"
)

func TestGoldenObligationCalendar(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		advisor, payroll, zm, founding bool
	}{
		{name: "ohne_beratung"},
		{name: "beratung_lohn_zm_gruendung", advisor: true, payroll: true, zm: true, founding: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			env.setSettings(t, func(c *domain.CompanySettings) {
				c.FederalState = "BY"
				c.VatPeriod = "month"
				c.PermanentExtension = true
				if tc.advisor {
					c.TaxAdvisor = domain.TaxAdvisor{Engaged: true, SinceYear: 2026,
						Declarations: []string{domain.DeclarationCorporateTax, domain.DeclarationEBilanz}}
				}
				if tc.founding {
					c.FoundedOn, c.RegisteredOn = "2025-09-15", "2025-10-20"
				}
			})
			svc := env.obligations(t)
			svc.SetFiscalYearSource(stubYearList{
				{Year: 2025, StartDate: "2025-09-15", EndDate: "2025-12-31", Status: domain.FiscalYearAdopted,
					PreparedOn: "2026-04-01", AdoptedOn: "2026-05-20", AdoptionNote: "Beschluss vom 20.05.2026"},
				{Year: 2026, StartDate: "2026-01-01", EndDate: "2026-12-31", Status: domain.FiscalYearOpen},
				{Year: 2027, StartDate: "2027-01-01", EndDate: "2027-12-31", Status: domain.FiscalYearOpen},
			})
			svc.SetSizeSource(stubSizes{class: domain.SizeSmall})
			svc.SetAppropriationSource(goldenAppropriationSource{2025: {
				Year: 2025, DecisionDate: "2026-05-20", Text: "Ausschüttung", Distribution: 1000000,
				WithholdingTax: 250000, SolidarityOnWithholding: 13750,
			}})
			if tc.payroll {
				env.post(t, "2026-01-31", "6020", "3730", 60000)
				env.post(t, "2026-09-30", "6020", "3730", 60000)
			}
			if tc.zm {
				customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
				env.euService(t, customer.ID, "2026-02-10", 500000)
			}
			env.post(t, "2026-02-10", "3830", domain.AccountBank, 120000)
			env.post(t, "2026-03-10", "7600", domain.AccountBank, 150000)
			env.post(t, "2026-02-16", "7610", domain.AccountBank, 80000)
			for _, kind := range []domain.ObligationEventKind{domain.EventShareholderChange, domain.EventBeneficialOwnerChange} {
				if _, err := svc.RecordEvent(context.Background(), kind, "2026-05-04", "Änderung der Beteiligung"); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := svc.Confirm(appCtx, "gesellschafterliste.ereignis-1", ConfirmInput{
				Date: "2026-05-05", Kind: domain.EvidenceOrderNumber, Reference: "HR-123",
			}); err != nil {
				t.Fatal(err)
			}
			// Die Bearbeiterkennung hängt vom Betriebssystem und Rechner ab.
			if err := env.db.Model(&domain.ObligationRecord{}).Where("decided_actor <> ?", "").Update("decided_actor", "test@buchfink").Error; err != nil {
				t.Fatal(err)
			}
			q := ObligationQuery{From: "2025-01-01", To: "2027-12-31", Today: "2026-10-17"}
			list := listObligations(t, svc, q)
			goldentest.Assert(t, "obligations_"+tc.name, list)

			// List sortiert nach Fälligkeit; die Erzeugungsreihenfolge ist ein eigener Vertrag.
			run, err := svc.begin(context.Background(), q.Today)
			if err != nil {
				t.Fatal(err)
			}
			from, to := run.span(q)
			run.dueFrom, run.dueTo = q.From, q.To
			generated, err := run.generate(from, to, nil)
			if err != nil {
				t.Fatal(err)
			}
			keys := make([]string, 0, len(generated))
			for _, obligation := range generated {
				keys = append(keys, obligation.Key)
			}
			goldentest.Assert(t, "obligation_order_"+tc.name, keys)

			// Jede Art ist auch einzeln über dieselbe öffentliche Abfrage erreichbar.
			for _, definition := range svc.Catalog() {
				filtered := q
				filtered.Kinds = []domain.ObligationKind{definition.Kind}
				want := []domain.Obligation{}
				for _, obligation := range list {
					if obligation.Kind == definition.Kind {
						want = append(want, obligation)
					}
				}
				got := listObligations(t, svc, filtered)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("Auswahl %s weicht von der Gesamtliste ab", definition.Kind)
				}
			}
		})
	}
}

type goldenAppropriationSource map[int]*domain.Appropriation

func (s goldenAppropriationSource) Appropriation(_ context.Context, year int) (*domain.Appropriation, error) {
	return s[year], nil
}

type goldenObligationVat struct {
	calls *[]string
	fail  bool
}

func (s goldenObligationVat) PeriodType(context.Context) domain.VatPeriodType {
	return domain.VatPeriodMonth
}

func (s goldenObligationVat) Periods(_ context.Context, year int) ([]VatPeriodStatus, error) {
	*s.calls = append(*s.calls, fmt.Sprintf("UStVA %d", year))
	if s.fail {
		return nil, errors.New("Voranmeldungen nicht lesbar")
	}
	return nil, nil
}

type goldenObligationDecisions struct{ calls *[]string }

func (s goldenObligationDecisions) Appropriation(_ context.Context, year int) (*domain.Appropriation, error) {
	*s.calls = append(*s.calls, fmt.Sprintf("Ergebnisverwendung %d", year))
	return nil, errors.New("Beschluss nicht lesbar")
}

func TestGoldenObligationErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		vatFails bool
		kinds    []domain.ObligationKind
	}{
		{name: "kalenderjahr", vatFails: true},
		{name: "geschaeftsjahr"},
		{name: "gefiltert", vatFails: true, kinds: []domain.ObligationKind{domain.ObligationDisclosure}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := []string{}
			svc := NewObligationService(nil, nil, nil)
			svc.SetFiscalYearSource(stubYearList{{Year: 2025, StartDate: "2025-09-15", EndDate: "2025-12-31"}})
			svc.SetVatSources(goldenObligationVat{calls: &calls, fail: tc.vatFails}, nil)
			svc.SetAppropriationSource(goldenObligationDecisions{calls: &calls})
			list, err := svc.List(context.Background(), ObligationQuery{From: "2025-01-01", To: "2027-12-31", Today: "2026-10-17", Kinds: tc.kinds})
			errText := ""
			if err != nil {
				errText = err.Error()
			}
			goldentest.Assert(t, "obligation_error_"+tc.name, struct {
				List  []domain.Obligation
				Error string
				Calls []string
			}{list, errText, calls})
		})
	}
}
