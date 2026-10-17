package accounting

import "github.com/buchfink/buchfink/internal/domain"

// Die Konten, auf denen das handelsrechtliche Ergebnis und die steuerliche
// Bemessungsgrundlage auseinandergehen, soweit der SKR04 sie eigens führt.
// Sie tragen die Schätzung der Steuerrückstellung (CalculateTaxProvision);
// was auf anderen Konten steht, kommt als Eingabe hinzu.

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

// TaxAdjustmentAccount ist ein Konto mit seiner steuerlichen Wirkung.
type TaxAdjustmentAccount struct {
	Account   string            `json:"account"`
	Kind      TaxAdjustmentKind `json:"kind"`
	Reference string            `json:"reference"`
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
	{"6620", TaxAdjustmentNonDeductible, refGifts},
	{"6621", TaxAdjustmentNonDeductible, refGifts},
	// Die Pauschsteuer nach § 37b EStG teilt das Schicksal des Geschenks.
	{"6622", TaxAdjustmentNonDeductible, refGifts},
	{"6644", TaxAdjustmentNonDeductible, refEntertainment},
	{"6645", TaxAdjustmentNonDeductible, "§ 4 Abs. 5 Satz 1 Nr. 3 und 4 EStG"},
	{"6642", TaxAdjustmentNonDeductible, "§ 4 Abs. 5 Satz 1 EStG"},
	{"6349", TaxAdjustmentNonDeductible, "§ 4 Abs. 5 Satz 1 Nr. 6b EStG"},
	{"6672", TaxAdjustmentNonDeductible, "§ 4 Abs. 5 Satz 1 Nr. 5 EStG"},
	{"6689", TaxAdjustmentNonDeductible, "§ 4 Abs. 5 Satz 1 Nr. 6 EStG"},
	// Geldbußen, Ordnungs- und Verwarnungsgelder, Geldstrafen.
	{"6968", TaxAdjustmentNonDeductible, "§ 4 Abs. 5 Satz 1 Nr. 8 EStG, § 10 Nr. 3 KStG"},
	{"6875", TaxAdjustmentNonDeductible, "§ 10 Nr. 4 KStG"},
	{"6390", TaxAdjustmentNonDeductible, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6394", TaxAdjustmentNonDeductible, "§ 4 Abs. 6 EStG"},
	// Nebenleistungen zu den Steuern vom Einkommen und zur Gewerbesteuer.
	{"6437", TaxAdjustmentNonDeductible, refSidelines},
	{"7302", TaxAdjustmentNonDeductible, "§ 4 Abs. 5b EStG"},
	{"7304", TaxAdjustmentNonDeductible, refSidelines},
	{"7306", TaxAdjustmentNonDeductible, refSidelines},
	{"7308", TaxAdjustmentNonDeductible, refSidelines},
	{"7313", TaxAdjustmentNonDeductible, "§ 4 Abs. 4a EStG"},
	{"7366", TaxAdjustmentNonDeductible, "§ 6 Abs. 1 Nr. 3a Buchst. e EStG"},
	// Gewinnminderungen aus Anteilen an Kapitalgesellschaften: Verluste aus
	// der Veräußerung, Teilwertabschreibungen, Abschreibungen auf Darlehen
	// eines wesentlich beteiligten Gesellschafters (§ 8b Abs. 3 Satz 4 KStG).
	{"6290", TaxAdjustmentNonDeductible, "§ 8b Abs. 3 Satz 4 KStG"},
	{"6858", TaxAdjustmentNonDeductible, refShareLosses},
	{"6892", TaxAdjustmentNonDeductible, refShareLosses},
	{"6898", TaxAdjustmentNonDeductible, refShareLosses},
	{"6904", TaxAdjustmentNonDeductible, refShareLosses},
	{"6906", TaxAdjustmentNonDeductible, refShareLosses},
	{"6908", TaxAdjustmentNonDeductible, refShareLosses},
	{"7204", TaxAdjustmentNonDeductible, refShareLosses},
	{"7214", TaxAdjustmentNonDeductible, refShareLosses},
	{"7255", TaxAdjustmentNonDeductible, refShareLosses},

	{"7005", TaxAdjustmentExemptDividend, "§ 8b Abs. 1 KStG"},
	{"7006", TaxAdjustmentExemptDividend, "§ 8b Abs. 1 KStG"},
	{"7015", TaxAdjustmentExemptDividend, "§ 8b Abs. 1 KStG"},
	{"7104", TaxAdjustmentExemptDividend, "§ 8b Abs. 1 KStG"},
	{"7014", TaxAdjustmentPortfolioDividend, "§ 8b Abs. 1 und 4 KStG"},
	{"7103", TaxAdjustmentPortfolioDividend, "§ 8b Abs. 1 und 4 KStG"},

	{"4852", TaxAdjustmentExemptGain, "§ 8b Abs. 2 KStG"},
	{"4858", TaxAdjustmentExemptGain, "§ 8b Abs. 2 KStG"},
	{"4904", TaxAdjustmentExemptGain, "§ 8b Abs. 2 KStG"},
	{"4906", TaxAdjustmentExemptGain, "§ 8b Abs. 2 KStG"},
	{"4914", TaxAdjustmentExemptGain, "§ 8b Abs. 2 KStG"},
	{"6857", TaxAdjustmentExemptGain, "§ 8b Abs. 2 Satz 2 KStG"},

	{"6391", TaxAdjustmentDonation, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6392", TaxAdjustmentDonation, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6393", TaxAdjustmentDonation, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6395", TaxAdjustmentDonation, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6397", TaxAdjustmentDonation, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
	{"6398", TaxAdjustmentDonation, "§ 9 Abs. 1 Satz 1 Nr. 2 KStG"},
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
