package accounting

import (
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

// PostingRuleVersion stamps every booking with the version of the mapping that
// produced it. When a group is later remapped to a different account, the old
// bookings stay explainable — which is what the Verfahrensdokumentation has to
// show.
const PostingRuleVersion = "2026.1"

// DeductibleQuota names a statutory limit on how much of an expense may be
// deducted. It is a key into the dated parameters, not the figure itself.
type DeductibleQuota string

const (
	// QuotaEntertainment is the entertainment share of § 4 Abs. 5 Satz 1 Nr. 2
	// EStG. Only the expense is split — the input tax stays fully deductible,
	// because § 15 Abs. 1a Satz 2 UStG takes entertainment out of the ban.
	QuotaEntertainment DeductibleQuota = "entertainment"
)

// ExpenseLimit benennt eine Freigrenze, die nicht den Betrag teilt, sondern das
// Konto entscheidet.
//
// Der Unterschied zur DeductibleQuota ist der Unterschied zwischen Freigrenze
// und Freibetrag. Bei der Bewirtung sind 70 % jedes Betrages abziehbar — der
// Aufwand wird geteilt. Beim Geschenk ist der ganze Betrag abziehbar oder gar
// keiner, und was von beidem gilt, entscheidet die Summe des Jahres je
// Empfänger. Eine Quote kann das nicht ausdrücken.
type ExpenseLimit string

const (
	// LimitGiftPerRecipient ist die Freigrenze des § 4 Abs. 5 Satz 1 Nr. 1 EStG
	// je Empfänger und Wirtschaftsjahr. Wird sie überschritten, sind sämtliche
	// Geschenke an diesen Empfänger nicht abziehbar — und mit ihnen entfällt
	// nach § 15 Abs. 1a UStG auch der Vorsteuerabzug.
	LimitGiftPerRecipient ExpenseLimit = "gift_per_recipient"
)

// PostingGroup is a fachliche Gruppe the user picks instead of an account
// number. The mapping to SKR04 is fixed and deterministic: no learning, no
// heuristics, no per-user drift.
type PostingGroup struct {
	Key       string           `json:"key"`
	Label     string           `json:"label"`
	Category  string           `json:"category"`
	Hint      string           `json:"hint,omitempty"`
	Direction domain.Direction `json:"direction"`

	// Account is the SKR04 account this group books to by default.
	Account string `json:"account"`
	// RateAccounts overrides the account for a specific VAT rate.
	RateAccounts map[domain.TaxRate]string `json:"-"`
	// TreatmentAccounts overrides the account for a specific Steuerfall. A
	// Reverse-Charge purchase of services does not belong on the same account as
	// a domestic one, because the VAT return reports them in different boxes.
	TreatmentAccounts map[domain.TaxTreatment]string `json:"-"`

	// NonDeductibleAccount carries the share of an expense that is an expense
	// under commercial law but only partly deductible for tax purposes. Booking
	// both shares to one account would make the Steuerbilanz wrong.
	NonDeductibleAccount string `json:"nonDeductibleAccount,omitempty"`
	// DeductibleQuota names which statutory share applies to this group. The
	// figure itself is dated and lives in tax_params.go.
	DeductibleQuota DeductibleQuota `json:"deductibleQuota,omitempty"`

	// Limit benennt eine Freigrenze, die über das Konto entscheidet statt den
	// Betrag zu teilen. Leer heißt: keine.
	Limit ExpenseLimit `json:"limit,omitempty"`
	// RecipientRequired verlangt die Angabe des Empfängers.
	//
	// Bei Geschenken ist sie keine Zierde, sondern die Voraussetzung des Abzugs:
	// § 4 Abs. 7 EStG lässt ihn nur zu, wenn die Aufwendungen einzeln und
	// getrennt aufgezeichnet sind, und ohne Empfänger ließe sich die Freigrenze
	// überdies nicht führen.
	RecipientRequired bool `json:"recipientRequired,omitempty"`
	// InputTaxExcluded sagt, dass zu dieser Gruppe kein Vorsteuerabzug gehört
	// (§ 15 Abs. 1a UStG). Die nicht abziehbare Vorsteuer gehört dann zum
	// Aufwand.
	InputTaxExcluded bool `json:"inputTaxExcluded,omitempty"`

	DefaultRate domain.TaxRate `json:"defaultRate"`
	// DefaultTreatment is the Steuerfall the group proposes. It matters for the
	// groups that carry no VAT: a rate of zero says "no rate applies", but not
	// *why*, and the booking core insists on the reason. Empty means
	// TaxTreatmentDomestic.
	DefaultTreatment domain.TaxTreatment `json:"defaultTreatment,omitempty"`
}

// Treatment returns the Steuerfall the group proposes, defaulting to a domestic
// taxable transaction.
func (g PostingGroup) Treatment() domain.TaxTreatment {
	if g.DefaultTreatment == "" {
		return domain.TaxTreatmentDomestic
	}
	return g.DefaultTreatment
}

// ResolveAccount returns the SKR04 account for a Steuerfall and rate.
func (g PostingGroup) ResolveAccount(treatment domain.TaxTreatment, rate domain.TaxRate) string {
	if acc, ok := g.TreatmentAccounts[treatment]; ok {
		return acc
	}
	if acc, ok := g.RateAccounts[rate]; ok {
		return acc
	}
	return g.Account
}

// postingGroups is the curated catalog. Every account number was checked
// against the DATEV SKR04 2026 template; the test in posting_groups_test.go
// re-checks it against the shipped catalog on every build.
var postingGroups = []PostingGroup{
	// --- Erträge ---------------------------------------------------------
	{
		Key: "erloese", Label: "Erlöse", Category: "Umsatzerlöse",
		Hint:      "Umsätze aus der gewöhnlichen Geschäftstätigkeit.",
		Direction: domain.DirectionOutgoing, Account: "4400", DefaultRate: domain.TaxRateStandard,
		RateAccounts: map[domain.TaxRate]string{
			domain.TaxRateReduced: "4300",
		},
		TreatmentAccounts: map[domain.TaxTreatment]string{
			domain.TaxTreatmentIntraCommunitySupply: "4125", // § 4 Nr. 1b UStG
			domain.TaxTreatmentExport:               "4120", // § 4 Nr. 1a UStG
			domain.TaxTreatmentReverseChargeSupply:  "4337", // § 13b UStG
			// Nullsteuersatz nach § 12 Abs. 3 UStG. Eigenes Konto, weil der
			// Umsatz steuerpflichtig ist und nicht zu den steuerfreien zählt.
			domain.TaxTreatmentZeroRated: "4290",
			domain.TaxTreatmentExempt:    "4150",
		},
	},
	{
		Key: "sonstige_ertraege", Label: "Sonstige betriebliche Erträge", Category: "Umsatzerlöse",
		Hint:      "Erträge außerhalb des eigentlichen Geschäftszwecks, z. B. Anlagenverkauf oder Erstattungen.",
		Direction: domain.DirectionOutgoing, Account: "4830", DefaultRate: domain.TaxRateStandard,
		TreatmentAccounts: map[domain.TaxTreatment]string{
			domain.TaxTreatmentExempt: "4842",
		},
	},

	// --- Material und Fremdleistungen ------------------------------------
	{
		Key: "wareneingang", Label: "Wareneingang", Category: "Material & Fremdleistungen",
		Hint:      "Eingekaufte Waren zum Weiterverkauf.",
		Direction: domain.DirectionIncoming, Account: "5400", DefaultRate: domain.TaxRateStandard,
		RateAccounts: map[domain.TaxRate]string{
			domain.TaxRateReduced: "5300",
		},
		TreatmentAccounts: map[domain.TaxTreatment]string{
			domain.TaxTreatmentReverseCharge: "5349",
		},
	},
	{
		Key: "fremdleistungen", Label: "Fremdleistungen", Category: "Material & Fremdleistungen",
		Hint:      "Zugekaufte Leistungen von Subunternehmern und Dienstleistern.",
		Direction: domain.DirectionIncoming, Account: "5906", DefaultRate: domain.TaxRateStandard,
		RateAccounts: map[domain.TaxRate]string{
			domain.TaxRateReduced: "5908",
			domain.TaxRateNone:    "5900",
		},
		TreatmentAccounts: map[domain.TaxTreatment]string{
			// § 13b: Leistung ohne ausgewiesene Vorsteuer, Steuerschuld beim
			// Empfänger. Das eigene Konto ist Voraussetzung dafür, dass die
			// UStVA die Kennzahlen 84/85 richtig füllt.
			domain.TaxTreatmentReverseCharge: "5909",
		},
	},

	// --- Raum- und Fahrzeugkosten ----------------------------------------
	{Key: "miete", Label: "Miete & Pacht", Category: "Raumkosten",
		Hint:      "Miete für Büro, Lager oder andere unbewegliche Wirtschaftsgüter. Nach § 4 Nr. 12 UStG steuerfrei, sofern der Vermieter nicht nach § 9 UStG zur Steuerpflicht optiert hat.",
		Direction: domain.DirectionIncoming, Account: "6310",
		DefaultRate: domain.TaxRateNone, DefaultTreatment: domain.TaxTreatmentExempt},
	{Key: "raumkosten", Label: "Nebenkosten & sonstige Raumkosten", Category: "Raumkosten",
		Hint:      "Strom, Heizung, Reinigung.",
		Direction: domain.DirectionIncoming, Account: "6345", DefaultRate: domain.TaxRateStandard},
	{Key: "fahrzeugkosten", Label: "Fahrzeugkosten", Category: "Fahrzeuge",
		Hint:      "Kraftstoff, Wartung, Kfz-Kosten.",
		Direction: domain.DirectionIncoming, Account: "6500", DefaultRate: domain.TaxRateStandard},

	// --- Verwaltung -------------------------------------------------------
	{Key: "buerobedarf", Label: "Bürobedarf", Category: "Verwaltung",
		Hint:      "Verbrauchsmaterial für das Büro.",
		Direction: domain.DirectionIncoming, Account: "6815", DefaultRate: domain.TaxRateStandard},
	{Key: "porto", Label: "Porto & Versand", Category: "Verwaltung",
		Direction: domain.DirectionIncoming, Account: "6800", DefaultRate: domain.TaxRateStandard},
	{Key: "telefon", Label: "Telefon & Internet", Category: "Verwaltung",
		Direction: domain.DirectionIncoming, Account: "6805", DefaultRate: domain.TaxRateStandard,
		TreatmentAccounts: map[domain.TaxTreatment]string{}},
	{Key: "software", Label: "Software & IT-Wartung", Category: "Verwaltung",
		Hint:      "SaaS-Abos, Hosting, Wartung von Hard- und Software. Anbieter im EU-Ausland führen meist zu § 13b.",
		Direction: domain.DirectionIncoming, Account: "6495", DefaultRate: domain.TaxRateStandard},
	{Key: "beratung", Label: "Rechts- & Beratungskosten", Category: "Verwaltung",
		Direction: domain.DirectionIncoming, Account: "6825", DefaultRate: domain.TaxRateStandard},
	{Key: "abschlusskosten", Label: "Abschluss- & Prüfungskosten", Category: "Verwaltung",
		Hint:      "Steuerberater, Jahresabschluss, Prüfung.",
		Direction: domain.DirectionIncoming, Account: "6827", DefaultRate: domain.TaxRateStandard},
	{Key: "versicherungen", Label: "Versicherungen", Category: "Verwaltung",
		Hint:      "Versicherungsprämien sind nach § 4 Nr. 10 UStG steuerfrei – hier fällt keine Vorsteuer an.",
		Direction: domain.DirectionIncoming, Account: "6400",
		DefaultRate: domain.TaxRateNone, DefaultTreatment: domain.TaxTreatmentExempt},
	{Key: "beitraege", Label: "Beiträge & Gebühren", Category: "Verwaltung",
		Hint:      "Kammer- und Verbandsbeiträge, behördliche Gebühren: meist nicht steuerbar.",
		Direction: domain.DirectionIncoming, Account: "6420",
		DefaultRate: domain.TaxRateNone, DefaultTreatment: domain.TaxTreatmentNotTaxable},
	{Key: "geldverkehr", Label: "Nebenkosten des Geldverkehrs", Category: "Verwaltung",
		Hint:      "Kontoführung, Überweisungsentgelte, Zahlungsdienstleister-Gebühren. Nach § 4 Nr. 8 UStG steuerfrei.",
		Direction: domain.DirectionIncoming, Account: "6855",
		DefaultRate: domain.TaxRateNone, DefaultTreatment: domain.TaxTreatmentExempt},

	// --- Personal und Vertrieb -------------------------------------------
	{Key: "gehaelter", Label: "Gehälter", Category: "Personal",
		Hint:      "Arbeitslohn ist kein Leistungsaustausch im Sinne des UStG.",
		Direction: domain.DirectionIncoming, Account: "6020",
		DefaultRate: domain.TaxRateNone, DefaultTreatment: domain.TaxTreatmentNotTaxable},
	{Key: "fortbildung", Label: "Fortbildung", Category: "Personal",
		Direction: domain.DirectionIncoming, Account: "6821", DefaultRate: domain.TaxRateStandard},
	{Key: "reisekosten", Label: "Reisekosten", Category: "Personal",
		Hint:      "Fahrt- und Übernachtungskosten. Verpflegungspauschalen gehören auf ein eigenes Konto.",
		Direction: domain.DirectionIncoming, Account: "6650", DefaultRate: domain.TaxRateStandard},
	{Key: "werbung", Label: "Werbekosten", Category: "Vertrieb",
		Direction: domain.DirectionIncoming, Account: "6600", DefaultRate: domain.TaxRateStandard},
	{Key: "bewirtung", Label: "Bewirtungskosten", Category: "Vertrieb",
		Hint:      "Nach § 4 Abs. 5 Satz 1 Nr. 2 EStG sind 70 % abziehbar; der Rest wird auf 6644 gebucht. Die Vorsteuer bleibt voll abziehbar. Ort, Tag, Teilnehmer und Anlass sind aufzuzeichnen.",
		Direction: domain.DirectionIncoming, Account: "6640",
		NonDeductibleAccount: "6644", DeductibleQuota: QuotaEntertainment,
		DefaultRate: domain.TaxRateStandard},
	{Key: "geschenke", Label: "Geschenke an Geschäftspartner", Category: "Vertrieb",
		Hint: "Nach § 4 Abs. 5 Satz 1 Nr. 1 EStG abziehbar, solange die Geschenke an einen Empfänger " +
			"im Wirtschaftsjahr die Freigrenze nicht übersteigen. Es ist eine Freigrenze: mit dem ersten " +
			"Cent darüber ist der gesamte Betrag nicht abziehbar, und mit ihm entfällt nach " +
			"§ 15 Abs. 1a UStG der Vorsteuerabzug. Der Empfänger ist aufzuzeichnen (§ 4 Abs. 7 EStG).",
		Direction: domain.DirectionIncoming, Account: "6610",
		NonDeductibleAccount: "6620", Limit: LimitGiftPerRecipient, RecipientRequired: true,
		DefaultRate: domain.TaxRateStandard},
	{Key: "geschenke_betrieblich", Label: "Geschenke zur ausschließlich betrieblichen Nutzung",
		Category: "Vertrieb",
		Hint: "Ein Gegenstand, den der Empfänger nur betrieblich nutzen kann, fällt nicht unter die " +
			"Freigrenze des § 4 Abs. 5 Satz 1 Nr. 1 EStG. Er ist unbegrenzt abziehbar — aufzuzeichnen " +
			"ist er trotzdem.",
		Direction: domain.DirectionIncoming, Account: "6625", RecipientRequired: true,
		DefaultRate: domain.TaxRateStandard},
	{Key: "repraesentation", Label: "Gästehaus, Jagd, Fischerei, Yacht", Category: "Vertrieb",
		Hint: "Aufwendungen nach § 4 Abs. 5 Satz 1 Nr. 3 und 4 EStG: Gästehäuser außerhalb des " +
			"Betriebsorts sowie Jagd, Fischerei, Segel- und Motoryachten samt der dazugehörigen " +
			"Bewirtung. Sie sind in keiner Höhe abziehbar, und § 15 Abs. 1a UStG nimmt ihnen auch den " +
			"Vorsteuerabzug — die Umsatzsteuer gehört deshalb in den Aufwand.",
		Direction: domain.DirectionIncoming, Account: "6645", InputTaxExcluded: true,
		DefaultRate: domain.TaxRateStandard},

	// --- Anlagen ----------------------------------------------------------
	{Key: "gwg", Label: "Geringwertige Wirtschaftsgüter (Sofortabschreibung)", Category: "Anlagen",
		Hint:      "Selbständig nutzbare Güter bis zur GWG-Grenze nach § 6 Abs. 2 EStG.",
		Direction: domain.DirectionIncoming, Account: "6260", DefaultRate: domain.TaxRateStandard},
	{Key: "sonstige_aufwendungen", Label: "Sonstige betriebliche Aufwendungen", Category: "Sonstiges",
		Hint:      "Nur verwenden, wenn keine passendere Gruppe zutrifft.",
		Direction: domain.DirectionIncoming, Account: "6300", DefaultRate: domain.TaxRateStandard},
}

var groupIndex = func() map[string]PostingGroup {
	m := make(map[string]PostingGroup, len(postingGroups))
	for _, g := range postingGroups {
		m[g.Key] = g
	}
	return m
}()

// PostingGroups returns the catalog for a direction, sorted by category then
// label so the UI can group them without re-sorting.
// RevenueAccountTreatments maps every Erlöskonto that stands for a particular
// Steuerfall back to that Steuerfall.
//
// It is derived from the catalogue rather than written out a second time. The
// Umsatzsteuer-Auswertung has to recognise these accounts, and a hand-kept copy
// of the same table drifts silently: a Steuerfall added here would simply stop
// appearing in the Voranmeldung, in no bucket at all.
func RevenueAccountTreatments() map[string]domain.TaxTreatment {
	out := map[string]domain.TaxTreatment{}
	for _, g := range postingGroups {
		if g.Direction != domain.DirectionOutgoing {
			continue
		}
		for treatment, account := range g.TreatmentAccounts {
			out[account] = treatment
		}
	}
	return out
}

// TaxableRevenueAccounts sind die Erlöskonten des steuerpflichtigen
// Inlandsumsatzes — die Konten, auf denen ein Steuerausweis richtig ist.
//
// Sie stehen als Gegenstück zu RevenueAccountTreatments: dort die Konten, deren
// Steuerfall keine Steuer entstehen lässt, hier die, die eine tragen. Wer über
// § 14c UStG entscheidet, braucht beide Seiten — sonst hielte er ein Konto ohne
// Steuerfall für ein steuerfreies.
func TaxableRevenueAccounts() map[string]bool {
	out := map[string]bool{}
	for _, g := range postingGroups {
		if g.Direction != domain.DirectionOutgoing {
			continue
		}
		if g.Account != "" {
			out[g.Account] = true
		}
		for _, account := range g.RateAccounts {
			out[account] = true
		}
	}
	return out
}

func PostingGroups(dir domain.Direction) []PostingGroup {
	// Belegt statt nil: die Liste geht als JSON an die Oberfläche, und eine
	// Richtung ohne Treffer käme dort sonst als `null` an.
	out := make([]PostingGroup, 0, len(postingGroups))
	for _, g := range postingGroups {
		if dir == "" || g.Direction == dir {
			out = append(out, g)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].Label < out[j].Label
	})
	return out
}

// AccountsRequiringGroup sind die Konten der beschränkt abziehbaren
// Betriebsausgaben, die nur über ihre Buchungsgruppe erreichbar sind.
//
// Der freie Kontoweg der Belegmaske ist der Notausgang für Fälle, die der
// Katalog nicht kennt. Für diese Konten darf es ihn nicht geben: an ihnen hängen
// die Aufzeichnungspflicht des § 4 Abs. 7 EStG, die Freigrenze je Empfänger und
// der Ausschluss der Vorsteuer. Wer 6610 von Hand wählt, bucht ein Geschenk ohne
// Empfänger und an der Freigrenze vorbei — und niemand sähe es je wieder.
//
// Die Liste wird aus dem Katalog gewonnen und nicht daneben geführt: eine neue
// Kategorie wäre sonst genau so lange ungeschützt, bis es jemandem auffällt.
func AccountsRequiringGroup() map[string]string {
	out := map[string]string{}
	for _, g := range postingGroups {
		if g.DeductibleQuota == "" && g.Limit == "" && !g.InputTaxExcluded && !g.RecipientRequired {
			continue
		}
		if g.Account != "" {
			out[g.Account] = g.Label
		}
		if g.NonDeductibleAccount != "" {
			out[g.NonDeductibleAccount] = g.Label
		}
	}
	return out
}

// LookupPostingGroup returns a group by key.
func LookupPostingGroup(key string) (PostingGroup, error) {
	g, ok := groupIndex[key]
	if !ok {
		return PostingGroup{}, fmt.Errorf("unbekannte Buchungsgruppe %q", key)
	}
	return g, nil
}
