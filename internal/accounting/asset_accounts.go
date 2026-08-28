package accounting

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// AssetAccount is one entry of the curated Anlagekonten catalog.
//
// The SKR04 holds about 120 accounts in Kontenklasse 0. Offering all of them
// would push the decision that actually matters — welcher Posten des
// Anlagevermögens ist das? — onto a list nobody can read. This catalog names the
// cases a normal company meets, and for each of them the two accounts that
// belong together: das Bestandskonto und das Konto, auf dem seine Abschreibung
// landet.
//
// It does not replace the chart of accounts. An asset may carry any bookable
// Klasse-0-Konto; the catalog only decides what the input mask proposes.
type AssetAccount struct {
	Number string            `json:"number"`
	Name   string            `json:"name"`
	Class  domain.AssetClass `json:"class"`
	// Group sorts the entries into the blocks the Bilanz shows.
	Group string `json:"group"`
	Hint  string `json:"hint,omitempty"`

	// DepreciationAccount is the Aufwandskonto of the planmäßigen AfA. The SKR04
	// separates them by asset type — Gebäude, Fahrzeuge und der Rest haben eigene
	// Konten, weil die GuV sie getrennt ausweist.
	DepreciationAccount string `json:"depreciationAccount,omitempty"`

	// Depreciable is false where nothing wears out: Grund und Boden, Anlagen im
	// Bau, und das gesamte Finanzanlagevermögen.
	Depreciable bool `json:"depreciable"`

	// Immovable marks the unbeweglichen Wirtschaftsgüter: Grund und Boden und
	// alles, was darauf steht. Sie sind vom Kontenkatalog her erkennbar und
	// nirgends sonst — und die Unterscheidung entscheidet mit: die degressive
	// AfA des § 7 Abs. 2 EStG und die Sonderabschreibung des § 7g Abs. 5 EStG
	// gibt es nur für bewegliche Wirtschaftsgüter.
	Immovable bool `json:"immovable,omitempty"`

	// InProgress marks the Sammelkonten, auf denen etwas liegt, das noch nicht
	// fertig ist: Anlagen im Bau und geleistete Anzahlungen. Von ihnen wird mit
	// der Fertigstellung auf das endgültige Anlagekonto umgebucht, und erst dann
	// beginnt die Abschreibung.
	InProgress bool `json:"inProgress,omitempty"`

	// DefaultUsefulLifeMonths is a proposal, never a rule. Die AfA-Tabellen sind
	// eine Verwaltungsanweisung; sie binden die Finanzverwaltung, nicht den
	// Steuerpflichtigen. Eine begründete abweichende Nutzungsdauer ist zulässig,
	// deshalb bleibt der Wert überschreibbar. Null heißt: kein Vorschlag.
	DefaultUsefulLifeMonths int    `json:"defaultUsefulLifeMonths,omitempty"`
	UsefulLifeSource        string `json:"usefulLifeSource,omitempty"`
}

// assetAccounts is the curated catalog. Every number is checked against the
// shipped DATEV SKR04 2026 catalog by TestAssetAccountsExistInSKR04.
var assetAccounts = []AssetAccount{
	// --- Immaterielle Vermögensgegenstände (§ 266 Abs. 2 A. I. HGB) --------
	{Number: "0135", Name: "EDV-Software", Class: domain.AssetClassIntangible,
		Group: "Konzessionen, Lizenzen und Software", DepreciationAccount: "6200", Depreciable: true,
		Hint: "Entgeltlich erworbene Software. Selbst geschaffene immaterielle Vermögensgegenstände " +
			"dürfen nach § 248 Abs. 2 HGB nur wahlweise aktiviert werden — steuerlich gar nicht."},
	{Number: "0140", Name: "Lizenzen an gewerblichen Schutzrechten", Class: domain.AssetClassIntangible,
		Group: "Konzessionen, Lizenzen und Software", DepreciationAccount: "6200", Depreciable: true},
	{Number: "0110", Name: "Konzessionen", Class: domain.AssetClassIntangible,
		Group: "Konzessionen, Lizenzen und Software", DepreciationAccount: "6200", Depreciable: true},
	{Number: "0120", Name: "Gewerbliche Schutzrechte", Class: domain.AssetClassIntangible,
		Group: "Konzessionen, Lizenzen und Software", DepreciationAccount: "6200", Depreciable: true},
	{Number: "0130", Name: "Ähnliche Rechte und Werte", Class: domain.AssetClassIntangible,
		Group: "Konzessionen, Lizenzen und Software", DepreciationAccount: "6200", Depreciable: true},
	{Number: "0150", Name: "Geschäfts- oder Firmenwert", Class: domain.AssetClassIntangible,
		Group: "Geschäfts- oder Firmenwert", DepreciationAccount: "6205", Depreciable: true,
		DefaultUsefulLifeMonths: 180, UsefulLifeSource: "§ 7 Abs. 1 Satz 3 EStG",
		Hint: "Steuerlich über 15 Jahre abzuschreiben (§ 7 Abs. 1 Satz 3 EStG). Eine Zuschreibung ist " +
			"hier ausgeschlossen: § 253 Abs. 5 Satz 2 HGB verbietet sie für den Geschäfts- oder Firmenwert."},
	{Number: "0170", Name: "Geleistete Anzahlungen auf immaterielle Vermögensgegenstände",
		Class: domain.AssetClassIntangible, Group: "Geleistete Anzahlungen", Depreciable: false,
		InProgress: true,
		Hint:       "Bis zum Zugang des Rechts wird nicht abgeschrieben; danach wird umgebucht."},

	// --- Sachanlagen (§ 266 Abs. 2 A. II. HGB) ----------------------------
	{Number: "0215", Name: "Unbebaute Grundstücke", Class: domain.AssetClassTangible,
		Group: "Grundstücke und Bauten", Depreciable: false, Immovable: true,
		Hint: "Grund und Boden nutzt sich nicht ab und wird nicht planmäßig abgeschrieben. " +
			"An Wert verlieren kann er nur außerplanmäßig (§ 253 Abs. 3 Satz 5 HGB)."},
	{Number: "0235", Name: "Grundstückswerte eigener bebauter Grundstücke", Class: domain.AssetClassTangible,
		Group: "Grundstücke und Bauten", Depreciable: false, Immovable: true,
		Hint: "Der Grund und Boden wird vom Gebäude getrennt geführt: das Gebäude wird abgeschrieben, " +
			"der Boden nicht. Der Kaufpreis ist dafür aufzuteilen."},
	{Number: "0240", Name: "Geschäftsbauten", Class: domain.AssetClassTangible,
		Group: "Grundstücke und Bauten", DepreciationAccount: "6221", Depreciable: true, Immovable: true,
		Hint: "Gebäude folgen nicht den AfA-Tabellen, sondern den festen Sätzen des § 7 Abs. 4 EStG — " +
			"für Betriebsgebäude in der Regel 3 %, also 33 Jahre."},
	{Number: "0250", Name: "Fabrikbauten", Class: domain.AssetClassTangible,
		Group: "Grundstücke und Bauten", DepreciationAccount: "6221", Depreciable: true, Immovable: true,
		Hint: "Feste AfA-Sätze nach § 7 Abs. 4 EStG."},
	{Number: "0260", Name: "Andere Bauten", Class: domain.AssetClassTangible,
		Group: "Grundstücke und Bauten", DepreciationAccount: "6221", Depreciable: true, Immovable: true},
	{Number: "0300", Name: "Wohnbauten", Class: domain.AssetClassTangible,
		Group: "Grundstücke und Bauten", DepreciationAccount: "6221", Depreciable: true, Immovable: true,
		Hint: "Für Wohngebäude gelten eigene Sätze des § 7 Abs. 4 EStG."},
	{Number: "0420", Name: "Technische Anlagen", Class: domain.AssetClassTangible,
		Group: "Technische Anlagen und Maschinen", DepreciationAccount: "6220", Depreciable: true,
		UsefulLifeSource: "AfA-Tabelle des BMF für den jeweiligen Wirtschaftszweig"},
	{Number: "0440", Name: "Maschinen", Class: domain.AssetClassTangible,
		Group: "Technische Anlagen und Maschinen", DepreciationAccount: "6220", Depreciable: true,
		UsefulLifeSource: "AfA-Tabelle des BMF für den jeweiligen Wirtschaftszweig"},
	{Number: "0460", Name: "Maschinengebundene Werkzeuge", Class: domain.AssetClassTangible,
		Group: "Technische Anlagen und Maschinen", DepreciationAccount: "6220", Depreciable: true},
	{Number: "0520", Name: "Pkw", Class: domain.AssetClassTangible,
		Group: "Fahrzeuge", DepreciationAccount: "6222", Depreciable: true,
		DefaultUsefulLifeMonths: 72, UsefulLifeSource: "AfA-Tabelle AV (BMF): Personenkraftwagen sechs Jahre"},
	{Number: "0540", Name: "Lkw", Class: domain.AssetClassTangible,
		Group: "Fahrzeuge", DepreciationAccount: "6222", Depreciable: true,
		DefaultUsefulLifeMonths: 108, UsefulLifeSource: "AfA-Tabelle AV (BMF): Lastkraftwagen neun Jahre"},
	{Number: "0560", Name: "Sonstige Transportmittel", Class: domain.AssetClassTangible,
		Group: "Fahrzeuge", DepreciationAccount: "6222", Depreciable: true},
	{Number: "0620", Name: "Werkzeuge", Class: domain.AssetClassTangible,
		Group: "Betriebs- und Geschäftsausstattung", DepreciationAccount: "6220", Depreciable: true},
	{Number: "0630", Name: "Betriebsausstattung", Class: domain.AssetClassTangible,
		Group: "Betriebs- und Geschäftsausstattung", DepreciationAccount: "6220", Depreciable: true},
	{Number: "0635", Name: "Geschäftsausstattung", Class: domain.AssetClassTangible,
		Group: "Betriebs- und Geschäftsausstattung", DepreciationAccount: "6220", Depreciable: true},
	{Number: "0640", Name: "Ladeneinrichtung", Class: domain.AssetClassTangible,
		Group: "Betriebs- und Geschäftsausstattung", DepreciationAccount: "6220", Depreciable: true,
		DefaultUsefulLifeMonths: 96, UsefulLifeSource: "AfA-Tabelle AV (BMF): Ladeneinbauten acht Jahre"},
	{Number: "0650", Name: "Büroeinrichtung", Class: domain.AssetClassTangible,
		Group: "Betriebs- und Geschäftsausstattung", DepreciationAccount: "6220", Depreciable: true,
		DefaultUsefulLifeMonths: 156, UsefulLifeSource: "AfA-Tabelle AV (BMF): Büromöbel dreizehn Jahre"},
	{Number: "0690", Name: "Sonstige Betriebs- und Geschäftsausstattung", Class: domain.AssetClassTangible,
		Group: "Betriebs- und Geschäftsausstattung", DepreciationAccount: "6220", Depreciable: true},
	{Number: "0670", Name: "Geringwertige Wirtschaftsgüter", Class: domain.AssetClassTangible,
		Group: "Geringwertige Wirtschaftsgüter", DepreciationAccount: "6260", Depreciable: true,
		Hint: "Für den Sofortabzug nach § 6 Abs. 2 EStG. Ab 250 € gehört das Gut in ein laufend " +
			"geführtes Verzeichnis (§ 6 Abs. 2 Satz 4 EStG) — das Anlagenverzeichnis erfüllt diese Pflicht."},
	{Number: "0675", Name: "Wirtschaftsgüter (Sammelposten)", Class: domain.AssetClassTangible,
		Group: "Geringwertige Wirtschaftsgüter", DepreciationAccount: "6264", Depreciable: true,
		Hint: "Der Sammelposten eines Wirtschaftsjahres nach § 6 Abs. 2a EStG, aufzulösen mit je einem Fünftel."},
	{Number: "0700", Name: "Geleistete Anzahlungen und Anlagen im Bau", Class: domain.AssetClassTangible,
		Group: "Anlagen im Bau", Depreciable: false, InProgress: true,
		Hint: "Solange die Anlage nicht fertig ist, wird nicht abgeschrieben. Mit der Fertigstellung " +
			"wird auf das endgültige Anlagekonto umgebucht und die AfA beginnt."},
	{Number: "0710", Name: "Geschäfts-, Fabrik- und andere Bauten im Bau auf eigenen Grundstücken",
		Class: domain.AssetClassTangible, Group: "Anlagen im Bau", Depreciable: false, InProgress: true},
	{Number: "0770", Name: "Technische Anlagen und Maschinen im Bau", Class: domain.AssetClassTangible,
		Group: "Anlagen im Bau", Depreciable: false, InProgress: true},
	{Number: "0785", Name: "Andere Anlagen, Betriebs- und Geschäftsausstattung im Bau",
		Class: domain.AssetClassTangible, Group: "Anlagen im Bau", Depreciable: false, InProgress: true},

	// --- Finanzanlagen (§ 266 Abs. 2 A. III. HGB) --------------------------
	{Number: "0800", Name: "Anteile an verbundenen Unternehmen", Class: domain.AssetClassFinancial,
		Group: "Anteile und Beteiligungen", Depreciable: false},
	{Number: "0820", Name: "Beteiligungen", Class: domain.AssetClassFinancial,
		Group: "Anteile und Beteiligungen", Depreciable: false,
		Hint: "Als Beteiligung gelten Anteile, die dem eigenen Geschäftsbetrieb dauernd dienen sollen. " +
			"Ab einem Fünftel des Kapitals vermutet § 271 Abs. 1 Satz 3 HGB das."},
	{Number: "0850", Name: "Beteiligungen an Kapitalgesellschaften", Class: domain.AssetClassFinancial,
		Group: "Anteile und Beteiligungen", Depreciable: false},
	{Number: "0860", Name: "Beteiligungen an Personengesellschaften", Class: domain.AssetClassFinancial,
		Group: "Anteile und Beteiligungen", Depreciable: false},
	{Number: "0900", Name: "Wertpapiere des Anlagevermögens", Class: domain.AssetClassFinancial,
		Group: "Wertpapiere", Depreciable: false,
		Hint: "Wertpapiere gehören nur ins Anlagevermögen, wenn sie dauernd dem Geschäftsbetrieb dienen " +
			"sollen. Sonst sind es Wertpapiere des Umlaufvermögens mit strengerem Niederstwertprinzip."},
	{Number: "0920", Name: "Festverzinsliche Wertpapiere", Class: domain.AssetClassFinancial,
		Group: "Wertpapiere", Depreciable: false},
	{Number: "0810", Name: "Ausleihungen an verbundene Unternehmen", Class: domain.AssetClassFinancial,
		Group: "Ausleihungen", Depreciable: false},
	{Number: "0880", Name: "Ausleihungen an Unternehmen mit Beteiligungsverhältnis",
		Class: domain.AssetClassFinancial, Group: "Ausleihungen", Depreciable: false},
	{Number: "0930", Name: "Übrige sonstige Ausleihungen", Class: domain.AssetClassFinancial,
		Group: "Ausleihungen", Depreciable: false},
	{Number: "0940", Name: "Darlehen", Class: domain.AssetClassFinancial,
		Group: "Ausleihungen", Depreciable: false},
	{Number: "0980", Name: "Genossenschaftsanteile zum langfristigen Verbleib",
		Class: domain.AssetClassFinancial, Group: "Anteile und Beteiligungen", Depreciable: false},
	{Number: "0990", Name: "Rückdeckungsansprüche aus Lebensversicherungen zum langfristigen Verbleib",
		Class: domain.AssetClassFinancial, Group: "Ausleihungen", Depreciable: false},
}

// AssetAccounts returns the catalog, optionally narrowed to one class.
func AssetAccounts(class domain.AssetClass) []AssetAccount {
	out := make([]AssetAccount, 0, len(assetAccounts))
	for _, a := range assetAccounts {
		if class == "" || a.Class == class {
			out = append(out, a)
		}
	}
	return out
}

// LookupAssetAccount returns the catalog entry of an account number.
func LookupAssetAccount(number string) (AssetAccount, bool) {
	for _, a := range assetAccounts {
		if a.Number == number {
			return a, true
		}
	}
	return AssetAccount{}, false
}

// GoodwillAccount is the Geschäfts- oder Firmenwert. It is named because
// § 253 Abs. 5 Satz 2 HGB singles it out: on every other asset a Zuschreibung is
// mandatory once the reason for the write-down is gone, on this one it is
// forbidden.
const GoodwillAccount = "0150"

// PoolAccount and GWGAccount are the two Sachanlagenkonten the Wertgrenzen of
// § 6 Abs. 2 und 2a EStG lead to.
const (
	GWGAccount        = "0670"
	GWGExpenseAccount = "6260"
	PoolAccount       = "0675"
	PoolExpenseAcc    = "6264"
)

// ImpairmentAccount returns the Aufwandskonto of an außerplanmäßige
// Abschreibung.
//
// The split between permanent and temporary is not a formality. § 253 Abs. 3
// Satz 5 HGB allows a write-down on Sachanlagen und immaterielle Vermögens-
// gegenstände only bei voraussichtlich dauernder Wertminderung; Satz 6 gives
// Finanzanlagen the extra option to write down auch bei einer nicht dauernden.
// Booking the two to the same account would lose the distinction the very next
// day.
func ImpairmentAccount(class domain.AssetClass, account string, permanent, taxPrivileged bool) (string, error) {
	switch class {
	case domain.AssetClassIntangible:
		if !permanent {
			return "", fmt.Errorf(
				"auf immaterielle Vermögensgegenstände darf nur bei voraussichtlich dauernder " +
					"Wertminderung außerplanmäßig abgeschrieben werden (§ 253 Abs. 3 Satz 5 HGB)")
		}
		if account == GoodwillAccount {
			return "6209", nil
		}
		return "6210", nil
	case domain.AssetClassTangible:
		if !permanent {
			return "", fmt.Errorf(
				"auf Sachanlagen darf nur bei voraussichtlich dauernder Wertminderung außerplanmäßig " +
					"abgeschrieben werden (§ 253 Abs. 3 Satz 5 HGB). Die Ausnahme für die nicht dauernde " +
					"Wertminderung gilt allein für Finanzanlagen (Satz 6)")
		}
		return "6230", nil
	case domain.AssetClassFinancial:
		if taxPrivileged {
			// § 8b Abs. 3 KStG bzw. § 3c Abs. 2 EStG: die Abschreibung auf einen
			// solchen Anteil bleibt außer Ansatz. Sie gehört deshalb auf ein
			// eigenes Konto, sonst ist die außerbilanzielle Korrektur später nicht
			// mehr auffindbar.
			return "7204", nil
		}
		if permanent {
			return "7200", nil
		}
		return "7201", nil
	default:
		return "", fmt.Errorf("unbekannte Anlagenklasse %q", class)
	}
}

// WriteUpAccount returns the Ertragskonto of a Zuschreibung.
func WriteUpAccount(class domain.AssetClass, account string, taxPrivileged bool) (string, error) {
	switch class {
	case domain.AssetClassIntangible:
		if account == GoodwillAccount {
			return "", fmt.Errorf(
				"auf den Geschäfts- oder Firmenwert darf nicht zugeschrieben werden (§ 253 Abs. 5 Satz 2 HGB)")
		}
		return "4911", nil
	case domain.AssetClassTangible:
		return "4910", nil
	case domain.AssetClassFinancial:
		if taxPrivileged {
			return "4913", nil
		}
		return "4912", nil
	default:
		return "", fmt.Errorf("unbekannte Anlagenklasse %q", class)
	}
}

// DisposalAccounts are the two accounts an Abgang needs: the one the Erlös lands
// on and the one the Restbuchwert is written off to.
type DisposalAccounts struct {
	// Revenue carries the Verkaufserlös. Empty on a Verschrottung.
	Revenue string `json:"revenue,omitempty"`
	// BookValue carries the Restbuchwert.
	BookValue string `json:"bookValue"`
	// Explanation says in one sentence why these two accounts and not the others.
	Explanation string `json:"explanation"`
}

// DisposalAccountsFor picks the accounts of an Abgang.
//
// This is the trap of the whole Anlagenbuchhaltung: **der SKR04 wählt das
// Erlöskonto nach dem Ergebnis, nicht nach dem Vorgang.** Derselbe Verkauf
// landet einmal unter den Erträgen (4845/4855) und einmal unter den
// Aufwendungen (6885/6895) — je nachdem, ob der Verkaufspreis über oder unter
// dem Restbuchwert lag. Die Kontierung hängt hier also am Rechenergebnis und
// nicht nur an der Eingabe des Nutzers, weshalb sie hier zentral fällt und nicht
// in der Oberfläche.
func DisposalAccountsFor(class domain.AssetClass, treatment domain.TaxTreatment, gain, taxPrivileged bool) (DisposalAccounts, error) {
	switch class {
	case domain.AssetClassTangible:
		if gain {
			return DisposalAccounts{
				Revenue:   tangibleRevenueAccount(treatment, true),
				BookValue: "4855",
				Explanation: "Der Verkaufserlös liegt über dem Restbuchwert: es entsteht ein Buchgewinn. " +
					"Der SKR04 führt Erlös und Restbuchwert dann unter den sonstigen betrieblichen Erträgen.",
			}, nil
		}
		return DisposalAccounts{
			Revenue:   tangibleRevenueAccount(treatment, false),
			BookValue: "6895",
			Explanation: "Der Verkaufserlös liegt unter dem Restbuchwert: es entsteht ein Buchverlust. " +
				"Derselbe Vorgang läuft im SKR04 dann über die sonstigen betrieblichen Aufwendungen.",
		}, nil
	case domain.AssetClassIntangible:
		if gain {
			return DisposalAccounts{Revenue: "4850", BookValue: "4856",
				Explanation: "Buchgewinn aus dem Abgang eines immateriellen Vermögensgegenstands."}, nil
		}
		return DisposalAccounts{Revenue: "6890", BookValue: "6896",
			Explanation: "Buchverlust aus dem Abgang eines immateriellen Vermögensgegenstands."}, nil
	case domain.AssetClassFinancial:
		switch {
		case gain && taxPrivileged:
			return DisposalAccounts{Revenue: "4852", BookValue: "4858",
				Explanation: "Buchgewinn aus dem Verkauf eines Anteils, dessen Gewinn dem Teileinkünfteverfahren " +
					"(§ 3 Nr. 40 EStG) bzw. § 8b Abs. 2 KStG unterliegt. Eigene Konten, damit die außerbilanzielle " +
					"Korrektur später auffindbar bleibt."}, nil
		case gain:
			return DisposalAccounts{Revenue: "4851", BookValue: "4857",
				Explanation: "Buchgewinn aus dem Abgang einer Finanzanlage."}, nil
		case taxPrivileged:
			return DisposalAccounts{Revenue: "6892", BookValue: "6898",
				Explanation: "Buchverlust aus dem Verkauf eines Anteils nach § 3 Nr. 40 EStG bzw. § 8b KStG; " +
					"der Verlust bleibt steuerlich außer Ansatz (§ 8b Abs. 3 KStG, § 3c Abs. 2 EStG)."}, nil
		default:
			return DisposalAccounts{Revenue: "6891", BookValue: "6897",
				Explanation: "Buchverlust aus dem Abgang einer Finanzanlage."}, nil
		}
	default:
		return DisposalAccounts{}, fmt.Errorf("unbekannte Anlagenklasse %q", class)
	}
}

// tangibleRevenueAccount picks the Erlöskonto of a Sachanlagenverkauf by
// Steuerfall. Only the Sachanlagen have treatment-specific accounts in the
// SKR04 — for the other two blocks the catalog knows one account per outcome.
func tangibleRevenueAccount(treatment domain.TaxTreatment, gain bool) string {
	switch treatment {
	case domain.TaxTreatmentExport:
		if gain {
			return "4844"
		}
		return "6884"
	case domain.TaxTreatmentIntraCommunitySupply:
		if gain {
			return "4848"
		}
		return "6888"
	case domain.TaxTreatmentDomestic:
		if gain {
			return "4845"
		}
		return "6885"
	default:
		// Steuerfrei nach § 4 Nr. 8 ff., nicht steuerbar oder zum Nullsteuersatz:
		// der SKR04 hält dafür das Konto ohne Steuerschlüssel bereit.
		if gain {
			return "4849"
		}
		return "6889"
	}
}

// SpecialDepreciationAccount picks the Aufwandskonto der Sonderabschreibung nach
// § 7g Abs. 5 EStG — und weist ab, wo es sie nicht gibt.
//
// Die Vorschrift gilt nur für **abnutzbare bewegliche** Wirtschaftsgüter des
// Anlagevermögens. Das ist keine Formalie: ein Gebäude ist eine Sachanlage wie
// eine Maschine, aber unbeweglich, und für es kommt die Sonderabschreibung nicht
// in Betracht. Woran das erkennbar ist, weiß allein der Kontenkatalog.
func SpecialDepreciationAccount(class domain.AssetClass, account string) (string, error) {
	if class != domain.AssetClassTangible {
		return "", fmt.Errorf(
			"die Sonderabschreibung nach § 7g Abs. 5 EStG gibt es nur für bewegliche Wirtschaftsgüter " +
				"des Sachanlagevermögens")
	}
	entry, known := LookupAssetAccount(account)
	if known && entry.Immovable {
		return "", fmt.Errorf(
			"%s (%s) trägt ein unbewegliches Wirtschaftsgut. § 7g Abs. 5 EStG begünstigt nur "+
				"bewegliche — für Gebäude gelten die festen Sätze des § 7 Abs. 4 EStG",
			account, entry.Name)
	}
	if known && entry.InProgress {
		return "", fmt.Errorf(
			"%s steht noch als Anlage im Bau. Die Sonderabschreibung setzt die Anschaffung oder "+
				"Herstellung voraus; buche zuerst die Fertigstellung um", account)
	}
	if known && entry.DepreciationAccount == "6222" {
		// Der SKR04 führt die Sonderabschreibung auf Fahrzeuge getrennt, weil die
		// GuV sie getrennt ausweist.
		return "6242", nil
	}
	return "6241", nil
}

// MaintenanceAccount picks the Aufwandskonto des Erhaltungsaufwands.
//
// Erhaltungsaufwand ist der Gegenbegriff zu den nachträglichen
// Herstellungskosten: was ein Wirtschaftsgut nur in seinem Zustand hält oder in
// zeitgemäßer Weise wiederherstellt, ist sofort abziehbarer Aufwand; was es
// erweitert oder über seinen ursprünglichen Zustand hinaus wesentlich
// verbessert, ist zu aktivieren (§ 255 Abs. 2 Satz 1 HGB). Die Abgrenzung ist
// eine Einschätzung und keine Rechnung — Buchfink fragt sie und bucht danach.
func MaintenanceAccount(class domain.AssetClass, account string) (string, error) {
	if class == domain.AssetClassFinancial {
		return "", fmt.Errorf(
			"eine Finanzanlage nutzt sich nicht ab und wird nicht instand gehalten; " +
				"Erhaltungsaufwand gibt es für sie nicht")
	}
	entry, known := LookupAssetAccount(account)
	switch {
	case known && entry.Immovable:
		return "6450", nil // Reparaturen und Instandhaltung von Bauten
	case known && entry.Group == "Fahrzeuge":
		return "6540", nil // Fahrzeug-Reparaturen
	case known && entry.Group == "Technische Anlagen und Maschinen":
		return "6460", nil // Reparaturen und Instandhaltung von technischen Anlagen und Maschinen
	case known:
		return "6470", nil // andere Anlagen und Betriebs- und Geschäftsausstattung
	default:
		return "6490", nil // Sonstige Reparaturen und Instandhaltung
	}
}

// AssetIncomeAccount is the Ertragskonto of a laufender Ertrag aus einer
// Finanzanlage.
//
// Der SKR04 trennt die Erträge des Finanzanlagevermögens nach ihrer Herkunft,
// weil die GuV sie nach § 275 Abs. 2 Nr. 9 bis 11 HGB getrennt ausweist:
// Erträge aus Beteiligungen, Erträge aus anderen Wertpapieren und Ausleihungen
// des Finanzanlagevermögens, sonstige Zinsen. Auf ein Sammelkonto gebucht wären
// sie in der GuV nicht mehr auseinanderzuhalten.
func AssetIncomeAccount(class domain.AssetClass, account string) (string, error) {
	if class != domain.AssetClassFinancial {
		return "", fmt.Errorf(
			"laufende Erträge werden hier nur für Finanzanlagen mit dem Anlagegut verknüpft")
	}
	entry, known := LookupAssetAccount(account)
	if !known {
		return "7100", nil // Sonstige Zinsen und ähnliche Erträge
	}
	switch entry.Group {
	case "Anteile und Beteiligungen":
		return "7000", nil // Erträge aus Beteiligungen
	case "Ausleihungen":
		return "7011", nil // Erträge aus Ausleihungen des Finanzanlagevermögens
	case "Wertpapiere":
		return "7010", nil // Erträge aus anderen Wertpapieren des Finanzanlagevermögens
	default:
		return "7100", nil
	}
}

// CurrencyTranslationAccounts are the two accounts of § 256a HGB: der Aufwand
// aus der Währungsumrechnung und der Ertrag daraus.
const (
	CurrencyLossAccount = "6880" // Aufwendungen aus der Währungsumrechnung
	CurrencyGainAccount = "4840" // Erträge aus der Währungsumrechnung
)

// WithholdingTaxAccount trägt die einbehaltene Kapitalertragsteuer.
//
// Sie ist kein Aufwand des Beteiligungsertrags, sondern eine Vorauszahlung auf
// die eigene Steuer — sie mindert den Zufluss, nicht den Ertrag. Der Ertrag
// steht deshalb brutto in der GuV, und die einbehaltene Steuer daneben.
const WithholdingTaxAccount = "7630"
