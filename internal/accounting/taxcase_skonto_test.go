package accounting

import (
	"reflect"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestTaxCaseSkonto(t *testing.T) {
	type correction struct {
		account, key string
		side         domain.Side
		amount       domain.Cents
	}
	for _, tc := range []struct {
		name       string
		treatment  domain.TaxTreatment
		direction  domain.Direction
		rate       domain.TaxRate
		gross, net domain.Cents
		account    string
		legs       []correction
	}{
		{"vorsteuer19", domain.TaxTreatmentDomestic, domain.DirectionIncoming, 1900, 1190, 1000, "5736", []correction{{"1406", "SKONTO_VST19", domain.SideCredit, 190}}},
		{"vorsteuer7", domain.TaxTreatmentDomestic, domain.DirectionIncoming, 700, 1070, 1000, "5731", []correction{{"1401", "SKONTO_VST7", domain.SideCredit, 70}}},
		{"umsatzsteuer19", domain.TaxTreatmentDomestic, domain.DirectionOutgoing, 1900, 1190, 1000, "4736", []correction{{"3806", "SKONTO_UST19", domain.SideDebit, 190}}},
		{"umsatzsteuer7", domain.TaxTreatmentDomestic, domain.DirectionOutgoing, 700, 1070, 1000, "4731", []correction{{"3801", "SKONTO_UST7", domain.SideDebit, 70}}},
		{"13b_abs1_19", domain.TaxTreatmentReverseChargeEU, domain.DirectionIncoming, 1900, 2000, 2000, "5730", []correction{{"1407", "SKONTO_RCEU19_VST", domain.SideCredit, 380}, {"3837", "SKONTO_RCEU19_UST", domain.SideDebit, 380}}},
		{"13b_abs1_7", domain.TaxTreatmentReverseChargeEU, domain.DirectionIncoming, 700, 2000, 2000, "5730", []correction{{"1408", "SKONTO_RCEU7_VST", domain.SideCredit, 140}, {"3835", "SKONTO_RCEU7_UST", domain.SideDebit, 140}}},
		{"13b_abs2_19", domain.TaxTreatmentReverseCharge, domain.DirectionIncoming, 1900, 2000, 2000, "5730", []correction{{"1407", "SKONTO_RC19_VST", domain.SideCredit, 380}, {"3837", "SKONTO_RC19_UST", domain.SideDebit, 380}}},
		{"13b_abs2_7", domain.TaxTreatmentReverseCharge, domain.DirectionIncoming, 700, 2000, 2000, "5730", []correction{{"1408", "SKONTO_RC7_VST", domain.SideCredit, 140}, {"3835", "SKONTO_RC7_UST", domain.SideDebit, 140}}},
		{"ig19", domain.TaxTreatmentIntraCommunityAcquisition, domain.DirectionIncoming, 1900, 2000, 2000, "5730", []correction{{"1404", "SKONTO_IG19_VST", domain.SideCredit, 380}, {"3804", "SKONTO_IG19_UST", domain.SideDebit, 380}}},
		{"ig7", domain.TaxTreatmentIntraCommunityAcquisition, domain.DirectionIncoming, 700, 2000, 2000, "5730", []correction{{"1402", "SKONTO_IG7_VST", domain.SideCredit, 140}, {"3802", "SKONTO_IG7_UST", domain.SideDebit, 140}}},
		{"13b_leistender", domain.TaxTreatmentReverseChargeSupply, domain.DirectionOutgoing, 0, 2000, 2000, "4741", nil},
		{"steuerfrei_eingang", domain.TaxTreatmentExempt, domain.DirectionIncoming, 0, 1190, 1190, "5730", nil},
		{"steuerfrei_ausgang", domain.TaxTreatmentExempt, domain.DirectionOutgoing, 0, 1190, 1190, "4734", nil},
		{"nicht_steuerbar_eingang", domain.TaxTreatmentNotTaxable, domain.DirectionIncoming, 0, 1190, 1190, "5730", nil},
		{"nicht_steuerbar_ausgang", domain.TaxTreatmentNotTaxable, domain.DirectionOutgoing, 0, 1190, 1190, "4734", nil},
		{"nullsteuersatz_eingang", domain.TaxTreatmentZeroRated, domain.DirectionIncoming, 0, 1190, 1190, "5730", nil},
		{"nullsteuersatz_ausgang", domain.TaxTreatmentZeroRated, domain.DirectionOutgoing, 0, 1190, 1190, "4734", nil},
		{"ohne_betrag", domain.TaxTreatmentDomestic, domain.DirectionIncoming, 1900, 0, 0, "5736", nil},
		{"ein_cent", domain.TaxTreatmentDomestic, domain.DirectionIncoming, 1900, 1, 1, "5736", nil},
		{"rundung", domain.TaxTreatmentDomestic, domain.DirectionIncoming, 1900, 3, 3, "5736", []correction{{"1406", "SKONTO_VST19", domain.SideCredit, 1}}},
		{"negativ", domain.TaxTreatmentDomestic, domain.DirectionIncoming, 1900, -1190, -1000, "5736", []correction{{"1406", "SKONTO_VST19", domain.SideCredit, -190}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := TaxCaseFor(tc.treatment).Skonto(tc.direction, tc.rate, tc.gross)
			if err != nil {
				t.Fatal(err)
			}
			want := TaxCaseSkonto{Account: tc.account, Net: tc.net, Legs: []domain.TaxLeg{}}
			for _, leg := range tc.legs {
				want.Legs = append(want.Legs, domain.TaxLeg{Account: leg.account, Key: leg.key, Side: leg.side, Amount: leg.amount, Base: tc.net})
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Skonto = %+v, erwartet %+v", got, want)
			}
		})
	}
}

func TestTaxCaseSkontoRejectsUnknownTax(t *testing.T) {
	for _, tc := range []struct {
		treatment domain.TaxTreatment
		direction domain.Direction
		rate      domain.TaxRate
	}{
		{"unbekannt", domain.DirectionIncoming, 1900},
		{"", domain.DirectionIncoming, 1900},
		{domain.TaxTreatmentDomestic, domain.DirectionIncoming, 0},
		{domain.TaxTreatmentDomestic, domain.DirectionOutgoing, 500},
		{domain.TaxTreatmentReverseCharge, domain.DirectionIncoming, 0},
		{domain.TaxTreatmentReverseChargeEU, domain.DirectionOutgoing, 1900},
		{domain.TaxTreatmentIntraCommunityAcquisition, domain.DirectionOutgoing, 1900},
		{domain.TaxTreatmentIntraCommunityAcquisition, domain.DirectionIncoming, 0},
	} {
		if _, err := TaxCaseFor(tc.treatment).Skonto(tc.direction, tc.rate, 1190); err == nil {
			t.Errorf("Steuerfall %q, Richtung %s, Satz %d: erwartet Fehler", tc.treatment, tc.direction, tc.rate)
		}
	}
}
