package declaration

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Ein Jahr einer Software-GmbH: Inlandsumsätze, eine EU-Leistung (Kz 21), ein
// Cloud-Abo eines EU-Anbieters (§ 13b Abs. 1) und Vorsteuer. Eine Buchung des
// ersten Quartals (Nr. 4: 1.000 € Umsatz, 190 € Steuer) kam nach der
// Übermittlung und steht in keiner Voranmeldung.
func vatYear() VatYearData {
	annual := &domain.VatReturn{Figures: []domain.VatReturnLine{
		{Code: "81", HasBase: true, HasTax: true, Base: eur(100000), Tax: eur(19000), ExpectedTax: eur(19000), EntryIDs: []uint{1, 2, 3, 4}},
		{Code: "21", HasBase: true, Base: eur(10000), EntryIDs: []uint{5}},
		{Code: "46", TaxCode: "47", HasBase: true, HasTax: true, Base: eur(1000), Tax: eur(190), EntryIDs: []uint{6}},
		{Code: "66", HasTax: true, Tax: eur(5000), EntryIDs: []uint{7}},
		{Code: "67", HasTax: true, Tax: eur(190), EntryIDs: []uint{6}},
	}}
	q1 := domain.VatReturn{PeriodKey: "2025-Q1", Status: domain.VatReturnSubmitted, Figures: []domain.VatReturnLine{
		{Code: "81", HasBase: true, HasTax: true, Base: eur(99000), Tax: eur(18810), EntryIDs: []uint{1, 2, 3}},
		{Code: "21", HasBase: true, Base: eur(10000), EntryIDs: []uint{5}},
		{Code: "46", TaxCode: "47", HasBase: true, HasTax: true, Base: eur(1000), Tax: eur(190), EntryIDs: []uint{6}},
		{Code: "66", HasTax: true, Tax: eur(5000), EntryIDs: []uint{7}},
		{Code: "67", HasTax: true, Tax: eur(190), EntryIDs: []uint{6}},
		{Code: "83", HasTax: true, Tax: eur(13810)},
	}}
	empty := func(key string) domain.VatReturn {
		return domain.VatReturn{PeriodKey: key, Status: domain.VatReturnSubmitted,
			Figures: []domain.VatReturnLine{{Code: "83", HasTax: true}}}
	}
	return VatYearData{
		PeriodType: domain.VatPeriodQuarter, Annual: annual,
		Submitted:       []domain.VatReturn{q1, empty("2025-Q2"), empty("2025-Q3"), empty("2025-Q4")},
		ExpectedPeriods: []string{"2025-Q1", "2025-Q2", "2025-Q3", "2025-Q4"},
		ZMServices:      eur(10000), ZMLines: 1,
		Ledger: Ledger{},
	}
}

func TestVatYearCarriesTheYearAndComputesSection(t *testing.T) {
	d, err := BuildVatYear(common(2025), vatYear())
	if err != nil {
		t.Fatal(err)
	}
	noInternalFindings(t, d)
	if d.FormYear != 2025 || d.Schema != Schema {
		t.Fatalf("Vordruckjahr %d, Schema %s", d.FormYear, d.Schema)
	}
	wantAmount(t, d, "ust.z22.kz177", eur(100000))
	wantAmount(t, d, "ust.z22.steuer", eur(19000))
	wantAmount(t, d, "ust.z37.summe", eur(19000))
	wantAmount(t, d, "ust.z65.kz846", eur(1000))
	wantAmount(t, d, "ust.z65.kz847", eur(190))
	wantAmount(t, d, "ust.z68.summe", eur(190))
	wantAmount(t, d, "ust.z74.kz721", eur(10000))
	wantAmount(t, d, "ust.z79.kz320", eur(5000))
	wantAmount(t, d, "ust.z83.kz467", eur(190))
	wantAmount(t, d, "ust.z87.summe", eur(5190))
	wantAmount(t, d, "ust.z108", eur(19190))
	wantAmount(t, d, "ust.z111", eur(14000))
	wantAmount(t, d, "ust.z118.kz816", eur(14000))
	// Vorauszahlungssoll: Summe Kz 83 der übermittelten Voranmeldungen.
	wantAmount(t, d, "ust.z119", eur(13810))
	// Die Abschlusszahlung ist genau die Steuer des Nachtrags.
	wantAmount(t, d, "ust.z120.kz820", eur(190))

	if l := line(t, d, "ust.z22.kz177"); l.Row != "22" || l.Code != "177" || !l.Check.OnForm || l.Status != StatusAuto {
		t.Errorf("Zeile 22: %+v", l)
	}
	if l := line(t, d, "ust.z19.kz133"); l.Value.Text == nil || *l.Value.Text != "1" {
		t.Errorf("Kz 133 bei Sollversteuerung: %+v", l.Value)
	}
	if v := line(t, d, "ust.z3.kz110").Value; v.Bool == nil || *v.Bool {
		t.Errorf("erstmalige Erklärung als berichtigt markiert")
	}
	if line(t, d, "ust.z1.steuernummer").Notes == nil {
		t.Errorf("die ELSTER-Steuernummer fehlt als Hinweis")
	}
	if hasFinding(d, "zm_abweichung") {
		t.Errorf("Kz 721 und ZM stimmen überein, trotzdem Befund")
	}
}

// Die Abstimmung nennt je Kennzahl die Differenz zu den Voranmeldungen und
// die Buchungen, die in keiner stehen.
func TestVatYearReconcilesWithTheReturns(t *testing.T) {
	d, err := BuildVatYear(common(2025), vatYear())
	if err != nil {
		t.Fatal(err)
	}
	var base, tax *ReconciliationRow
	for i := range d.Reconciliation {
		r := &d.Reconciliation[i]
		if r.Code == "81" && r.Field == "bemessungsgrundlage" {
			base = r
		}
		if r.Code == "81" && r.Field == "steuer" {
			tax = r
		}
		if r.Code != "81" && r.Difference != 0 {
			t.Errorf("Kz %s %s: Differenz %s ohne Grund", r.Code, r.Field, r.Difference)
		}
	}
	if base == nil || base.Difference != eur(1000) || len(base.UnreportedIDs) != 1 || base.UnreportedIDs[0] != 4 {
		t.Fatalf("Abstimmung Kz 81 Grundlage: %+v", base)
	}
	if tax == nil || tax.Difference != eur(190) || tax.Explanation == "" {
		t.Fatalf("Abstimmung Kz 81 Steuer: %+v", tax)
	}
	if hasFinding(d, "abstimmung_differenz") {
		t.Errorf("eine erklärte Differenz ist keine Warnung")
	}
}

func TestVatYearWarnsAboutMissingReturnsAndZM(t *testing.T) {
	data := vatYear()
	data.Submitted = data.Submitted[:3]
	data.ZMServices = eur(9000)
	d, err := BuildVatYear(common(2025), data)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(d, "voranmeldung_fehlt") {
		t.Errorf("die fehlende Voranmeldung 2025-Q4 wird nicht gemeldet")
	}
	if l := line(t, d, "ust.z119"); l.Status != StatusProposed {
		t.Errorf("Vorauszahlungssoll ohne alle Voranmeldungen: Stand %s", l.Status)
	}
	if !hasFinding(d, "zm_abweichung") {
		t.Errorf("Kz 721 (10.000 €) gegen ZM (9.000 €) ohne Befund")
	}
	if !line(t, d, "ust.z120.kz820").Provisional {
		t.Errorf("Zeile 120 rechnet aus einer vorgeschlagenen Zeile und ist nicht vorläufig")
	}
}

// Die Sondervorauszahlung gehört zum Vorauszahlungssoll; sie steht als Kz 39
// in der Dezember-Voranmeldung (ELSTER-Anleitung zu Zeile 119).
func TestVatYearPrepaymentTargetIncludesSpecialPrepayment(t *testing.T) {
	data := vatYear()
	data.Submitted[3].Figures = []domain.VatReturnLine{
		{Code: "39", HasTax: true, Tax: eur(1200)},
		{Code: "83", HasTax: true, Tax: eur(-1200)},
	}
	d, err := BuildVatYear(common(2025), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "ust.z119", eur(13810))
}

// Ein Fall, den Buchfink nicht bucht, sperrt die Erklärung, statt still in
// einer Zeile zu landen.
func TestVatYearBlocksUnsupportedCases(t *testing.T) {
	data := vatYear()
	data.Annual.Figures = append(data.Annual.Figures,
		domain.VatReturnLine{Code: "41", HasBase: true, Base: eur(5000), EntryIDs: []uint{9}})
	d, err := BuildVatYear(common(2025), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "ust.z38.kz741", eur(5000))
	if !hasFinding(d, "fall_nicht_unterstuetzt") {
		t.Errorf("innergemeinschaftliche Lieferungen ohne Befund übernommen")
	}
	if l := line(t, d, "ust.z59.kz742"); l.Status != StatusNotApplicable {
		t.Errorf("Dreiecksgeschäft ohne Umsatz: Stand %s", l.Status)
	}
}

// Eine Eingabe des Agenten bleibt vorgeschlagen, bis ein Mensch sie
// bestätigt; die bestätigte steht als Eingabe in der Zeile.
func TestVatYearInputsReplaceTheBooks(t *testing.T) {
	d, err := BuildVatYear(common(2025,
		proposed("ust.z7.art", Text("Softwareentwicklung")),
		confirmed("ust.z23.kz178", Cents(eur(1000))),
	), vatYear())
	if err != nil {
		t.Fatal(err)
	}
	if l := line(t, d, "ust.z7.art"); l.Status != StatusProposed || l.Value.Text == nil {
		t.Errorf("Eingabe des Agenten: %+v", l)
	}
	if l := line(t, d, "ust.z23.kz178"); l.Status != StatusInput || l.Origin.Kind != OriginInput {
		t.Errorf("bestätigte Eingabe: %+v", l)
	}
	wantAmount(t, d, "ust.z23.steuer", eur(190))
	wantAmount(t, d, "ust.z37.summe", eur(19190))
	if !hasFinding(d, "zeile_vorgeschlagen") {
		t.Errorf("ein Vorschlag sperrt die Erklärung nicht")
	}
}

// Eine Eingabe zu einer Zeile, die nur aus den Büchern rechnet, wirkt nicht.
func TestVatYearIgnoresInputsOnBookLines(t *testing.T) {
	d, err := BuildVatYear(common(2025, confirmed("ust.z22.kz177", Cents(eur(1)))), vatYear())
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "ust.z22.kz177", eur(100000))
	if !hasFinding(d, "eingabe_ohne_wirkung") {
		t.Errorf("die wirkungslose Eingabe bleibt unerwähnt")
	}
}

// Ist die Erklärung übermittelt und ändern sich danach die Zahlen, wird die
// neue zur berichtigten Erklärung (Kz 110).
func TestVatYearBecomesACorrectionAfterSubmission(t *testing.T) {
	first, err := BuildVatYear(common(2025), vatYear())
	if err != nil {
		t.Fatal(err)
	}
	c := common(2025)
	c.Records = []RecordState{{Status: StatusSubmitted, FiguresHash: first.FiguresHash,
		Submission: &Submission{At: "2026-05-02", TransferTicket: "et123", FiguresHash: first.FiguresHash}}}

	same, err := BuildVatYear(c, vatYear())
	if err != nil {
		t.Fatal(err)
	}
	if same.Status != StatusSubmitted || same.FiguresHash != first.FiguresHash {
		t.Fatalf("unveränderte Erklärung: Stand %s, Hash gleich %t", same.Status, same.FiguresHash == first.FiguresHash)
	}
	if v := line(t, same, "ust.z3.kz110").Value; *v.Bool {
		t.Errorf("die übermittelte Erklärung ist sich selbst keine Berichtigung")
	}

	changed := vatYear()
	changed.Annual.Figures[3].Tax = eur(4000)
	corr, err := BuildVatYear(c, changed)
	if err != nil {
		t.Fatal(err)
	}
	if v := line(t, corr, "ust.z3.kz110").Value; !*v.Bool {
		t.Errorf("geänderte Erklärung ohne Kz 110")
	}
	if corr.Status != StatusDraft || !hasFinding(corr, "seit_uebermittlung_geaendert") {
		t.Errorf("Stand %s, Befund Änderung fehlt", corr.Status)
	}
}

// Für ein Jahr ohne eigenen Katalog nimmt Buchfink den letzten geprüften und
// sagt, dass die Zeilen nicht am Vordruck des Jahres geprüft sind.
func TestVatYearWithoutCatalogOfTheYear(t *testing.T) {
	d, err := BuildVatYear(common(2026), vatYear())
	if err != nil {
		t.Fatal(err)
	}
	if d.FormYear != 2025 || !hasFinding(d, "vordruck_nicht_geprueft") {
		t.Fatalf("Vordruckjahr %d", d.FormYear)
	}
	if line(t, d, "ust.z22.kz177").Check.OnForm {
		t.Errorf("Zeile trägt den Prüfvermerk eines anderen Jahres")
	}
}

// § 15a UStG: Grundstücke in Zeile 99, andere Wirtschaftsgüter in Zeile 100,
// nachträglich abziehbar und zurückzuzahlen getrennt.
func TestVatYearInputTaxCorrectionSplitsByAsset(t *testing.T) {
	data := vatYear()
	data.Annual.Figures = append(data.Annual.Figures,
		domain.VatReturnLine{Code: "64", HasTax: true, Tax: eur(-300), EntryIDs: []uint{11, 12}})
	e1, e2 := uint(11), uint(12)
	data.Corrections = []domain.InputTaxCorrection{
		{Immovable: true, Usages: []domain.InputTaxUsage{{FiscalYear: 2025, Amount: eur(200), EntryID: &e1}}},
		{Immovable: false, Usages: []domain.InputTaxUsage{{FiscalYear: 2025, Amount: eur(-500), EntryID: &e2}}},
	}
	d, err := BuildVatYear(common(2025, confirmed("ust.z93.grund", Text("Nutzungsänderung"))), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "ust.z99.abziehbar", eur(200))
	wantAmount(t, d, "ust.z100.zurueck", eur(500))
	wantAmount(t, d, "ust.z102.kz357", eur(200))
	wantAmount(t, d, "ust.z102.kz359", eur(500))
	// 19.190 − 5.190 − 200 + 500
	wantAmount(t, d, "ust.z118.kz816", eur(14300))
	if v := line(t, d, "ust.z90.kz371").Value; !*v.Bool {
		t.Errorf("Zeile 90 ohne Ja trotz Grundstücksberichtigung")
	}
}
