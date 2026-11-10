package mcpserver

import (
	"context"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"testing"
)

func TestAnhangText(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	var out, replay anhangOut
	a.ok("anhang_text", a.args(map[string]any{"jahr": 2026}), &out)
	if len(out.Abschnitte) != 9 || out.Gesperrt || !out.Abschnitte[0].Pflicht {
		t.Fatalf("Abschnitte: %+v", out)
	}
	args := a.args(map[string]any{"jahr": 2026, "idempotenz_schluessel": "anhang-2026", "texte": map[string]any{"organkredite": "Keine Organkredite."}})
	a.ok("anhang_text", args, &out)
	a.ok("anhang_text", args, &replay)
	if out.Abschnitte[0].Text != "Keine Organkredite." || !replay.Wiederholt {
		t.Fatalf("Speichern: %+v / %+v", out, replay)
	}
	reader := h.mustSession(h.login("Nur lesen", "buchfink:lesen").access)
	if res := callTool(t, reader, "anhang_text", a.args(map[string]any{"jahr": 2026})); res.IsError {
		t.Fatal(resultText(res))
	}
	if res := callTool(t, reader, "anhang_text", args); !res.IsError {
		t.Fatal("Schreiben mit Leserecht")
	}

	a.ok("anhang_text", a.args(map[string]any{"jahr": 2026, "texte": map[string]any{"haftungsverhaeltnisse": "Keine.", "weitere_angaben": "Keine weiteren Angaben erforderlich."}}), &out)
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-01", "")
		return err
	})
	a.ok("anhang_text", a.args(map[string]any{"jahr": 2026}), &out)
	if !out.Gesperrt {
		t.Fatal("Aufstellung sperrt Texte nicht")
	}
	a.fail("anhang_text", a.args(map[string]any{"jahr": 2026, "texte": map[string]any{"organkredite": "Geändert"}}), CodeRule)
	until := harnessNow().AddDate(0, 0, 5).Format("2006-01-02")
	if _, err := h.host.EnableExaminer(uiCtx, until, "Betriebsprüfung"); err != nil {
		t.Fatal(err)
	}
	a.ok("anhang_text", a.args(map[string]any{"jahr": 2026}), &out)
	a.fail("anhang_text", a.args(map[string]any{"jahr": 2026, "texte": map[string]any{"methoden": "Neu"}}), CodeExaminer)
}
