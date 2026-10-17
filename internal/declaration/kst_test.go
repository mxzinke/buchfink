package declaration

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// settled sind die Angaben, die der Mensch oder der Agent mit Rückfrage zu
// einer schlichten GmbH ohne Vorjahresverluste macht.
func settled(extra ...Input) []Input {
	in := []Input{
		confirmed("kst.gk.vga", Cents(0)),
		confirmed("kst.gk.einlage", Cents(0)),
		confirmed("kst.gk.nab_weitere", Cents(0)),
		confirmed("kst.v.z14", Cents(0)),
		confirmed("kst.wa.z11e", Bool(false)),
		confirmed("kst.f.bestand_vorjahr", Cents(0)),
	}
	return append(in, extra...)
}

func confirmedWithholding(kapest, solz domain.Cents) []Input {
	return []Input{confirmed("kst.wa.kapest", Cents(kapest)), confirmed("kst.wa.solz", Cents(solz))}
}

// Beispiel A der Referenz § 8b KStG (buchfink-beteiligungen): Dividende einer
// 100-%-Tochter 100.000 € auf 7006, Kapitalertragsteuer 25.000 € auf 7630,
// Solidaritätszuschlag 1.375 € auf 7633. Wirkung auf das Einkommen gegenüber
// dem Jahresüberschuss: − 68.625 €.
func subsidiaryDividend() CorporateData {
	l := Ledger{}
	book(l, "7006", -eur(100000))
	book(l, "7630", eur(25000))
	book(l, "7633", eur(1375))
	return CorporateData{
		Ledger: l, NetIncome: eur(73625), BalanceResult: eur(73625),
		PriorKnown: true, PriorEquity: eur(500000), PriorSubscribed: eur(25000),
		Holdings: []Holding{{AssetID: 1, Name: "Tochter A GmbH", QuotaPermille: 1000,
			AcquiredOn: "2020-01-01", Dividends: eur(100000)}},
		TradeTaxRate: 400,
	}
}

func TestCorporateTaxSubsidiaryDividend(t *testing.T) {
	inputs := append(settled(confirmed("kst.gk.8b_streubesitz", Cents(0))), confirmedWithholding(eur(25000), eur(1375))...)
	d, err := BuildCorporateTax(common(2025, inputs...), subsidiaryDividend())
	if err != nil {
		t.Fatal(err)
	}
	noInternalFindings(t, d)
	wantAmount(t, d, "kst.gk.z11", eur(73625))
	wantAmount(t, d, "kst.gk.8b_frei", eur(100000))
	wantAmount(t, d, "kst.gk.8b_5_bezuege", eur(5000))
	wantAmount(t, d, "kst.gk.kst", eur(25000))
	wantAmount(t, d, "kst.gk.z82", eur(1375))
	wantAmount(t, d, "kst.gk.summe", eur(5000))
	if effect := amount(t, d, "kst.gk.summe") - amount(t, d, "kst.gk.z11"); effect != -eur(68625) {
		t.Errorf("Wirkung auf das Einkommen %s €, erwartet −68.625 €", effect)
	}
	wantAmount(t, d, "kst.zve.zve", eur(5000))
	// 15 % von 5.000 € und 5,5 % darauf.
	wantAmount(t, d, "kst.k.kst", eur(750))
	wantAmount(t, d, "kst.k.solz", eur(41, 25))
	wantAmount(t, d, "kst.k.kapest", eur(25000))
	wantAmount(t, d, "kst.k.ergebnis", -eur(25583, 75))
	if len(d.Blocking()) != 0 {
		t.Errorf("blockierende Befunde: %+v", d.Blocking())
	}
	if l := line(t, d, "kst.gk.z82"); l.Row != "82" || l.Check.OnForm {
		t.Errorf("Zeile 82 ist nur über die Anleitung belegt: %+v", l.Check)
	}
	if l := line(t, d, "kst.gk.kst"); l.Row != "" {
		t.Errorf("unbelegte Zeile trägt die Nummer %s", l.Row)
	}
}

// Ohne Bestätigung der Quote zu Beginn des Kalenderjahres bleibt der
// Streubesitz ein Vorschlag aus der Kartei.
func TestCorporateTaxProposesStreubesitzFromTheRegister(t *testing.T) {
	d, err := BuildCorporateTax(common(2025, settled()...), subsidiaryDividend())
	if err != nil {
		t.Fatal(err)
	}
	if l := line(t, d, "kst.gk.8b_streubesitz"); l.Status != StatusProposed || l.Value.Amount() != 0 {
		t.Errorf("Streubesitz aus der Kartei: %+v", l)
	}

	data := subsidiaryDividend()
	data.Holdings = nil
	d, err = BuildCorporateTax(common(2025, settled()...), data)
	if err != nil {
		t.Fatal(err)
	}
	if l := line(t, d, "kst.gk.8b_streubesitz"); l.Status != StatusOpen {
		t.Errorf("Bezüge ohne Anteil in der Kartei: Stand %s", l.Status)
	}
	if !line(t, d, "kst.zve.zve").Provisional {
		t.Errorf("das Einkommen hängt am offenen Streubesitz und ist nicht vorläufig")
	}
}

// Beispiel C: Streubesitz unter 10 % ist voll steuerpflichtig, ohne 5 %.
func TestCorporateTaxSmallHolding(t *testing.T) {
	l := Ledger{}
	book(l, "7014", -eur(1000))
	book(l, "7630", eur(250))
	book(l, "7633", eur(13, 75))
	data := CorporateData{Ledger: l, NetIncome: eur(736, 25), BalanceResult: eur(736, 25), PriorKnown: true,
		Holdings: []Holding{{Name: "Aktien X AG", QuotaPermille: 20, AcquiredOn: "2018-05-01", Dividends: eur(1000)}}}
	d, err := BuildCorporateTax(common(2025, settled()...), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "kst.gk.8b_streubesitz", eur(1000))
	wantAmount(t, d, "kst.gk.8b_frei", 0)
	wantAmount(t, d, "kst.gk.8b_5_bezuege", 0)
	wantAmount(t, d, "kst.gk.summe", eur(1000))
}

// Beispiel B: Verkauf einer Beteiligung, Erlös 200.000 €, Restbuchwert
// 50.000 €, Kosten 5.000 €: Gewinn 145.000 € außer Ansatz, 7.250 €
// nicht abziehbar.
func TestCorporateTaxSaleOfAShare(t *testing.T) {
	l := Ledger{}
	book(l, "4852", -eur(200000))
	book(l, "4858", eur(50000))
	book(l, "6857", eur(5000))
	data := CorporateData{Ledger: l, NetIncome: eur(145000), BalanceResult: eur(145000), PriorKnown: true}
	d, err := BuildCorporateTax(common(2025, settled()...), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "kst.gk.8b_gewinne", eur(145000))
	wantAmount(t, d, "kst.gk.8b_5_gewinne", eur(7250))
	wantAmount(t, d, "kst.gk.summe", eur(7250))
}

// Ein Verlust aus dem Verkauf und eine Teilwertabschreibung sind nicht
// abziehbar (§ 8b Abs. 3 Satz 3 KStG).
func TestCorporateTaxShareLossesAreAddedBack(t *testing.T) {
	l := Ledger{}
	book(l, "6892", -eur(10000))
	book(l, "6898", eur(30000))
	book(l, "7204", eur(5000))
	data := CorporateData{Ledger: l, NetIncome: eur(75000), BalanceResult: eur(75000), PriorKnown: true}
	d, err := BuildCorporateTax(common(2025, settled()...), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "kst.gk.8b_minderungen", eur(25000))
	wantAmount(t, d, "kst.gk.summe", eur(100000))
}

// Nicht abziehbare Aufwendungen und Steuern stehen je Posten mit ihren Konten.
func TestCorporateTaxNonDeductibleExpenses(t *testing.T) {
	l := Ledger{}
	book(l, "6620", eur(300))
	book(l, "6644", eur(450))
	book(l, "6645", eur(1000))
	book(l, "6875", eur(2000))
	book(l, "7610", eur(8000))
	book(l, "7643", -eur(500))
	book(l, "7600", eur(9000))
	book(l, "7608", eur(495))
	book(l, "6393", eur(1000))
	book(l, "6394", eur(200))
	data := CorporateData{Ledger: l, NetIncome: eur(50000), BalanceResult: eur(50000), PriorKnown: true,
		Revenue: eur(500000), Wages: eur(200000)}
	d, err := BuildCorporateTax(common(2025, settled()...), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "kst.gk.geschenke", eur(300))
	wantAmount(t, d, "kst.gk.bewirtung", eur(450))
	wantAmount(t, d, "kst.gk.repraesentation", eur(1000))
	wantAmount(t, d, "kst.gk.aufsichtsrat", eur(2000))
	wantAmount(t, d, "kst.gk.gewst", eur(7500))
	wantAmount(t, d, "kst.gk.kst", eur(9000))
	wantAmount(t, d, "kst.gk.z82", eur(495))
	wantAmount(t, d, "kst.gk.spenden", eur(1200))
	// 50.000 + 300 + 450 + 1.000 + 2.000 + 7.500 + 9.000 + 495 + 1.200
	wantAmount(t, d, "kst.gk.summe", eur(71945))
	// Parteispenden sind nicht abziehbar; 1.000 € liegen unter dem Höchstbetrag.
	wantAmount(t, d, "kst.z.z5", eur(1000))
	wantAmount(t, d, "kst.zve.zuwendungen", eur(1000))
	wantAmount(t, d, "kst.zve.zve", eur(70945))
	if line(t, d, "kst.gk.gewst").Origin.Accounts == nil {
		t.Errorf("die Gewerbesteuer nennt ihre Konten nicht")
	}
}

// § 9 Abs. 1 Nr. 2 KStG: Höchstbetrag ist der höhere Betrag aus 20 % des
// Einkommens und 4 Promille der Umsätze, Löhne und Gehälter; der Rest wird
// vorgetragen.
func TestCorporateTaxDonationCap(t *testing.T) {
	l := Ledger{}
	book(l, "6393", eur(30000))
	data := CorporateData{Ledger: l, NetIncome: eur(70000), BalanceResult: eur(70000), PriorKnown: true,
		Revenue: eur(1000000), Wages: eur(500000)}
	d, err := BuildCorporateTax(common(2025, settled(confirmed("kst.z.z10", Cents(eur(1000000))))...), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "kst.zve.z32", eur(100000))
	// 20 % von 100.000 € = 20.000 €; 4 ‰ von 1,5 Mio. € = 6.000 €.
	wantAmount(t, d, "kst.z.z7", eur(20000))
	wantAmount(t, d, "kst.zve.zuwendungen", eur(20000))
	wantAmount(t, d, "kst.z.vortrag_ende", eur(10000))
	wantAmount(t, d, "kst.zve.zve", eur(80000))
	if f := findForm(d, "Z"); f == nil || f.Required != RequirementMandatory {
		t.Errorf("Anlage Z bei Zuwendungen nicht verlangt")
	}
}

// § 10d Abs. 2 EStG: bis 1 Mio. € voll, darüber 70 % (ab 2024).
func TestCorporateTaxLossCarryForward(t *testing.T) {
	data := CorporateData{Ledger: Ledger{}, NetIncome: eur(2000000), BalanceResult: eur(2000000), PriorKnown: true}
	in := settled()
	in[3] = confirmed("kst.v.z14", Cents(eur(3000000)))
	d, err := BuildCorporateTax(common(2025, in...), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "kst.zve.verlustabzug", eur(1700000))
	wantAmount(t, d, "kst.zve.zve", eur(300000))
	wantAmount(t, d, "kst.v.ende", eur(1300000))
	if f := findForm(d, "Verluste"); f == nil || f.Required != RequirementMandatory {
		t.Errorf("Anlage Verluste bei Verlustvortrag nicht verlangt")
	}
}

func TestCorporateTaxLossOfTheYear(t *testing.T) {
	data := CorporateData{Ledger: Ledger{}, NetIncome: -eur(40000), BalanceResult: -eur(40000), PriorKnown: true}
	d, err := BuildCorporateTax(common(2025, settled(confirmed("kst.v.z20", Cents(0)))...), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "kst.zve.zve", -eur(40000))
	wantAmount(t, d, "kst.k.kst", 0)
	wantAmount(t, d, "kst.v.ende", eur(40000))
}

// Der Steuersatz folgt dem Veranlagungszeitraum (§ 23 Abs. 1 KStG); der
// Betrag wird zugunsten auf volle Euro gerundet (§ 31 Abs. 1 Satz 2 KStG).
func TestCorporateTaxRateFollowsTheYear(t *testing.T) {
	data := CorporateData{Ledger: Ledger{}, NetIncome: eur(10000, 99), BalanceResult: eur(10000, 99), PriorKnown: true}
	d, err := BuildCorporateTax(common(2028, settled()...), data)
	if err != nil {
		t.Fatal(err)
	}
	// 14 % von 10.000,99 € = 1.400,1386 € → 1.400 €; SolZ 77 €.
	wantAmount(t, d, "kst.k.kst", eur(1400))
	wantAmount(t, d, "kst.k.solz", eur(77))
	if v := line(t, d, "kst.k.satz").Value; v.Text == nil || *v.Text != "14 %" {
		t.Errorf("Satz 2028: %+v", v)
	}
}

// Gründungsjahr: kein Vorjahr, Einlagekonto beginnt mit null, das Aufgeld ist
// eine Einlage des Jahres.
func TestCorporateTaxFoundingYear(t *testing.T) {
	l := Ledger{}
	book(l, "2925", -eur(5000))
	data := CorporateData{Ledger: l, NetIncome: eur(1000), BalanceResult: eur(1000), Founded: true}
	in := []Input{confirmed("kst.gk.vga", Cents(0)), confirmed("kst.gk.einlage", Cents(0)),
		confirmed("kst.gk.nab_weitere", Cents(0)), confirmed("kst.wa.z11e", Bool(false)),
		confirmed("kst.f.einlagen", Cents(eur(5000)))}
	d, err := BuildCorporateTax(common(2025, in...), data)
	if err != nil {
		t.Fatal(err)
	}
	if l := line(t, d, "kst.v.z14"); l.Status != StatusAuto {
		t.Errorf("Verlustvortrag im Gründungsjahr: Stand %s", l.Status)
	}
	if l := line(t, d, "kst.f.bestand_vorjahr"); l.Status != StatusNotApplicable {
		t.Errorf("Einlagekonto des Vorjahres im Gründungsjahr: Stand %s", l.Status)
	}
	wantAmount(t, d, "kst.f.bestand_ende", eur(5000))
	if len(d.Blocking()) != 0 {
		t.Errorf("blockierende Befunde im Gründungsjahr: %+v", d.Blocking())
	}
}

// Leistungen über dem ausschüttbaren Gewinn sind Einlagenrückgewähr; das
// sperrt mit Verweis auf den Steuerberater.
func TestCorporateTaxContributionAccount(t *testing.T) {
	data := CorporateData{Ledger: Ledger{}, NetIncome: eur(10000), BalanceResult: eur(10000), PriorKnown: true,
		PriorEquity: eur(100000), PriorSubscribed: eur(25000),
		Distributions: []Distribution{{ForYear: 2024, DecisionDate: "2025-04-10", Amount: eur(70000)}}}
	in := settled()
	in[5] = confirmed("kst.f.bestand_vorjahr", Cents(eur(10000)))
	d, err := BuildCorporateTax(common(2025, in...), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "kst.f.leistungen", eur(70000))
	// 100.000 − 25.000 − 10.000
	wantAmount(t, d, "kst.f.ausschuettbar", eur(65000))
	wantAmount(t, d, "kst.f.verwendung", eur(5000))
	wantAmount(t, d, "kst.f.bestand_ende", eur(5000))
	if !hasFinding(d, "einlagenrueckgewaehr") {
		t.Errorf("Einlagenrückgewähr ohne Befund")
	}
}

func TestCorporateTaxChecks(t *testing.T) {
	l := Ledger{}
	book(l, "6067", eur(1000))
	data := CorporateData{Ledger: l, NetIncome: eur(10000), BalanceResult: eur(9000), PriorKnown: true}
	c := common(2025)
	c.Basis.ClosingState = string(domain.FiscalYearOpen)
	d, err := BuildCorporateTax(c, data)
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"ergebnis_ungleich_bilanz", "abschluss_offen", "vga_pruefen", "zeile_offen"} {
		if !hasFinding(d, code) {
			t.Errorf("Befund %s fehlt", code)
		}
	}
	if l := line(t, d, "kst.gk.vga"); l.Status != StatusOpen {
		t.Errorf("vGA bei bebuchtem Kandidatenkonto: Stand %s", l.Status)
	}
	if d.Overview().Complete {
		t.Errorf("Erklärung mit offenen Zeilen gilt als abgabereif")
	}
}

func findForm(d *Declaration, id string) *Form {
	for i := range d.Forms {
		if d.Forms[i].ID == id {
			return &d.Forms[i]
		}
	}
	return nil
}
