package invoice

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Pflichtangaben im erzeugten Datensatz.
//
// Geprüft wird das Erzeugnis und nicht das Modell dahinter: was der Empfänger
// bekommt, ist die Datei. Der Weg über RenderInvoiceXML schließt die Prüfung
// gegen EN 16931 mit ein — ein Testfall, der ein ungültiges Dokument erzeugt,
// scheitert schon dort.

func testSeller() *domain.CompanySettings {
	return &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH",
		Street:      "Hauptstraße 1",
		ZipCity:     "80331 München",
		Country:     "Deutschland",
		TaxNumber:   "143/815/08151",
		VatID:       "DE123456789",
		IBAN:        "DE02120300000000202051",
		BIC:         "BYLADEM1001",
		BankName:    "Musterbank",
		// Ansprechpartner: bei XRechnung Pflicht (BR-DE-2 bis BR-DE-7).
		ContactName:    "Frieda Fink",
		ContactPhone:   "089 1234567",
		ContactEmail:   "rechnung@pfennig.example",
		RegisterCourt:  "Amtsgericht München",
		RegisterNumber: "HRB 123456",
	}
}

func testBuyer() *domain.Contact {
	return &domain.Contact{
		Type: domain.ContactTypeCustomer, Name: "Kunde GmbH",
		Street: "Kundenweg 2", PostalCode: "10115", City: "Berlin", CountryCode: "DE",
		LedgerAccount: "10001",
	}
}

func testInvoice() *domain.Invoice {
	inv := &domain.Invoice{
		ContactID: 1, ContactName: "Kunde GmbH", InvoiceNumber: "RE-2026-0001",
		Date: "2026-03-01", ServiceDateFrom: "2026-02-01", ServiceDateTo: "2026-02-28",
		DueDate: "2026-03-31", Currency: "EUR",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Kind:         domain.InvoiceKindInvoice,
		Items: []domain.InvoiceItem{{
			Position: 1, Description: "Beratung", QuantityMilli: 12000,
			Unit: "HUR", UnitPrice: 15000, TaxRate: domain.TaxRateStandard,
		}},
	}
	inv.Recalculate()
	return inv
}

func renderFor(t *testing.T, inv *domain.Invoice, seller *domain.CompanySettings, buyer *domain.Contact, profile domain.EInvoiceProfile) string {
	t.Helper()
	xml, validation, err := RenderInvoiceXML(inv, seller, buyer, profile)
	if err != nil {
		t.Fatalf("Rechnungsdatensatz erzeugen: %v", err)
	}
	if validation.Errors != 0 {
		t.Fatalf("der erzeugte Datensatz hat %d Befunde: %s", validation.Errors, validation.Findings)
	}
	if validation.At == "" || validation.Ruleset == "" {
		t.Error("der Validierungsbericht muss Zeitpunkt und Regelwerk nennen")
	}
	return xml
}

// Die Mengeneinheit steht als Schlüssel aus UN/ECE Rec. 20 an der Position
// (BT-130). Fest C62 zu schreiben hieße, zwölf Stunden als zwölf Stück zu
// verschicken.
func TestUnitCodeReachesTheXML(t *testing.T) {
	xml := renderFor(t, testInvoice(), testSeller(), testBuyer(), domain.EInvoiceProfileZUGFeRD)
	if !strings.Contains(xml, `unitCode="HUR"`) {
		t.Error("die Einheit HUR fehlt im Datensatz")
	}
	if strings.Contains(xml, `unitCode="C62"`) {
		t.Error("es wird weiterhin fest C62 geschrieben")
	}
}

// § 14 Abs. 4 Nr. 2 UStG lässt die Wahl zwischen Steuernummer und USt-IdNr.
// Liegt keine USt-IdNr. vor, gehört die Steuernummer in BT-32 — und die
// Registernummer in BT-30, weil EN 16931 für die maschinelle Zuordnung des
// Verkäufers eine Kennung aus BT-29/30/31 verlangt (BR-CO-26).
func TestTaxNumberIsWrittenWhenThereIsNoVatID(t *testing.T) {
	seller := testSeller()
	seller.VatID = ""
	xml := renderFor(t, testInvoice(), seller, testBuyer(), domain.EInvoiceProfileZUGFeRD)

	if !strings.Contains(xml, `schemeID="FC">143/815/08151`) {
		t.Error("die Steuernummer fehlt als BT-32 im Datensatz")
	}
	if !strings.Contains(xml, "HRB 123456") {
		t.Error("ohne USt-IdNr. muss die Registernummer als BT-30 einspringen (BR-CO-26)")
	}

	// Mit USt-IdNr. stehen beide da: die Norm erlaubt es, und der Empfänger
	// braucht die USt-IdNr. für seine Prüfung.
	both := renderFor(t, testInvoice(), testSeller(), testBuyer(), domain.EInvoiceProfileZUGFeRD)
	if !strings.Contains(both, `schemeID="VA">DE123456789`) || !strings.Contains(both, `schemeID="FC">143/815/08151`) {
		t.Error("liegen beide vor, gehören beide in den Datensatz")
	}
}

// Die im Voraus vereinbarte Entgeltminderung ist Pflichtangabe
// (§ 14 Abs. 4 Nr. 7 UStG) und steht in BT-20 — und im PDF, das der Empfänger
// liest.
func TestPaymentTermsReachXMLAndDocument(t *testing.T) {
	inv := testInvoice()
	inv.Terms = domain.PaymentTerms{DueDays: 30, DiscountPermille: 20, DiscountDays: 10}

	xml := renderFor(t, inv, testSeller(), testBuyer(), domain.EInvoiceProfileZUGFeRD)
	if !strings.Contains(xml, "2 % Skonto") {
		t.Errorf("BT-20 nennt die Skontovereinbarung nicht:\n%s", xml)
	}

	typ := GenerateTypstTemplate(inv, testSeller(), testBuyer())
	if !strings.Contains(typ, "Skonto") || !strings.Contains(typ, "11.03.2026") {
		t.Error("die Zahlungsbedingung fehlt im Dokument")
	}
}

// § 14a Abs. 1 UStG: bei der innergemeinschaftlichen Lieferung gehört die
// USt-IdNr. des Empfängers auf die Rechnung — in BT-48 und in den Adressblock.
func TestBuyerVatIDOnIntraCommunitySupply(t *testing.T) {
	inv := testInvoice()
	inv.TaxTreatment = domain.TaxTreatmentIntraCommunitySupply
	inv.Items[0].TaxRate = domain.TaxRateNone
	inv.Recalculate()

	buyer := testBuyer()
	buyer.CountryCode = "AT"
	buyer.VatID = "ATU12345678"

	xml := renderFor(t, inv, testSeller(), buyer, domain.EInvoiceProfileZUGFeRD)
	// Die USt-IdNr. des Erwerbers steht in der BuyerTradeParty.
	buyerBlock := xml[strings.Index(xml, "BuyerTradeParty"):]
	if !strings.Contains(buyerBlock[:strings.Index(buyerBlock, "/ram:BuyerTradeParty")], "ATU12345678") {
		t.Error("BT-48 fehlt: die USt-IdNr. des Empfängers steht nicht in der Erwerberpartei")
	}
	if !strings.Contains(xml, "innergemeinschaftliche Lieferung") {
		t.Error("der Befreiungsgrund (BT-120) fehlt")
	}

	typ := GenerateTypstTemplate(inv, testSeller(), buyer)
	if !strings.Contains(typ, "ATU12345678") {
		t.Error("die USt-IdNr. des Empfängers fehlt im Adressblock des Dokuments")
	}
}

// Die Stornorechnung trägt den Typschlüssel 384, negierte Beträge und den Bezug
// auf die Rechnung, die sie storniert (BG-3). Ohne den Bezug kommt sie beim
// Empfänger als unverbundene zweite Rechnung an.
func TestCancellationCarriesTypeCodeAndReference(t *testing.T) {
	inv := testInvoice()
	inv.InvoiceNumber = "RE-2026-0002"
	inv.Kind = domain.InvoiceKindCancellation
	inv.CorrectsInvoiceNumber = "RE-2026-0001"
	inv.CorrectsInvoiceDate = "2026-03-01"
	inv.Negate()

	xml := renderFor(t, inv, testSeller(), testBuyer(), domain.EInvoiceProfileZUGFeRD)
	if !strings.Contains(xml, "<ram:TypeCode>384</ram:TypeCode>") {
		t.Error("die Stornorechnung muss BT-3 = 384 tragen")
	}
	if !strings.Contains(xml, "<ram:IssuerAssignedID>RE-2026-0001</ram:IssuerAssignedID>") {
		t.Error("der Bezug auf die stornierte Rechnung (BT-25) fehlt")
	}
	if !strings.Contains(xml, "20260301") {
		t.Error("das Datum der stornierten Rechnung (BT-26) fehlt")
	}
	if !strings.Contains(xml, "<ram:LineTotalAmount>-1800.00</ram:LineTotalAmount>") ||
		!strings.Contains(xml, "<ram:GrandTotalAmount>-2142.00</ram:GrandTotalAmount>") {
		t.Errorf("die Beträge der Stornorechnung sind nicht negiert:\n%s", xml)
	}

	typ := GenerateTypstTemplate(inv, testSeller(), testBuyer())
	if !strings.Contains(typ, "STORNORECHNUNG") {
		t.Error("das Dokument muss sich als Stornorechnung ausweisen")
	}
	if !strings.Contains(typ, "Storno zu Rechnung RE-2026-0001") {
		t.Error("der Bezug fehlt im lesbaren Teil")
	}
	// Das Wort „Gutschrift" gehört nicht auf ein Storno: eine Gutschrift im
	// Sinne des § 14 Abs. 2 Satz 2 UStG ist die Abrechnung des
	// Leistungsempfängers.
	if strings.Contains(typ, "Gutschrift") {
		t.Error("ein Storno darf nicht als Gutschrift bezeichnet werden (§ 14c Abs. 2 UStG)")
	}
}

// Die Schlussrechnung setzt die Anzahlungen ab: BT-113 nennt den bereits
// gezahlten Betrag, BG-3 die Abschlagsrechnungen, und BT-115 bleibt der Rest.
func TestFinalInvoiceStatesPrepaidAmount(t *testing.T) {
	inv := testInvoice()
	inv.Kind = domain.InvoiceKindFinal
	inv.PrepaidAmount = 47600
	inv.PrecedingRefs = []domain.InvoiceReference{{Number: "RE-2026-0005", Date: "2026-01-15"}}

	xml := renderFor(t, inv, testSeller(), testBuyer(), domain.EInvoiceProfileZUGFeRD)
	if !strings.Contains(xml, "<ram:TotalPrepaidAmount>476.00</ram:TotalPrepaidAmount>") {
		t.Errorf("BT-113 fehlt oder ist falsch:\n%s", xml)
	}
	if !strings.Contains(xml, "<ram:DuePayableAmount>1666.00</ram:DuePayableAmount>") {
		t.Error("BT-115 muss der Restbetrag nach Abzug der Anzahlungen sein")
	}
	if !strings.Contains(xml, "RE-2026-0005") {
		t.Error("die abgesetzte Abschlagsrechnung fehlt in BG-3")
	}

	typ := GenerateTypstTemplate(inv, testSeller(), testBuyer())
	if !strings.Contains(typ, "abzüglich Anzahlungen") || !strings.Contains(typ, "Zahlbetrag") {
		t.Error("die Verrechnung der Anzahlungen fehlt im Dokument")
	}
}

// Die XRechnung ist eine eigene Ausprägung: Kennung, Leitweg-ID und die
// CIUS-Regeln müssen zusammen aufgehen, sonst weist der Rechnungseingang der
// Verwaltung sie zurück.
func TestXRechnungProfileIsValid(t *testing.T) {
	buyer := testBuyer()
	buyer.LeitwegID = "991-33333TEST-33"
	buyer.EInvoiceProfile = domain.EInvoiceProfileXRechnungCII

	inv := testInvoice()
	inv.Terms = domain.PaymentTerms{DueDays: 30}

	xml, validation, err := RenderInvoiceXML(inv, testSeller(), buyer, domain.EInvoiceProfileXRechnungCII)
	if err != nil {
		t.Fatalf("XRechnung erzeugen: %v", err)
	}
	if validation.Errors != 0 {
		t.Fatalf("die XRechnung hat %d Befunde: %s", validation.Errors, validation.Findings)
	}
	if !strings.Contains(xml, "xrechnung_3.0") && !strings.Contains(xml, "xrechnung") {
		t.Error("die Kennung der deutschen Ausprägung (BT-24) fehlt")
	}
	if !strings.Contains(xml, "<ram:BuyerReference>991-33333TEST-33</ram:BuyerReference>") {
		t.Error("die Leitweg-ID (BT-10) fehlt — ohne sie ist die Rechnung nicht zustellbar (BR-DE-15)")
	}
	if !strings.Contains(validation.Ruleset, "xrechnung") {
		t.Errorf("der Bericht muss das geprüfte Regelwerk nennen, enthält aber %q", validation.Ruleset)
	}
}

// Ein Verstoß verhindert die Ausstellung. Ohne Leitweg-ID fehlt der XRechnung
// die Käuferreferenz (BR-DE-15), und eine Rechnung, die der Empfänger
// zurückweist, soll gar nicht erst entstehen.
func TestXRechnungWithoutRouteIDIsRejected(t *testing.T) {
	_, validation, err := RenderInvoiceXML(testInvoice(), testSeller(), testBuyer(), domain.EInvoiceProfileXRechnungCII)
	if err == nil {
		t.Fatal("eine XRechnung ohne Leitweg-ID darf nicht erzeugt werden")
	}
	if !strings.Contains(err.Error(), "XRechnung") {
		t.Errorf("die Meldung nennt das Regelwerk nicht: %v", err)
	}
	if validation.Errors == 0 {
		t.Error("der Bericht muss die Befunde enthalten, auch wenn die Ausstellung scheitert")
	}
}

// Die Anschrift des Empfängers steht in Feldern und nicht in einer Zeile:
// § 14 Abs. 4 Nr. 1 UStG verlangt sie vollständig, EN 16931 verlangt sie
// getrennt (BT-50, BT-52, BT-53).
func TestBuyerAddressIsStructured(t *testing.T) {
	xml := renderFor(t, testInvoice(), testSeller(), testBuyer(), domain.EInvoiceProfileZUGFeRD)
	for _, want := range []string{
		"<ram:PostcodeCode>10115</ram:PostcodeCode>",
		"<ram:LineOne>Kundenweg 2</ram:LineOne>",
		"<ram:CityName>Berlin</ram:CityName>",
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("im Datensatz fehlt %s", want)
		}
	}
	// Und die des Verkäufers ebenso, aus dem einzeiligen Feld zerlegt.
	if !strings.Contains(xml, "<ram:PostcodeCode>80331</ram:PostcodeCode>") ||
		!strings.Contains(xml, "<ram:CityName>München</ram:CityName>") {
		t.Error("die Anschrift des Verkäufers wurde nicht in PLZ und Ort zerlegt")
	}
}

// Eine Kleinbetragsrechnung ohne erfassten Kunden bekommt „Barverkauf" als
// Namen: EN 16931 verlangt den Namen des Erwerbers (BR-07), auch wo § 33 UStDV
// ihn erlässt — die Norm kennt die Kleinbetragsrechnung nicht.
// Die Kleinbetragsrechnung ohne Empfänger bekommt keinen erfundenen Erwerber.
//
// EN 16931 verlangt seinen Namen (BR-07), § 33 UStDV erlässt ihn — und nimmt
// die Kleinbetragsrechnung zugleich von der E-Rechnungspflicht aus. Einen Namen
// zu erfinden, um die Norm zu bedienen, hieße eine Pflichtangabe zu behaupten:
// der Datensatz wäre maschinenlesbar falsch. Also entsteht keiner, und das
// Dokument sagt, warum.
func TestSmallAmountInvoiceWithoutContact(t *testing.T) {
	inv := testInvoice()
	inv.ContactID = 0
	inv.ContactName = ""
	inv.SmallAmount = true
	inv.Items[0].QuantityMilli = 1000
	inv.Items[0].UnitPrice = 10000
	inv.Recalculate()

	_, _, err := RenderInvoiceXML(inv, testSeller(), nil, domain.EInvoiceProfileZUGFeRD)
	if err == nil {
		t.Fatal("ohne Erwerber darf kein strukturierter Datensatz entstehen (BR-07)")
	}
	if !strings.Contains(err.Error(), "§ 33 UStDV") {
		t.Errorf("die Meldung nennt den Ausweg nicht: %v", err)
	}

	typ := GenerateTypstTemplate(inv, testSeller(), nil)
	if !strings.Contains(typ, "Kleinbetragsrechnung nach § 33 UStDV") {
		t.Error("der Hinweis auf § 33 UStDV fehlt im Dokument")
	}
	if !strings.Contains(typ, "E-Rechnung ausgenommen") {
		t.Error("der Hinweis auf die Ausnahme von der E-Rechnungspflicht fehlt im Dokument")
	}
}
