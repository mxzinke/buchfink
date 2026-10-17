package service

import "github.com/buchfink/buchfink/internal/domain"

// obligationCatalog ist der Katalog der Pflichtarten. Fristen und Fundstellen
// sind am Wortlaut geprüft (Stand Oktober 2026); die Herleitung je Art steht
// in obligation_rules.go.
var obligationCatalog = []domain.ObligationType{
	{
		Kind: domain.ObligationVatReturn, Title: "Umsatzsteuer-Voranmeldung",
		Basis:   "§ 18 Abs. 1 und 2 UStG, § 46 UStDV",
		Rhythm:  "monatlich oder vierteljährlich",
		Channel: domain.ChannelElster, Responsibility: domain.ResponsibilityAgentPrepares,
		Evidence: domain.EvidenceFromData, ShiftsToWorkday: true,
		Description: "Übermittlung und Zahlung bis zum 10. Tag nach Ablauf des Voranmeldungszeitraums, " +
			"mit Dauerfristverlängerung einen Monat später. Erledigt mit dem Transferticket an der Voranmeldung.",
	},
	{
		Kind: domain.ObligationSpecialPrepayment, Title: "Sondervorauszahlung",
		Basis:   "§ 47 Abs. 1, § 48 Abs. 2 UStDV",
		Rhythm:  "jährlich",
		Channel: domain.ChannelElster, Responsibility: domain.ResponsibilityAgentPrepares,
		Evidence: domain.EvidenceFromEither, ShiftsToWorkday: true,
		Description: "Bei monatlicher Voranmeldung mit Dauerfristverlängerung: ein Elftel der Vorauszahlungen des " +
			"Vorjahres, zu berechnen, anzumelden und zu entrichten bis zum gesetzlichen Abgabetag der ersten " +
			"Voranmeldung, dem 10. Februar. Erledigt mit der Buchung auf 3830 oder der Bestätigung mit Transferticket.",
	},
	{
		Kind: domain.ObligationECSalesList, Title: "Zusammenfassende Meldung",
		Basis:   "§ 18a Abs. 2 UStG",
		Rhythm:  "vierteljährlich, soweit sonstige Leistungen ins übrige Gemeinschaftsgebiet ausgeführt wurden",
		Channel: domain.ChannelBZSt, Responsibility: domain.ResponsibilityAgentPrepares,
		Evidence: domain.EvidenceFromData, ShiftsToWorkday: true,
		Description: "Übermittlung an das Bundeszentralamt für Steuern bis zum 25. Tag nach Ablauf des " +
			"Kalendervierteljahres. Die Dauerfristverlängerung gilt für sie nicht. Erledigt mit dem Transferticket.",
	},
	{
		Kind: domain.ObligationCorporatePrepay, Title: "Körperschaftsteuer-Vorauszahlung",
		Basis:   "§ 37 Abs. 1 EStG i. V. m. § 31 Abs. 1 und 2 KStG",
		Rhythm:  "10. März, 10. Juni, 10. September, 10. Dezember",
		Channel: domain.ChannelInternal, Responsibility: domain.ResponsibilityHumanConfirms,
		Evidence: domain.EvidenceFromEither,
		Description: "Zahlung an das Finanzamt in der Höhe des Vorauszahlungsbescheids, mit Solidaritätszuschlag. " +
			"Erledigt mit der Buchung auf 7600 oder 7608.",
	},
	{
		Kind: domain.ObligationTradePrepay, Title: "Gewerbesteuer-Vorauszahlung",
		Basis:   "§ 19 Abs. 1 und 2 GewStG",
		Rhythm:  "15. Februar, 15. Mai, 15. August, 15. November",
		Channel: domain.ChannelInternal, Responsibility: domain.ResponsibilityHumanConfirms,
		Evidence:    domain.EvidenceFromEither,
		Description: "Zahlung an die Gemeinde in der Höhe des Vorauszahlungsbescheids. Erledigt mit der Buchung auf 7610.",
	},
	{
		Kind: domain.ObligationPayrollTax, Title: "Lohnsteuer-Anmeldung",
		Basis:   "§ 41a Abs. 1 und 2 EStG",
		Rhythm:  "monatlich; vierteljährlich oder jährlich bei geringer Lohnsteuer des Vorjahres",
		Channel: domain.ChannelPayrollProvider, Responsibility: domain.ResponsibilityHumanConfirms,
		Evidence: domain.EvidenceFromEither, ShiftsToWorkday: true,
		Description: "Anmeldung und Abführung bis zum 10. Tag nach Ablauf des Anmeldungszeitraums, meist durch den " +
			"Lohnanbieter. Erledigt, sobald das Lohnjournal des Zeitraums gebucht ist.",
	},
	{
		Kind: domain.ObligationMonthCommit, Title: "Monatsfestschreibung",
		Basis:   "§ 146 Abs. 4 AO, GoBD Rz. 107",
		Rhythm:  "monatlich",
		Channel: domain.ChannelInternal, Responsibility: domain.ResponsibilityHumanConfirms,
		Evidence: domain.EvidenceFromData,
		Description: "Bis zum Ende des Folgemonats, mit Dauerfristverlängerung einen Monat später, zuzüglich der " +
			"Nachfrist aus den Einstellungen. Nach der Festschreibung nimmt der Monat keine Buchung mehr auf.",
	},
	{
		Kind: domain.ObligationClosingBookings, Title: "Abschlussbuchungen zum Bilanzstichtag",
		Basis:   "§ 242 Abs. 1, § 264 Abs. 1 HGB",
		Rhythm:  "jährlich",
		Channel: domain.ChannelInternal, Responsibility: domain.ResponsibilityAgentPrepares,
		Evidence: domain.EvidenceFromData,
		Description: "Abschreibungen, Abgrenzungen, Rückstellungen und Umsatzsteuer-Verrechnung zum Stichtag, " +
			"fällig mit der Aufstellung. Erledigt mit der Festschreibung des Geschäftsjahres.",
	},
	{
		Kind: domain.ObligationStatementPrep, Title: "Aufstellung des Jahresabschlusses",
		Basis:   "§ 264 Abs. 1 Satz 3 und 4 HGB",
		Rhythm:  "jährlich",
		Channel: domain.ChannelInternal, Responsibility: domain.ResponsibilityHumanConfirms,
		Evidence: domain.EvidenceFromData,
		Description: "Innerhalb der ersten drei Monate des folgenden Geschäftsjahres; kleine und " +
			"Kleinstkapitalgesellschaften innerhalb der ersten sechs Monate, wenn das einem ordnungsgemäßen " +
			"Geschäftsgang entspricht. Erledigt mit dem Stand „aufgestellt“.",
	},
	{
		Kind: domain.ObligationAdoption, Title: "Feststellung des Jahresabschlusses",
		Basis:   "§ 42a Abs. 2 GmbHG",
		Rhythm:  "jährlich",
		Channel: domain.ChannelInternal, Responsibility: domain.ResponsibilityHumanConfirms,
		Evidence: domain.EvidenceFromData,
		Description: "Beschluss der Gesellschafter bis zum Ablauf der ersten acht Monate des folgenden " +
			"Geschäftsjahres, bei kleinen Gesellschaften der ersten elf Monate; der Gesellschaftsvertrag kann die " +
			"Frist nicht verlängern. Erledigt mit dem Stand „festgestellt“.",
	},
	{
		Kind: domain.ObligationAppropriation, Title: "Beschluss über die Ergebnisverwendung",
		Basis:   "§ 42a Abs. 2, § 46 Nr. 1 GmbHG",
		Rhythm:  "jährlich",
		Channel: domain.ChannelInternal, Responsibility: domain.ResponsibilityHumanConfirms,
		Evidence:    domain.EvidenceFromData,
		Description: "In derselben Frist wie die Feststellung. Erledigt, sobald der Beschluss in Buchfink erfasst ist.",
	},
	{
		Kind: domain.ObligationEBilanz, Title: "E-Bilanz",
		Basis:   "§ 5b Abs. 1 EStG, § 60 Abs. 1 EStDV",
		Rhythm:  "jährlich, mit der Körperschaftsteuererklärung",
		Channel: domain.ChannelElster, Responsibility: domain.ResponsibilityAgentPrepares,
		Evidence: domain.EvidenceFromHuman, ShiftsToWorkday: true,
		Description: "Bilanz und Gewinn- und Verlustrechnung mit Kontennachweisen nach amtlichem Datensatz; " +
			"fällig mit der Körperschaftsteuererklärung. Buchfink exportiert die XBRL-Datei, die Übermittlung " +
			"braucht ein Programm mit ERiC.",
	},
	{
		Kind: domain.ObligationCorporateTaxReturn, Title: "Körperschaftsteuererklärung",
		Basis:   "§ 31 Abs. 1a KStG, § 149 Abs. 2 und 3 AO",
		Rhythm:  "jährlich",
		Channel: domain.ChannelElster, Responsibility: domain.ResponsibilityAgentPrepares,
		Evidence: domain.EvidenceFromEither, ShiftsToWorkday: true,
		Description: "Mit der Erklärung zum steuerlichen Einlagekonto. Sieben Monate nach Ablauf des " +
			"Kalenderjahres; erstellt ein Steuerberater sie, bis zum letzten Tag des Februar des zweiten Folgejahres.",
	},
	{
		Kind: domain.ObligationTradeTaxReturn, Title: "Gewerbesteuererklärung",
		Basis:   "§ 14a GewStG, § 149 Abs. 2 und 3 AO",
		Rhythm:  "jährlich",
		Channel: domain.ChannelElster, Responsibility: domain.ResponsibilityAgentPrepares,
		Evidence: domain.EvidenceFromEither, ShiftsToWorkday: true,
		Description: "Erklärung zur Festsetzung des Steuermessbetrags. Fristen wie bei der Körperschaftsteuer.",
	},
	{
		Kind: domain.ObligationAnnualVatReturn, Title: "Umsatzsteuer-Jahreserklärung",
		Basis:   "§ 18 Abs. 3 UStG, § 149 Abs. 2 und 3 AO",
		Rhythm:  "jährlich",
		Channel: domain.ChannelElster, Responsibility: domain.ResponsibilityAgentPrepares,
		Evidence: domain.EvidenceFromEither, ShiftsToWorkday: true,
		Description: "Steueranmeldung für das Kalenderjahr. Fristen wie bei der Körperschaftsteuer.",
	},
	{
		Kind: domain.ObligationDisclosure, Title: "Offenlegung im Unternehmensregister",
		Basis:   "§ 325 Abs. 1 und 1a, § 326 HGB",
		Rhythm:  "jährlich",
		Channel: domain.ChannelCompanyRegister, Responsibility: domain.ResponsibilityHumanConfirms,
		Evidence: domain.EvidenceFromData,
		Description: "Übermittlung des festgestellten Abschlusses spätestens ein Jahr nach dem Abschlussstichtag. " +
			"Kleine Gesellschaften legen Bilanz und Anhang offen, Kleinstgesellschaften nur die Bilanz und können " +
			"sie hinterlegen. Erledigt mit dem Stand „offengelegt“.",
	},
	{
		Kind: domain.ObligationTransparency, Title: "Mitteilung an das Transparenzregister",
		Basis:   "§ 20 Abs. 1 GwG",
		Rhythm:  "nach der Eintragung und bei jeder Änderung der wirtschaftlich Berechtigten",
		Channel: domain.ChannelTransparencyRegister, Responsibility: domain.ResponsibilityHumanConfirms,
		Evidence:    domain.EvidenceFromHuman,
		Description: "Die Angaben zu den wirtschaftlich Berechtigten sind unverzüglich mitzuteilen und aktuell zu halten.",
	},
	{
		Kind: domain.ObligationShareholderList, Title: "Neue Gesellschafterliste",
		Basis:   "§ 40 Abs. 1 und 2 GmbHG",
		Rhythm:  "bei jeder Veränderung der Gesellschafter oder ihrer Beteiligung",
		Channel: domain.ChannelNotary, Responsibility: domain.ResponsibilityHumanConfirms,
		Evidence: domain.EvidenceFromHuman,
		Description: "Unverzüglich nach Wirksamwerden der Veränderung zum Handelsregister einzureichen; hat ein " +
			"Notar mitgewirkt, reicht er die Liste ein.",
	},
	{
		Kind: domain.ObligationTaxRegistration, Title: "Fragebogen zur steuerlichen Erfassung",
		Basis:   "§ 137, § 138 Abs. 1b und 4 AO",
		Rhythm:  "einmalig nach der Gründung",
		Channel: domain.ChannelElster, Responsibility: domain.ResponsibilityAgentPrepares,
		Evidence: domain.EvidenceFromHuman, ShiftsToWorkday: true,
		Description: "Anzeige der Gründung und Auskünfte für die Besteuerung nach amtlichem Datensatz binnen eines " +
			"Monats nach der Gründung.",
	},
	{
		Kind: domain.ObligationTradeRegistration, Title: "Gewerbeanzeige",
		Basis:   "§ 14 Abs. 1 GewO",
		Rhythm:  "einmalig zum Beginn des Betriebs",
		Channel: domain.ChannelTradeOffice, Responsibility: domain.ResponsibilityHumanConfirms,
		Evidence:    domain.EvidenceFromHuman,
		Description: "Anzeige an das Gewerbeamt gleichzeitig mit dem Beginn des Betriebs.",
	},
	{
		Kind: domain.ObligationWithholdingTax, Title: "Kapitalertragsteuer auf die Ausschüttung",
		Basis:   "§ 43 Abs. 1 Satz 1 Nr. 1, § 44 Abs. 1 Satz 5, Abs. 2, § 45a Abs. 1 EStG",
		Rhythm:  "bei jeder Ausschüttung",
		Channel: domain.ChannelElster, Responsibility: domain.ResponsibilityAgentPrepares,
		Evidence:    domain.EvidenceFromHuman,
		Description: "Einbehalt, Anmeldung und Abführung im Zeitpunkt des Zuflusses an die Gesellschafter.",
	},
}

// ObligationTypes liefert den Katalog der Pflichtarten mit Skill-Verweis.
func ObligationTypes() []domain.ObligationType {
	out := make([]domain.ObligationType, len(obligationCatalog))
	for i, t := range obligationCatalog {
		t.SkillRef = t.Kind.SkillRef()
		out[i] = t
	}
	return out
}

// obligationType liefert den Katalogeintrag einer Pflichtart.
func obligationType(kind domain.ObligationKind) (domain.ObligationType, bool) {
	for _, t := range obligationCatalog {
		if t.Kind == kind {
			t.SkillRef = t.Kind.SkillRef()
			return t, true
		}
	}
	return domain.ObligationType{}, false
}
