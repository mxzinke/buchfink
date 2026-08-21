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

	// Zahlungsdifferenzen
	AccountErhalteneSkonti19 = "5736" // Erhaltene Skonti 19 % Vorsteuer
	AccountGewaehrteSkonti19 = "4736" // Gewährte Skonti 19 % USt
	AccountNebenkostenGeld   = "6855" // Nebenkosten des Geldverkehrs
)

// LiquidAccounts lists the accounts treated as liquid funds for the cashflow
// view: Kasse, the bank accounts 1800-1850 provided by SKR04, and Geldtransit.
func LiquidAccounts() []string {
	return []string{
		AccountKasse, "1610", "1620",
		AccountBank, "1810", "1820", "1830", "1840", "1850",
		AccountGeldtransit,
	}
}
