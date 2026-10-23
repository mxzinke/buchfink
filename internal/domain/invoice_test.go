package domain

import (
	"strings"
	"testing"
)

// Die Mengeneinheit muss ein Schlüssel aus UN/ECE Rec. 20 sein (BT-130);
// „Stunde" ist dort keiner.
func TestResolveUnitCode(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "C62", true},
		{"Stück", "C62", true},
		{"Stunde", "HUR", true},
		{"HUR", "HUR", true},
		{"h", "", false},
		{"Monat", "MON", true},
		{"Furlong", "", false},
	}
	for _, c := range cases {
		got, ok := ResolveUnitCode(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("ResolveUnitCode(%q) = %q, %v — erwartet %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
	if UnitLabel("HUR") != "Stunde" {
		t.Errorf("UnitLabel(HUR) = %q, erwartet Stunde", UnitLabel("HUR"))
	}
	if len(UnitCodes()) == 0 {
		t.Error("die Einheitenliste darf nicht leer sein")
	}
}

// § 14 Abs. 4 Nr. 1 UStG verlangt die vollständige Anschrift des Empfängers,
// Nr. 2 die Steuernummer oder USt-IdNr. des Ausstellers. Fehlt eine davon, ist
// die Rechnung nicht ordnungsmäßig — und der Empfänger verliert den Abzug.
func TestValidateParties(t *testing.T) {
	inv := &Invoice{TaxTreatment: TaxTreatmentDomestic}
	buyer := &Contact{Name: "Kunde GmbH", Street: "Kundenweg 2", PostalCode: "10115", City: "Berlin"}

	if err := inv.ValidateParties("", "", buyer); err == nil {
		t.Error("ohne Steuernummer und ohne USt-IdNr. darf keine Rechnung entstehen")
	} else if !strings.Contains(err.Error(), "§ 14 Abs. 4 Nr. 2") {
		t.Errorf("die Meldung nennt die Norm nicht: %v", err)
	}
	if err := inv.ValidateParties("143/815/08151", "", buyer); err != nil {
		t.Errorf("die Steuernummer allein genügt nach § 14 Abs. 4 Nr. 2 UStG: %v", err)
	}

	noStreet := &Contact{Name: "Kunde GmbH", PostalCode: "10115", City: "Berlin"}
	err := inv.ValidateParties("143/815/08151", "", noStreet)
	if err == nil || !strings.Contains(err.Error(), "Straße") {
		t.Errorf("die fehlende Straße muss beim Namen genannt werden, erhalten: %v", err)
	}
	noCity := &Contact{Name: "Kunde GmbH", Street: "Kundenweg 2", PostalCode: "10115"}
	if err := inv.ValidateParties("143/815/08151", "", noCity); err == nil ||
		!strings.Contains(err.Error(), "Ort") {
		t.Errorf("der fehlende Ort muss gemeldet werden, erhalten: %v", err)
	}

	// § 14a Abs. 1 UStG: bei der Steuerschuldnerschaft des Leistungsempfängers
	// gehört dessen USt-IdNr. auf die Rechnung.
	rc := &Invoice{TaxTreatment: TaxTreatmentReverseChargeSupply}
	if err := rc.ValidateParties("143/815/08151", "DE123456789", buyer); err == nil {
		t.Error("ohne USt-IdNr. des Empfängers darf keine Leistung nach § 13b abgerechnet werden")
	}
	buyer.VatID = "ATU12345678"
	if err := rc.ValidateParties("143/815/08151", "DE123456789", buyer); err != nil {
		t.Errorf("mit USt-IdNr. muss die Leistung nach § 13b durchgehen: %v", err)
	}

	// § 33 UStDV: die Kleinbetragsrechnung braucht die Anschrift des
	// Empfängers nicht.
	small := &Invoice{TaxTreatment: TaxTreatmentDomestic, SmallAmount: true}
	if err := small.ValidateParties("143/815/08151", "", noCity); err != nil {
		t.Errorf("die Kleinbetragsrechnung kommt ohne Anschrift aus: %v", err)
	}
}

// Der Storno negiert die Beträge. Ein Storno mit den ursprünglichen Beträgen und
// einem Hinweis „bitte nicht beachten" ist keiner: das System des Empfängers
// bucht, was die Zahlen sagen.
func TestNegateTurnsAnInvoiceIntoItsStorno(t *testing.T) {
	inv := &Invoice{
		TaxTreatment: TaxTreatmentDomestic,
		Items: []InvoiceItem{
			{Position: 1, Description: "Beratung", QuantityMilli: 2000, UnitPrice: 50000, TaxRate: TaxRateStandard},
		},
	}
	inv.Recalculate()
	net, tax, gross := inv.NetAmount, inv.TaxAmount, inv.GrossAmount

	inv.Negate()
	if inv.NetAmount != -net || inv.TaxAmount != -tax || inv.GrossAmount != -gross {
		t.Errorf("negiert = %s / %s / %s, erwartet %s / %s / %s",
			inv.NetAmount, inv.TaxAmount, inv.GrossAmount, -net, -tax, -gross)
	}
	if inv.Items[0].QuantityMilli != -2000 {
		t.Errorf("die Menge muss negiert sein, ist aber %d", inv.Items[0].QuantityMilli)
	}
	// Der Preis bleibt positiv: ein negativer Nettopreis ist nach BR-27 kein
	// zulässiger Rechnungsinhalt.
	if inv.Items[0].UnitPrice != 50000 {
		t.Errorf("der Einzelpreis darf nicht negiert werden, ist aber %s", inv.Items[0].UnitPrice)
	}
}

// Die Dokumentart entscheidet BT-3, und der Empfänger bucht danach.
func TestInvoiceKindTypeCodes(t *testing.T) {
	cases := map[InvoiceKind]string{
		InvoiceKindInvoice:      "380",
		InvoiceKindFinal:        "380",
		InvoiceKindAdvance:      "386",
		InvoiceKindCorrection:   "384",
		InvoiceKindCancellation: "384",
	}
	for kind, want := range cases {
		if got := kind.TypeCode(); got != want {
			t.Errorf("%s → BT-3 %q, erwartet %q", kind, got, want)
		}
	}
	if InvoiceKindAdvance.BooksOnIssue() {
		t.Error("die Abschlagsrechnung wird beim Ausstellen nicht gebucht — die Steuer entsteht mit der Vereinnahmung")
	}
	if !InvoiceKindFinal.BooksOnIssue() {
		t.Error("die Schlussrechnung wird beim Ausstellen gebucht")
	}
}

// Die im Voraus vereinbarte Entgeltminderung ist Pflichtangabe
// (§ 14 Abs. 4 Nr. 7 UStG). Sie muss als Satz auf dem Dokument stehen und in
// BT-20, nicht als drei Zahlen, die niemand deutet.
func TestPaymentTermsNote(t *testing.T) {
	terms := PaymentTerms{DueDays: 30, DiscountPermille: 20, DiscountDays: 10}
	note := terms.Note("2026-03-01")
	for _, want := range []string{"30 Tagen", "11.03.2026", "2 %"} {
		if !strings.Contains(note, want) {
			t.Errorf("im Zahlungsbedingungstext fehlt %q: %s", want, note)
		}
	}
	if (PaymentTerms{}).Note("2026-03-01") != "" {
		t.Error("ohne vereinbarte Bedingung darf kein Text entstehen")
	}
	if got := (PaymentTerms{DiscountPermille: 15, DiscountDays: 7}).DiscountPercent(); got != "1,5" {
		t.Errorf("Skontosatz = %q, erwartet 1,5", got)
	}
}

// Die Mengeneinheit wird beim Ausstellen geprüft: ein Schlüssel, den EN 16931
// nicht kennt, ließe die Rechnung erst beim Empfänger scheitern.
func TestValidateRejectsUnknownUnit(t *testing.T) {
	inv := &Invoice{
		ContactID: 1, Date: "2026-03-01", ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01",
		TaxTreatment: TaxTreatmentDomestic,
		Items: []InvoiceItem{
			{Position: 1, Description: "Beratung", QuantityMilli: 1000, UnitPrice: 10000, Unit: "Furlong"},
		},
	}
	err := inv.Validate()
	if err == nil || !strings.Contains(err.Error(), "Mengeneinheit") {
		t.Errorf("eine unbekannte Mengeneinheit muss abgewiesen werden, erhalten: %v", err)
	}
}

// Die Auswahllisten der Oberfläche kommen aus dem Fachmodell.
//
// Versandweg und Lückengrund sind Wertelisten mit fester Bedeutung; ihre
// Beschriftungen lagen zusätzlich als Kopie in der Rechnungsseite. Der Test
// hält fest, dass jede Ausprägung genau einmal in der Auswahl steht und die
// Beschriftung des Grundes dieselbe ist, die der Lückenbericht zeigt — sonst
// wählt der Anwender „Probelauf" und liest danach ein anderes Wort.
func TestOptionListsCoverEveryValue(t *testing.T) {
	vias := InvoiceSentViaOptions()
	if len(vias) != 4 {
		t.Fatalf("erwartet werden vier Versandwege, erhalten: %d", len(vias))
	}
	seen := map[InvoiceSentVia]bool{}
	for _, option := range vias {
		if option.Label == "" {
			t.Errorf("der Versandweg %q hat keine Beschriftung", option.Via)
		}
		if seen[option.Via] {
			t.Errorf("der Versandweg %q steht zweimal in der Auswahl", option.Via)
		}
		seen[option.Via] = true
	}
	for _, via := range []InvoiceSentVia{
		InvoiceSentViaEmail, InvoiceSentViaPortal, InvoiceSentViaPost, InvoiceSentViaOther,
	} {
		if !seen[via] {
			t.Errorf("der Versandweg %q fehlt in der Auswahl", via)
		}
	}

	reasons := NumberGapReasons()
	if len(reasons) != 4 {
		t.Fatalf("erwartet werden vier Lückengründe, erhalten: %d", len(reasons))
	}
	for _, option := range reasons {
		if option.Label != option.Reason.Label() {
			t.Errorf("der Grund %q trägt in der Auswahl %q, im Bericht aber %q",
				option.Reason, option.Label, option.Reason.Label())
		}
	}
	// Gegenprobe: der Bericht setzt dieselbe Beschriftung, die die Auswahl
	// angeboten hat.
	report := NumberGapReport{Gaps: []NumberGapEntry{{
		Reason: reasons[1].Reason, Label: reasons[1].Reason.Label(),
	}}}
	if report.Gaps[0].Label != reasons[1].Label {
		t.Errorf("Auswahl und Bericht benennen denselben Grund verschieden: %q / %q",
			reasons[1].Label, report.Gaps[0].Label)
	}
}
