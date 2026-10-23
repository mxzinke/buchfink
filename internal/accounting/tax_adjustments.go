package accounting

import "github.com/buchfink/buchfink/internal/domain"

// Die Konten, auf denen das handelsrechtliche Ergebnis und die steuerliche
// Bemessungsgrundlage auseinandergehen, soweit der SKR04 sie eigens führt.
// Sie tragen die Steuerrückstellung (CalculateTaxProvision) und die Anlage GK
// der Körperschaftsteuererklärung; was auf anderen Konten steht, kommt als
// Eingabe hinzu.

// TaxAdjustmentKind ist die Art der steuerlichen Korrektur eines Kontos.
type TaxAdjustmentKind string

const (
	// TaxAdjustmentNonDeductible: der Aufwand mindert das Einkommen nicht und
	// wird hinzugerechnet.
	TaxAdjustmentNonDeductible TaxAdjustmentKind = "nicht_abziehbar"
	// TaxAdjustmentExemptDividend: Bezüge nach § 8b Abs. 1 KStG, die außer
	// Ansatz bleiben, wenn die Beteiligung zu Beginn des Kalenderjahres
	// mindestens 10 % betrug (Abs. 4).
	TaxAdjustmentExemptDividend TaxAdjustmentKind = "steuerfreie_bezuege"
	// TaxAdjustmentPortfolioDividend: Bezüge auf Konten, die der SKR04 auch für
	// Streubesitz unter 10 % führt (§ 8b Abs. 4 KStG). Ohne Angabe gelten sie
	// als steuerpflichtig.
	TaxAdjustmentPortfolioDividend TaxAdjustmentKind = "bezuege_streubesitz"
	// TaxAdjustmentExemptGain: Gewinne aus der Veräußerung von Anteilen an
	// Kapitalgesellschaften (§ 8b Abs. 2 KStG); die Konten tragen Erlös,
	// Buchwert und Veräußerungskosten.
	TaxAdjustmentExemptGain TaxAdjustmentKind = "steuerfreie_gewinne"
	// TaxAdjustmentDonation: Zuwendungen, die im Rahmen der Höchstbeträge des
	// § 9 Abs. 1 Satz 1 Nr. 2 KStG abziehbar sind.
	TaxAdjustmentDonation TaxAdjustmentKind = "zuwendungen"
)

// TaxAdjustmentGroup ordnet ein nicht abziehbares Konto der Art seiner
// Hinzurechnung zu. Die Steuerrückstellung rechnet alle Arten gleich; die
// Körperschaftsteuererklärung führt jede Art in ihrer Zeile.
type TaxAdjustmentGroup string

const (
	// GroupGifts: Geschenke über der Freigrenze (§ 4 Abs. 5 Satz 1 Nr. 1 EStG).
	GroupGifts TaxAdjustmentGroup = "geschenke"
	// GroupEntertainment: der nicht abziehbare Teil der Bewirtung (Nr. 2).
	GroupEntertainment TaxAdjustmentGroup = "bewirtung"
	// GroupHospitality: Gästehäuser, Jagd, Jachten (Nr. 3 und 4).
	GroupHospitality TaxAdjustmentGroup = "repraesentation"
	// GroupRestricted: die übrigen eingeschränkt abziehbaren Betriebsausgaben
	// des § 4 Abs. 5 EStG.
	GroupRestricted TaxAdjustmentGroup = "eingeschraenkt"
	// GroupFinesAndOther: Geldbußen und Geldstrafen (§ 4 Abs. 5 Satz 1 Nr. 8
	// EStG, § 10 Nr. 3 KStG) und weitere nicht abziehbare Aufwendungen
	// außerhalb des § 4 Abs. 5 EStG.
	GroupFinesAndOther TaxAdjustmentGroup = "weitere"
	// GroupTaxSidelines: Nebenleistungen zu den Steuern vom Einkommen und zur
	// Gewerbesteuer (§ 10 Nr. 2 KStG, § 4 Abs. 5b EStG).
	GroupTaxSidelines TaxAdjustmentGroup = "nebenleistungen"
	// GroupSupervisoryBoard: die Hälfte der Aufsichtsratsvergütungen (§ 10
	// Nr. 4 KStG).
	GroupSupervisoryBoard TaxAdjustmentGroup = "aufsichtsrat"
	// GroupDonations: Zuwendungen; die abziehbaren gehen in den Höchstbetrag.
	GroupDonations TaxAdjustmentGroup = "zuwendungen"
	// GroupShareLosses: Gewinnminderungen aus Anteilen (§ 8b Abs. 3 Satz 3
	// KStG).
	GroupShareLosses TaxAdjustmentGroup = "anteile"
	// GroupShareLoans: Gewinnminderungen aus Darlehen an eine Gesellschaft mit
	// Beteiligung von mehr als einem Viertel (§ 8b Abs. 3 Satz 4 KStG).
	GroupShareLoans TaxAdjustmentGroup = "darlehen"
	// GroupDividends: Bezüge nach § 8b Abs. 1 KStG.
	GroupDividends TaxAdjustmentGroup = "bezuege"
	// GroupShareGains: Gewinne nach § 8b Abs. 2 KStG.
	GroupShareGains TaxAdjustmentGroup = "gewinne"
)

// TaxAdjustmentAccount ist ein Konto mit seiner steuerlichen Wirkung.
type TaxAdjustmentAccount struct {
	Account   string             `json:"account"`
	Kind      TaxAdjustmentKind  `json:"kind"`
	Group     TaxAdjustmentGroup `json:"group"`
	Reference string             `json:"reference"`
}

const (
	refGifts         = "§ 4 Abs. 5 Satz 1 Nr. 1 EStG"
	refEntertainment = "§ 4 Abs. 5 Satz 1 Nr. 2 EStG"
	refSidelines     = "§ 10 Nr. 2 KStG"
	refShareLosses   = "§ 8b Abs. 3 Satz 3 KStG"
)

// taxAdjustmentAccounts ist der Katalog. Die Nummern stehen im
// SKR04-Katalog; TestTaxAdjustmentAccountsExist hält das fest.
var taxAdjustmentAccounts = []TaxAdjustmentAccount{
	{"6620", TaxAdjustmentNonDeductible, GroupGifts, refGifts},
	{"6621", TaxAdjustmentNonDeductible, GroupGifts, refGifts},
	// Die Pauschsteuer nach § 37b EStG teilt das Schicksal des Geschenks.
	{"6622", TaxAdjustmentNonDeductible, GroupGifts, refGifts},
	{"6644", TaxAdjustmentNonDeductible, GroupEntertainment, refEntertainment},
	{"6645", TaxAdjustmentNonDeductible, GroupHospitality, "§ 4 Abs. 5 Satz 1 Nr. 3 und 4 EStG"},
	{"6642", TaxAdjustmentNonDeductible, GroupRestricted, "§ 4 Abs. 5 Satz 1 EStG"},
	{"6349", TaxAdjustmentNonDeductible, GroupRestricted, "§ 4 Abs. 5 Satz 1 Nr. 6b EStG"},
	{"6672", TaxAdjustmentNonDeductible, GroupRestricted, "§ 4 Abs. 5 Satz 1 Nr. 5 EStG"},
	{"6689", TaxAdjustmentNonDeductible, GroupRestricted, "§ 4 Abs. 5 Satz 1 Nr. 6 EStG"},
	// Geldbußen, Ordnungs- und Verwarnungsgelder, Geldstrafen.
	{"6968", TaxAdjustmentNonDeductible, GroupFinesAndOther, "§ 4 Abs. 5 Satz 1 Nr. 8 EStG, § 10 Nr. 3 KStG"},
	{"6875", TaxAdjustmentNonDeductible, GroupSupervisoryBoard, "§ 10 Nr. 4 KStG"},
	{"6390", TaxAdjustmentNonDeductible, GroupDonations, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6394", TaxAdjustmentNonDeductible, GroupDonations, "§ 4 Abs. 6 EStG"},
	// Nebenleistungen zu den Steuern vom Einkommen und zur Gewerbesteuer.
	{"6437", TaxAdjustmentNonDeductible, GroupTaxSidelines, refSidelines},
	{"7302", TaxAdjustmentNonDeductible, GroupTaxSidelines, "§ 4 Abs. 5b EStG"},
	{"7304", TaxAdjustmentNonDeductible, GroupTaxSidelines, refSidelines},
	{"7306", TaxAdjustmentNonDeductible, GroupTaxSidelines, refSidelines},
	{"7308", TaxAdjustmentNonDeductible, GroupTaxSidelines, refSidelines},
	{"7313", TaxAdjustmentNonDeductible, GroupFinesAndOther, "§ 4 Abs. 4a EStG"},
	{"7366", TaxAdjustmentNonDeductible, GroupFinesAndOther, "§ 6 Abs. 1 Nr. 3a Buchst. e EStG"},
	// Gewinnminderungen aus Anteilen an Kapitalgesellschaften: Verluste aus
	// der Veräußerung, Teilwertabschreibungen, Abschreibungen auf Darlehen
	// eines wesentlich beteiligten Gesellschafters (§ 8b Abs. 3 Satz 4 KStG).
	{"6290", TaxAdjustmentNonDeductible, GroupShareLoans, "§ 8b Abs. 3 Satz 4 KStG"},
	{"6858", TaxAdjustmentNonDeductible, GroupShareLosses, refShareLosses},
	{"6892", TaxAdjustmentNonDeductible, GroupShareLosses, refShareLosses},
	{"6898", TaxAdjustmentNonDeductible, GroupShareLosses, refShareLosses},
	{"6904", TaxAdjustmentNonDeductible, GroupShareLosses, refShareLosses},
	{"6906", TaxAdjustmentNonDeductible, GroupShareLosses, refShareLosses},
	{"6908", TaxAdjustmentNonDeductible, GroupShareLosses, refShareLosses},
	{"7204", TaxAdjustmentNonDeductible, GroupShareLosses, refShareLosses},
	{"7214", TaxAdjustmentNonDeductible, GroupShareLosses, refShareLosses},
	{"7255", TaxAdjustmentNonDeductible, GroupShareLosses, refShareLosses},

	{"7005", TaxAdjustmentExemptDividend, GroupDividends, "§ 8b Abs. 1 KStG"},
	{"7006", TaxAdjustmentExemptDividend, GroupDividends, "§ 8b Abs. 1 KStG"},
	{"7015", TaxAdjustmentExemptDividend, GroupDividends, "§ 8b Abs. 1 KStG"},
	{"7104", TaxAdjustmentExemptDividend, GroupDividends, "§ 8b Abs. 1 KStG"},
	{"7014", TaxAdjustmentPortfolioDividend, GroupDividends, "§ 8b Abs. 1 und 4 KStG"},
	{"7103", TaxAdjustmentPortfolioDividend, GroupDividends, "§ 8b Abs. 1 und 4 KStG"},

	{"4852", TaxAdjustmentExemptGain, GroupShareGains, "§ 8b Abs. 2 KStG"},
	{"4858", TaxAdjustmentExemptGain, GroupShareGains, "§ 8b Abs. 2 KStG"},
	{"4904", TaxAdjustmentExemptGain, GroupShareGains, "§ 8b Abs. 2 KStG"},
	{"4906", TaxAdjustmentExemptGain, GroupShareGains, "§ 8b Abs. 2 KStG"},
	{"4914", TaxAdjustmentExemptGain, GroupShareGains, "§ 8b Abs. 2 KStG"},
	{"6857", TaxAdjustmentExemptGain, GroupShareGains, "§ 8b Abs. 2 Satz 2 KStG"},

	{"6391", TaxAdjustmentDonation, GroupDonations, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6392", TaxAdjustmentDonation, GroupDonations, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6393", TaxAdjustmentDonation, GroupDonations, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6395", TaxAdjustmentDonation, GroupDonations, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6397", TaxAdjustmentDonation, GroupDonations, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6398", TaxAdjustmentDonation, GroupDonations, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
}

// TaxAdjustmentAccountsOf liefert die Konten einer Art und, wenn angegeben,
// einer Gruppe.
func TaxAdjustmentAccountsOf(kind TaxAdjustmentKind, groups ...TaxAdjustmentGroup) []string {
	out := []string{}
	for _, a := range taxAdjustmentAccounts {
		if a.Kind != kind {
			continue
		}
		if len(groups) == 0 {
			out = append(out, a.Account)
			continue
		}
		for _, g := range groups {
			if a.Group == g {
				out = append(out, a.Account)
				break
			}
		}
	}
	return out
}

// TaxAdjustmentFor nennt die steuerliche Wirkung eines Kontos.
func TaxAdjustmentFor(account string) (TaxAdjustmentAccount, bool) {
	for _, a := range taxAdjustmentAccounts {
		if a.Account == account {
			return a, true
		}
	}
	return TaxAdjustmentAccount{}, false
}

// TaxAdjustmentAccounts liefert den Katalog als Kopie.
func TaxAdjustmentAccounts() []TaxAdjustmentAccount {
	out := make([]TaxAdjustmentAccount, len(taxAdjustmentAccounts))
	copy(out, taxAdjustmentAccounts)
	return out
}

// Die Konten der Steuern vom Einkommen und Ertrag, die die Abrechnung der
// Steuerrückstellung braucht.
const (
	// AccountKapitalertragsteuer25 trägt die einbehaltene Kapitalertragsteuer,
	// AccountSoliKapitalertragsteuer den Solidaritätszuschlag darauf. Beide
	// werden auf die Körperschaftsteuer und den Solidaritätszuschlag
	// angerechnet (§ 36 Abs. 2 Nr. 2 EStG über § 31 Abs. 1 KStG, § 1 Abs. 2
	// SolZG).
	AccountKapitalertragsteuer25        = "7630"
	AccountSoliKapitalertragsteuer      = "7633"
	AccountKoerperschaftsteuerForderung = "1450"
	AccountGewerbesteuerForderung       = "1435"
)

// IsIncomeTaxAccount meldet, ob ein Konto die Steuern vom Einkommen und Ertrag
// trägt (§ 275 Abs. 2 Nr. 14 HGB). Der SKR04 führt sie im Bereich 7600 bis
// 7649; sie mindern ihre eigene Bemessungsgrundlage nicht (§ 10 Nr. 2 KStG,
// § 4 Abs. 5b EStG), und ihre Erstattungen erhöhen sie nicht.
func IsIncomeTaxAccount(account string) bool {
	return len(account) == 4 && account >= "7600" && account <= "7649"
}

// IsWageAccount meldet, ob ein Konto unter den Löhnen und Gehältern steht
// (§ 275 Abs. 2 Nr. 6 Buchst. a HGB).
func IsWageAccount(acc domain.Account) bool {
	key, ok := StatementKeyForAccount(acc)
	return ok && key == "guv.6.a"
}

// IsRevenueAccount meldet, ob ein Konto unter den Umsatzerlösen steht
// (§ 275 Abs. 2 Nr. 1 HGB).
func IsRevenueAccount(acc domain.Account) bool {
	key, ok := StatementKeyForAccount(acc)
	return ok && key == "guv.1"
}
