package xrechnung

import (
	"sort"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/einvoice"
)

// validXRechnung builds a document that satisfies both EN 16931 and the German
// CIUS. Every test below changes exactly one thing.
func validXRechnung() *einvoice.Invoice {
	return &einvoice.Invoice{
		Syntax:          einvoice.SyntaxCII,
		SpecificationID: Current,
		Number:          "RE-2026-0001",
		IssueDate:       einvoice.NewDate("2026-03-15"),
		TypeCode:        "380",
		Currency:        "EUR",
		BuyerReference:  "991-33333TEST-33",
		Seller: einvoice.Party{
			Name:          "Muster GmbH",
			VATIdentifier: "DE123456789",
			Address:       &einvoice.Address{City: "Berlin", PostCode: "10115", CountryCode: "DE"},
			Contact: &einvoice.Contact{
				Name: "Anna Muster", Phone: "+49 30 1234567", Email: "rechnung@muster.example",
			},
		},
		Buyer: einvoice.Party{
			Name:    "Kunde AG",
			Address: &einvoice.Address{City: "Hamburg", PostCode: "20095", CountryCode: "DE"},
		},
		Delivery: &einvoice.Delivery{Date: einvoice.NewDate("2026-03-15")},
		PaymentMeans: []einvoice.PaymentMeans{{
			TypeCode:       "58",
			CreditTransfer: []einvoice.CreditTransfer{{AccountID: "DE02120300000000202051"}},
		}},
		Lines: []einvoice.Line{{
			ID:        "1",
			Quantity:  einvoice.NewAmount("1"),
			UnitCode:  "C62",
			NetAmount: einvoice.NewAmount("1000.00"),
			Price:     einvoice.Price{NetPrice: einvoice.NewAmount("1000.00")},
			VAT:       einvoice.LineVAT{CategoryCode: "S", Rate: einvoice.NewAmount("19.00")},
			Item:      einvoice.Item{Name: "Beratung"},
		}},
		VATBreakdown: []einvoice.VATBreakdown{{
			TypeCode: "VAT", TaxableAmount: einvoice.NewAmount("1000.00"),
			TaxAmount: einvoice.NewAmount("190.00"), CategoryCode: "S",
			Rate: einvoice.NewAmount("19.00"),
		}},
		Totals: einvoice.Totals{
			LineTotal: einvoice.NewAmount("1000.00"), TaxBasisTotal: einvoice.NewAmount("1000.00"),
			TaxTotal: einvoice.NewAmount("190.00"), TaxTotalCount: 1,
			GrandTotal: einvoice.NewAmount("1190.00"), DuePayableAmount: einvoice.NewAmount("1190.00"),
		},
	}
}

// Der Ausgangspunkt muss beide Ebenen bestehen — die Norm und die deutsche
// Ausprägung. Sonst prüfen die Tests darunter nichts.
func TestValidXRechnungHasNoFindings(t *testing.T) {
	result := einvoice.ValidateWith(validXRechnung(), Ruleset())
	for _, f := range result.Findings {
		t.Errorf("%s %s: %s", f.Rule, f.Where, f.Message)
	}
	want := []string{einvoice.RulesetVersion, Version}
	if strings.Join(result.Rulesets, " ") != strings.Join(want, " ") {
		t.Errorf("gelaufen sind %v, erwartet %v", result.Rulesets, want)
	}
}

func mustReport(t *testing.T, rule string, change func(*einvoice.Invoice)) {
	t.Helper()
	inv := validXRechnung()
	change(inv)
	result := einvoice.ValidateWith(inv, Ruleset())
	if len(result.ByRule(rule)) == 0 {
		var got []string
		for _, f := range result.Findings {
			got = append(got, f.Rule)
		}
		t.Errorf("%s hätte anschlagen müssen, gemeldet wurde: %s", rule, strings.Join(got, " "))
	}
}

// XRechnung macht den Verkäufer erreichbar. Das ist keine Förmlichkeit: ein
// öffentlicher Auftraggeber, der zu einer Rechnung nicht nachfragen kann, muss
// sie zurückweisen.
func TestSellerMustBeReachable(t *testing.T) {
	mustReport(t, "BR-DE-2", func(i *einvoice.Invoice) { i.Seller.Contact = nil })
	mustReport(t, "BR-DE-3", func(i *einvoice.Invoice) { i.Seller.Address.City = "" })
	mustReport(t, "BR-DE-4", func(i *einvoice.Invoice) { i.Seller.Address.PostCode = "" })
	mustReport(t, "BR-DE-5", func(i *einvoice.Invoice) { i.Seller.Contact.Name = "" })
	mustReport(t, "BR-DE-6", func(i *einvoice.Invoice) { i.Seller.Contact.Phone = "" })
	mustReport(t, "BR-DE-7", func(i *einvoice.Invoice) { i.Seller.Contact.Email = "" })
	mustReport(t, "BR-DE-27", func(i *einvoice.Invoice) { i.Seller.Contact.Phone = "n. a." })
	mustReport(t, "BR-DE-28", func(i *einvoice.Invoice) { i.Seller.Contact.Email = "rechnung.muster.example" })
}

func TestBuyerAndDeliveryAddressesAreRequired(t *testing.T) {
	mustReport(t, "BR-DE-8", func(i *einvoice.Invoice) { i.Buyer.Address.City = "" })
	mustReport(t, "BR-DE-9", func(i *einvoice.Invoice) { i.Buyer.Address.PostCode = "" })
	mustReport(t, "BR-DE-10", func(i *einvoice.Invoice) {
		i.Delivery.Address = &einvoice.Address{PostCode: "10115", CountryCode: "DE"}
	})
	mustReport(t, "BR-DE-11", func(i *einvoice.Invoice) {
		i.Delivery.Address = &einvoice.Address{City: "Berlin", CountryCode: "DE"}
	})
}

// Ohne Käuferreferenz kommt die Rechnung im Behördennetz nicht an: bei
// öffentlichen Auftraggebern steht dort die Leitweg-ID.
func TestBuyerReferenceIsMandatory(t *testing.T) {
	mustReport(t, "BR-DE-15", func(i *einvoice.Invoice) { i.BuyerReference = "" })
}

// Eine Korrektur ohne Bezug lässt offen, was sie korrigiert.
func TestCorrectionNeedsItsReference(t *testing.T) {
	mustReport(t, "BR-DE-26", func(i *einvoice.Invoice) { i.TypeCode = "384" })
	mustReport(t, "BR-DE-17", func(i *einvoice.Invoice) { i.TypeCode = "386" })

	inv := validXRechnung()
	inv.TypeCode = "384"
	inv.PrecedingInvoices = []einvoice.PrecedingInvoice{{Number: "RE-2025-0042"}}
	if findings := einvoice.ValidateWith(inv, Ruleset()).ByRule("BR-DE-26"); len(findings) > 0 {
		t.Errorf("mit Bezug darf BR-DE-26 nicht anschlagen: %s", findings[0].Message)
	}
}

// Eine Zahlungsanweisung muss ausführbar sein. Ein Beleg, der "Karte" sagt und
// dann ein Bankkonto nennt, ist keine Anweisung, sondern ein Rätsel.
func TestPaymentInstructionsMustBeExecutable(t *testing.T) {
	mustReport(t, "BR-DE-1", func(i *einvoice.Invoice) { i.PaymentMeans = nil })
	mustReport(t, "BR-DE-23-a", func(i *einvoice.Invoice) {
		i.PaymentMeans[0].CreditTransfer = nil
	})
	mustReport(t, "BR-DE-23-b", func(i *einvoice.Invoice) {
		i.PaymentMeans[0].Card = &einvoice.CardInformation{PrimaryAccountNumber: "1234"}
	})
	mustReport(t, "BR-DE-24-a", func(i *einvoice.Invoice) {
		i.PaymentMeans[0] = einvoice.PaymentMeans{TypeCode: "48"}
	})
	mustReport(t, "BR-DE-25-a", func(i *einvoice.Invoice) {
		i.PaymentMeans[0] = einvoice.PaymentMeans{TypeCode: "59"}
	})
	mustReport(t, "BR-DE-30", func(i *einvoice.Invoice) {
		i.PaymentMeans[0] = einvoice.PaymentMeans{
			TypeCode:    "59",
			DirectDebit: &einvoice.DirectDebit{MandateReference: "M1", DebitedAccount: "DE02120300000000202051"},
		}
	})
	mustReport(t, "BR-DE-31", func(i *einvoice.Invoice) {
		i.CreditorReference = "DE98ZZZ09999999999"
		i.PaymentMeans[0] = einvoice.PaymentMeans{
			TypeCode:    "59",
			DirectDebit: &einvoice.DirectDebit{MandateReference: "M1"},
		}
	})
}

// Die Prüfziffer fängt genau den Fehler, der wirklich passiert: zwei vertauschte
// Stellen — und der schickt das Geld zu einem Fremden.
func TestIBANIsCheckedProperly(t *testing.T) {
	valid := []string{
		"DE02120300000000202051",
		"DE02 1203 0000 0000 2020 51",
		"AT026000000001349870",
		"NL91ABNA0417164300",
	}
	for _, iban := range valid {
		if !ValidIBAN(iban) {
			t.Errorf("%q ist eine gültige IBAN", iban)
		}
	}
	invalid := []string{
		"DE02120300000000202052", // Prüfziffer falsch
		"DE20120300000000202051", // vertauschte Stellen
		"DE0212030000000020205",  // zu kurz für DE, Prüfziffer bricht
		"1234567890",             // keine Länderkennung
		"",
	}
	for _, iban := range invalid {
		if ValidIBAN(iban) {
			t.Errorf("%q ist keine gültige IBAN", iban)
		}
	}

	mustReport(t, "BR-DE-19", func(i *einvoice.Invoice) {
		i.PaymentMeans[0].CreditTransfer[0].AccountID = "DE02120300000000202052"
	})
}

// Skonto hat in EN 16931 kein Feld. XRechnung schreibt deshalb eine Form für
// die Zahlungsbedingungen vor, damit es maschinell lesbar bleibt statt in Prosa
// zu verschwinden.
func TestSkontoFollowsItsFormat(t *testing.T) {
	good := validXRechnung()
	good.PaymentTermsNote = "Zahlbar bis 15.04.2026\n#SKONTO#TAGE=14#PROZENT=2.00#"
	if findings := einvoice.ValidateWith(good, Ruleset()).ByRule("BR-DE-18"); len(findings) > 0 {
		t.Errorf("die vorgeschriebene Form darf nicht anschlagen: %s", findings[0].Message)
	}
	withBase := validXRechnung()
	withBase.PaymentTermsNote = "#SKONTO#TAGE=14#PROZENT=2.00#BASISBETRAG=1190.00#"
	if findings := einvoice.ValidateWith(withBase, Ruleset()).ByRule("BR-DE-18"); len(findings) > 0 {
		t.Errorf("die Form mit Basisbetrag darf nicht anschlagen: %s", findings[0].Message)
	}

	mustReport(t, "BR-DE-18", func(i *einvoice.Invoice) {
		i.PaymentTermsNote = "#SKONTO#TAGE=14#PROZENT=2#"
	})
}

// XRechnung ist strenger als die Norm: den Steuersatz verlangt sie auch dort,
// wo EN 16931 ihn weglässt.
func TestVATRateIsAlwaysRequired(t *testing.T) {
	mustReport(t, "BR-DE-14", func(i *einvoice.Invoice) {
		i.Lines[0].VAT = einvoice.LineVAT{CategoryCode: "O"}
		i.VATBreakdown[0] = einvoice.VATBreakdown{
			TypeCode: "VAT", CategoryCode: "O",
			TaxableAmount: einvoice.NewAmount("1000.00"), TaxAmount: einvoice.NewAmount("0.00"),
			ExemptionReason: "nicht steuerbar",
		}
		i.Seller.VATIdentifier = ""
		i.Totals.TaxTotal = einvoice.NewAmount("0.00")
		i.Totals.GrandTotal = einvoice.NewAmount("1000.00")
		i.Totals.DuePayableAmount = einvoice.NewAmount("1000.00")
	})
	mustReport(t, "BR-DE-16", func(i *einvoice.Invoice) { i.Seller.VATIdentifier = "" })
}

// Zwei Anlagen mit demselben Dateinamen: eine davon überschreibt die andere,
// sobald jemand sie ablegt.
func TestAttachmentsNeedDistinctNames(t *testing.T) {
	mustReport(t, "BR-DE-22", func(i *einvoice.Invoice) {
		i.SupportingDocs = []einvoice.SupportingDocument{
			{Reference: "A1", Filename: "nachweis.pdf", MimeCode: "application/pdf", Attachment: []byte("x")},
			{Reference: "A2", Filename: "nachweis.pdf", MimeCode: "application/pdf", Attachment: []byte("y")},
		}
	})
	mustReport(t, "BR-DEX-01", func(i *einvoice.Invoice) {
		i.SupportingDocs = []einvoice.SupportingDocument{
			{Reference: "A1", Filename: "a.exe", MimeCode: "application/octet-stream", Attachment: []byte("x")},
		}
	})
	mustReport(t, "BR-TMP-2", func(i *einvoice.Invoice) {
		i.SupportingDocs = []einvoice.SupportingDocument{{Reference: "A1", ExternalURI: "irgendwo/datei.pdf"}}
	})
}

// Die Regeln für saubere Fahrzeuge laufen nur auf Verlangen — dem Dokument
// sieht man das Vergabeszenario nicht an.
func TestCleanVehicleRulesAreOptional(t *testing.T) {
	inv := validXRechnung()
	if findings := einvoice.ValidateWith(inv, Ruleset()).ByRule("BR-DE-CVD-01"); len(findings) > 0 {
		t.Error("ohne Anforderung dürfen die CVD-Regeln nicht laufen")
	}

	result := einvoice.ValidateWith(inv, Ruleset(WithCleanVehicles()))
	for _, rule := range []string{"BR-DE-CVD-01", "BR-DE-CVD-02", "BR-DE-CVD-03"} {
		if len(result.ByRule(rule)) == 0 {
			t.Errorf("%s hätte bei eingeschalteter Prüfung anschlagen müssen", rule)
		}
	}

	vehicle := validXRechnung()
	vehicle.ContractReference = "V-2026-1"
	vehicle.TenderReference = "A-2026-1"
	vehicle.Lines[0].Item.Classifications = []einvoice.Identifier{{Value: "M1", Scheme: "CVD"}}
	vehicle.Lines[0].Item.Attributes = []einvoice.ItemAttribute{{Name: "cva", Value: "cvd-1"}}
	for _, f := range einvoice.ValidateWith(vehicle, Ruleset(WithCleanVehicles())).Findings {
		if strings.HasPrefix(f.Rule, "BR-DE-CVD") {
			t.Errorf("%s: %s", f.Rule, f.Message)
		}
	}

	wrong := validXRechnung()
	wrong.ContractReference = "V-2026-1"
	wrong.TenderReference = "A-2026-1"
	wrong.Lines[0].Item.Classifications = []einvoice.Identifier{{Value: "X9", Scheme: "CVD"}}
	wrong.Lines[0].Item.Attributes = []einvoice.ItemAttribute{{Name: "cva", Value: "cvd-9"}}
	result = einvoice.ValidateWith(wrong, Ruleset(WithCleanVehicles()))
	for _, rule := range []string{"BR-DE-CVD-04", "BR-DE-CVD-05"} {
		if len(result.ByRule(rule)) == 0 {
			t.Errorf("%s hätte anschlagen müssen", rule)
		}
	}
}

// Was nicht geprüft wird, wird benannt. Schweigen über eine Lücke ist schlimmer
// als die Lücke.
func TestUncheckedRulesAreNamedWithAReason(t *testing.T) {
	checked := map[string]bool{}
	for _, rule := range CheckedRules() {
		checked[rule] = true
	}
	unchecked := UncheckedRules()

	var missing []string
	for rule := range xrechnungRules {
		if !checked[rule] && unchecked[rule] == "" {
			missing = append(missing, rule)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("weder geprüft noch begründet: %s", strings.Join(missing, " "))
	}
	t.Logf("XRechnung: %d von %d Regeln geprüft", len(checked), len(xrechnungRules))
	for rule, reason := range unchecked {
		if reason == "" {
			t.Errorf("%s ist ohne Begründung ausgenommen", rule)
		}
	}
}

// Jede zugesagte Regel muss es in der Spezifikation geben.
func TestClaimedRulesExistInTheSpecification(t *testing.T) {
	for _, rule := range CheckedRules() {
		if _, ok := xrechnungRules[rule]; !ok {
			t.Errorf("%q wird geprüft, steht aber nicht im Regelverzeichnis", rule)
		}
	}
}

func TestProfileIdentifiersAreRecognised(t *testing.T) {
	for _, id := range []string{IdentifierV12, IdentifierV20, IdentifierV23, IdentifierV30, IdentifierV30Legacy} {
		if !Applies(&einvoice.Invoice{SpecificationID: id}) {
			t.Errorf("%s wird nicht als XRechnung erkannt", id)
		}
	}
	if Applies(&einvoice.Invoice{SpecificationID: einvoice.ProfileEN16931}) {
		t.Error("reines EN 16931 ist keine XRechnung")
	}
	if !UsesExtension(&einvoice.Invoice{SpecificationID: IdentifierV22 + "#conformant#urn:xoev-de:kosit:extension:xrechnung_2.2"}) {
		t.Error("die Extension wird nicht erkannt")
	}
}
