package domain

import (
	"strings"
	"testing"
)

// Das Nummernformat ist einstellbar, weil ein Mandant mit vorhandener
// Buchhaltung seine Systematik fortführen muss. Was es nicht darf, ist den
// Zähler verlieren: § 14 Abs. 4 Nr. 4 UStG verlangt eine einmalige,
// fortlaufende Nummer, und ein Format ohne {NR} gäbe jeder Rechnung dieselbe.
func TestInvoiceNumberFormat(t *testing.T) {
	cases := []struct {
		format string
		seq    int64
		want   string
	}{
		{DefaultInvoiceNumberFormat, 7, "RE-2026-0007"},
		{"{JAHR}-{NR:6}", 42, "2026-000042"},
		{"R{NR}", 5, "R5"},
		{"AR/{JAHR}/{NR:3}", 128, "AR/2026/128"},
		// Ein unbrauchbares Format fällt auf die Voreinstellung zurück, statt
		// eine Nummer ohne Zähler zu erzeugen.
		{"RE-{JAHR}", 7, "RE-2026-0007"},
	}
	for _, c := range cases {
		if got := FormatInvoiceNumberWith(c.format, 2026, c.seq); got != c.want {
			t.Errorf("FormatInvoiceNumberWith(%q, 2026, %d) = %q, erwartet %q", c.format, c.seq, got, c.want)
		}
	}

	if err := ValidateInvoiceNumberFormat("RE-{JAHR}"); err == nil {
		t.Error("ein Format ohne {NR} muss abgewiesen werden — jede Rechnung trüge dieselbe Nummer")
	}
	if err := ValidateInvoiceNumberFormat(""); err == nil {
		t.Error("ein leeres Nummernformat muss abgewiesen werden")
	}
	if err := ValidateInvoiceNumberFormat("{JAHR}-{NR:4}"); err != nil {
		t.Errorf("ein gültiges Format wurde abgewiesen: %v", err)
	}
}

// Der Lückenbericht muss aus der Nummer den Zählerstand zurücklesen, sonst
// vergleicht er nichts.
func TestParseInvoiceSequence(t *testing.T) {
	cases := []struct {
		number string
		want   int64
		ok     bool
	}{
		{"RE-2026-0007", 7, true},
		{"2026-000042", 42, true},
		{"AR/2026/128", 128, true},
		// Nur das Geschäftsjahr und sonst keine Ziffer: da ist kein Zähler.
		{"RE-2026", 0, false},
		{"ohne Ziffern", 0, false},
	}
	for _, c := range cases {
		got, ok := ParseInvoiceSequence(c.number, 2026)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseInvoiceSequence(%q) = %d, %v — erwartet %d, %v", c.number, got, ok, c.want, c.ok)
		}
	}
}

// Der Bericht vergleicht Zählerstand und vergebene Nummern. Der Zähler ist der
// Maßstab: eine gelöschte Zeile ändert ihn nicht, und genau darum geht es.
func TestNumberGapReport(t *testing.T) {
	report := BuildNumberGapReport(2026, 5,
		[]string{"RE-2026-0001", "RE-2026-0003", "RE-2026-0004"},
		[]NumberGap{{Sequence: 2, Reason: NumberGapTest, Detail: "Probelauf", RecordedAt: "2026-03-01T10:00:00Z"}},
		DefaultInvoiceNumberFormat)

	if report.Issued != 4 {
		t.Errorf("ausgegeben = %d, erwartet 4", report.Issued)
	}
	if len(report.Gaps) != 1 {
		t.Fatalf("erwartet genau eine Lücke, erhalten %d", len(report.Gaps))
	}
	gap := report.Gaps[0]
	if gap.Sequence != 2 || gap.Number != "RE-2026-0002" {
		t.Errorf("die Lücke ist %d (%s), erwartet 2 (RE-2026-0002)", gap.Sequence, gap.Number)
	}
	if gap.Reason != NumberGapTest || gap.Label != "Probelauf" {
		t.Errorf("der dokumentierte Grund fehlt: %+v", gap)
	}

	// Ohne dokumentierten Grund bleibt die Lücke sichtbar — und als
	// unbegründet ausgewiesen. Das ist der Befund, um den es der Prüfung geht.
	plain := BuildNumberGapReport(2026, 3, []string{"RE-2026-0001"}, nil, DefaultInvoiceNumberFormat)
	if len(plain.Gaps) != 1 || plain.Gaps[0].Reason != NumberGapUnknown {
		t.Errorf("eine unbegründete Lücke muss als solche erscheinen: %+v", plain.Gaps)
	}
	// Eine leere Liste ist leer und nicht nil: sie geht als JSON ins Frontend.
	full := BuildNumberGapReport(2026, 2, []string{"RE-2026-0001"}, nil, DefaultInvoiceNumberFormat)
	if full.Gaps == nil {
		t.Error("die Lückenliste muss leer statt nil sein")
	}
}

// Die Mengeneinheit muss ein Schlüssel aus UN/ECE Rec. 20 sein (BT-130).
// „Stunde" ist dort keiner — und eine Beratungsrechnung über 12 Stunden kam
// beim Empfänger bisher als 12 Stück an.
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
		{"h", "HUR", true},
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

// Die einzeilige Anschrift der Bestandsdaten wird zerlegt — und was sich nicht
// sicher zerlegen lässt, bleibt liegen. Eine geratene Straße auf einer Rechnung
// wäre ein Formfehler, den der Empfänger mit seinem Vorsteuerabzug bezahlt.
func TestParsePostalAddress(t *testing.T) {
	street, code, city := ParsePostalAddress("Kundenweg 2, 10115 Berlin")
	if street != "Kundenweg 2" || code != "10115" || city != "Berlin" {
		t.Errorf("zerlegt zu %q / %q / %q", street, code, city)
	}

	street, code, city = ParsePostalAddress("Hauptstraße 1\nGebäude B\n80331 München")
	if street != "Hauptstraße 1, Gebäude B" || code != "80331" || city != "München" {
		t.Errorf("mehrzeilig zerlegt zu %q / %q / %q", street, code, city)
	}

	if street, _, _ := ParsePostalAddress("Irgendwo im Nirgendwo"); street != "" {
		t.Error("eine Zeile ohne PLZ darf nicht als Anschrift durchgehen")
	}

	c := &Contact{Address: "Kundenweg 2, 10115 Berlin"}
	if !c.MigrateAddress() || !c.HasCompleteAddress() {
		t.Error("die Übernahme der Altanschrift ist fehlgeschlagen")
	}
	unparsable := &Contact{Address: "postlagernd"}
	if unparsable.MigrateAddress() {
		t.Error("eine unlesbare Anschrift darf nicht übernommen werden")
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

	// § 14a Abs. 1 UStG: bei der innergemeinschaftlichen Lieferung gehört die
	// USt-IdNr. des Empfängers auf die Rechnung.
	ig := &Invoice{TaxTreatment: TaxTreatmentIntraCommunitySupply}
	if err := ig.ValidateParties("143/815/08151", "DE123456789", buyer); err == nil {
		t.Error("ohne USt-IdNr. des Empfängers darf keine ig. Lieferung abgerechnet werden")
	}
	buyer.VatID = "ATU12345678"
	if err := ig.ValidateParties("143/815/08151", "DE123456789", buyer); err != nil {
		t.Errorf("mit USt-IdNr. muss die ig. Lieferung durchgehen: %v", err)
	}

	// § 33 UStDV: die Kleinbetragsrechnung braucht den Empfänger nicht.
	small := &Invoice{TaxTreatment: TaxTreatmentDomestic, SmallAmount: true}
	if err := small.ValidateParties("143/815/08151", "", nil); err != nil {
		t.Errorf("die Kleinbetragsrechnung kommt ohne Empfänger aus: %v", err)
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

// Die Invarianten des Rechnungsverbunds. Sie sind die Konstruktion, an der
// § 14c hängt: eine Schlussrechnung ohne Verrechnung weist die Steuer zweimal
// aus.
func TestInvoiceGroupInvariants(t *testing.T) {
	group := &InvoiceGroup{
		Title: "Umbau", TotalNet: 1000000, TaxRate: TaxRateStandard,
		Advances: []AdvanceItem{
			{NetAmount: 400000, TaxAmount: 76000, GrossAmount: 476000, SettledAt: "2026-04-15"},
			{NetAmount: 300000, TaxAmount: 57000, GrossAmount: 357000},
		},
	}

	p := group.ComputeProgress()
	if p.BilledNet != 700000 {
		t.Errorf("abgerechnet = %s, erwartet 7000,00", p.BilledNet)
	}
	if p.ReceivedNet != 400000 || p.ReceivedTax != 76000 {
		t.Errorf("vereinnahmt = %s / %s, erwartet 4000,00 / 760,00", p.ReceivedNet, p.ReceivedTax)
	}
	if p.OpenNet != 300000 {
		t.Errorf("noch nicht abgerechnet = %s, erwartet 3000,00", p.OpenNet)
	}

	// Nur der berechnete *und* vereinnahmte Abschlag ist abzusetzen
	// (§ 14 Abs. 5 Satz 2 UStG).
	if got := group.DeductibleAdvances(); len(got) != 1 || got[0].NetAmount != 400000 {
		t.Errorf("abzusetzen sind %d Abschläge, erwartet den einen vereinnahmten", len(got))
	}

	// Die Summe der Abschläge darf den Gesamtbetrag nicht überschreiten.
	if err := group.EnsureAdvanceFits(200000); err != nil {
		t.Errorf("ein passender Abschlag wurde abgewiesen: %v", err)
	}
	if err := group.EnsureAdvanceFits(400000); err == nil {
		t.Error("ein Abschlag über den Gesamtbetrag hinaus muss abgewiesen werden")
	}

	// Ein stornierter Abschlag fällt aus der Verrechnung heraus: mit ihm
	// entfällt die Rechnung im Sinne des § 14 Abs. 5 Satz 2 UStG.
	group.Advances[0].Cancelled = true
	if len(group.DeductibleAdvances()) != 0 {
		t.Error("ein stornierter Abschlag darf nicht mehr abgesetzt werden")
	}
	if group.ComputeProgress().BilledNet != 300000 {
		t.Errorf("der stornierte Abschlag zählt nicht mehr mit: %s", group.ComputeProgress().BilledNet)
	}

	// Nach der Schlussrechnung nimmt der Verbund keine Abschläge mehr auf.
	group.Closed = true
	if err := group.EnsureAdvanceFits(1000); err == nil {
		t.Error("ein abgeschlossener Verbund darf keinen weiteren Abschlag aufnehmen")
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
