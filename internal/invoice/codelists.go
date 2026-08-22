// ACHTUNG: Dieser Prüfer ist abgelöst.
//
// Die EN-16931-Prüfung liegt in `internal/einvoice`. Sie läuft auf einem
// semantischen Modell statt auf CII-Structs, deckt alle 223 Geschäftsregeln ab
// statt 170, liest neben CII auch UBL, und XRechnung und ZUGFeRD sitzen als
// Schichten darüber.
//
// Was hier steht, hängt nur noch am Buchungspfad (`internal/service`), der
// weiterhin die CIIInvoice-Struktur verwendet. Das Umhängen ist der zweite
// Schritt und für sich zu machen — bis dahin gilt: **neue Regeln kommen ins
// Modul, nicht hierher.** Zwei Prüfer im Baum sind genau die Stelle, an der
// jemand den falschen bearbeitet.

package invoice

import "strings"

// The code lists EN 16931 references. They are embedded rather than fetched:
// a validation that needs the network is a validation that fails on a train, and
// these lists change rarely enough to be reviewed with a release.

// vatCategoryCodes is the subset of UNTDID 5305 that EN 16931 permits (BR-CL-17).
//
// The list is deliberately short. UNTDID 5305 knows more codes; the norm allows
// exactly these, and accepting others would mean accepting a document whose tax
// treatment Buchfink cannot map.
var vatCategoryCodes = map[string]string{
	"S":  "Regelsteuersatz",
	"Z":  "Nullsteuersatz",
	"E":  "steuerbefreit",
	"AE": "Steuerschuldnerschaft des Leistungsempfängers",
	"K":  "steuerfreie innergemeinschaftliche Lieferung",
	"G":  "steuerfreie Ausfuhrlieferung",
	"O":  "nicht steuerbar",
	"L":  "Kanarische Inseln (IGIC)",
	"M":  "Ceuta und Melilla (IPSI)",
}

func isVATCategoryCode(code string) bool {
	_, ok := vatCategoryCodes[strings.ToUpper(strings.TrimSpace(code))]
	return ok
}

// VATCategoryLabel names a category code for display.
func VATCategoryLabel(code string) string {
	if label, ok := vatCategoryCodes[strings.ToUpper(strings.TrimSpace(code))]; ok {
		return label
	}
	return code
}

// iso4217 lists the active currency codes (BR-CL-03).
var iso4217 = newCodeSet(`AED AFN ALL AMD ANG AOA ARS AUD AWG AZN BAM BBD BDT BGN BHD BIF BMD BND
BOB BOV BRL BSD BTN BWP BYN BZD CAD CDF CHE CHF CHW CLF CLP CNY COP COU CRC CUP CVE CZK
DJF DKK DOP DZD EGP ERN ETB EUR FJD FKP GBP GEL GHS GIP GMD GNF GTQ GYD HKD HNL HTG HUF
IDR ILS INR IQD IRR ISK JMD JOD JPY KES KGS KHR KMF KPW KRW KWD KYD KZT LAK LBP LKR LRD
LSL LYD MAD MDL MGA MKD MMK MNT MOP MRU MUR MVR MWK MXN MXV MYR MZN NAD NGN NIO NOK NPR
NZD OMR PAB PEN PGK PHP PKR PLN PYG QAR RON RSD RUB RWF SAR SBD SCR SDG SEK SGD SHP SLE
SOS SRD SSP STN SVC SYP SZL THB TJS TMT TND TOP TRY TTD TWD TZS UAH UGX USD USN UYI UYU
UYW UZS VED VES VND VUV WST XAF XCD XCG XDR XOF XPF XSU XUA YER ZAR ZMW ZWG`)

func isCurrencyCode(code string) bool {
	return iso4217[strings.ToUpper(strings.TrimSpace(code))]
}

// iso3166Alpha2 lists the country codes (BR-CL-10, BR-CL-11).
var iso3166Alpha2 = newCodeSet(`AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH
BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW
CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI
GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO
JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH
MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA
PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL
SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US
UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW`)

func isCountryCode(code string) bool {
	return iso3166Alpha2[strings.ToUpper(strings.TrimSpace(code))]
}

func newCodeSet(list string) map[string]bool {
	set := map[string]bool{}
	for _, code := range strings.Fields(list) {
		set[code] = true
	}
	return set
}
