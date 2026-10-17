package accounting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Vordruck USt 1 A als Tabelle.
//
// Buchfink kennt nur einen Teil der Tatbestände, die der Vordruck vorsieht. Die
// übrigen Kennziffern stehen trotzdem hier und werden mit null geführt: ein
// Blatt, auf dem nur die befüllten Zeilen stehen, sieht aus wie eine vollständige
// Anmeldung und ist keine — wer es abtippt, übersieht die Zeile, die er von Hand
// hätte füllen müssen.

// vatCodeSpec beschreibt eine Zeile des Vordrucks.
type vatCodeSpec struct {
	code      string
	label     string
	reference string
	hasBase   bool
	hasTax    bool
	// taxCode ist die eigene Kennziffer des Steuerbetrags, wo der Vordruck zwei
	// vergibt (35/36, 46/47, 95/98, 73/74, 84/85, 76/80, 94/96).
	taxCode string
	// rate ist der Steuersatz, aus dem sich die rechnerische Steuer ergibt. Nur
	// gesetzt, wo der Vordruck den Satz vorgibt.
	rate domain.TaxRate
}

// vatCodes ist der Vordruck USt 1 A 2026 (Formular 2026USt1A561) in seiner
// Reihenfolge; die Kennzahlen 87, 90, 50 und 37 und die Trennung 46/73/84 gelten
// laut ELSTER-Hilfe schon für 2025.
var vatCodes = []vatCodeSpec{
	// A. Steuerpflichtige Umsätze
	{code: "81", label: "Steuerpflichtige Umsätze zum Steuersatz von 19 %", reference: "§ 12 Abs. 1 UStG", hasBase: true, hasTax: true, rate: domain.TaxRateStandard},
	{code: "86", label: "Steuerpflichtige Umsätze zum Steuersatz von 7 %", reference: "§ 12 Abs. 2 UStG", hasBase: true, hasTax: true, rate: domain.TaxRateReduced},
	{code: "87", label: "Steuerpflichtige Umsätze zum Steuersatz von 0 %", reference: "§ 12 Abs. 3 UStG", hasBase: true},
	{code: "35", label: "Steuerpflichtige Umsätze zu anderen Steuersätzen", reference: "§ 12 UStG", hasBase: true, hasTax: true, taxCode: "36"},
	{code: "77", label: "Lieferungen land- und forstwirtschaftlicher Betriebe an Abnehmer mit USt-IdNr.", reference: "§ 24 UStG", hasBase: true},
	{code: "76", label: "Umsätze, für die eine Steuer nach § 24 UStG zu entrichten ist", reference: "§ 24 Abs. 1 Satz 1 Nr. 2 UStG", hasBase: true, hasTax: true, taxCode: "80"},

	// B. Steuerfreie Umsätze
	{code: "41", label: "Innergemeinschaftliche Lieferungen an Abnehmer mit USt-IdNr.", reference: "§ 4 Nr. 1 Buchst. b, § 6a UStG", hasBase: true},
	{code: "44", label: "Innergemeinschaftliche Lieferungen neuer Fahrzeuge an Abnehmer ohne USt-IdNr.", reference: "§ 6a Abs. 1 Satz 1 Nr. 2 Buchst. c UStG", hasBase: true},
	{code: "49", label: "Innergemeinschaftliche Lieferungen neuer Fahrzeuge außerhalb eines Unternehmens", reference: "§ 2a UStG", hasBase: true},
	{code: "43", label: "Weitere steuerfreie Umsätze mit Vorsteuerabzug (u. a. Ausfuhrlieferungen)", reference: "§ 4 Nr. 1 Buchst. a, Nr. 2 bis 7 UStG", hasBase: true},
	{code: "48", label: "Steuerfreie Umsätze ohne Vorsteuerabzug", reference: "§ 4 Nr. 8 bis 29, § 19 Abs. 1 UStG", hasBase: true},

	// C. Innergemeinschaftliche Erwerbe
	{code: "91", label: "Steuerfreie innergemeinschaftliche Erwerbe", reference: "§§ 4b, 25c UStG", hasBase: true},
	{code: "89", label: "Innergemeinschaftliche Erwerbe zum Steuersatz von 19 %", reference: "§ 1a, § 12 Abs. 1 UStG", hasBase: true, hasTax: true, rate: domain.TaxRateStandard},
	{code: "93", label: "Innergemeinschaftliche Erwerbe zum Steuersatz von 7 %", reference: "§ 1a, § 12 Abs. 2 UStG", hasBase: true, hasTax: true, rate: domain.TaxRateReduced},
	{code: "90", label: "Innergemeinschaftliche Erwerbe zum Steuersatz von 0 %", reference: "§ 1a, § 12 Abs. 3 UStG", hasBase: true},
	{code: "95", label: "Innergemeinschaftliche Erwerbe zu anderen Steuersätzen", reference: "§ 1a UStG", hasBase: true, hasTax: true, taxCode: "98"},
	{code: "94", label: "Innergemeinschaftliche Erwerbe neuer Fahrzeuge von Lieferern ohne USt-IdNr.", reference: "§ 1b Abs. 2 und 3 UStG", hasBase: true, hasTax: true, taxCode: "96"},

	// D. Leistungsempfänger als Steuerschuldner
	{code: "46", label: "Sonstige Leistungen nach § 3a Abs. 2 UStG eines im übrigen Gemeinschaftsgebiet ansässigen Unternehmers", reference: "§ 13b Abs. 1 UStG", hasBase: true, hasTax: true, taxCode: "47"},
	{code: "73", label: "Umsätze, die unter das Grunderwerbsteuergesetz fallen", reference: "§ 13b Abs. 2 Nr. 3 UStG", hasBase: true, hasTax: true, taxCode: "74"},
	{code: "84", label: "Andere Leistungen, für die der Leistungsempfänger die Steuer schuldet", reference: "§ 13b Abs. 2 Nr. 1, 2, 4 bis 12 UStG", hasBase: true, hasTax: true, taxCode: "85"},

	// E. Ergänzende Angaben zu Umsätzen
	{code: "42", label: "Lieferungen des ersten Abnehmers bei innergemeinschaftlichen Dreiecksgeschäften", reference: "§ 25b UStG", hasBase: true},
	{code: "60", label: "Steuerpflichtige Umsätze, für die der Leistungsempfänger die Steuer schuldet", reference: "§ 13b Abs. 5 UStG", hasBase: true},
	{code: "21", label: "Nicht steuerbare sonstige Leistungen im übrigen Gemeinschaftsgebiet", reference: "§ 18b Satz 1 Nr. 2 UStG", hasBase: true},
	{code: "45", label: "Übrige nicht steuerbare Umsätze (Leistungsort nicht im Inland)", reference: "§ 1 Abs. 1 Nr. 1 UStG", hasBase: true},

	// F. Abziehbare Vorsteuerbeträge
	{code: "66", label: "Vorsteuerbeträge aus Rechnungen von anderen Unternehmern", reference: "§ 15 Abs. 1 Satz 1 Nr. 1 UStG", hasTax: true},
	{code: "61", label: "Vorsteuerbeträge aus dem innergemeinschaftlichen Erwerb", reference: "§ 15 Abs. 1 Satz 1 Nr. 3 UStG", hasTax: true},
	{code: "62", label: "Entstandene Einfuhrumsatzsteuer", reference: "§ 15 Abs. 1 Satz 1 Nr. 2 UStG", hasTax: true},
	{code: "67", label: "Vorsteuerbeträge aus Leistungen nach § 13b UStG", reference: "§ 15 Abs. 1 Satz 1 Nr. 4 UStG", hasTax: true},
	{code: "63", label: "Vorsteuerbeträge nach allgemeinen Durchschnittssätzen", reference: "§ 23a UStG", hasTax: true},
	{code: "59", label: "Vorsteuerabzug für innergemeinschaftliche Lieferungen neuer Fahrzeuge außerhalb eines Unternehmens und von Kleinunternehmern", reference: "§ 2a, § 15 Abs. 4a UStG", hasTax: true},
	{code: "64", label: "Berichtigung des Vorsteuerabzugs", reference: "§ 15a UStG", hasTax: true},

	// G. Andere Steuerbeträge
	{code: "65", label: "Steuer infolge Wechsels der Besteuerungsform sowie Nachsteuer wegen Steuersatzänderung", reference: "§ 19 Abs. 2, § 24 Abs. 4, § 27 Abs. 1 UStG", hasTax: true},
	{code: "69", label: "Unrichtig oder unberechtigt ausgewiesene Steuerbeträge und weitere geschuldete Beträge", reference: "§ 14c, § 6a Abs. 4, § 17 Abs. 1 Satz 7, § 25b Abs. 2 UStG", hasTax: true},

	// H. Vorauszahlung
	{code: "39", label: "Abzug der festgesetzten Sondervorauszahlung für Dauerfristverlängerung", reference: "§ 48 Abs. 4 UStDV", hasTax: true},
	{code: "83", label: "Verbleibende Umsatzsteuer-Vorauszahlung (Überschuss mit Minuszeichen)", reference: "§ 18 Abs. 1 UStG", hasTax: true},

	// I. Ergänzende Angaben zu Minderungen wegen Uneinbringlichkeit. Beide
	// Beträge sind in den Zeilen darüber schon enthalten und gehen in keine
	// Summe ein.
	{code: "50", label: "Minderung der Bemessungsgrundlage wegen Uneinbringlichkeit (in Kz 81, 86, 87, 35, 77 und 76 enthalten)", reference: "§ 17 Abs. 2 Nr. 1 Satz 1 UStG", hasBase: true},
	{code: "37", label: "Minderung der abziehbaren Vorsteuerbeträge wegen Uneinbringlichkeit (in Kz 66, 63 und 59 enthalten)", reference: "§ 17 Abs. 2 Nr. 1 Satz 1 UStG", hasTax: true},
}

// Kennziffern, auf die sich der Rest des Programms bezieht.
const (
	VatCodeStandardRate        = "81"
	VatCodeReducedRate         = "86"
	VatCodeZeroRate            = "87" // § 12 Abs. 3 UStG
	VatCodeOtherRates          = "35"
	VatCodeExemptNoDeduction   = "48"
	VatCodeAcquisition19       = "89"
	VatCodeAcquisition7        = "93"
	VatCodeReverseChargeEU     = "46" // § 13b Abs. 1 UStG
	VatCodeReverseChargeOther  = "84" // § 13b Abs. 2 UStG ohne Nr. 3
	VatCodeRecipientLiable     = "60" // § 13b Abs. 5 UStG, eigener Umsatz
	VatCodeEUServices          = "21" // nicht steuerbare sonstige Leistungen § 18b
	VatCodeNotTaxable          = "45"
	VatCodeInputTax            = "66"
	VatCodeInputTaxAcquisition = "61"
	VatCodeInputTaxReverse     = "67"
	VatCodeInputTaxCorrection  = "64" // Berichtigung § 15a UStG
	VatCodeUnlawfulTax         = "69" // § 14c UStG
	VatCodeSpecialPrepayment   = "39"
	VatCodePayable             = "83"
	VatCodeBadDebtBase         = "50" // § 17 Abs. 2 Nr. 1 UStG, Umsatzseite
	VatCodeBadDebtInputTax     = "37" // § 17 Abs. 2 Nr. 1 UStG, Vorsteuerseite
)

// owedTaxCodes sind die Kennziffern, deren Steuerbeträge in die Zahllast
// eingehen; deductibleTaxCodes die, die sie mindern.
var (
	owedTaxCodes       = []string{"81", "86", "35", "76", "89", "93", "95", "94", "46", "73", "84", "65", "69"}
	deductibleTaxCodes = []string{"66", "61", "62", "67", "63", "59", "64"}
)

// VatReturnSource ist alles, was ein Kennziffernblatt braucht und nicht im
// Journal steht.
//
// Die drei Funktionen sind bewusst Funktionen und keine Karten: sie beantworten
// Fragen an Stammdaten und an bereits übermittelte Anmeldungen, und der
// Kennziffernrechner soll weder Repositories kennen noch einen Kontext führen.
type VatReturnSource struct {
	Entries []domain.JournalEntry
	// ReceivedAt liefert den Belegeingang zu einer Beleg-ID; "" wenn unbekannt.
	ReceivedAt func(receiptID uint) string
	// EURecipient meldet, ob ein Geschäftspartner im übrigen Gemeinschaftsgebiet
	// sitzt und eine USt-IdNr. hat. Davon hängt ab, ob eine Leistung mit
	// Steuerschuld des Empfängers in Kz 21 oder in Kz 45 gehört.
	EURecipient func(contactID uint) bool
	// SubmittedPeriod meldet, ob der Zeitraum, in den ein Datum fällt, bereits
	// als übermittelt bestätigt ist.
	SubmittedPeriod func(date string) bool
	// ReportedEntry meldet, ob eine Buchung in der übermittelten Anmeldung ihres
	// Zeitraums bereits enthalten war.
	//
	// Ohne diese Frage wäre jede Buchung eines übermittelten Zeitraums ein
	// Nachtrag — auch die, die selbst gemeldet wurde. Nachtrag ist nur, was der
	// Anmeldung damals fehlte.
	ReportedEntry func(date string, entryID uint) bool
	// DomesticRecipient meldet, ob ein Geschäftspartner im Inland sitzt. Eine
	// Leistung mit Steuerschuld des Empfängers an ihn ist ein Umsatz nach
	// § 13b Abs. 5 UStG (Kz 60) und kein nicht steuerbarer.
	DomesticRecipient func(contactID uint) bool
	// Allocations sind die Zahlungszuordnungen zu offenen Posten. Aus ihnen
	// folgt, was vor der Leistung vereinnahmt oder bezahlt wurde
	// (Mindest-Istversteuerung) und welche Buchung eine Forderung als
	// uneinbringlich ausbucht (Kz 50).
	Allocations []domain.PaymentAllocation
	// SpecialPrepayment ist die anzurechnende Sondervorauszahlung (Kz 39). Sie
	// wird nur im letzten Zeitraum des Jahres angerechnet (§ 48 Abs. 4 UStDV);
	// darüber entscheidet der Aufrufer, nicht diese Funktion.
	SpecialPrepayment domain.Cents
}

// vatMovement ist der Beitrag einer Journalzeile zu einer Kennziffer.
type vatMovement struct {
	entryIndex int
	date       string
	code       string
	base       domain.Cents
	tax        domain.Cents
	// trigger ist die Zahlungsbuchung, die den Betrag vor der Leistung in
	// diesen Zeitraum gezogen hat; 0, wo die Buchung selbst entscheidet.
	trigger uint
}

// BuildVatReturn rechnet das Kennziffernblatt eines Zeitraums aus dem Journal.
//
// Zugeordnet wird nach VatPeriodFor und damit nach dem Entstehen der Steuer,
// nicht nach dem Buchungsdatum. Was in einen bereits übermittelten Zeitraum
// gehört, landet in der Liste der Nachträge — der Anwender entscheidet dann
// über eine Berichtigung.
func BuildVatReturn(period VatPeriod, src VatReturnSource) *domain.VatReturn {
	movements := vatMovements(src)

	ret := &domain.VatReturn{
		FiscalYear: period.Year,
		PeriodType: period.Type,
		PeriodKey:  period.Key,
		PeriodFrom: period.From,
		PeriodTo:   period.To,
		Status:     domain.VatReturnDraft,
	}

	type bucket struct {
		base domain.Cents
		tax  domain.Cents
		ids  map[uint]bool
		list []uint
	}
	buckets := map[string]*bucket{}
	add := func(code string, m vatMovement, entryID uint) {
		b := buckets[code]
		if b == nil {
			b = &bucket{ids: map[uint]bool{}}
			buckets[code] = b
		}
		b.base += m.base
		b.tax += m.tax
		for _, id := range []uint{entryID, m.trigger} {
			if id != 0 && !b.ids[id] {
				b.ids[id] = true
				b.list = append(b.list, id)
			}
		}
	}

	yearStart := fmt.Sprintf("%d-01-01", period.Year)
	for _, m := range movements {
		entry := &src.Entries[m.entryIndex]
		switch {
		case m.date >= period.From && m.date <= period.To:
			add(m.code, m, entry.ID)

		// Nachträge werden auf das Geschäftsjahr des Zeitraums begrenzt. Eine
		// Buchung, die in ein früheres Jahr gehört, ist kein Nachtrag zu diesem
		// Blatt: sie berichtigt eine Anmeldung, die längst in einer
		// Jahreserklärung aufgegangen ist.
		case m.date >= yearStart && m.date < period.From && submitted(src, m.date) && !reportedMovement(src, m, entry.ID):
			// Nachtrag: der Zeitraum ist übermittelt, die Buchung kam später.
			periodKey := m.date
			if p, err := VatPeriodOf(m.date, period.Type); err == nil {
				periodKey = p.Key
			}
			ret.LateEntries = append(ret.LateEntries, domain.VatLateEntry{
				EntryID:     entry.ID,
				EntryNumber: entry.EntryNumber,
				BookingDate: entry.BookingDate,
				PeriodKey:   periodKey,
				Description: entry.Description,
				Code:        m.code,
				Base:        TruncToEuro(m.base),
				Tax:         m.tax,
			})
		}
	}

	// Die Sondervorauszahlung ist keine Buchung dieses Zeitraums, sondern eine
	// Anrechnung auf ihn (§ 48 Abs. 4 UStDV).
	if src.SpecialPrepayment != 0 {
		add(VatCodeSpecialPrepayment, vatMovement{tax: src.SpecialPrepayment}, 0)
	}

	var owed, deductible domain.Cents
	ret.Figures = make([]domain.VatReturnLine, 0, len(vatCodes))
	for _, spec := range vatCodes {
		line := domain.VatReturnLine{
			Code:      spec.code,
			Label:     spec.label,
			Reference: spec.reference,
			HasBase:   spec.hasBase,
			HasTax:    spec.hasTax,
			TaxCode:   spec.taxCode,
		}
		if b := buckets[spec.code]; b != nil {
			// Bemessungsgrundlagen führt der Vordruck in vollen Euro. Abgerundet
			// wird zum Betrag hin — bei einer Gutschrift ist das dieselbe
			// Richtung, sonst wüchse eine Korrektur beim Runden.
			line.Base = TruncToEuro(b.base)
			line.Tax = b.tax
			line.EntryIDs = b.list
			if spec.rate != domain.TaxRateNone && spec.hasBase {
				// Die rechnerische Steuer entsteht aus der *ungerundeten*
				// Grundlage. Die Abrundung auf volle Euro ist eine Vorschrift
				// über das Blatt und keine Aussage über die Steuer: aus der
				// gerundeten Grundlage gerechnet zeigte die Abweichung bis zu
				// 19 Cent je Kennziffer an, obwohl die gebuchte Steuer stimmt —
				// und eine Anzeige, die immer ausschlägt, wird nicht mehr
				// gelesen.
				line.ExpectedTax = spec.rate.Tax(b.base)
			}
		}
		ret.Figures = append(ret.Figures, line)
		switch {
		case contains(owedTaxCodes, spec.code):
			owed += line.Tax
		case contains(deductibleTaxCodes, spec.code):
			deductible += line.Tax
		}
	}

	// Kz 83: geschuldete Steuer abzüglich Vorsteuer und Sondervorauszahlung.
	payable := owed - deductible - ret.Tax(VatCodeSpecialPrepayment)
	for i := range ret.Figures {
		if ret.Figures[i].Code == VatCodePayable {
			ret.Figures[i].Tax = payable
		}
	}
	ret.Payable = payable

	unrounded := make(map[string]domain.Cents, len(buckets))
	for code, b := range buckets {
		unrounded[code] = b.base
	}
	ret.Findings = vatReturnFindings(ret, unrounded)

	sort.SliceStable(ret.LateEntries, func(i, j int) bool {
		if ret.LateEntries[i].PeriodKey != ret.LateEntries[j].PeriodKey {
			return ret.LateEntries[i].PeriodKey < ret.LateEntries[j].PeriodKey
		}
		return ret.LateEntries[i].EntryID < ret.LateEntries[j].EntryID
	})
	ret.EnsureLists()
	return ret
}

func submitted(src VatReturnSource, date string) bool {
	if src.SubmittedPeriod == nil {
		return false
	}
	return src.SubmittedPeriod(date)
}

func reported(src VatReturnSource, date string, entryID uint) bool {
	if src.ReportedEntry == nil {
		return false
	}
	return src.ReportedEntry(date, entryID)
}

// reportedMovement meldet, ob ein Beitrag in der übermittelten Anmeldung
// seines Zeitraums enthalten war. Hat eine Zahlung ihn vorgezogen, muss auch
// sie gemeldet worden sein: eine spätere Zahlung auf eine schon gemeldete
// Rechnung ist ein Nachtrag.
func reportedMovement(src VatReturnSource, m vatMovement, entryID uint) bool {
	if !reported(src, m.date, entryID) {
		return false
	}
	return m.trigger == 0 || reported(src, m.date, m.trigger)
}

// vatMovements zerlegt die Buchungen in ihre Beiträge zu den Kennziffern.
func vatMovements(src VatReturnSource) []vatMovement {
	treatments := RevenueTreatments()
	paid := indexPrepayments(src.Entries, src.Allocations)
	out := make([]vatMovement, 0, len(src.Entries))

	for i := range src.Entries {
		entry := &src.Entries[i]
		// Der Saldenvortrag bringt den Bestand der Steuerkonten ins neue Jahr.
		// Er ist kein Umsatz dieses Zeitraums; ohne diese Zeile stünde die
		// Vorsteuer des Vorjahres ein zweites Mal in der Anmeldung des Januars.
		if entry.Source == domain.EntrySourceOpening {
			continue
		}
		// Die Abschlussbuchungen bleiben ebenso außen vor — die
		// Umsatzsteuer-Jahresverrechnung stellt die Steuerkonten zum
		// Bilanzstichtag auf null und ist kein Umsatz des letzten
		// Voranmeldungszeitraums. Mit einer Ausnahme: die Vorsteuerberichtigung
		// nach § 15a UStG ist eine Abschlussbuchung *und* gehört in die
		// Anmeldung. Sie wird deshalb nicht am Kopf der Buchung ausgesondert,
		// sondern Zeile für Zeile — ihre Zeilen haben den Steuerschlüssel, alle
		// anderen einer Abschlussbuchung nicht.
		closing := entry.Source == domain.EntrySourceClosing
		receivedAt := ""
		if entry.ReceiptID != nil && src.ReceivedAt != nil {
			receivedAt = src.ReceivedAt(*entry.ReceiptID)
		}
		gross, payments := paid.paymentsOf(entry)
		writeOff := paid.isWriteOff(entry)

		for _, line := range entry.Lines {
			if closing && line.TaxKey != TaxKeyInputTaxCorrection {
				continue
			}
			// Vorzeichen in der natürlichen Richtung des Kontos: die
			// Bemessungsgrundlage folgt derselben Seitenlogik wie der
			// Steuerbetrag, sonst senkte ein Skonto die Steuer und erhöhte
			// zugleich den Umsatz, aus dem sie stammt.
			credit, base := line.Amount, line.TaxBase
			if line.Side == domain.SideDebit {
				credit, base = -credit, -base
			}
			debit := -credit

			date := VatPeriodFor(entry, line, receivedAt)
			m := vatMovement{entryIndex: i, date: date}
			// earlier sagt, ab wann eine Zahlung vor der Leistung die Steuer
			// entstehen lässt oder den Abzug erlaubt; nil, wo sie es nicht tut.
			var earlier func(string) string

			// Die Skonto-Steuerkorrektur läuft in dieselbe Kennziffer wie der
			// Umsatz, den sie mindert: § 17 Abs. 1 UStG berichtigt Steuer und
			// Bemessungsgrundlage im Zeitraum der Änderung (Satz 8). Der eigene
			// Schlüssel trennt die Korrektur im Journal; das Vorzeichen kommt aus
			// der Seite der Buchung.
			key := strings.TrimPrefix(line.TaxKey, TaxKeySkontoPrefix)

			switch {
			case key == "":
				treatment := treatments[line.Account]
				if treatment == "" {
					continue
				}
				m.base = credit
				switch treatment {
				case domain.TaxTreatmentExempt:
					m.code = VatCodeExemptNoDeduction
				case domain.TaxTreatmentZeroRated:
					// § 12 Abs. 3 UStG ist keine Befreiung, sondern ein
					// steuerpflichtiger Umsatz zum Satz null: Kz 87.
					m.code = VatCodeZeroRate
					earlier = paymentDate
				case domain.TaxTreatmentReverseChargeSupply:
					m.code = reverseChargeSupplyCode(src, entry)
				case domain.TaxTreatmentNotTaxable:
					m.code = VatCodeNotTaxable
				default:
					continue
				}

			case key == "UST19":
				m.code, m.base, m.tax, earlier = VatCodeStandardRate, base, credit, paymentDate
			case key == "UST7":
				m.code, m.base, m.tax, earlier = VatCodeReducedRate, base, credit, paymentDate
			case key == TaxKeyUnlawful:
				// § 14c UStG: der Betrag wird geschuldet, obwohl kein
				// steuerpflichtiger Umsatz dahintersteht. Er hat deshalb keine
				// Bemessungsgrundlage im Vordruck.
				m.code, m.tax = VatCodeUnlawfulTax, credit
			case key == "IG19_UST":
				m.code, m.base, m.tax = VatCodeAcquisition19, base, credit
			case key == "IG7_UST":
				m.code, m.base, m.tax = VatCodeAcquisition7, base, credit
			case isReverseChargeKey(key, TaxKeyReverseChargeEUPrefix, "_UST"):
				m.code, m.base, m.tax, earlier = VatCodeReverseChargeEU, base, credit, paymentDate
			case isReverseChargeKey(key, TaxKeyReverseChargePrefix, "_UST"):
				m.code, m.base, m.tax, earlier = VatCodeReverseChargeOther, base, credit, paymentDate
			case key == "VST19", key == "VST7":
				m.code, m.tax = VatCodeInputTax, debit
				docDate := entry.DocumentDate
				earlier = func(payment string) string { return latest(payment, docDate, receivedAt) }
			case key == "IG19_VST", key == "IG7_VST":
				m.code, m.tax = VatCodeInputTaxAcquisition, debit
			case isReverseChargeKey(key, TaxKeyReverseChargeEUPrefix, "_VST"),
				isReverseChargeKey(key, TaxKeyReverseChargePrefix, "_VST"):
				m.code, m.tax, earlier = VatCodeInputTaxReverse, debit, paymentDate
			case key == TaxKeyInputTaxCorrection:
				// Die Berichtigung hat keine Bemessungsgrundlage im Vordruck: sie
				// ist ein Steuerbetrag, der eine frühere Grundlage nachträglich
				// anders bewertet. Die zurückzuzahlende Vorsteuer steht im Haben
				// und mindert damit die abziehbaren Beträge der Kennziffer 64.
				m.code, m.tax = VatCodeInputTaxCorrection, debit
			default:
				continue
			}

			parts := splitByPayments(m, gross, payments, earlier)
			out = append(out, parts...)

			// Die Ausbuchung einer uneinbringlichen Forderung mindert die
			// Bemessungsgrundlage in Kz 81 oder 86 und wird zusätzlich in Kz 50
			// genannt, als Betrag der Minderung.
			if writeOff && (key == "UST19" || key == "UST7") && line.TaxKey == key {
				out = append(out, vatMovement{entryIndex: i, date: date, code: VatCodeBadDebtBase, base: -base})
			}
		}
	}
	return out
}

// paymentDate: bei der Umsatzsteuer und beim § 13b entsteht die Steuer auf
// eine Zahlung vor der Leistung mit Ablauf des Zeitraums der Zahlung.
func paymentDate(date string) string { return date }

// isReverseChargeKey erkennt die Schlüssel eines § 13b-Absatzes. Der Präfix
// „RC" ist in „RCEU" enthalten; geprüft wird deshalb der Satz dazwischen.
func isReverseChargeKey(key, prefix, suffix string) bool {
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
		return false
	}
	rate := strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
	if rate == "" {
		return false
	}
	for _, r := range rate {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// reverseChargeSupplyCode ordnet die eigene Leistung mit Steuerschuld des
// Empfängers nach dessen Sitz: im übrigen Gemeinschaftsgebiet mit USt-IdNr.
// eine sonstige Leistung nach § 18b Satz 1 Nr. 2 UStG (Kz 21), im Inland ein
// Umsatz nach § 13b Abs. 5 UStG (Kz 60), sonst ein übriger nicht steuerbarer
// Umsatz (Kz 45).
func reverseChargeSupplyCode(src VatReturnSource, entry *domain.JournalEntry) string {
	if entry.ContactID == nil {
		return VatCodeNotTaxable
	}
	switch {
	case src.EURecipient != nil && src.EURecipient(*entry.ContactID):
		return VatCodeEUServices
	case src.DomesticRecipient != nil && src.DomesticRecipient(*entry.ContactID):
		return VatCodeRecipientLiable
	}
	return VatCodeNotTaxable
}

// RevenueTreatments ordnet jedes Erlöskonto seinem Steuerfall zu.
//
// Der Katalog der Buchungsgruppen liefert die Zuordnung; dazu kommen die
// steuerfreien Erlöskonten des SKR04, die keine Gruppe anbietet, eine
// Handbuchung aber erreicht. Die Tabelle steht einmal — eine zweite Kopie in der
// Auswertung driftete stillschweigend, und ein neuer Steuerfall fiele einfach
// aus der Voranmeldung heraus.
func RevenueTreatments() map[string]domain.TaxTreatment {
	out := RevenueAccountTreatments()
	for _, account := range []string{"4110", "4160", "4165"} {
		out[account] = domain.TaxTreatmentExempt
	}
	// Das gewährte Skonto auf eine Leistung mit Steuerschuld des Empfängers
	// mindert deren Bemessungsgrundlage in Kz 21, 45 oder 60 und in der
	// Zusammenfassenden Meldung (§ 17 Abs. 1 Satz 8, § 18a Abs. 7 Satz 2 UStG).
	out[ReverseChargeSupplySkontoAccount] = domain.TaxTreatmentReverseChargeSupply
	return out
}

// ReverseChargeSupplySkontoAccount ist das Skontokonto zum Erlöskonto 4337
// (SKR04: Gewährte Skonti aus Leistungen, für die der Leistungsempfänger die
// Umsatzsteuer nach § 13b UStG schuldet).
const ReverseChargeSupplySkontoAccount = "4741"

// TruncToEuro schneidet die Cent ab, wie es die amtlichen Vordrucke für die
// Bemessungsgrundlagen und für die Zusammenfassende Meldung vorsehen. Gerundet
// wird zum Betrag hin, damit eine negative Grundlage (Gutschrift, Storno) nicht
// größer wird als der Umsatz, den sie zurücknimmt.
func TruncToEuro(c domain.Cents) domain.Cents {
	return c - c%100
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
