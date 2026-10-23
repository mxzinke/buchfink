package mcpserver

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/goldentest"
)

func TestGoldenAdapterReceiptPosting(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	vendor := h.vendor()
	a := h.agent("")
	var filed ablageOut
	result := a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-04-02", "belegdatum": "2026-03-31", "betreff": "Server", "dateien": []any{map[string]any{"pfad": h.inboxFile("kopf.pdf", pdf("Kopf"))}}}), &filed)
	goldenAdapterResult(t, "adapter_ablage", result, h)
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Receipts.SaveHeader(ctx, filed.Beleg.ID, domain.ReceiptHeader{Kind: domain.ReceiptKindOther, DocumentDate: "2026-03-31", Subject: "Server", Currency: "EUR"})
		return err
	})
	args := a.args(bookArgs(filed.Beleg.ID, vendor, "2026-03-31", "100.00"))
	args["nur_vorschau"] = true
	goldenAdapterResult(t, "adapter_rueckfrage_vorschau", a.ok("beleg_buchen_mit_rueckfrage", args, nil), h)
	goldenAdapterResult(t, "adapter_direkt_vorschau", a.ok("beleg_buchen", args, nil), h)
	h.read(func(ctx context.Context, s *core.Services) error {
		r, err := s.Receipts.Get(ctx, filed.Beleg.ID)
		if err == nil && (r.DocumentDate != "2026-03-31" || r.IssuerName != "" || r.GrossAmount != 0) {
			t.Fatal("Vorschau speichert Kopfdaten")
		}
		return err
	})
	delete(args, "nur_vorschau")
	goldenAdapterResult(t, "adapter_rueckfrage_buchung", a.ok("beleg_buchen_mit_rueckfrage", args, nil), h)
	h.read(func(ctx context.Context, s *core.Services) error {
		r, err := s.Receipts.Get(ctx, filed.Beleg.ID)
		if err == nil {
			goldentest.Assert(t, "adapter_ergaenzter_kopf", domain.ReceiptHeader{Kind: r.Kind, DocumentDate: r.DocumentDate, IssuerName: r.IssuerName, GrossAmount: r.GrossAmount, TaxAmount: r.TaxAmount, Currency: r.Currency, Subject: r.Subject})
		}
		return err
	})
	for _, direction := range []string{"alle", "forderung", "verbindlichkeit"} {
		goldenAdapterResult(t, "adapter_posten_"+direction, a.ok("offene_posten", map[string]any{"stichtag": "2026-06-30", "richtung": direction, "mit_altersstruktur": true}, nil), h)
	}
	for _, period := range []string{"2026-03", "2026-Q1", "2026"} {
		goldenAdapterResult(t, "adapter_erklaerung_"+period, a.ok("erklaerung", map[string]any{"art": "ustva", "zeitraum": period}, nil), h)
	}
}

func TestGoldenAdapterReceiptPeriod(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	vendor := h.vendor()
	a := h.agent("")
	var filed ablageOut
	goldenAdapterResult(t, "adapter_ablage_zeitraum", a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-04-02", "dateien": []any{map[string]any{"pfad": h.eInvoice("PERIOD", "2026-03-31", 10000)}}}), &filed), h)
	for _, tool := range []string{"beleg_buchen", "beleg_buchen_mit_rueckfrage"} {
		goldentest.Assert(t, "adapter_"+tool+"_kontakt", a.fail(tool, a.args(bookArgs(filed.Beleg.ID, 999, "2026-03-31", "100.00")), CodeNotFound))
		goldentest.Assert(t, "adapter_"+tool+"_beleg", a.fail(tool, a.args(bookArgs(999, vendor, "2026-03-31", "100.00")), CodeNotFound))
	}
	goldenAdapterResult(t, "adapter_beleg_lesen", a.ok("beleg_lesen", map[string]any{"beleg_id": filed.Beleg.ID}, nil), h)
}

// Testordner und Monat der Dateiübernahme werden vereinheitlicht.
func goldenAdapterResult(t *testing.T, name string, result map[string]any, h *harness) {
	t.Helper()
	var normalize func(any) any
	normalize = func(v any) any {
		switch x := v.(type) {
		case string:
			return regexp.MustCompile(`/Übernommen/\d{4}-\d{2}/`).ReplaceAllString(strings.ReplaceAll(x, h.tenant.DataDir, "<Datenordner>"), "/Übernommen/<Monat>/")
		case map[string]any:
			for k, value := range x {
				x[k] = normalize(value)
			}
		case []any:
			for i, value := range x {
				x[i] = normalize(value)
			}
		}
		return v
	}
	goldentest.Assert(t, name, normalize(result))
}

func TestGoldenAdapterMissingHeader(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	vendor := h.vendor()
	a := h.agent("")
	for _, field := range []string{"belegdatum", "aussteller", "brutto"} {
		args := map[string]any{"eingang_am": "2026-03-31", "belegdatum": "2026-03-31", "aussteller": "Netzwerk GmbH", "brutto": "119.00", "dateien": []any{map[string]any{"pfad": h.inboxFile(field+".pdf", pdf(field))}}}
		delete(args, field)
		var filed ablageOut
		a.ok("beleg_ablegen", a.args(args), &filed)
		book := a.args(bookArgs(filed.Beleg.ID, vendor, "2026-03-31", "100.00"))
		for _, preview := range []bool{true, false} {
			book["nur_vorschau"] = preview
			goldentest.Assert(t, "adapter_kopf_fehlt_"+field, a.fail("beleg_buchen_mit_rueckfrage", book, CodeRule))
		}
	}
}
