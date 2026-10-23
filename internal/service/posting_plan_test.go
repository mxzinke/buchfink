package service

import (
	"context"
	"slices"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestIncomingPlanSpecials(t *testing.T) {
	want := map[string][]string{
		"bewirtung_70_30":           {"gruppe_mit_sonderregel", "bewirtung"},
		"geschenk_unter_grenze":     {"geschenk", "gruppe_mit_sonderregel"},
		"geschenk_ueber_grenze":     {"geschenk", "gruppe_mit_sonderregel"},
		"geschenk_grenze_kumuliert": {"geschenk", "gruppe_mit_sonderregel"},
		"vorsteueranteil":           {"vorsteueranteil"},
		"vorsteuerausschluss":       {"gruppe_mit_sonderregel"},
		"anzahlung":                 {"anzahlung"},
		"rueckstellungsverbrauch":   {"konto_statt_gruppe", "rueckstellung"},
		"gesellschafterauslage":     {"auslage"},
		"fremdwaehrung":             {"fremdbetrag"},
		"pflichtangabe_fehlt":       {"befund_vorsteuer"},
		"uebersteuerung":            {"uebersteuerung", "befund_vorsteuer"},
		"konto_statt_gruppe":        {"konto_statt_gruppe"},
		"e_rechnung_befunde":        {"befund_vorsteuer"},
		"e_rechnung_uebersteuerung": {"uebersteuerung", "befund_vorsteuer"},
	}
	for _, name := range incomingGoldenCases {
		t.Run(name, func(t *testing.T) {
			env, req := incomingGoldenFixture(t, name)
			ctx := context.Background()
			audit := repository.NewAuditRepository(env.db)
			before, err := audit.Count(ctx)
			if err != nil {
				t.Fatal(err)
			}
			preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			if got := preview.Specials(); !slices.Equal(got, want[name]) {
				t.Fatalf("Sonderangaben: %v, erwartet %v", got, want[name])
			}
			if copy := preview.Specials(); len(copy) > 0 {
				copy[0] = "verändert"
				if !slices.Equal(preview.Specials(), want[name]) {
					t.Fatal("Sonderangaben lassen sich von außen verändern")
				}
			}
			after, err := audit.Count(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if before != after {
				t.Fatal("Planung hat das Protokoll geändert")
			}
			receipt, err := env.receipts.Get(ctx, req.ReceiptID)
			if err != nil {
				t.Fatal(err)
			}
			if !receipt.IsOpen() || receipt.JournalEntryID != nil {
				t.Fatal("Planung hat den Beleg gebucht")
			}
		})
	}
}

func TestPositionRulesKeepErrorOrder(t *testing.T) {
	env := newTestEnv(t)
	vendor := env.vendor(t, "Lieferant GmbH", "DE", "")
	req := env.receipt(t, vendor.ID, "", 10000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Positions[0].InputTaxShare = -1
	req.Positions = append(req.Positions, ReceiptPosition{Net: -1})
	for _, tc := range []struct {
		share int
		want  string
	}{
		{-1, "Position 1: der abziehbare Vorsteueranteil liegt zwischen 0 und 100 %"},
		{0, "Position 1: weder eine Buchungsgruppe noch ein Konto angegeben"},
	} {
		req.Positions[0].InputTaxShare = tc.share
		_, err := env.posting.PreviewIncomingReceipt(context.Background(), req)
		if err == nil || err.Error() != tc.want {
			t.Fatalf("Fehler: %v, erwartet %q", err, tc.want)
		}
	}
}
