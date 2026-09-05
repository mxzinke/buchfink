package accounting

import (
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

// TaxKeyInfo beschreibt einen Steuerschlüssel im Klartext.
//
// Der Schlüssel steht an jeder Steuerzeile des Journals. Ohne ein Verzeichnis
// dazu ist er eine Zeichenkette, die nur Buchfink versteht — und die
// Datenüberlassung verlangt, dass sich jede Verschlüsselung im Klartext auflösen
// lässt (GoBD Rz. 95, § 147 Abs. 6 AO).
type TaxKeyInfo struct {
	Key       string              `json:"key"`
	Label     string              `json:"label"`
	Account   string              `json:"account"`
	Direction domain.Direction    `json:"direction"`
	Treatment domain.TaxTreatment `json:"treatment"`
	Rate      domain.TaxRate      `json:"rate"`
	Side      domain.Side         `json:"side"`
	// VatCode ist die Kennziffer des Vordrucks USt 1 A, in die der Betrag
	// eingeht. Leer, wo der Schlüssel in keine Kennziffer läuft.
	VatCode string `json:"vatCode"`
	// VatCodeLabel ist die Zeile des Vordrucks im Klartext.
	VatCodeLabel string `json:"vatCodeLabel"`
}

// vatCodeForTaxKey ordnet jedem Steuerschlüssel seine Kennziffer zu.
//
// Die Zuordnung ist dieselbe wie in vatMovements. Sie steht hier ein zweites
// Mal, weil dort über Buchungen gelaufen wird und hier über den Katalog; der
// Test TestTaxKeyCatalogMatchesVatReturn hält beide zusammen, damit ein neuer
// Schlüssel nicht in der einen Liste auftaucht und in der anderen fehlt.
var vatCodeForTaxKey = map[string]string{
	"UST19":        VatCodeStandardRate,
	"UST7":         VatCodeReducedRate,
	"VST19":        VatCodeInputTax,
	"VST7":         VatCodeInputTax,
	"IG19_UST":     VatCodeAcquisition19,
	"IG7_UST":      VatCodeAcquisition7,
	"IG19_VST":     VatCodeInputTaxAcquisition,
	"IG7_VST":      VatCodeInputTaxAcquisition,
	"RC19_UST":     VatCodeReverseCharge,
	"RC7_UST":      VatCodeReverseCharge,
	"RC19_VST":     VatCodeInputTaxReverse,
	"RC7_VST":      VatCodeInputTaxReverse,
	TaxKeyUnlawful: VatCodeUnlawfulTax,
}

// taxKeyLabels sind die Klartexte der Schlüssel.
var taxKeyLabels = map[string]string{
	"UST19":        "Umsatzsteuer 19 % auf einen steuerpflichtigen Inlandsumsatz",
	"UST7":         "Umsatzsteuer 7 % auf einen steuerpflichtigen Inlandsumsatz",
	"VST19":        "Vorsteuer 19 % aus der Rechnung eines anderen Unternehmers",
	"VST7":         "Vorsteuer 7 % aus der Rechnung eines anderen Unternehmers",
	"IG19_UST":     "Erwerbsteuer 19 % auf einen innergemeinschaftlichen Erwerb",
	"IG7_UST":      "Erwerbsteuer 7 % auf einen innergemeinschaftlichen Erwerb",
	"IG19_VST":     "Vorsteuer 19 % aus dem innergemeinschaftlichen Erwerb",
	"IG7_VST":      "Vorsteuer 7 % aus dem innergemeinschaftlichen Erwerb",
	"RC19_UST":     "Geschuldete Steuer 19 % als Leistungsempfänger nach § 13b UStG",
	"RC7_UST":      "Geschuldete Steuer 7 % als Leistungsempfänger nach § 13b UStG",
	"RC19_VST":     "Vorsteuer 19 % aus der Leistung nach § 13b UStG",
	"RC7_VST":      "Vorsteuer 7 % aus der Leistung nach § 13b UStG",
	TaxKeyUnlawful: "Unrichtig oder unberechtigt ausgewiesene Steuer nach § 14c UStG",
}

// TaxKeyCatalog listet jeden Steuerschlüssel, den Buchfink erzeugen kann.
//
// Er wird nicht von Hand gepflegt, sondern aus dem Steuerrechner gewonnen: für
// jede Kombination aus Richtung, Steuerfall und Satz wird gefragt, welche
// Steuerzeilen entstehen. Eine Liste, die daneben geführt würde, verlöre beim
// nächsten neuen Steuerfall den Anschluss — und niemand merkte es, weil sie
// nirgends gegen die Wirklichkeit gehalten wird.
func TaxKeyCatalog() []TaxKeyInfo {
	resolver := NewSKR04TaxResolver()
	// Ein Musterbetrag, aus dem sich in jedem Fall eine Steuerzeile ergibt. Er
	// dient nur dazu, den Rechner laufen zu lassen; ausgewertet werden Konto,
	// Seite und Schlüssel, nicht der Betrag.
	const sample domain.Cents = 100_00

	seen := map[string]TaxKeyInfo{}
	for _, dir := range []domain.Direction{domain.DirectionIncoming, domain.DirectionOutgoing} {
		for _, treatment := range domain.AllTaxTreatments() {
			for _, rate := range []domain.TaxRate{domain.TaxRateReduced, domain.TaxRateStandard} {
				legs, err := resolver.Resolve(dir, treatment, rate, sample)
				if err != nil {
					continue
				}
				for _, leg := range legs {
					if _, ok := seen[leg.Key]; ok {
						continue
					}
					seen[leg.Key] = TaxKeyInfo{
						Key:       leg.Key,
						Label:     taxKeyLabels[leg.Key],
						Account:   leg.Account,
						Direction: dir,
						Treatment: treatment,
						Rate:      rate,
						Side:      leg.Side,
						VatCode:   vatCodeForTaxKey[leg.Key],
					}
				}
			}
		}
	}

	// Der § 14c-Schlüssel entsteht aus keinem Steuerfall — er ist der Weg, eine
	// Steuer zu buchen, die ohne steuerpflichtigen Umsatz geschuldet wird.
	seen[TaxKeyUnlawful] = TaxKeyInfo{
		Key:       TaxKeyUnlawful,
		Label:     taxKeyLabels[TaxKeyUnlawful],
		Account:   domain.AccountUmsatzsteuer14c,
		Direction: domain.DirectionOutgoing,
		Side:      domain.SideCredit,
		VatCode:   vatCodeForTaxKey[TaxKeyUnlawful],
	}

	out := make([]TaxKeyInfo, 0, len(seen))
	for _, info := range seen {
		info.VatCodeLabel = VatCodeLabel(info.VatCode)
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// VatCodeLabel liefert die Zeile des Vordrucks zu einer Kennziffer.
func VatCodeLabel(code string) string {
	if code == "" {
		return ""
	}
	for _, spec := range vatCodes {
		if spec.code == code {
			return spec.label
		}
		if spec.taxCode == code {
			return spec.label + " (Steuerbetrag)"
		}
	}
	return ""
}
