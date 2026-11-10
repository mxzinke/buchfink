package core

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Steuerrückstellung und Erklärungen rechnen aus denselben Büchern dieselben
// Steuern: dieselben Konten, Höchstbeträge, Rundungen und dieselbe Anrechnung.
//
// Die Bücher 2026: Umsatz 300.000 €, Aufwand 50.000 €, nicht abziehbar 3.400 €
// (Bewirtung, Geldbuße, Verlust aus Anteilen des Umlaufvermögens,
// Arbeitszimmer, Darlehen an die Tochter, Geschenke, Schuldzinsen,
// Parteispende), Spende 3.000 €, Dividende der Tochter 10.000 € mit 2.500,40 €
// Kapitalertragsteuer und 137,52 € Solidaritätszuschlag, Vorauszahlungen
// 10.000 € KSt, 550 € SolZ und 8.000 € GewSt, Verlustvortrag 50.000 €,
// Hebesatz 410 %.
//
// Einkommen vor Zuwendungen 250.500 €, nach Zuwendungen 247.500 €, nach
// Verlustabzug 197.500 €: KSt 29.625,00 €, SolZ 1.629,37 €, Gewerbeertrag
// 197.500 €, Messbetrag 6.912,50 €, GewSt 28.341,25 €. Angerechnet werden
// 2.501 € Kapitalertragsteuer (§ 36 Abs. 3 EStG).
func TestTaxProvisionAgreesWithTheDeclarations(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)

	line := func(side domain.Side, account string, amount domain.Cents) domain.JournalLine {
		return domain.JournalLine{Side: side, Account: account, Amount: amount}
	}
	d, c := domain.SideDebit, domain.SideCredit
	entries := map[string][]domain.JournalLine{
		"Umsatz":        {line(d, "1800", 30_000_000), line(c, "4400", 30_000_000)},
		"Aufwand":       {line(d, "6815", 5_000_000), line(c, "1800", 5_000_000)},
		"Bewirtung":     {line(d, "6644", 100_000), line(c, "1800", 100_000)},
		"Geldbuße":      {line(d, "6968", 50_000), line(c, "1800", 50_000)},
		"Wertpapiere":   {line(d, "6904", 70_000), line(c, "1800", 70_000)},
		"Arbeitszimmer": {line(d, "6349", 30_000), line(c, "1800", 30_000)},
		"Tochter":       {line(d, "6290", 40_000), line(c, "1800", 40_000)},
		"Geschenke":     {line(d, "6620", 20_000), line(c, "1800", 20_000)},
		"Schuldzinsen":  {line(d, "7313", 10_000), line(c, "1800", 10_000)},
		"Spende":        {line(d, "6391", 300_000), line(c, "1800", 300_000)},
		"Parteispende":  {line(d, "6394", 20_000), line(c, "1800", 20_000)},
		"Dividende": {line(d, "1800", 736_208), line(d, accounting.AccountKapitalertragsteuer25, 250_040),
			line(d, accounting.AccountSoliKapitalertragsteuer, 13_752), line(c, "7006", 1_000_000)},
		"Vorauszahlungen": {line(d, domain.AccountKoerperschaftsteuer, 1_000_000),
			line(d, domain.AccountSolidaritaetszuschlag, 55_000), line(d, domain.AccountGewerbesteuer, 800_000),
			line(c, "1800", 1_855_000)},
	}
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		if _, err := s.ClosingSettings.Save(ctx, service.ClosingSettings{
			TradeTaxRatePercent: 410, AccrualMethod: "monthly", AccrualRelease: "yearly"}); err != nil {
			return err
		}
		for text, lines := range entries {
			if _, err := s.Journal.Post(ctx, &domain.JournalEntry{
				BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
				ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
				Description: text, Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
				Lines: lines,
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for kind, key := range map[declaration.Kind]string{
		declaration.KindCorporateTax: "kst.v.z14", declaration.KindTradeTax: "gewst.verlust_vorjahr",
	} {
		if _, err := tenant.SetDeclarationInput(uiCtx, service.InputRequest{Kind: kind, Year: 2026, Key: key,
			Value: declaration.Cents(5_000_000), Source: declaration.InputFromNotice}); err != nil {
			t.Fatal(err)
		}
	}

	opts := service.TaxProvisionOptions{LossCarryforwardCorporate: 5_000_000, LossCarryforwardTrade: 5_000_000}
	var preview *service.TaxProvisionPreview
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) (err error) {
		preview, err = s.ClosingBookings.PreviewTaxProvision(ctx, 2026, opts)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	kst, err := tenant.Declaration(uiCtx, declaration.KindCorporateTax, 2026)
	if err != nil {
		t.Fatal(err)
	}
	gewst, err := tenant.Declaration(uiCtx, declaration.KindTradeTax, 2026)
	if err != nil {
		t.Fatal(err)
	}
	amount := func(d *declaration.Declaration, key string) domain.Cents {
		l, ok := d.Line(key)
		if !ok {
			t.Fatalf("Zeile %s fehlt", key)
		}
		return l.Value.Amount()
	}

	for _, c := range []struct {
		what                  string
		provision, decl, want domain.Cents
	}{
		{"Einkommen vor Zuwendungen", preview.IncomeBeforeDonations, amount(kst, "kst.zve.z32"), 25_050_000},
		{"Gesamtbetrag der Einkünfte", preview.IncomeBeforeLoss, amount(kst, "kst.zve.gesamtbetrag"), 24_750_000},
		{"zu versteuerndes Einkommen", preview.TaxableIncome, amount(kst, "kst.zve.zve"), 19_750_000},
		{"Körperschaftsteuer", preview.CorporateTax, amount(kst, "kst.k.kst"), 2_962_500},
		{"Solidaritätszuschlag", preview.Solidarity, amount(kst, "kst.k.solz"), 162_937},
		{"Gewerbeertrag", preview.TradeIncome, amount(gewst, "gewst.k.ertrag_gerundet"), 19_750_000},
		{"Messbetrag", preview.TradeBase, amount(gewst, "gewst.k.messbetrag"), 691_250},
		{"Gewerbesteuer", preview.TradeTax, amount(gewst, "gewst.k.steuer"), 2_834_125},
		{"Abrechnung KSt und SolZ", preview.CorporateDue + preview.SolidarityDue, amount(kst, "kst.k.ergebnis"), 1_806_585},
	} {
		if c.provision != c.want || c.decl != c.want {
			t.Errorf("%s: Rückstellung %s €, Erklärung %s €, erwartet %s €", c.what, c.provision, c.decl, c.want)
		}
	}

	// Nach dem Buchen der Rückstellung meldet keine Kontrollrechnung eine
	// Abweichung.
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.ClosingBookings.BookTaxProvision(ctx, service.TaxProvisionRequest{FiscalYear: 2026, Options: opts})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []declaration.Kind{declaration.KindCorporateTax, declaration.KindTradeTax} {
		d, err := tenant.Declaration(uiCtx, kind, 2026)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range d.Findings {
			if f.Code == "rueckstellung_abweichung" {
				t.Errorf("%s: %s", kind, f.Text)
			}
		}
	}
}
