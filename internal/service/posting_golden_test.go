package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestGoldenPreviewIncomingReceipt(t *testing.T) {
	cases := []string{
		"inland_19", "inland_7", "inland_gemischt", "bewirtung_70_30",
		"geschenk_unter_grenze", "geschenk_ueber_grenze", "geschenk_grenze_kumuliert",
		"vorsteueranteil", "vorsteueranteil_null", "vorsteuerausschluss",
		"anzahlung", "rueckstellungsverbrauch", "gesellschafterauslage", "firmenkreditkarte",
		"fremdwaehrung", "13b_inland", "13b_eu", "ig_erwerb",
		"kleinbetragsrechnung", "pflichtangabe_fehlt", "uebersteuerung", "konto_statt_gruppe",
		"e_rechnung_befunde", "e_rechnung_uebersteuerung", "leistungsnachweis_fehlt",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			env := newTestEnv(t)
			country, vatID := "DE", ""
			switch name {
			case "13b_eu":
				country, vatID = "IE", "IE6388047V"
			case "ig_erwerb":
				country, vatID = "NL", "NL123456789B01"
			case "fremdwaehrung":
				country = "US"
			}
			vendor := env.vendor(t, "Lieferant GmbH", country, vatID)
			req := env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
			switch name {
			case "inland_7":
				req.Positions[0].TaxRate = domain.TaxRateReduced
			case "inland_gemischt":
				req.Positions = []ReceiptPosition{{PostingGroup: "reisekosten", Net: 20000, TaxRate: domain.TaxRateReduced}, {PostingGroup: "reisekosten", Net: 5000, TaxRate: domain.TaxRateStandard}}
			case "bewirtung_70_30":
				req.Positions[0].PostingGroup = "bewirtung"
				req.Entertainment = entertainmentRecord()
			case "geschenk_unter_grenze", "geschenk_ueber_grenze", "geschenk_grenze_kumuliert":
				env.gifts(t)
				req.Positions[0].PostingGroup = "geschenke"
				req.Positions[0].Gift = &GiftInput{Name: "Dr. Meyer", Occasion: "Jubiläum"}
				req.Positions[0].Net = 4000
				if name == "geschenk_ueber_grenze" {
					req.Positions[0].Net = 6000
				}
				if name == "geschenk_grenze_kumuliert" {
					if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
						t.Fatal(err)
					}
					req = env.giftReceipt(t, vendor.ID, "Dr. Meyer", 2000)
				}
			case "vorsteueranteil", "vorsteueranteil_null":
				req.Positions[0].PostingGroup = "fahrzeugkosten"
				req.Positions[0].InputTaxShare = 600
				req.Positions[0].InputTaxShareReason = "60 % steuerpflichtige Nutzung nach Fahrtenbuch"
				if name == "vorsteueranteil_null" {
					req.Positions[0].InputTaxShare = 0
				}
			case "vorsteuerausschluss":
				env.gifts(t)
				req.Positions[0].PostingGroup = "repraesentation"
			case "anzahlung":
				env.postingWithAdvances(t)
				req.AdvanceTarget = accounting.AdvanceTargetInventory
				req.Settlement, req.PaymentAccount = SettlementPaid, domain.AccountBank
			case "rueckstellungsverbrauch":
				modules := env.closingModules(t)
				provision, err := modules.provisions.BookFormation(ctx, ProvisionRequest{FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Jahresabschluss 2026", Amount: 300000, ExpectedOn: "2027-06-30", Reason: "Angebot der Kanzlei"})
				if err != nil {
					t.Fatal(err)
				}
				req.ProvisionID = provision.ID
				req.Positions = []ReceiptPosition{{Account: domain.AccountAbschlusskosten, Net: 350000, TaxRate: domain.TaxRateStandard, Text: "Jahresabschluss 2026"}}
			case "gesellschafterauslage":
				env.withShareholders(t)
				person := env.person(t, "Mara Beispiel")
				req.Settlement, req.CreditorContactID = SettlementShareholderExpense, person.ID
				req.Positions[0].PostingGroup = "buerobedarf"
			case "firmenkreditkarte":
				req.Settlement = SettlementCreditCard
			case "fremdwaehrung":
				// Der vorhandene Fake zählt Netzaufrufe; beide benötigten Kurse werden
				// vorab gespeichert. Die Vorschau muss vollständig aus dem Repository lesen.
				server, calls := ecbServer(t, "1.0850", "2026-03-10")
				currency := env.currencyPosting(t, server.URL)
				if _, err := currency.SaveRate(ctx, domain.ExchangeRate{Currency: "USD", Date: "2026-03-10", RateMicros: 1085000, Source: "EZB-Referenzkurs aus Testbestand"}); err != nil {
					t.Fatal(err)
				}
				if _, err := currency.SaveVatRate(ctx, domain.VatExchangeRate{Month: "2026-03", Currency: "USD", RateMicros: 1100000}); err != nil {
					t.Fatal(err)
				}
				req.Currency, req.ForeignAmount, req.Positions[0].Net = "USD", 108500, 108500
				t.Cleanup(func() {
					if *calls != 0 {
						t.Errorf("Vorschau hat %d Netzaufrufe ausgeführt", *calls)
					}
				})
			case "13b_inland":
				req.TaxTreatment = domain.TaxTreatmentReverseCharge
			case "13b_eu":
				req.TaxTreatment = domain.TaxTreatmentReverseChargeEU
			case "ig_erwerb":
				req.TaxTreatment = domain.TaxTreatmentIntraCommunityAcquisition
				req.Positions[0].PostingGroup = "wareneingang"
			case "kleinbetragsrechnung", "pflichtangabe_fehlt", "uebersteuerung":
				incomplete := env.incompleteVendor(t, "Tankstelle Nord")
				receipt := env.fileSmall(t, domain.ReceiptKindInvoice, "Tankstelle Nord", 5950, "")
				req = smallRequest(incomplete.ID, receipt.ID, "fahrzeugkosten", 5000, domain.TaxRateStandard)
				if name != "kleinbetragsrechnung" {
					req.Positions[0].Net = 30000
				}
				if name == "uebersteuerung" {
					req.OverrideReason = "Berichtigte Rechnung liegt vor"
				}
			case "konto_statt_gruppe":
				req.Positions[0].PostingGroup = ""
				req.Positions[0].Account = "6815"
			case "e_rechnung_befunde", "e_rechnung_uebersteuerung":
				receipt := env.structuredReceipt(t, 3)
				req.ReceiptID = receipt.ID
				if name == "e_rechnung_uebersteuerung" {
					req.OverrideReason = "Pflichtangaben anhand des Originals geprüft"
				}
			case "leistungsnachweis_fehlt":
				settings := repository.NewSettingsRepository(env.db)
				cfg, err := settings.GetCompanySettings(ctx)
				if err != nil {
					t.Fatal(err)
				}
				cfg.InvoiceCheckThreshold = 10000
				if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
					t.Fatal(err)
				}
				env.posting.SetSettingsSource(settings)
			}
			preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			// Auto-IDs kennzeichnen hier nur die isolierten Fixtures. Der Rollenbezug
			// bleibt als feste ID erhalten; Satzreihenfolge, Konten und Beträge bleiben.
			for i := range preview.Lines {
				line := &preview.Lines[i]
				line.ID, line.EntryID = 0, 0
				if line.ContactID != nil {
					normalized := uint(1)
					switch *line.ContactID {
					case req.ContactID:
					case req.CreditorContactID:
						normalized = 2
					default:
						t.Fatalf("unbekannter Kontakt in der Vorschau: %d", *line.ContactID)
					}
					line.ContactID = &normalized
				}
			}
			// Erfassungszeiten und die Datenbank-ID eines Kurses ändern sich je Lauf;
			// Kursdatum, Quelle, Sätze und sämtliche Rechenergebnisse bleiben erhalten.
			if preview.Conversion != nil {
				preview.Conversion.Rate.ID = 0
				preview.Conversion.Rate.CreatedAt = time.Time{}
				if preview.Conversion.VatRate != nil {
					preview.Conversion.VatRate.UpdatedAt = time.Time{}
				}
			}
			// Die automatisch vergebene Belegnummer in Hinweisen ist kein
			// Bestandteil der Berechnung; der Verweis auf diesen Beleg bleibt.
			receipt, err := env.receipts.Get(ctx, req.ReceiptID)
			if err != nil {
				t.Fatal(err)
			}
			for i := range preview.Warnings {
				warning := &preview.Warnings[i]
				warning.Detail = strings.ReplaceAll(warning.Detail, receipt.ReceiptNumber, "<Beleg>")
				warning.SupplierNote = strings.ReplaceAll(warning.SupplierNote, receipt.ReceiptNumber, "<Beleg>")
			}

			for i := range preview.InputTaxFindings {
				finding := &preview.InputTaxFindings[i]
				finding.Detail = strings.ReplaceAll(finding.Detail, receipt.ReceiptNumber, "<Beleg>")
			}

			assertGolden(t, "posting_preview/"+name, preview)
		})
	}
}
