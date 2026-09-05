package accounting

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Abschreibungsregeln als Ressource.
//
// Sie standen bisher als Go-Literale in afa.go. Der Gesetzgeber öffnet und
// schließt die degressive Abschreibung im Abstand weniger Jahre, führt Staffeln
// für einzelne Wirtschaftsgüter ein und ändert die Wertgrenzen — jedes Mal war
// das eine Codeänderung, die niemand außerhalb des Quelltextes nachlesen konnte.
// Als eingebettete Datei ist die Regelmenge sichtbar, versioniert und von den
// Tests lesbar: sie prüfen gegen dieselbe Datei, aus der gerechnet wird, und
// nicht gegen eine zweite Abschrift der Zahlen.
//
// Eingebettet und nicht daneben gelegt: Buchfink ist eine einzelne Datei, und
// eine Regelmenge, die zur Laufzeit fehlen kann, wäre eine Abschreibung, die
// beim Kunden nicht rechnet.

//go:embed afa_rules.json
var afaRulesJSON []byte

// AfARules ist die eingebettete Regelmenge.
type AfARules struct {
	Version string `json:"version"`
	Source  string `json:"source"`
	Note    string `json:"note"`

	// InvestmentDeductionNote erklärt, warum der Investitionsabzugsbetrag des
	// § 7g Abs. 1 EStG in Buchfink nicht vorkommt.
	//
	// Er fehlt nicht, er gehört nicht hierher: der Abzug wird außerhalb der
	// Bilanz vorgenommen und in der Steuererklärung geltend gemacht, ebenso wie
	// seine Hinzurechnung bei der Anschaffung (§ 7g Abs. 2 EStG). Ohne diesen
	// Satz sähe die Maske aus, als kenne Buchfink den § 7g nur zur Hälfte — die
	// Sonderabschreibung des Absatzes 5 steht dort ja.
	InvestmentDeductionNote string `json:"investmentDeductionNote"`

	ParameterSets     []AfAParameters         `json:"parameterSets"`
	DegressiveWindows []DegressiveWindow      `json:"degressiveWindows"`
	ElectricVehicle   []ElectricVehicleWindow `json:"electricVehicleWindows"`
	BuildingRates     []BuildingRate          `json:"buildingRates"`
}

// ElectricVehicleWindow ist die Staffel des § 7 Abs. 2a EStG für ein
// Anschaffungsfenster.
//
// Sie ist keine Methode mit einem Satz, sondern eine Folge fester Prozentsätze
// der Anschaffungskosten: 75 % im Jahr der Anschaffung, dann 10, 5, 5, 3 und
// 2 %. Deshalb steht sie als Liste und nicht als Faktor — ein Faktor käme auf
// diese Zahlen nie.
type ElectricVehicleWindow struct {
	From  string `json:"from"`
	Until string `json:"until"`
	// PermillePerYear sind die Sätze in Promille der Anschaffungskosten, Jahr
	// für Jahr ab dem Anschaffungsjahr.
	PermillePerYear []int64 `json:"permillePerYear"`
	Source          string  `json:"source"`
	Note            string  `json:"note"`
}

// BuildingRate ist ein fester AfA-Satz des § 7 Abs. 4 EStG.
//
// Gebäude folgen nicht den AfA-Tabellen und keiner geschätzten Nutzungsdauer,
// sondern festen Prozentsätzen. Welcher gilt, hängt an zwei Merkmalen: ob das
// Gebäude Wohnzwecken dient und an einem Stichtag — dem Bauantrag beim
// Betriebsgebäude, der Fertigstellung beim Wohngebäude.
type BuildingRate struct {
	Key         string `json:"key"`
	Residential bool   `json:"residential"`
	// ReferenceFrom ist der früheste Stichtag, für den dieser Satz gilt. Leer
	// heißt: kein unteres Ende — der Eintrag fängt auf, was keiner der früheren
	// Einträge getroffen hat.
	ReferenceFrom string `json:"referenceFrom,omitempty"`
	Permille      int64  `json:"permille"`
	Source        string `json:"source"`
	Label         string `json:"label"`
	// Note benennt die Vereinfachung, wo der Eintrag eine trägt.
	Note string `json:"note,omitempty"`
}

var afaRules = mustLoadAfARules()

func mustLoadAfARules() AfARules {
	var rules AfARules
	if err := json.Unmarshal(afaRulesJSON, &rules); err != nil {
		panic(fmt.Sprintf("afa_rules.json ist nicht lesbar: %v", err))
	}
	if len(rules.ParameterSets) == 0 || len(rules.DegressiveWindows) == 0 {
		panic("afa_rules.json enthält keine Wertgrenzen oder keine Fenster der degressiven Abschreibung")
	}
	return rules
}

// AfARuleSet liefert die geladene Regelmenge — für die Oberfläche, die die Sätze
// anzeigt, und für die Tests, die gegen dieselbe Datei prüfen.
func AfARuleSet() AfARules {
	out := afaRules
	out.ParameterSets = append([]AfAParameters(nil), afaRules.ParameterSets...)
	out.DegressiveWindows = append([]DegressiveWindow(nil), afaRules.DegressiveWindows...)
	out.ElectricVehicle = append([]ElectricVehicleWindow(nil), afaRules.ElectricVehicle...)
	out.BuildingRates = append([]BuildingRate(nil), afaRules.BuildingRates...)
	return out
}

// ElectricVehicleWindowFor liefert die Staffel, in die ein Anschaffungsdatum
// fällt.
func ElectricVehicleWindowFor(acquisitionDate string) (ElectricVehicleWindow, bool) {
	for _, w := range afaRules.ElectricVehicle {
		if acquisitionDate >= w.From && acquisitionDate <= w.Until {
			return w, true
		}
	}
	return ElectricVehicleWindow{}, false
}

// BuildingRateFor liefert den festen Satz des § 7 Abs. 4 EStG.
//
// referenceDate ist der Stichtag, an dem der Satz hängt: der Bauantrag beim
// Betriebsgebäude, die Fertigstellung beim Wohngebäude. Er ist Pflicht, und ohne
// ihn kommt ein Fehler statt einer Näherung: das Anschaffungsdatum liegt zwar
// nie vor der Fertigstellung, aber der Satz steigt mit dem Stichtag — ein
// Betriebsgebäude mit Bauantrag vor dem 01.04.1985, das 2026 erworben wird,
// bekäme daraus 3 % statt der 2 %, die § 7 Abs. 4 Satz 1 Nr. 2 Buchst. b EStG
// ihm zubilligt.
func BuildingRateFor(residential bool, referenceDate string) (BuildingRate, error) {
	if referenceDate == "" {
		return BuildingRate{}, fmt.Errorf(
			"ohne Stichtag lässt sich der Gebäudesatz nicht bestimmen. § 7 Abs. 4 EStG knüpft ihn beim " +
				"Betriebsgebäude an den Bauantrag und beim Wohngebäude an die Fertigstellung")
	}
	for _, r := range afaRules.BuildingRates {
		if r.Residential != residential {
			continue
		}
		if r.ReferenceFrom != "" && referenceDate < r.ReferenceFrom {
			continue
		}
		return r, nil
	}
	return BuildingRate{}, fmt.Errorf(
		"für den %s ist kein Gebäudesatz hinterlegt", germanDate(referenceDate))
}

// electricVehicleSchedule rechnet die Staffel des § 7 Abs. 2a EStG aus.
//
// Der letzte Jahresbetrag wird nicht gerechnet, sondern als Rest genommen: sechs
// gerundete Prozentsätze summieren sich sonst an den Anschaffungskosten vorbei,
// und ein Fahrzeug bliebe mit ein paar Cent Restbuchwert stehen.
func electricVehicleSchedule(cost domain.Cents, window ElectricVehicleWindow) []domain.Cents {
	out := make([]domain.Cents, 0, len(window.PermillePerYear))
	remaining := cost
	for i, permille := range window.PermillePerYear {
		amount := domain.MulRound(cost, permille, 1000)
		if i == len(window.PermillePerYear)-1 || amount > remaining {
			amount = remaining
		}
		if amount < 0 {
			amount = 0
		}
		remaining -= amount
		out = append(out, amount)
	}
	return out
}
