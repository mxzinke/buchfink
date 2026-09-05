package accounting

import (
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

// TaxParameters are the statutory values a booking depends on that the
// legislator changes over time.
//
// They are keyed by the period they apply to rather than being editable master
// data. The distinction matters: these are not choices the user makes. Making
// them editable would invite a wrong entry to produce a quietly wrong booking,
// while a fixed constant in the code would silently misbook a reworked prior
// year. A dated table does neither, and PostingRuleVersion on every entry says
// which set produced it.
type TaxParameters struct {
	// ValidFrom is the first day this set applies to, as ISO date.
	ValidFrom string

	// EntertainmentDeductiblePermille is the deductible share of entertainment
	// expenses, in permille (700 = 70 %). § 4 Abs. 5 Satz 1 Nr. 2 EStG. The input
	// tax is unaffected — § 15 Abs. 1a Satz 2 UStG takes entertainment out of the
	// deduction ban.
	EntertainmentDeductiblePermille int64

	// SmallAmountInvoiceLimit is the gross ceiling below which a Kleinbetrags-
	// rechnung is enough (§ 33 UStDV). Relevant for the e-invoice obligation: a
	// small-amount invoice may always be a sonstige Rechnung.
	SmallAmountInvoiceLimit domain.Cents

	// CorporateTaxPermille ist der Körperschaftsteuersatz in Promille
	// (150 = 15 %, § 23 Abs. 1 KStG).
	CorporateTaxPermille int64
	// SolidarityPermille ist der Solidaritätszuschlag auf die
	// Körperschaftsteuer (55 = 5,5 %, § 4 SolZG).
	SolidarityPermille int64
	// TradeTaxBasePermille ist die Steuermesszahl der Gewerbesteuer
	// (35 = 3,5 %, § 11 Abs. 2 GewStG). Der Hebesatz kommt von der Gemeinde und
	// steht deshalb in den Einstellungen, nicht hier.
	TradeTaxBasePermille int64
	// WithholdingTaxPermille ist der Kapitalertragsteuersatz auf eine
	// Ausschüttung (250 = 25 %, § 43a Abs. 1 Satz 1 Nr. 1 EStG).
	WithholdingTaxPermille int64

	// GiftDeductibleLimit ist die Freigrenze für Geschenke an Personen, die
	// nicht Arbeitnehmer sind (§ 4 Abs. 5 Satz 1 Nr. 1 EStG), je Empfänger und
	// Wirtschaftsjahr, netto.
	//
	// Es ist eine Freigrenze und kein Freibetrag: wird sie überschritten, ist
	// nicht der übersteigende Teil nicht abziehbar, sondern der ganze Betrag —
	// und mit ihm nach § 15 Abs. 1a UStG auch die Vorsteuer.
	GiftDeductibleLimit domain.Cents

	// Die drei Bagatellgrenzen des § 44 UStDV zur Vorsteuerberichtigung nach
	// § 15a UStG. Sie stehen zusammen, weil sie aufeinander aufbauen: die erste
	// entscheidet, ob ein Wirtschaftsgut überhaupt ins Verzeichnis gehört, die
	// zweite über die Berichtigung eines einzelnen Jahres, die dritte nur noch
	// über den Zeitpunkt.

	// InputTaxCorrectionFloor ist die Vorsteuer je Wirtschaftsgut, bis zu der
	// eine Berichtigung ganz entfällt (§ 44 Abs. 1 UStDV: 1.000 €).
	InputTaxCorrectionFloor domain.Cents
	// InputTaxCorrectionMinorPoints ist die Änderung des Verwendungsanteils in
	// Prozentpunkten, unterhalb derer § 44 Abs. 2 UStDV die Jahresberichtigung
	// erlässt — sofern auch der Betrag klein bleibt.
	InputTaxCorrectionMinorPoints int64
	// InputTaxCorrectionMinorAmount ist der zugehörige Betrag (§ 44 Abs. 2
	// UStDV: 1.000 €). Beide Bedingungen müssen zusammentreffen.
	InputTaxCorrectionMinorAmount domain.Cents
	// InputTaxCorrectionAnnualAmount ist der Betrag, bis zu dem die Berichtigung
	// erst bei der Steuerberechnung für das Kalenderjahr vorzunehmen ist
	// (§ 44 Abs. 3 UStDV: 6.000 €). Sie entfällt dadurch nicht — sie wandert nur
	// aus dem Voranmeldungszeitraum ans Jahresende.
	InputTaxCorrectionAnnualAmount domain.Cents

	// NearAcquisitionPermille ist der Anteil der Gebäude-Anschaffungskosten, ab
	// dem Instandsetzungs- und Modernisierungsaufwendungen der ersten drei Jahre
	// zu anschaffungsnahen Herstellungskosten werden (§ 6 Abs. 1 Nr. 1a EStG:
	// 15 %, netto gerechnet).
	NearAcquisitionPermille int64
	// NearAcquisitionYears ist der Zeitraum, über den dieser Aufwand summiert
	// wird: drei Jahre nach der Anschaffung.
	NearAcquisitionYears int
}

// taxParameterSets are ordered by ValidFrom, oldest first.
//
// Historical values are kept, not overwritten: a booking dated into a closed
// period has to be reproducible with the values that applied then.
var taxParameterSets = []TaxParameters{
	{
		ValidFrom:                       "2007-01-01",
		EntertainmentDeductiblePermille: 700,
		// § 33 UStDV: 150 € until 2016, 250 € from 2017.
		SmallAmountInvoiceLimit: 15000,
		// Bis 2007 betrug der Körperschaftsteuersatz 25 % (§ 23 Abs. 1 KStG
		// a. F.); die Unternehmensteuerreform 2008 hat ihn auf 15 % gesenkt und
		// zugleich die Gewerbesteuer-Messzahl von 5 % auf 3,5 % vereinheitlicht.
		CorporateTaxPermille:   250,
		SolidarityPermille:     55,
		TradeTaxBasePermille:   50,
		WithholdingTaxPermille: 200,
		GiftDeductibleLimit:    3500,
	},
	{
		ValidFrom:                       "2008-01-01",
		EntertainmentDeductiblePermille: 700,
		SmallAmountInvoiceLimit:         15000,
		CorporateTaxPermille:            150,
		SolidarityPermille:              55,
		TradeTaxBasePermille:            35,
		// Die Abgeltungsteuer von 25 % gilt seit dem 1. Januar 2009
		// (§ 43a Abs. 1 Satz 1 Nr. 1 EStG i. d. F. des UntStRefG 2008).
		WithholdingTaxPermille: 200,
		GiftDeductibleLimit:    3500,
	},
	{
		ValidFrom:                       "2009-01-01",
		EntertainmentDeductiblePermille: 700,
		SmallAmountInvoiceLimit:         15000,
		CorporateTaxPermille:            150,
		SolidarityPermille:              55,
		TradeTaxBasePermille:            35,
		WithholdingTaxPermille:          250,
		GiftDeductibleLimit:             3500,
	},
	{
		ValidFrom:                       "2017-01-01",
		EntertainmentDeductiblePermille: 700,
		SmallAmountInvoiceLimit:         25000,
		CorporateTaxPermille:            150,
		SolidarityPermille:              55,
		TradeTaxBasePermille:            35,
		WithholdingTaxPermille:          250,
		GiftDeductibleLimit:             3500,
	},
	{
		// Das Wachstumschancengesetz hat die Freigrenze des § 4 Abs. 5 Satz 1
		// Nr. 1 EStG für Wirtschaftsjahre, die nach dem 31.12.2023 beginnen, von
		// 35 € auf 50 € angehoben. Alles andere blieb, wie es war.
		ValidFrom:                       "2024-01-01",
		EntertainmentDeductiblePermille: 700,
		SmallAmountInvoiceLimit:         25000,
		CorporateTaxPermille:            150,
		SolidarityPermille:              55,
		TradeTaxBasePermille:            35,
		WithholdingTaxPermille:          250,
		GiftDeductibleLimit:             5000,
	},
}

// withStableParameters ergänzt die Werte, die seit Beginn der geführten Zeit
// unverändert gelten.
//
// Sie stehen nicht in jedem Satz, weil sie sich nie geändert haben und eine
// vierfach wiederholte Zahl beim nächsten Satz genau einmal vergessen würde.
// Ändert der Gesetzgeber einen von ihnen, wandert er in die Sätze — die Struktur
// dafür steht schon.
func withStableParameters(sets []TaxParameters) []TaxParameters {
	for i := range sets {
		// § 44 UStDV in seiner seit der Euro-Umstellung geltenden Fassung.
		sets[i].InputTaxCorrectionFloor = 100_000
		sets[i].InputTaxCorrectionMinorPoints = 10
		sets[i].InputTaxCorrectionMinorAmount = 100_000
		sets[i].InputTaxCorrectionAnnualAmount = 600_000
		// § 6 Abs. 1 Nr. 1a EStG, eingefügt durch das StÄndG 2003 und seither
		// unverändert.
		sets[i].NearAcquisitionPermille = 150
		sets[i].NearAcquisitionYears = 3
	}
	return sets
}

func init() { taxParameterSets = withStableParameters(taxParameterSets) }

// TaxParametersFor returns the values that applied on a date.
func TaxParametersFor(date string) (TaxParameters, error) {
	if date == "" {
		return TaxParameters{}, fmt.Errorf("ohne Datum lassen sich die steuerlichen Werte nicht bestimmen")
	}
	idx := sort.Search(len(taxParameterSets), func(i int) bool {
		return taxParameterSets[i].ValidFrom > date
	})
	if idx == 0 {
		return TaxParameters{}, fmt.Errorf("für den %s sind keine steuerlichen Werte hinterlegt", date)
	}
	return taxParameterSets[idx-1], nil
}

// EInvoiceTransition says whether a sonstige Rechnung — paper or a plain PDF —
// was still permitted for a transaction on a given date.
//
// § 27 Abs. 38 UStG knows three cases, and the middle one is the reason this
// returns a status rather than a boolean: from 2027 the transitional rule only
// covers issuers whose total turnover in the previous year did not exceed
// 800.000 €. That figure belongs to the supplier, and Buchfink cannot know it.
// Claiming certainty either way would be worse than naming the condition.
type EInvoiceTransition string

const (
	// EInvoiceTransitionAllowed: a sonstige Rechnung is still permitted
	// (§ 27 Abs. 38 Nr. 1 UStG, transactions before 01.01.2027).
	EInvoiceTransitionAllowed EInvoiceTransition = "allowed"
	// EInvoiceTransitionConditional: permitted only if the issuer's total
	// turnover in the previous year was at most 800.000 €
	// (§ 27 Abs. 38 Nr. 2 UStG, transactions in 2027).
	EInvoiceTransitionConditional EInvoiceTransition = "conditional"
	// EInvoiceTransitionExpired: no transitional rule applies any more.
	EInvoiceTransitionExpired EInvoiceTransition = "expired"
)

// EInvoiceTransitionFor classifies a document date against § 27 Abs. 38 UStG.
func EInvoiceTransitionFor(documentDate string) EInvoiceTransition {
	switch {
	case documentDate < "2027-01-01":
		return EInvoiceTransitionAllowed
	case documentDate < "2028-01-01":
		return EInvoiceTransitionConditional
	default:
		return EInvoiceTransitionExpired
	}
}

// EInvoiceTransitionRevenueLimit ist die Umsatzgrenze des § 27 Abs. 38 Nr. 2
// UStG: 800.000 € Gesamtumsatz im vorangegangenen Kalenderjahr. Bis dahin darf
// im Jahr 2027 noch eine sonstige Rechnung ausgestellt werden.
const EInvoiceTransitionRevenueLimit = domain.Cents(80_000_000)

// EInvoiceIssueTransitionFor beantwortet dieselbe Frage für die Ausstellerseite:
// darf *diese* Rechnung noch ohne strukturierten Datensatz hinausgehen?
//
// Der Unterschied zur Empfangsseite ist der Vorjahresumsatz. Bei einem
// eingehenden Beleg gehört er dem Lieferanten und ist unbekannt — deshalb
// liefert EInvoiceTransitionFor dort nur die Bedingung. Beim eigenen Umsatz
// kennt Buchfink die Zahl (Einstellung `prior_year_revenue` am Geschäftsjahr,
// aus der GuV des Vorjahres vorbelegt) und kann die Frage entscheiden statt sie
// weiterzureichen.
//
// Ein Vorjahresumsatz von null heißt „nicht erfasst" und nicht „kein Umsatz":
// die Übergangsregel gilt dann, weil ein Mandant ohne Vorjahr die Grenze nicht
// überschreiten kann und ein nicht gepflegtes Feld keine Rechnung blockieren
// soll, die das Gesetz erlaubt.
func EInvoiceIssueTransitionFor(documentDate string, priorYearRevenue domain.Cents) EInvoiceTransition {
	switch EInvoiceTransitionFor(documentDate) {
	case EInvoiceTransitionAllowed:
		return EInvoiceTransitionAllowed
	case EInvoiceTransitionConditional:
		if priorYearRevenue > EInvoiceTransitionRevenueLimit {
			return EInvoiceTransitionExpired
		}
		return EInvoiceTransitionAllowed
	default:
		return EInvoiceTransitionExpired
	}
}
