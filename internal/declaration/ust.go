package declaration

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// VatYearData sind die Größen der Umsatzsteuererklärung.
type VatYearData struct {
	PeriodType domain.VatPeriodType
	// Annual ist das Kennziffernblatt der Voranmeldung über das ganze
	// Kalenderjahr: dieselbe Zuordnung nach dem Entstehen der Steuer, nur ohne
	// Zeitraumgrenzen.
	Annual *domain.VatReturn
	// Submitted sind die übermittelten Voranmeldungen des Jahres, je Zeitraum
	// die jüngste.
	Submitted []domain.VatReturn
	// ExpectedPeriods sind die Zeiträume, für die eine Voranmeldung
	// abzugeben war.
	ExpectedPeriods []string
	// ZMServices ist die Summe der übermittelten Zusammenfassenden Meldungen
	// (Meldeart sonstige Leistung); ZMMissing die Zeiträume mit Umsätzen, aber
	// ohne übermittelte Meldung.
	ZMServices domain.Cents
	ZMLines    int
	ZMMissing  []string
	// Corrections ist das Verzeichnis nach § 15a UStG mit den Berichtigungen.
	Corrections []domain.InputTaxCorrection
	// Ledger liefert die Zahlungen auf die Vorauszahlungskonten.
	Ledger Ledger
}

// Zuordnung der Voranmeldung (USt 1 A) zur Jahreserklärung (USt 2 A).
type vatMapping struct {
	code  string // Kennzahl der Voranmeldung
	tax   bool   // Steuer statt Bemessungsgrundlage
	key   string // Zeile der Jahreserklärung
	label string
	// unsupported: Fall, den Buchfink für die Zielgruppe nicht bucht.
	unsupported bool
	// review: der Wert braucht eine Zuordnung, die die Voranmeldung nicht
	// trifft.
	review string
}

var vatMappings = []vatMapping{
	{code: "81", key: "ust.z22.kz177", label: "Umsätze 19 %"},
	{code: "81", tax: true, key: "ust.z22.steuer", label: "Steuer 19 %"},
	{code: "86", key: "ust.z25.kz275", label: "Umsätze 7 %"},
	{code: "86", tax: true, key: "ust.z25.steuer", label: "Steuer 7 %"},
	{code: "87", key: "ust.z28.kz157", label: "Umsätze 0 %"},
	{code: "35", key: "ust.z31.kz155", label: "Umsätze zu anderen Steuersätzen"},
	{code: "35", tax: true, key: "ust.z31.kz156", label: "Steuer zu anderen Steuersätzen"},
	{code: "77", key: "ust.z32.kz777", label: "Land- und Forstwirtschaft", unsupported: true},
	{code: "76", key: "ust.z33.kz346", label: "Land- und Forstwirtschaft", unsupported: true},
	{code: "76", tax: true, key: "ust.z33.kz347", label: "Land- und Forstwirtschaft", unsupported: true},
	{code: "65", tax: true, key: "ust.z35.kz317", label: "Nachsteuer",
		review: "Die Voranmeldung führt Nachsteuer wegen Wechsels der Besteuerungsform und wegen Steuersatzänderung in einer Kennzahl; die Jahreserklärung trennt sie in Zeile 35 und 36."},
	{code: "41", key: "ust.z38.kz741", label: "innergemeinschaftliche Lieferungen", unsupported: true},
	{code: "44", key: "ust.z39.kz744", label: "neue Fahrzeuge", unsupported: true},
	{code: "49", key: "ust.z40.kz749", label: "neue Fahrzeuge", unsupported: true},
	{code: "43", key: "ust.z41.kz752", label: "Ausfuhrlieferungen",
		review: "Kz 43 der Voranmeldung umfasst Ausfuhrlieferungen und weitere steuerfreie Umsätze mit Vorsteuerabzug; die Jahreserklärung trennt sie in Zeile 41 und 42."},
	{code: "48", key: "ust.z49.kz240", label: "steuerfreie Umsätze ohne Vorsteuerabzug",
		review: "Die Jahreserklärung trennt steuerfreie Umsätze ohne Vorsteuerabzug nach Vorschrift und danach, ob sie zum Gesamtumsatz gehören (Zeilen 46, 47 und 49)."},
	{code: "91", key: "ust.z50.kz791", label: "steuerfreie innergemeinschaftliche Erwerbe"},
	{code: "89", key: "ust.z51.kz781", label: "innergemeinschaftliche Erwerbe 19 %"},
	{code: "89", tax: true, key: "ust.z51.steuer", label: "Steuer innergemeinschaftliche Erwerbe 19 %"},
	{code: "93", key: "ust.z52.kz793", label: "innergemeinschaftliche Erwerbe 7 %"},
	{code: "93", tax: true, key: "ust.z52.steuer", label: "Steuer innergemeinschaftliche Erwerbe 7 %"},
	{code: "90", key: "ust.z53.kz780", label: "innergemeinschaftliche Erwerbe 0 %"},
	{code: "95", key: "ust.z54.kz798", label: "innergemeinschaftliche Erwerbe zu anderen Sätzen"},
	{code: "95", tax: true, key: "ust.z54.kz799", label: "Steuer innergemeinschaftliche Erwerbe zu anderen Sätzen"},
	{code: "94", key: "ust.z55.kz794", label: "Erwerb neuer Fahrzeuge", unsupported: true},
	{code: "94", tax: true, key: "ust.z55.kz796", label: "Erwerb neuer Fahrzeuge", unsupported: true},
	{code: "42", key: "ust.z59.kz742", label: "Dreiecksgeschäfte", unsupported: true},
	{code: "46", key: "ust.z65.kz846", label: "§ 13b Abs. 1 UStG"},
	{code: "46", tax: true, key: "ust.z65.kz847", label: "Steuer § 13b Abs. 1 UStG"},
	{code: "73", key: "ust.z66.kz873", label: "§ 13b Abs. 2 Nr. 3 UStG", unsupported: true},
	{code: "73", tax: true, key: "ust.z66.kz874", label: "§ 13b Abs. 2 Nr. 3 UStG", unsupported: true},
	{code: "84", key: "ust.z67.kz877", label: "andere Leistungen § 13b UStG"},
	{code: "84", tax: true, key: "ust.z67.kz878", label: "Steuer andere Leistungen § 13b UStG"},
	{code: "60", key: "ust.z70.kz209", label: "§ 13b Abs. 5 UStG (eigene Umsätze)"},
	{code: "21", key: "ust.z74.kz721", label: "nicht steuerbare sonstige Leistungen § 18b"},
	{code: "45", key: "ust.z75.kz205", label: "übrige nicht steuerbare Umsätze"},
	{code: "50", key: "ust.z78.kz650", label: "Minderung uneinbringliche Forderungen"},
	{code: "66", tax: true, key: "ust.z79.kz320", label: "Vorsteuer aus Rechnungen"},
	{code: "61", tax: true, key: "ust.z80.kz761", label: "Vorsteuer innergemeinschaftliche Erwerbe"},
	{code: "62", tax: true, key: "ust.z81.kz762", label: "Einfuhrumsatzsteuer"},
	{code: "67", tax: true, key: "ust.z83.kz467", label: "Vorsteuer § 13b UStG"},
	{code: "63", tax: true, key: "ust.z84.kz334", label: "Durchschnittssätze", unsupported: true},
	{code: "59", tax: true, key: "ust.z85.kz759", label: "§ 15 Abs. 4a UStG", unsupported: true},
	{code: "37", tax: true, key: "ust.z88.kz637", label: "Minderung der Vorsteuer"},
	{code: "69", tax: true, key: "ust.z113.kz318", label: "§ 14c UStG"},
}

// Kennzahlen der Voranmeldung, die nur abgestimmt und nicht übertragen werden.
var vatReconcileOnly = []vatMapping{
	{code: "64", tax: true, label: "Berichtigung des Vorsteuerabzugs nach § 15a UStG"},
}

func vatValue(ret *domain.VatReturn, code string, tax bool) (domain.Cents, []uint) {
	if ret == nil {
		return 0, nil
	}
	l, ok := ret.Line(code)
	if !ok {
		return 0, nil
	}
	if tax {
		return l.Tax, l.EntryIDs
	}
	if !l.HasBase {
		return l.Tax, l.EntryIDs
	}
	return l.Base, l.EntryIDs
}

// BuildVatYear rechnet die Umsatzsteuererklärung. Ist die Erklärung des Jahres
// schon übermittelt und weichen die Zahlen davon ab, ist sie eine berichtigte
// Erklärung (Kz 110).
func BuildVatYear(common Common, data VatYearData) (*Declaration, error) {
	d, err := buildVatYear(common, data, false)
	if err != nil {
		return nil, err
	}
	submitted := false
	for _, r := range common.Records {
		if r.Status != StatusSubmitted {
			continue
		}
		submitted = true
		if r.FiguresHash == d.FiguresHash {
			return d, nil
		}
	}
	if !submitted {
		return d, nil
	}
	return buildVatYear(common, data, true)
}

func buildVatYear(common Common, data VatYearData, correction bool) (*Declaration, error) {
	cat, exact, err := CatalogFor(KindVatYear, common.Year)
	if err != nil {
		return nil, err
	}
	c := newCalc(cat, common.Inputs)
	subject := subjectOf(common.Settings)
	settings := origin(OriginSettings, "Unternehmensdaten")

	// Kopf und allgemeine Angaben.
	c.taxNumberLine("ust.z1.steuernummer", subject)
	if correction {
		c.books("ust.z3.kz110", Bool(true), Origin{Kind: OriginComputation,
			Text: "Für das Jahr ist schon eine Erklärung übermittelt; diese weicht ab"})
	} else {
		c.books("ust.z3.kz110", Bool(false), Origin{Kind: OriginComputation, Text: "Erstmalige Erklärung"})
	}
	c.resolve("ust.z4.wid", figure{status: StatusOpen, origin: Origin{Kind: OriginNone}})
	c.textLine("ust.z5.name", firmName(common.Settings), "Der Firmenname fehlt in den Unternehmensdaten.")
	c.resolve("ust.z6.firma", figure{status: StatusNotApplicable, origin: Origin{Kind: OriginNone}})
	c.open("ust.z7.art", "Die Art des Unternehmens steht nicht in den Unternehmensdaten.")
	c.textLine("ust.z8.strasse", subject.Address.Street, "Die Straße fehlt in den Unternehmensdaten.")
	c.optionalText("ust.z9.hausnummer", subject.Address.HouseNumber)
	c.optionalText("ust.z9.adressergaenzung", subject.Address.Addition)
	c.textLine("ust.z10.plz", subject.Address.PostalCode, "Die Postleitzahl fehlt in den Unternehmensdaten.")
	c.textLine("ust.z10.ort", subject.Address.City, "Der Ort fehlt in den Unternehmensdaten.")
	c.optionalText("ust.z12.telefon", common.Settings.ContactPhone)
	c.optionalText("ust.z13.email", common.Settings.ContactEmail)
	c.books("ust.z14.kz125", Bool(false), settings, "Die Gesellschaft hat ihren Sitz im Inland.")
	c.books("ust.z15.kz126", Bool(false), settings)
	c.none("ust.z16.kz136", Value{}, "Eine GmbH oder UG als Kleinunternehmerin bildet Buchfink nicht ab.")
	c.durationOfBusiness(common)
	switch strings.ToUpper(strings.TrimSpace(common.Settings.TaxationType)) {
	case "IST":
		c.books("ust.z19.kz133", Text("2"), settings, "Besteuerung nach vereinnahmten Entgelten (§ 20 UStG).")
	default:
		c.books("ust.z19.kz133", Text("1"), settings, "Besteuerung nach vereinbarten Entgelten (§ 16 Abs. 1 Satz 1 UStG).")
	}
	c.resolve("ust.z20.kz129", figure{value: Bool(false), status: StatusAuto, origin: Origin{Kind: OriginNone}})
	c.resolve("ust.z21.kz123", figure{value: Bool(false), status: StatusAuto, origin: Origin{Kind: OriginNone}})

	// Umsätze und Steuern aus dem Kennziffernblatt des Jahres.
	for _, m := range vatMappings {
		c.vatLine(m, data.Annual)
	}
	c.wertabgabe("ust.z23.kz178", "ust.z23.steuer", 19)
	c.wertabgabe("ust.z24.kz179", "ust.z24.steuer", 19)
	c.wertabgabe("ust.z26.kz195", "ust.z26.steuer", 7)
	c.wertabgabe("ust.z27.kz196", "ust.z27.steuer", 7)
	c.resolve("ust.z29.kz158", noneFigure())
	c.resolve("ust.z30.kz159", noneFigure())
	c.none("ust.z34.kz361", Value{})
	c.resolve("ust.z36.kz319", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone},
		notes: []string{"Nachsteuer wegen Steuersatzänderung führt die Voranmeldung zusammen mit Zeile 35 in Kz 65."}})
	c.sum("ust.z37.summe", "ust.z22.steuer", "ust.z23.steuer", "ust.z24.steuer", "ust.z25.steuer",
		"ust.z26.steuer", "ust.z27.steuer", "ust.z31.kz156", "ust.z33.kz347", "ust.z35.kz317", "ust.z36.kz319")

	c.resolve("ust.z42.vorschrift", noneFigure())
	c.resolve("ust.z42.betrag", noneFigure())
	c.resolve("ust.z43.betrag", noneFigure())
	c.resolve("ust.z44.betrag", noneFigure())
	c.sum("ust.z45.kz237", "ust.z42.betrag", "ust.z43.betrag", "ust.z44.betrag")
	c.resolve("ust.z46.kz286", noneFigure())
	c.resolve("ust.z47.vorschrift", noneFigure())
	c.resolve("ust.z47.kz287", noneFigure())
	c.none("ust.z48.kz236", Value{})
	if c.amount("ust.z49.kz240") != 0 {
		c.resolve("ust.z49.vorschrift", figure{status: StatusOpen, origin: Origin{Kind: OriginNone},
			notes: []string{"Vorschrift der Steuerbefreiung nennen, etwa „4 Nr. 8“."}})
	} else {
		c.resolve("ust.z49.vorschrift", noneFigure())
	}
	c.sum("ust.z56.summe", "ust.z51.steuer", "ust.z52.steuer", "ust.z54.kz799", "ust.z55.kz796")
	c.none("ust.z57.kz852", Value{})
	c.none("ust.z57.kz853", Value{})
	c.sum("ust.z58.summe", "ust.z57.kz853")
	for _, key := range []string{"ust.z60.kz751", "ust.z60.steuer", "ust.z61.kz746", "ust.z61.steuer",
		"ust.z62.kz750", "ust.z63.kz747", "ust.z63.kz748"} {
		c.none(key, Value{})
	}
	c.sum("ust.z64.summe", "ust.z60.steuer", "ust.z61.steuer", "ust.z63.kz748")
	c.sum("ust.z68.summe", "ust.z65.kz847", "ust.z66.kz874", "ust.z67.kz878")

	c.resolve("ust.z69.betrag", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone},
		notes: []string{"Buchfink kennt keine Option zur Steuerpflicht nach § 9 UStG."}})
	c.resolve("ust.z71.kz213", noneFigure("Nur bei Leistungen an Nichtunternehmer im übrigen Gemeinschaftsgebiet unter der Schwelle von 10.000 €."))
	c.resolve("ust.z72.kz214", noneFigure("Nur bei Leistungen an Nichtunternehmer im übrigen Gemeinschaftsgebiet, die dort zu versteuern sind."))
	c.resolve("ust.z73.kz211", noneFigure())
	if c.amount("ust.z74.kz721")+c.amount("ust.z75.kz205") != 0 {
		c.resolve("ust.z76.kz204", figure{value: Cents(0), status: StatusProposed, origin: Origin{Kind: OriginNone},
			notes: []string{"Nicht steuerbare Umsätze liegen vor. Schließen sie den Vorsteuerabzug aus (§ 15 Abs. 2 und 3 UStG), etwa Finanz- oder Versicherungsumsätze im Ausland, ist der Betrag hier einzutragen."}})
	} else {
		c.resolve("ust.z76.kz204", noneFigure())
	}
	c.none("ust.z77.kz212", Value{})
	c.none("ust.z82.kz466", Value{})
	c.none("ust.z86.kz760", Value{})
	c.sum("ust.z87.summe", "ust.z79.kz320", "ust.z80.kz761", "ust.z81.kz762", "ust.z82.kz466",
		"ust.z83.kz467", "ust.z84.kz334", "ust.z85.kz759", "ust.z86.kz760")

	c.inputTaxCorrection(common.Year, data)

	c.sum("ust.z103", "ust.z37.summe")
	c.sum("ust.z104", "ust.z56.summe")
	c.sum("ust.z105", "ust.z58.summe")
	c.sum("ust.z106", "ust.z64.summe")
	c.sum("ust.z107", "ust.z68.summe")
	c.sum("ust.z108", "ust.z103", "ust.z104", "ust.z105", "ust.z106", "ust.z107")
	c.sum("ust.z109", "ust.z87.summe")
	c.sum("ust.z110", "ust.z102.kz357")
	c.books("ust.z111", Cents(c.amount("ust.z108")-c.amount("ust.z109")-c.amount("ust.z110")),
		fromLines("Zeile 108 − Zeile 109 − Zeile 110", "ust.z108", "ust.z109", "ust.z110"))
	c.sum("ust.z112", "ust.z102.kz359")
	c.resolve("ust.z114.kz331", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone},
		notes: []string{"Nur bei Berichtigung einer Steuer nach § 17 Abs. 1 Satz 7 UStG."}})
	c.none("ust.z115.kz391", Value{})
	c.sum("ust.z116", "ust.z111", "ust.z112", "ust.z113.kz318", "ust.z114.kz331", "ust.z115.kz391")
	c.books("ust.z117", Cents(0), Origin{Kind: OriginNone}, "Anlage UN entfällt für eine im Inland ansässige Gesellschaft.")
	c.books("ust.z118.kz816", Cents(c.amount("ust.z116")-c.amount("ust.z117")),
		fromLines("Zeile 116 − Zeile 117", "ust.z116", "ust.z117"))
	c.prepaymentTarget(data)
	c.books("ust.z120.kz820", Cents(c.amount("ust.z118.kz816")-c.amount("ust.z119")),
		fromLines("Zeile 118 − Zeile 119", "ust.z118.kz816", "ust.z119"))
	advisor := common.Settings.TaxAdvisor.Prepares(string(KindVatYear), common.Year)
	c.books("ust.z121.berater", Bool(advisor), Origin{Kind: OriginSettings, Text: "Steuerliche Beratung laut Unternehmensdaten"})

	c.form("USt2A", RequirementMandatory, "§ 18 Abs. 3 UStG")
	rows := reconcileVat(c, data)
	c.checkZM(data)
	c.checkExpectedTax(data.Annual)

	d := finish(KindVatYear, common, cat, exact, c)
	d.Reconciliation = rows
	return d, nil
}

func origin(kind OriginKind, text string) Origin { return Origin{Kind: kind, Text: text} }

func noneFigure(notes ...string) figure {
	return figure{status: StatusNotApplicable, origin: Origin{Kind: OriginNone}, notes: notes}
}

func firmName(s domain.CompanySettings) string {
	name := strings.TrimSpace(s.CompanyName)
	if name == "" || s.LegalForm == "" {
		return name
	}
	lower := strings.ToLower(name)
	for _, suffix := range []string{"gmbh", "ug (haftungsbeschränkt)", "ug"} {
		if strings.HasSuffix(lower, suffix) {
			return name
		}
	}
	return name + " " + s.LegalForm
}

func (c *calc) optionalText(key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		c.books(key, Value{}, Origin{Kind: OriginSettings, Text: "Unternehmensdaten ohne Angabe"})
		return
	}
	c.books(key, Text(value), Origin{Kind: OriginSettings, Text: "Unternehmensdaten"})
}

// sum setzt eine Rechenzeile als Summe anderer Zeilen.
func (c *calc) sum(key string, from ...string) figure {
	var total domain.Cents
	for _, k := range from {
		total += c.amount(k)
	}
	formula := "Summe der Zeilen"
	if len(from) == 1 {
		formula = "Übertrag"
	}
	return c.books(key, Cents(total), fromLines(formula, from...))
}

// durationOfBusiness füllt Zeile 17, wenn die Unternehmereigenschaft nicht
// das ganze Jahr bestand: im Gründungsjahr ab der Gründung (§ 16 Abs. 3 UStG).
func (c *calc) durationOfBusiness(common Common) {
	start := fmt.Sprintf("%d-01-01", common.Year)
	end := fmt.Sprintf("%d-12-31", common.Year)
	founded := common.Settings.FoundedOn
	if founded > start && founded <= end {
		o := Origin{Kind: OriginSettings, Text: "Tag der Gründung laut Unternehmensdaten"}
		c.resolve("ust.z17.von", figure{value: Date(founded), status: StatusProposed, origin: o,
			notes: []string{"Die Unternehmereigenschaft beginnt mit der ersten nach außen erkennbaren, auf eine Unternehmertätigkeit gerichteten Tätigkeit; der Tag der Gründung ist ein Vorschlag."}})
		c.resolve("ust.z17.bis", figure{value: Date(end), status: StatusAuto, origin: o})
		return
	}
	c.resolve("ust.z17.von", noneFigure("Die Unternehmereigenschaft bestand das ganze Jahr."))
	c.resolve("ust.z17.bis", noneFigure())
}

// vatLine überträgt eine Kennzahl der Voranmeldung.
func (c *calc) vatLine(m vatMapping, annual *domain.VatReturn) {
	amount, ids := vatValue(annual, m.code, m.tax)
	o := Origin{Kind: OriginEntries, EntryIDs: ids,
		Formula: fmt.Sprintf("Kz %s der Voranmeldung über das Kalenderjahr", m.code)}
	switch {
	case amount == 0 && m.unsupported:
		c.resolve(m.key, noneFigure())
	case m.unsupported:
		c.books(m.key, Cents(amount), o)
		c.finding(SeverityBlocking, "fall_nicht_unterstuetzt", m.key,
			fmt.Sprintf("In den Büchern stehen %s € für %s. Diesen Fall bildet Buchfink für eine GmbH der Zielgruppe nicht ab; bitte mit einem Steuerberater klären.",
				amount, m.label), "")
	case amount != 0 && m.review != "":
		c.propose(m.key, Cents(amount), o, m.review)
	default:
		c.books(m.key, Cents(amount), o)
	}
}

// wertabgabe setzt eine unentgeltliche Wertabgabe: die Bemessungsgrundlage
// nur als Eingabe, die Steuer daraus.
func (c *calc) wertabgabe(baseKey, taxKey string, rate int64) {
	base := c.resolve(baseKey, noneFigure("Unentgeltliche Wertabgaben bucht Buchfink nicht gesondert; nur eintragen, wenn welche vorliegen."))
	if base.value.Empty() {
		c.none(taxKey, Value{})
		return
	}
	c.books(taxKey, Cents(domain.MulRound(base.value.Amount(), rate, 100)),
		fromLines(fmt.Sprintf("Bemessungsgrundlage × %d %%", rate), baseKey))
}

// inputTaxCorrection füllt Abschnitt J aus dem Verzeichnis nach § 15a UStG.
// Grundstücke gehören in Zeile 99, alle anderen Wirtschaftsgüter in Zeile 100.
func (c *calc) inputTaxCorrection(year int, data VatYearData) {
	type bucket struct {
		up, down domain.Cents
		ids      []uint
	}
	var land, other bucket
	for _, corr := range data.Corrections {
		u, ok := corr.UsageFor(year)
		if !ok || !u.Booked() || u.Amount == 0 {
			continue
		}
		b := &other
		if corr.Immovable {
			b = &land
		}
		if u.Amount > 0 {
			b.up += u.Amount
		} else {
			b.down += -u.Amount
		}
		b.ids = append(b.ids, *u.EntryID)
	}
	kz64, kzIDs := vatValue(data.Annual, "64", true)
	if land.up-land.down+other.up-other.down != kz64 {
		// Die Kennziffer zählt, was gebucht ist; das Verzeichnis erklärt es nur.
		other = bucket{ids: kzIDs}
		land = bucket{}
		if kz64 > 0 {
			other.up = kz64
		} else {
			other.down = -kz64
		}
		if kz64 != 0 {
			c.finding(SeverityWarning, "vorsteuerberichtigung_ohne_verzeichnis", "ust.z100.abziehbar",
				fmt.Sprintf("Die gebuchte Berichtigung des Vorsteuerabzugs (%s €) stimmt nicht mit dem Verzeichnis nach § 15a UStG überein; sie steht ungeteilt in Zeile 100.", kz64),
				"§ 15a UStG")
		}
	}
	entries := func(b bucket) Origin {
		return Origin{Kind: OriginEntries, EntryIDs: b.ids, Formula: "Berichtigungen nach § 15a UStG des Jahres"}
	}
	c.books("ust.z99.abziehbar", Cents(land.up), entries(land))
	c.books("ust.z99.zurueck", Cents(land.down), entries(land))
	c.books("ust.z100.abziehbar", Cents(other.up), entries(other))
	c.books("ust.z100.zurueck", Cents(other.down), entries(other))
	c.resolve("ust.z101.abziehbar", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone}})
	c.resolve("ust.z101.zurueck", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone}})
	c.sum("ust.z102.kz357", "ust.z99.abziehbar", "ust.z100.abziehbar", "ust.z101.abziehbar")
	c.sum("ust.z102.kz359", "ust.z99.zurueck", "ust.z100.zurueck", "ust.z101.zurueck")

	landChanged := land.up+land.down != 0
	otherChanged := other.up+other.down != 0
	o := Origin{Kind: OriginComputation, FromLines: []string{"ust.z99.abziehbar", "ust.z100.abziehbar"},
		Text: "Verzeichnis nach § 15a UStG"}
	c.resolve("ust.z89.kz370", figure{value: Bool(false), status: StatusAuto, origin: Origin{Kind: OriginNone}})
	c.resolve("ust.z90.kz371", figure{value: Bool(landChanged), status: StatusAuto, origin: o})
	c.resolve("ust.z91.kz372", figure{value: Bool(otherChanged), status: StatusAuto, origin: o})
	c.resolve("ust.z92.kz369", figure{value: Bool(false), status: StatusAuto, origin: Origin{Kind: OriginNone}})
	if landChanged || otherChanged {
		reason := c.resolve("ust.z93.grund", figure{status: StatusOpen, origin: Origin{Kind: OriginNone},
			notes: []string{"Grund der Änderung ankreuzen: Veräußerung, Lieferung nach § 3 Abs. 1b UStG, Wechsel der Besteuerungsform oder Nutzungsänderung."}})
		if reason.status == StatusOpen {
			c.finding(SeverityBlocking, "zeile_offen", "ust.z93.grund",
				"Der Grund der Berichtigung nach § 15a UStG fehlt (Zeilen 93 bis 98).", "§ 15a UStG")
		}
	} else {
		c.resolve("ust.z93.grund", noneFigure())
	}
}

// prepaymentTarget setzt das Vorauszahlungssoll: die Summe der angemeldeten
// Vorauszahlungen und Überschüsse einschließlich der angerechneten
// Sondervorauszahlung.
func (c *calc) prepaymentTarget(data VatYearData) {
	var soll domain.Cents
	submitted := map[string]bool{}
	periods := []string{}
	for _, r := range data.Submitted {
		soll += r.Tax("83") + r.Tax("39")
		submitted[r.PeriodKey] = true
		periods = append(periods, r.PeriodKey)
	}
	sort.Strings(periods)
	missing := []string{}
	for _, p := range data.ExpectedPeriods {
		if !submitted[p] {
			missing = append(missing, p)
		}
	}
	o := Origin{Kind: OriginVatReturns,
		Formula: "Summe Kz 83 und Kz 39 der übermittelten Voranmeldungen",
		Text:    "Voranmeldungen " + strings.Join(periods, ", ")}
	if len(data.ExpectedPeriods) == 0 {
		o.Text = "Keine Voranmeldungen abzugeben"
	}
	if len(missing) > 0 {
		c.resolve("ust.z119", figure{value: Cents(soll), status: StatusProposed, origin: o,
			notes: []string{fmt.Sprintf("Für %s ist keine Voranmeldung als übermittelt erfasst.", strings.Join(missing, ", "))}})
		c.finding(SeverityWarning, "voranmeldung_fehlt", "ust.z119",
			fmt.Sprintf("Für %s ist keine Voranmeldung als übermittelt erfasst. Das Vorauszahlungssoll enthält sie nicht.",
				strings.Join(missing, ", ")), "§ 18 Abs. 1 UStG")
	} else {
		c.resolve("ust.z119", figure{value: Cents(soll), status: StatusAuto, origin: o})
	}

	paid, paidOrigin := data.Ledger.sum(full("3820", "3830"), true, AccountTotals.NetBeforeClosing)
	if len(data.ExpectedPeriods) > 0 && paid != c.amount("ust.z119") {
		c.finding(SeverityHint, "vorauszahlungen_gezahlt", "ust.z119",
			fmt.Sprintf("Auf den Vorauszahlungskonten (%s) sind im Geschäftsjahr %s € gebucht, angemeldet sind %s €. "+
				"Die Differenz sind offene, erst im Folgejahr gezahlte oder erstattete Vorauszahlungen.",
				accountList(paidOrigin), paid, c.amount("ust.z119")), "")
	}
}

func accountList(o Origin) string {
	out := []string{}
	for _, a := range o.Accounts {
		out = append(out, a.Account)
	}
	if len(out) == 0 {
		return "3820, 3830"
	}
	return strings.Join(out, ", ")
}

// reconcileVat vergleicht je Kennzahl das Jahr mit der Summe der übermittelten
// Voranmeldungen.
func reconcileVat(c *calc, data VatYearData) []ReconciliationRow {
	if len(data.Submitted) == 0 {
		return nil
	}
	rows := []ReconciliationRow{}
	seen := map[string]bool{}
	all := append(append([]vatMapping{}, vatMappings...), vatReconcileOnly...)
	for _, m := range all {
		id := fmt.Sprintf("%s/%t", m.code, m.tax)
		if seen[id] {
			continue
		}
		seen[id] = true
		year, yearIDs := vatValue(data.Annual, m.code, m.tax)
		var submitted domain.Cents
		reported := map[uint]bool{}
		for i := range data.Submitted {
			v, ids := vatValue(&data.Submitted[i], m.code, m.tax)
			submitted += v
			for _, id := range ids {
				reported[id] = true
			}
		}
		if year == 0 && submitted == 0 {
			continue
		}
		field := "bemessungsgrundlage"
		if m.tax {
			field = "steuer"
		}
		row := ReconciliationRow{Code: m.code, Label: m.label, Field: field,
			Year: year, Submitted: submitted, Difference: year - submitted}
		for _, id := range yearIDs {
			if !reported[id] {
				row.UnreportedIDs = append(row.UnreportedIDs, id)
			}
		}
		if row.Difference != 0 {
			row.Explanation = explainVatDifference(row, m, len(data.Submitted))
			if row.Explanation == "" {
				row.Explanation = "Grund nicht erkennbar."
				c.finding(SeverityWarning, "abstimmung_differenz", m.key,
					fmt.Sprintf("Kz %s (%s): Jahreswert %s €, Summe der Voranmeldungen %s €, Differenz %s € ohne erkennbaren Grund.",
						m.code, field, year, submitted, row.Difference), "§ 18 Abs. 3 UStG")
			} else {
				c.finding(SeverityHint, "abstimmung_differenz_erklaert", m.key,
					fmt.Sprintf("Kz %s (%s): Differenz %s € zur Summe der Voranmeldungen. %s", m.code, field, row.Difference, row.Explanation), "")
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func explainVatDifference(row ReconciliationRow, m vatMapping, periods int) string {
	switch {
	case len(row.UnreportedIDs) > 0:
		return fmt.Sprintf("%d Buchung(en) stehen in keiner übermittelten Voranmeldung (Nachtrag ohne berichtigte Voranmeldung).",
			len(row.UnreportedIDs))
	case m.code == "64":
		return "Berichtigungen nach § 15a UStG bis 6.000 € je Wirtschaftsgut gehören erst in die Jahreserklärung (§ 44 Abs. 3 UStDV)."
	case !m.tax && row.Difference.Abs() < domain.Cents(100*periods):
		return "Rundung der Bemessungsgrundlage auf volle Euro in jeder Voranmeldung."
	}
	return ""
}

// checkZM gleicht Kz 721 mit den Zusammenfassenden Meldungen ab.
func (c *calc) checkZM(data VatYearData) {
	kz721 := c.amount("ust.z74.kz721")
	if len(data.ZMMissing) > 0 {
		c.finding(SeverityWarning, "zm_fehlt", "ust.z74.kz721",
			fmt.Sprintf("Für %s ist keine Zusammenfassende Meldung als übermittelt erfasst.", strings.Join(data.ZMMissing, ", ")),
			"§ 18a UStG")
	}
	// Kz 721 ist auf volle Euro abgeschnitten, jede Zeile der Meldung ebenso;
	// je Zeile kann der Unterschied knapp einen Euro betragen.
	tolerance := domain.Cents(100 * (data.ZMLines + 1))
	if diff := kz721 - data.ZMServices; diff.Abs() >= tolerance {
		c.finding(SeverityWarning, "zm_abweichung", "ust.z74.kz721",
			fmt.Sprintf("Kz 721 (%s €) weicht von der Summe der Zusammenfassenden Meldungen (%s €) ab.", kz721, data.ZMServices),
			"§ 18a, § 18b Satz 1 Nr. 2 UStG")
	}
}

// checkExpectedTax vergleicht die gebuchte Steuer mit Bemessungsgrundlage mal
// Satz.
func (c *calc) checkExpectedTax(annual *domain.VatReturn) {
	if annual == nil {
		return
	}
	for _, l := range annual.Figures {
		if dev := l.Deviation(); dev.Abs() > 100 {
			c.finding(SeverityWarning, "steuer_abweichung", "",
				fmt.Sprintf("Kz %s: gebuchte Steuer %s €, aus der Bemessungsgrundlage gerechnet %s € (Abweichung %s €).",
					l.Code, l.Tax, l.ExpectedTax, dev), "")
		}
	}
}
