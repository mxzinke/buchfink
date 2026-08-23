package einvoice

import (
	"strings"
	"testing"
)

// Die Regeln, für die das Artefakt keine Testdatei mitbringt, bekommen eigene.
//
// Ohne sie stünden 32 Prüfungen im Umfang, ohne dass je gezeigt wäre, dass sie
// anschlagen — und eine Prüfung, die nie ausgelöst wurde, ist eine Behauptung.

// validInvoice returns a document that violates nothing, as the starting point
// for the negative tests. Every test below changes exactly one thing.
func validInvoice() *Invoice {
	return &Invoice{
		Syntax:          SyntaxCII,
		SpecificationID: "urn:cen.eu:en16931:2017",
		Number:          "RE-2026-0001",
		IssueDate:       NewDate("2026-03-15"),
		TypeCode:        "380",
		Currency:        "EUR",
		Seller: Party{
			Name:          "Muster GmbH",
			VATIdentifier: "DE123456789",
			Address:       &Address{City: "Berlin", CountryCode: "DE"},
		},
		Buyer: Party{
			Name:    "Kunde AG",
			Address: &Address{City: "Hamburg", CountryCode: "DE"},
			Identifiers: []Identifier{
				{Value: "4025678", Scheme: "0088"},
			},
		},
		Lines: []Line{{
			ID:        "1",
			Quantity:  NewAmount("1"),
			UnitCode:  "C62",
			NetAmount: NewAmount("1000.00"),
			Price:     Price{NetPrice: NewAmount("1000.00")},
			VAT:       LineVAT{CategoryCode: "S", Rate: NewAmount("19.00")},
			Item:      Item{Name: "Beratung"},
		}},
		VATBreakdown: []VATBreakdown{{
			TypeCode:      "VAT",
			TaxableAmount: NewAmount("1000.00"),
			TaxAmount:     NewAmount("190.00"),
			CategoryCode:  "S",
			Rate:          NewAmount("19.00"),
		}},
		Totals: Totals{
			LineTotal:        NewAmount("1000.00"),
			TaxBasisTotal:    NewAmount("1000.00"),
			TaxTotal:         NewAmount("190.00"),
			TaxTotalCount:    1,
			GrandTotal:       NewAmount("1190.00"),
			DuePayableAmount: NewAmount("1190.00"),
		},
	}
}

// Der Ausgangspunkt muss sauber sein, sonst prüfen die Tests darunter nichts.
func TestValidInvoiceHasNoFindings(t *testing.T) {
	for _, f := range Validate(validInvoice()).Findings {
		t.Errorf("%s %s: %s", f.Rule, f.Where, f.Message)
	}
}

// mustReport runs the check and requires the named rule to fire.
func mustReport(t *testing.T, rule string, change func(*Invoice)) {
	t.Helper()
	inv := validInvoice()
	change(inv)
	result := Validate(inv)
	if len(result.ByRule(rule)) == 0 {
		var got []string
		for _, f := range result.Findings {
			got = append(got, f.Rule)
		}
		t.Errorf("%s hätte anschlagen müssen, gemeldet wurde: %s", rule, strings.Join(got, " "))
	}
}

// mustNotReport requires the named rule to stay silent.
func mustNotReport(t *testing.T, rule string, change func(*Invoice)) {
	t.Helper()
	inv := validInvoice()
	change(inv)
	if findings := Validate(inv).ByRule(rule); len(findings) > 0 {
		t.Errorf("%s hätte nicht anschlagen dürfen: %s", rule, findings[0].Message)
	}
}

// Die BR-DEC-Familie: kein Betrag der Rechnung trägt mehr als zwei
// Nachkommastellen. Ein dritter bedeutet einen Betrag, den kein Konto halten
// kann — und jede Summenregel danach würde ihn stillschweigend überspringen.
func TestDecimalLimitsAreEnforced(t *testing.T) {
	tooPrecise := NewAmount("100.005")

	cases := []struct {
		rule   string
		change func(*Invoice)
	}{
		{"BR-DEC-01", func(i *Invoice) { i.Allowances = []AllowanceCharge{allowance(tooPrecise)} }},
		{"BR-DEC-02", func(i *Invoice) {
			a := allowance(NewAmount("100.00"))
			a.BaseAmount = tooPrecise
			i.Allowances = []AllowanceCharge{a}
		}},
		{"BR-DEC-05", func(i *Invoice) { i.Charges = []AllowanceCharge{allowance(tooPrecise)} }},
		{"BR-DEC-06", func(i *Invoice) {
			a := allowance(NewAmount("100.00"))
			a.BaseAmount = tooPrecise
			i.Charges = []AllowanceCharge{a}
		}},
		{"BR-DEC-09", func(i *Invoice) { i.Totals.LineTotal = tooPrecise }},
		{"BR-DEC-10", func(i *Invoice) { i.Totals.AllowanceTotal = tooPrecise }},
		{"BR-DEC-11", func(i *Invoice) { i.Totals.ChargeTotal = tooPrecise }},
		{"BR-DEC-12", func(i *Invoice) { i.Totals.TaxBasisTotal = tooPrecise }},
		{"BR-DEC-13", func(i *Invoice) { i.Totals.TaxTotal = tooPrecise }},
		{"BR-DEC-14", func(i *Invoice) { i.Totals.GrandTotal = tooPrecise }},
		{"BR-DEC-15", func(i *Invoice) { i.Totals.TaxTotalInTaxCurr = tooPrecise }},
		{"BR-DEC-16", func(i *Invoice) { i.Totals.PrepaidAmount = tooPrecise }},
		{"BR-DEC-17", func(i *Invoice) { i.Totals.RoundingAmount = tooPrecise }},
		{"BR-DEC-18", func(i *Invoice) { i.Totals.DuePayableAmount = tooPrecise }},
		{"BR-DEC-19", func(i *Invoice) { i.VATBreakdown[0].TaxableAmount = tooPrecise }},
		{"BR-DEC-20", func(i *Invoice) { i.VATBreakdown[0].TaxAmount = tooPrecise }},
		{"BR-DEC-23", func(i *Invoice) { i.Lines[0].NetAmount = tooPrecise }},
		{"BR-DEC-24", func(i *Invoice) { i.Lines[0].Allowances = []AllowanceCharge{allowance(tooPrecise)} }},
		{"BR-DEC-25", func(i *Invoice) {
			a := allowance(NewAmount("100.00"))
			a.BaseAmount = tooPrecise
			i.Lines[0].Allowances = []AllowanceCharge{a}
		}},
		{"BR-DEC-27", func(i *Invoice) { i.Lines[0].Charges = []AllowanceCharge{allowance(tooPrecise)} }},
		{"BR-DEC-28", func(i *Invoice) {
			a := allowance(NewAmount("100.00"))
			a.BaseAmount = tooPrecise
			i.Lines[0].Charges = []AllowanceCharge{a}
		}},
	}
	for _, c := range cases {
		t.Run(c.rule, func(t *testing.T) { mustReport(t, c.rule, c.change) })
	}
}

func allowance(amount Amount) AllowanceCharge {
	return AllowanceCharge{
		Amount:      amount,
		VATCategory: "S",
		VATRate:     NewAmount("19.00"),
		Reason:      "Rabatt",
	}
}

// Die Codelisten, die der Korpus nicht berührt.
func TestRemainingCodeListsAreEnforced(t *testing.T) {
	cases := []struct {
		rule   string
		change func(*Invoice)
	}{
		{"BR-CL-08", func(i *Invoice) {
			i.Notes = []Note{{SubjectCode: "KEIN-CODE", Text: "Hinweis"}}
		}},
		{"BR-CL-22", func(i *Invoice) {
			i.VATBreakdown[0].CategoryCode = "E"
			i.VATBreakdown[0].Rate = NewAmount("0")
			i.VATBreakdown[0].TaxAmount = NewAmount("0.00")
			i.VATBreakdown[0].ExemptionReasonCode = "kein-schluessel"
		}},
		{"BR-CL-25", func(i *Invoice) {
			i.Seller.ElectronicAddress = Identifier{Value: "de@example.org", Scheme: "9999"}
		}},
		{"BR-CL-26", func(i *Invoice) {
			i.Delivery = &Delivery{
				LocationID: Identifier{Value: "L1", Scheme: "9999"},
				Address:    &Address{CountryCode: "DE"},
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.rule, func(t *testing.T) { mustReport(t, c.rule, c.change) })
	}
}

// Split Payment ist eine italienische Regelung. Auf einer deutschen Rechnung
// hat sie nichts zu suchen — und Buchfink hätte kein Konto dafür.
func TestSplitPaymentIsBoundToItaly(t *testing.T) {
	mustReport(t, "BR-B-01", func(i *Invoice) {
		i.VATBreakdown[0].CategoryCode = "B"
		i.Lines[0].VAT.CategoryCode = "B"
	})
	mustReport(t, "BR-B-02", func(i *Invoice) {
		i.Seller.Address.CountryCode = "IT"
		i.Buyer.Address.CountryCode = "IT"
		i.VATBreakdown = append(i.VATBreakdown, VATBreakdown{
			TypeCode: "VAT", CategoryCode: "B", Rate: NewAmount("22.00"),
			TaxableAmount: NewAmount("0.00"), TaxAmount: NewAmount("0.00"),
		})
	})
	// Eine rein italienische Rechnung ohne Regelsteuersatz ist in Ordnung.
	mustNotReport(t, "BR-B-01", func(i *Invoice) {
		i.Seller.Address.CountryCode = "IT"
		i.Buyer.Address.CountryCode = "IT"
		i.VATBreakdown[0].CategoryCode = "B"
		i.Lines[0].VAT.CategoryCode = "B"
	})
}

// Fehlt bei einer innergemeinschaftlichen Lieferung der Befreiungsgrund, ist die
// Rechnung nach § 14 Abs. 4 Nr. 8 UStG unvollständig — und EN 16931 verlangt ihn
// ebenso.
func TestIntraCommunityNeedsItsExemptionReason(t *testing.T) {
	intraCommunity := func(i *Invoice) {
		i.Buyer.Address.CountryCode = "NL"
		i.Buyer.VATIdentifier = "NL123456789B01"
		i.Delivery = &Delivery{
			Date:    NewDate("2026-03-15"),
			Address: &Address{CountryCode: "NL"},
		}
		i.Lines[0].VAT = LineVAT{CategoryCode: "K", Rate: NewAmount("0")}
		i.VATBreakdown[0] = VATBreakdown{
			TypeCode: "VAT", CategoryCode: "K", Rate: NewAmount("0"),
			TaxableAmount: NewAmount("1000.00"), TaxAmount: NewAmount("0.00"),
		}
		i.Totals.TaxTotal = NewAmount("0.00")
		i.Totals.GrandTotal = NewAmount("1000.00")
		i.Totals.DuePayableAmount = NewAmount("1000.00")
	}

	mustReport(t, "BR-IC-10", intraCommunity)
	mustNotReport(t, "BR-IC-10", func(i *Invoice) {
		intraCommunity(i)
		i.VATBreakdown[0].ExemptionReason = "Innergemeinschaftliche Lieferung"
	})
}

// Vier Regeln der Norm sind auch im Referenzprüfer `true()`: sie verlangen, dass
// der Schlüssel eines Grundes und derselbe Grund im Klartext dasselbe bedeuten.
// Das ist
// maschinell nicht entscheidbar. Buchfink führt sie im Umfang und meldet nichts
// — dieser Test hält fest, dass das Absicht ist und nicht Vergesslichkeit.
func TestReasonAgreementRulesAreDeliberatelySilent(t *testing.T) {
	for _, rule := range []string{"BR-CO-05", "BR-CO-06", "BR-CO-07", "BR-CO-08"} {
		mustNotReport(t, rule, func(i *Invoice) {
			a := allowance(NewAmount("100.00"))
			a.ReasonCode = "95" // Rabatt
			a.Reason = "Zuschlag für Eilzustellung"
			i.Allowances = []AllowanceCharge{a}
			i.Charges = []AllowanceCharge{a}
			i.Lines[0].Allowances = []AllowanceCharge{a}
			i.Lines[0].Charges = []AllowanceCharge{a}
		})
	}
}

// Zwei Steuergesamtbeträge in derselben Währung lassen den Empfänger wählen —
// und was er wählt, ist geraten.
func TestTwoTaxTotalsInOneCurrencyAreReported(t *testing.T) {
	mustReport(t, "BR-CO-15", func(i *Invoice) { i.Totals.TaxTotalCount = 2 })
}

// Ein Steuerbetrag, der um genau eine Währungseinheit danebenliegt, ist ein
// Verstoß; die Toleranz der Norm ist offen, nicht einschließend.
func TestRoundingToleranceIsExclusive(t *testing.T) {
	mustReport(t, "BR-CO-17", func(i *Invoice) {
		i.VATBreakdown[0].TaxAmount = NewAmount("191.00")
		i.Totals.TaxTotal = NewAmount("191.00")
		i.Totals.GrandTotal = NewAmount("1191.00")
		i.Totals.DuePayableAmount = NewAmount("1191.00")
	})
	mustNotReport(t, "BR-CO-17", func(i *Invoice) {
		i.VATBreakdown[0].TaxAmount = NewAmount("190.99")
		i.Totals.TaxTotal = NewAmount("190.99")
		i.Totals.GrandTotal = NewAmount("1190.99")
		i.Totals.DuePayableAmount = NewAmount("1190.99")
	})
}
