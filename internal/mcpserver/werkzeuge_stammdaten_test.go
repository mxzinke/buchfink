package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Ein Vorschlag ändert nichts; der Mensch übernimmt ihn in der App so, wie die
// Vorschau ihn zeigt, oder lehnt ihn ab, und der Agent sieht die Entscheidung
// in status.
func TestStammdatenVorschlagUebernehmenUndAblehnen(t *testing.T) {
	h := newHarness(t)
	a := h.agent("")

	var out stammdatenOut
	a.ok("stammdaten_vorschlagen", a.args(map[string]any{
		"idempotenz_schluessel": "nordlicht-gruendung",
		"begruendung":           "Gesellschaftsvertrag vom 03.08.2026",
		"unternehmen": map[string]any{
			"strasse": "Am Kaiserkai", "hausnummer": "19", "plz": "20457", "ort": "Hamburg",
			"geschaeftsfuehrung": "Jana Petersen", "notar": "Dr. Henrike Albers", "urkundennummer": "1187/2026 A",
			"steuernummer": "47/716/02345", "bundesland": "HH", "ust_zeitraum": "monat", "stammkapital": "25000.00",
		},
		"gruendung": map[string]any{
			"beurkundet_am": "2026-08-03", "stammkapital": "25000.00", "gruendungsaufwand_bis": "2500.00",
			"gesellschafter": []any{
				map[string]any{"name": "Jana Petersen", "anteil": "15000.00", "nicht_eingefordert": "7500.00"},
				map[string]any{"name": "Ole Brandt", "anteil": "10000.00", "nicht_eingefordert": "5000.00"},
			},
		},
		"bankkonten": []any{map[string]any{"iban": "DE42 2004 1155 0742 1388 00", "name": "Geschäftskonto",
			"bic": "HATBDEHHXXX", "bank": "Hanseatische Testbank AG"}},
	}), &out)
	if out.Stand != "offen" || len(out.Aenderungen) == 0 {
		t.Fatalf("Vorschlag: %+v", out)
	}
	if !hasChange(out.Aenderungen, "Bankkonten", "DE42 2004 1155 0742 1388 00", "Konto 1800 · Konto für Rechnungen") ||
		!hasChange(out.Aenderungen, "Gründung", "Gesellschafter 2", "nicht eingefordert 5.000,00 €") ||
		!hasChange(out.Aenderungen, "Unternehmensdaten", "Voranmeldungszeitraum", "monatlich") ||
		hasChange(out.Aenderungen, "Unternehmensdaten", "Stammkapital", "") {
		t.Fatalf("Vorschau: %+v", out.Aenderungen)
	}

	// Nichts ist geschrieben.
	h.read(func(ctx context.Context, s *core.Services) error {
		cfg, err := s.Settings.GetCompanySettings(ctx)
		if err != nil {
			return err
		}
		accounts, err := s.Bank.Accounts(ctx)
		if err != nil {
			return err
		}
		state, err := s.Foundation.GetState(ctx)
		if err != nil {
			return err
		}
		if cfg.Street != "" || cfg.TaxNumber != "" || len(accounts) != 0 || state.HasFoundation {
			t.Fatalf("der Vorschlag hat geschrieben: %+v %+v %+v", cfg, accounts, state)
		}
		return nil
	})
	var st statusOut
	a.ok("status", nil, &st)
	if len(st.Vorschlaege) != 1 || st.Vorschlaege[0].Stand != "offen" {
		t.Fatalf("status: %+v", st.Vorschlaege)
	}

	// Eine Vorschau, die nicht mehr gilt, wird nicht übernommen.
	var pending []service.MasterDataProposalView
	h.read(func(ctx context.Context, s *core.Services) error {
		var err error
		pending, err = s.MasterData.Pending(ctx)
		return err
	})
	if len(pending) != 1 {
		t.Fatalf("offene Vorschläge: %+v", pending)
	}
	stale := pending[0].Fingerprint
	h.ui(func(ctx context.Context, s *core.Services) error {
		cfg, err := s.Settings.GetCompanySettings(ctx)
		if err != nil {
			return err
		}
		cfg.Street = "Großer Grasbrook"
		return s.Settings.UpdateCompanySettings(ctx, cfg)
	})
	err := h.host.Write(uiCtx, func(ctx context.Context, s *core.Services) error {
		_, err := s.MasterData.Accept(ctx, pending[0].ID, stale)
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "seit der Vorschau geändert") {
		t.Fatalf("veraltete Vorschau übernommen: %v", err)
	}

	h.ui(func(ctx context.Context, s *core.Services) error {
		fresh, err := s.MasterData.Pending(ctx)
		if err != nil {
			return err
		}
		if !hasLine(fresh[0].Lines, "Straße", "Großer Grasbrook", "Am Kaiserkai") {
			t.Fatalf("neue Vorschau: %+v", fresh[0].Lines)
		}
		_, err = s.MasterData.Accept(ctx, fresh[0].ID, fresh[0].Fingerprint)
		return err
	})
	h.read(func(ctx context.Context, s *core.Services) error {
		cfg, err := s.Settings.GetCompanySettings(ctx)
		if err != nil {
			return err
		}
		if cfg.Street != "Am Kaiserkai" || cfg.TaxNumber != "47/716/02345" || cfg.VatPeriod != "month" ||
			cfg.FoundedOn != "2026-08-03" || cfg.IBAN != "DE42200411550742138800" {
			t.Fatalf("Unternehmensdaten nach dem Übernehmen: %+v", cfg)
		}
		state, err := s.Foundation.GetState(ctx)
		if err != nil {
			return err
		}
		if !state.HasFoundation || len(state.Foundation.Shareholders) != 2 ||
			state.Foundation.Shareholders[1].Uncalled != 500_000 || state.Foundation.FoundationCostCap != 250_000 {
			t.Fatalf("Gründung: %+v", state.Foundation)
		}
		tasks, err := s.Tasks.Tasks(ctx, service.TaskOptions{})
		if err != nil {
			return err
		}
		for _, task := range append(tasks.Open, tasks.Overdue...) {
			if task.Key == domain.TaskKeyMasterDataProposal {
				t.Fatalf("übernommener Vorschlag als Aufgabe: %+v", task)
			}
		}
		return nil
	})

	// Abgelehnt: der Agent sieht den Grund.
	a.ok("stammdaten_vorschlagen", a.args(map[string]any{
		"idempotenz_schluessel": "nordlicht-ustid", "begruendung": "Mitteilung des BZSt",
		"unternehmen": map[string]any{"ust_id": "DE123456789"},
	}), &out)
	h.read(func(ctx context.Context, s *core.Services) error {
		tasks, err := s.Tasks.Tasks(ctx, service.TaskOptions{})
		if err != nil {
			return err
		}
		for _, task := range tasks.Open {
			if task.Key == domain.TaskKeyMasterDataProposal && task.Target.Area == domain.AreaCompany {
				return nil
			}
		}
		t.Fatalf("offener Vorschlag fehlt in der Aufgabenliste: %+v", tasks.Open)
		return nil
	})
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.MasterData.Reject(ctx, out.VorschlagID, "Nummer gehört zu einem anderen Unternehmen")
		return err
	})
	a.ok("status", nil, &st)
	if len(st.Vorschlaege) != 2 || st.Vorschlaege[0].Stand != "abgelehnt" ||
		st.Vorschlaege[0].Vermerk != "Nummer gehört zu einem anderen Unternehmen" || st.Vorschlaege[1].Stand != "uebernommen" {
		t.Fatalf("status nach den Entscheidungen: %+v", st.Vorschlaege)
	}
}

// Was keines der Formulare annähme, weist schon der Vorschlag ab.
func TestStammdatenVorschlagPrueftDieEingabe(t *testing.T) {
	h := newHarness(t)
	a := h.agent("")
	propose := func(key string, in map[string]any) map[string]any {
		in["idempotenz_schluessel"], in["begruendung"] = key, "Test"
		return a.args(in)
	}
	a.fail("stammdaten_vorschlagen", propose("unbekannt-1-vorschlag", map[string]any{
		"unternehmen": map[string]any{"rechtsform": "AG"}}), CodeInvalidInput)
	a.fail("stammdaten_vorschlagen", propose("steuernr-1-vorschlag", map[string]any{
		"unternehmen": map[string]any{"steuernummer": "123", "bundesland": "HH"}}), CodeRule)
	a.fail("stammdaten_vorschlagen", propose("iban-1-vorschlag", map[string]any{
		"bankkonten": []any{map[string]any{"iban": "DE00 1234", "name": "Konto"}}}), CodeRule)
	a.fail("stammdaten_vorschlagen", propose("ug-1-vorschlag", map[string]any{
		"gruendung": map[string]any{"beurkundet_am": "2026-08-03", "stammkapital": "0.50",
			"gesellschafter": []any{map[string]any{"name": "A", "anteil": "0.50"}}}}), CodeRule)
	a.fail("stammdaten_vorschlagen", propose("leer-1-vorschlag", map[string]any{}), CodeRule)
	a.fail("stammdaten_vorschlagen", propose("kapital-1-vorschlag", map[string]any{
		"unternehmen": map[string]any{"stammkapital": "30000.00"},
		"gruendung": map[string]any{"beurkundet_am": "2026-08-03", "stammkapital": "25000.00",
			"gesellschafter": []any{map[string]any{"name": "A", "anteil": "25000.00"}}}}), CodeRule)

}

func hasChange(changes []aenderung, area, field, after string) bool {
	for _, c := range changes {
		if c.Bereich == area && c.Feld == field && strings.Contains(c.Neu, after) {
			return true
		}
	}
	return false
}

func hasLine(lines []service.MasterDataChangeLine, field, before, after string) bool {
	for _, l := range lines {
		if l.Field == field && l.Before == before && l.After == after {
			return true
		}
	}
	return false
}
