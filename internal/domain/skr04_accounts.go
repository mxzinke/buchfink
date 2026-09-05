package domain

// Standard SKR04 accounts referenced by the booking logic.
//
// Every number here was checked against the DATEV SKR04 2026 template
// (Art.-Nr. 11175, assets/DATEV-SKR04-BilrUg-2026.pdf) and the extracted catalog
// in internal/accounting/skr04_2026.json. SKR03 numbers differ throughout —
// 1600 is Kasse here and Verbindlichkeiten aus LuL there — so these constants
// exist to keep the difference from leaking back into the code.
const (
	// Bestandskonten
	AccountForderungenLuL       = "1200" // Forderungen aus Lieferungen und Leistungen
	AccountKasse                = "1600" // Kasse
	AccountBank                 = "1800" // Bank
	AccountGeldtransit          = "1460" // Geldtransit
	AccountVerbindlichkeitenLuL = "3300" // Verbindlichkeiten aus Lieferungen und Leistungen
	AccountDurchlaufendePosten  = "1370" // Durchlaufende Posten

	// Rechnungsabgrenzung (§ 250 HGB)
	AccountAktiveRAP  = "1900" // Aktive Rechnungsabgrenzung
	AccountPassiveRAP = "3900" // Passive Rechnungsabgrenzung

	// Kapital (Gründung / Eröffnungsbilanz)
	AccountGezeichnetesKapital       = "2900" // Gezeichnetes Kapital
	AccountAusstehendeEinlagenOffen  = "2910" // Ausstehende Einlagen, nicht eingefordert
	AccountAusstehendeEinlagenGeford = "1298" // Ausstehende Einlagen, eingefordert (Forderung)
	AccountSaldenvortraegeSachkonten = "9000" // Saldenvorträge, Sachkonten
	AccountSaldenvortraegeDebitoren  = "9008" // Saldenvorträge, Debitoren
	AccountSaldenvortraegeKreditoren = "9009" // Saldenvorträge, Kreditoren

	// Ergebnisvortrag der Kapitalgesellschaft (§ 266 Abs. 3 A. IV HGB).
	//
	// Beide Konten bilden denselben Bilanzposten „Gewinn-/Verlustvortrag" ab und
	// unterscheiden sich nur in der Richtung; der SKR04-Katalog führt sie unter
	// derselben position_id bilanz.passiva_a_iv.gewinn_verlustvortrag. Der Name
	// des Kontos 2978 trägt im Katalog noch die Überschrift des folgenden
	// Abschnitts („Sonderposten mit Rücklageanteil") mit sich — gemeint ist der
	// Verlustvortrag vor Verwendung.
	//
	// „Vor Verwendung" ist wörtlich zu nehmen: hierher kommt das Jahresergebnis
	// des abgelaufenen Jahres beim Saldenvortrag. Was die Gesellschafter daraus
	// machen — Einstellung in Rücklagen oder Ausschüttung —, ist ein eigener
	// Beschluss und eine eigene Buchung.
	AccountGewinnvortrag  = "2970" // Gewinnvortrag vor Verwendung
	AccountVerlustvortrag = "2978" // Verlustvortrag vor Verwendung

	// Umsatzsteuer / Vorsteuer
	AccountVorsteuer         = "1400" // Abziehbare Vorsteuer
	AccountVorsteuer7        = "1401" // Abziehbare Vorsteuer 7 %
	AccountVorsteuerIG       = "1402" // Abziehbare Vorsteuer aus innergemeinschaftlichem Erwerb
	AccountVorsteuerIG19     = "1404" // Abziehbare Vorsteuer aus innergemeinschaftlichem Erwerb 19 %
	AccountVorsteuer19       = "1406" // Abziehbare Vorsteuer 19 %
	AccountVorsteuer13b19    = "1407" // Abziehbare Vorsteuer nach § 13b UStG 19 %
	AccountVorsteuer13b      = "1408" // Abziehbare Vorsteuer nach § 13b UStG
	AccountUmsatzsteuer      = "3800" // Umsatzsteuer
	AccountUmsatzsteuer7     = "3801" // Umsatzsteuer 7 %
	AccountUmsatzsteuerIG    = "3802" // Umsatzsteuer aus innergemeinschaftlichem Erwerb
	AccountUmsatzsteuerIG19  = "3804" // Umsatzsteuer aus innergemeinschaftlichem Erwerb 19 %
	AccountUmsatzsteuer19    = "3806" // Umsatzsteuer 19 %
	AccountUmsatzsteuer13b   = "3835" // Umsatzsteuer nach § 13b UStG
	AccountUmsatzsteuer13b19 = "3837" // Umsatzsteuer nach § 13b UStG 19 %

	// AccountUmsatzsteuer14c nimmt die Beträge auf, die nach § 14c UStG
	// geschuldet werden, obwohl kein steuerpflichtiger Umsatz dahintersteht.
	//
	// Der SKR04 führt das Konto ausdrücklich als „In Rechnung unrichtig oder
	// unberechtigt ausgewiesene Steuerbeträge, UStVA Kz. 69" — die Kennziffer
	// steht im Kontennamen, und genau dorthin geht der Betrag.
	AccountUmsatzsteuer14c = "3851"
	// AccountSondervorauszahlung ist die Sondervorauszahlung von 1/11 der
	// Vorauszahlungen des Vorjahres, die die Dauerfristverlängerung voraussetzt
	// (§ 47 Abs. 1 UStDV). Der SKR04 nennt das Konto „Umsatzsteuer-
	// Vorauszahlungen 1/11"; 3820 ist die laufende Vorauszahlung und nicht sie.
	AccountSondervorauszahlung = "3830"

	// Rückstellungen (§ 249 HGB) und ihre Gegenkonten.
	//
	// Die Nummern sind dem Katalog entnommen und nicht der Erinnerung: 3095 ist
	// im SKR04 die Rückstellung für Abschluss- und Prüfungskosten und 3074 die
	// für Personalkosten — genau umgekehrt zu dem, was der Schema-Vergleich mit
	// anderen Kontenrahmen nahelegt.
	AccountRueckstellungPensionen       = "3000" // Rückstellungen für Pensionen und ähnliche Verpflichtungen
	AccountSteuerrueckstellungen        = "3020" // Steuerrückstellungen (Sammelkonto)
	AccountRueckstellungGewerbesteuer   = "3035" // Gewerbesteuerrückstellung nach § 4 Abs. 5b EStG
	AccountRueckstellungKoerperschaft   = "3040" // Körperschaftsteuerrückstellung
	AccountRueckstellungSonstige        = "3070" // Sonstige Rückstellungen
	AccountRueckstellungPersonal        = "3074" // Rückstellungen für Personalkosten
	AccountRueckstellungInstandhaltung  = "3075" // Unterlassene Instandhaltung, Nachholung in den ersten drei Monaten
	AccountRueckstellungGewaehrleistung = "3090" // Rückstellungen für Gewährleistungen
	AccountRueckstellungDrohverlust     = "3092" // Rückstellungen für drohende Verluste aus schwebenden Geschäften
	AccountRueckstellungAbschluss       = "3095" // Rückstellungen für Abschluss- und Prüfungskosten
	AccountRueckstellungAufbewahrung    = "3096" // Rückstellungen zur Erfüllung der Aufbewahrungspflichten

	AccountAufwandGewaehrleistung = "6790" // Aufwand für Gewährleistung
	AccountAbschlusskosten        = "6827" // Abschluss- und Prüfungskosten
	AccountSonstigerAufwand       = "6300" // Sonstige betriebliche Aufwendungen
	AccountInstandhaltung         = "6490" // Sonstige Reparaturen und Instandhaltung
	AccountPersonalaufwandUrlaub  = "6076" // Aufwendungen aus der Veränderung von Urlaubsrückstellungen
	// AccountErtragAufloesungRueckstellungen nimmt die Auflösung auf: was
	// zurückgestellt und nicht gebraucht wurde, ist ein Ertrag des Jahres, in
	// dem der Grund entfällt.
	AccountErtragAufloesungRueckstellungen = "4930" // Erträge aus der Auflösung von Rückstellungen
	// AccountZinsaufwandAbzinsung ist der Gegenposten der Aufzinsung: der
	// Barwert wächst, weil die Fälligkeit näher rückt, und § 277 Abs. 5 Satz 1
	// HGB verlangt den Ausweis unter den Zinsen.
	AccountZinsaufwandAbzinsung = "7362" // Zinsaufwendungen aus der Abzinsung von Rückstellungen
	// AccountZinsaufwandLangfristig trägt die Auflösung eines Disagios: das
	// Damnum ist vorausbezahlter Zins (§ 250 Abs. 3 HGB).
	AccountZinsaufwandLangfristig = "7320" // Zinsaufwendungen für langfristige Verbindlichkeiten

	// Steuern vom Einkommen und Ertrag
	AccountKoerperschaftsteuer   = "7600" // Körperschaftsteuer
	AccountSolidaritaetszuschlag = "7608" // Solidaritätszuschlag
	AccountGewerbesteuer         = "7610" // Gewerbesteuer

	// Umsatzsteuer-Jahresverrechnung
	AccountUmsatzsteuerVorauszahlungen = "3820" // Umsatzsteuer-Vorauszahlungen
	AccountUmsatzsteuerVorjahr         = "3841" // Umsatzsteuerverbindlichkeiten Vorjahr
	AccountUmsatzsteuerforderung       = "1420" // Forderungen aus Umsatzsteuer-Vorauszahlungen

	// Bestandsveränderungen der Vorräte (§ 275 Abs. 2 Nr. 2 und Nr. 5 HGB)
	AccountBestandFertige        = "4800" // Bestandsveränderungen fertige Erzeugnisse
	AccountBestandUnfertige      = "4810" // Bestandsveränderungen unfertige Erzeugnisse
	AccountBestandUnfertigeLeist = "4815" // Bestandsveränderungen unfertige Leistungen
	AccountBestandRHBWaren       = "5880" // Bestandsveränderungen Roh-, Hilfs- und Betriebsstoffe sowie bezogene Waren

	// Ergebnisverwendung (§ 29 GmbHG, § 5a Abs. 3 GmbHG)
	AccountGesetzlicheRuecklage   = "2930" // Gesetzliche Rücklage
	AccountAndereGewinnruecklagen = "2960" // Andere Gewinnrücklagen
	AccountAusschuettung          = "3519" // Verbindlichkeiten gegenüber Gesellschaftern für offene Ausschüttungen
	AccountKapitalertragsteuer    = "3760" // Verbindlichkeiten aus Einbehaltungen (KapESt und SolZ) für offene Ausschüttungen

	// Zahlungsdifferenzen
	AccountErhalteneSkonti19 = "5736" // Erhaltene Skonti 19 % Vorsteuer
	AccountGewaehrteSkonti19 = "4736" // Gewährte Skonti 19 % USt
	AccountNebenkostenGeld   = "6855" // Nebenkosten des Geldverkehrs
)

// ResultCarryForwardAccount benennt das Eigenkapitalkonto, auf das das
// Jahresergebnis beim Saldenvortrag ins Folgejahr gebracht wird. Ein Gewinn
// steht im Haben auf 2970, ein Verlust im Soll auf 2978.
func ResultCarryForwardAccount(netIncome Cents) string {
	if netIncome < 0 {
		return AccountVerlustvortrag
	}
	return AccountGewinnvortrag
}

// IsCarryForwardAccount meldet, ob ein Konto eines der statistischen
// Vortragskonten ist.
//
// Sie sind das Gegenkonto des Saldenvortrags und werden deshalb selbst nie
// vorgetragen: täte man es, trüge das neue Jahr den Vortrag des alten ein
// zweites Mal.
func IsCarryForwardAccount(account string) bool {
	switch account {
	case AccountSaldenvortraegeSachkonten, AccountSaldenvortraegeDebitoren, AccountSaldenvortraegeKreditoren:
		return true
	}
	return false
}

// CollectiveAccounts are the balance sheet positions that Personenkonten roll
// up into. They must not be booked to directly.
//
// The open item of a business partner belongs on that partner's Personenkonto,
// never on 1200 or 3300. A booking straight onto the collective account would
// land in the balance sheet but in no OPOS list — two sources of truth for the
// same figure, and the difference only shows up when someone wonders why a
// customer's account does not add up to the receivables position.
func CollectiveAccounts() map[string]ContactType {
	return map[string]ContactType{
		AccountForderungenLuL:       ContactTypeCustomer,
		AccountVerbindlichkeitenLuL: ContactTypeVendor,
	}
}

// IsCollectiveAccount reports whether an account is a Sammelkonto for
// Personenkonten.
func IsCollectiveAccount(account string) bool {
	_, ok := CollectiveAccounts()[account]
	return ok
}

// LiquidAccounts lists the accounts treated as liquid funds for the cashflow
// view: Kasse, the bank accounts 1800-1850 provided by SKR04, and Geldtransit.
func LiquidAccounts() []string {
	return []string{
		AccountKasse, "1610", "1620",
		AccountBank, "1810", "1820", "1830", "1840", "1850",
		AccountGeldtransit,
	}
}
