package accounting

import (
	"reflect"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestTaxCaseCatalog(t *testing.T) {
	type leg struct{ account, key, code string }
	rows := []struct {
		treatment domain.TaxTreatment
		direction domain.Direction
		legs      []leg
		date      string
		code      string
		category  string
		skonto    string
		vatID, eu bool
	}{
		{domain.TaxTreatmentDomestic, domain.DirectionIncoming, []leg{{"1406", "VST19", "66"}}, "2026-04-01", "", "S", "5736", false, false},
		{domain.TaxTreatmentDomestic, domain.DirectionOutgoing, []leg{{"3806", "UST19", "81"}}, "2026-01-31", "", "S", "4736", false, false},
		{domain.TaxTreatmentReverseChargeEU, domain.DirectionIncoming, []leg{{"1407", "RCEU19_VST", "67"}, {"3837", "RCEU19_UST", "46"}}, "2026-01-31", "", "S", "5730", true, true},
		{domain.TaxTreatmentReverseCharge, domain.DirectionIncoming, []leg{{"1407", "RC19_VST", "67"}, {"3837", "RC19_UST", "84"}}, "2026-02-28", "", "S", "5730", false, false},
		{domain.TaxTreatmentIntraCommunityAcquisition, domain.DirectionIncoming, []leg{{"1404", "IG19_VST", "61"}, {"3804", "IG19_UST", "89"}}, "2026-02-28", "", "S", "5730", true, true},
		{domain.TaxTreatmentReverseChargeSupply, domain.DirectionOutgoing, nil, "2026-01-31", "45", "AE", "4741", true, false},
		{domain.TaxTreatmentZeroRated, domain.DirectionIncoming, nil, "2026-01-31", "87", "Z", "5730", false, false},
		{domain.TaxTreatmentZeroRated, domain.DirectionOutgoing, nil, "2026-01-31", "87", "Z", "4734", false, false},
		{domain.TaxTreatmentExempt, domain.DirectionIncoming, nil, "2026-01-31", "48", "E", "5730", false, false},
		{domain.TaxTreatmentExempt, domain.DirectionOutgoing, nil, "2026-01-31", "48", "E", "4734", false, false},
		{domain.TaxTreatmentNotTaxable, domain.DirectionIncoming, nil, "2026-01-31", "45", "O", "5730", false, false},
		{domain.TaxTreatmentNotTaxable, domain.DirectionOutgoing, nil, "2026-01-31", "45", "O", "4734", false, false},
	}
	covered := map[domain.TaxTreatment]bool{}
	entry := &domain.JournalEntry{BookingDate: "2026-05-10", DocumentDate: "2026-03-10", ServiceDateTo: "2026-01-31"}
	for _, row := range rows {
		t.Run(string(row.treatment)+"/"+string(row.direction), func(t *testing.T) {
			covered[row.treatment] = true
			c := TaxCaseFor(row.treatment)
			legs, err := c.Legs(row.direction, domain.TaxRateStandard, 10000)
			if err != nil {
				t.Fatal(err)
			}
			var got []leg
			for _, l := range legs {
				got = append(got, leg{l.Account, l.Key, c.VatCode(l.Key).Code})
				if l.Amount != 1900 || l.Base != 10000 {
					t.Fatalf("Steuerzeile: %+v", l)
				}
				if date := c.Timing(entry, domain.JournalLine{TaxKey: l.Key}, "2026-04-01"); date != row.date {
					t.Errorf("Datum %s: %s, erwartet %s", l.Key, date, row.date)
				}
			}
			if !reflect.DeepEqual(got, row.legs) {
				t.Errorf("Steuerzeilen: %+v, erwartet %+v", got, row.legs)
			}
			if len(legs) == 0 && c.Timing(entry, domain.JournalLine{}, "2026-04-01") != row.date {
				t.Error("Datum der Erlöszeile stimmt nicht")
			}
			if c.VatCode("").Code != row.code || c.InvoiceCategory() != row.category || c.MayShowTax() != (row.treatment == domain.TaxTreatmentDomestic) {
				t.Errorf("Eigenschaften: Kennzahl %+v, Kategorie %s, Steuerausweis %t", c.VatCode(""), c.InvoiceCategory(), c.MayShowTax())
			}
			check := c.CheckPartner(row.direction)
			if !check.Allowed || check.RequiresVatID != row.vatID || check.RequiresEU != row.eu {
				t.Errorf("Partnerprüfung: %+v", check)
			}
			if check.RequiresVatIDForEU != (row.treatment == domain.TaxTreatmentReverseCharge) || check.ConfirmEUVatID != (row.treatment == domain.TaxTreatmentReverseChargeSupply) {
				t.Errorf("EU-Partnerprüfung: %+v", check)
			}
			skonto, err := c.Skonto(row.direction, domain.TaxRateStandard, 1190)
			if err != nil {
				t.Fatal(err)
			}
			wantNet := domain.Cents(1190)
			if row.treatment == domain.TaxTreatmentDomestic {
				wantNet = 1000
			}
			if skonto.Account != row.skonto || skonto.Net != wantNet || len(skonto.Legs) != len(legs) {
				t.Errorf("Skonto: %+v", skonto)
			}
			for i, l := range skonto.Legs {
				if l.Account != legs[i].Account || l.Side != legs[i].Side.Opposite() || l.Key != "SKONTO_"+legs[i].Key || l.Base != wantNet || l.Amount != domain.TaxRateStandard.Tax(wantNet) {
					t.Errorf("Skonto-Steuerzeile: %+v", l)
				}
			}
		})
	}
	for _, treatment := range domain.AllTaxTreatments() {
		if !covered[treatment] {
			t.Errorf("Steuerfall %s fehlt in der Tabelle", treatment)
		}
	}
}

func TestTaxCaseReportingRules(t *testing.T) {
	for _, row := range []struct {
		key, code string
		side      domain.Side
		base      bool
		payment   TaxPrepayment
		longTerm  bool
		badDebt   bool
	}{
		{"UST19", "81", domain.SideCredit, true, TaxPrepaymentOnPayment, false, true},
		{"UST7", "86", domain.SideCredit, true, TaxPrepaymentOnPayment, false, true},
		{"VST19", "66", domain.SideDebit, false, TaxPrepaymentWithInvoice, false, false},
		{"VST7", "66", domain.SideDebit, false, TaxPrepaymentWithInvoice, false, false},
		{"IG19_UST", "89", domain.SideCredit, true, TaxPrepaymentNone, false, false},
		{"IG7_UST", "93", domain.SideCredit, true, TaxPrepaymentNone, false, false},
		{"IG7_VST", "61", domain.SideDebit, false, TaxPrepaymentNone, false, false},
		{"RC7_UST", "84", domain.SideCredit, true, TaxPrepaymentOnPayment, true, false},
		{"RCEU7_UST", "46", domain.SideCredit, true, TaxPrepaymentOnPayment, true, false},
		{"RC7_VST", "67", domain.SideDebit, false, TaxPrepaymentOnPayment, true, false},
		{"RCEU7_VST", "67", domain.SideDebit, false, TaxPrepaymentOnPayment, true, false},
		{"UST14C", "69", domain.SideCredit, false, TaxPrepaymentNone, false, false},
		{"VST15A", "64", domain.SideDebit, false, TaxPrepaymentNone, false, false},
		{"RC05_UST", "84", domain.SideCredit, true, TaxPrepaymentOnPayment, true, false},
		{"RCEU0_VST", "67", domain.SideDebit, false, TaxPrepaymentOnPayment, true, false},
		{"RCEU_UST", "", "", false, TaxPrepaymentNone, false, false},
		{"UNBEKANNT", "", "", false, TaxPrepaymentNone, false, false},
		{"SKONTO_UST19", "81", domain.SideCredit, true, TaxPrepaymentOnPayment, false, false},
		{"SKONTO_SKONTO_UST19", "", "", false, TaxPrepaymentNone, false, false},
	} {
		t.Run(row.key, func(t *testing.T) {
			want := TaxCaseVAT{Code: row.code, TaxSide: row.side, HasBase: row.base, Prepayment: row.payment, LongTerm: row.longTerm, BadDebtBase: row.badDebt}
			want.AllowClosing = row.key == "VST15A"
			if got := TaxCaseFor("").VatCode(row.key); got != want {
				t.Errorf("Melderegel: %+v, erwartet %+v", got, want)
			}
		})
	}
}

func TestTaxCaseSpecialKeysAndRecipients(t *testing.T) {
	for _, key := range []string{TaxKeyUnlawful, TaxKeyInputTaxCorrection} {
		if _, err := TaxCaseFor(domain.TaxTreatment(key)).Legs(domain.DirectionOutgoing, 1900, 10000); err == nil {
			t.Errorf("%s darf kein automatischer Steuerfall sein", key)
		}
	}
	for _, row := range []struct{ key, code string }{{"UST14C", "69"}, {"VST15A", "64"}} {
		if got := TaxCaseFor("").VatCode(row.key); got.Code != row.code || got.HasBase {
			t.Errorf("manueller Schlüssel %s: %+v", row.key, got)
		}
	}
	c := TaxCaseFor(domain.TaxTreatmentReverseChargeSupply)
	for _, row := range []struct {
		eu, domestic bool
		code, zm     string
	}{{false, false, "45", ""}, {false, true, "60", ""}, {true, false, "21", "S"}, {true, true, "21", "S"}} {
		got := c.VatCode("").ForRecipient(row.eu, row.domestic)
		if got.Code != row.code || string(got.ZMKind) != row.zm {
			t.Errorf("Empfänger EU=%t, Inland=%t: %+v", row.eu, row.domestic, got)
		}
	}
}
