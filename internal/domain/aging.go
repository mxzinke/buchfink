package domain

// AgingBucketKey benennt ein Band der Altersstruktur.
type AgingBucketKey string

const (
	AgingNotDue      AgingBucketKey = "not_due"
	AgingUpTo30      AgingBucketKey = "1_30"
	AgingUpTo60      AgingBucketKey = "31_60"
	AgingUpTo90      AgingBucketKey = "61_90"
	AgingOver90      AgingBucketKey = "over_90"
	AgingWithoutDate AgingBucketKey = "no_due_date"
)

// AgingBucket ist ein Band der Altersstruktur mit seinen Posten.
type AgingBucket struct {
	Key    AgingBucketKey `json:"key"`
	Label  string         `json:"label"`
	Amount Cents          `json:"amount"`
	Items  int            `json:"items"`
}

// MaturityBandKey benennt ein Band der Restlaufzeit.
type MaturityBandKey string

const (
	MaturityUpToOneYear   MaturityBandKey = "up_to_1y"
	MaturityOneToFive     MaturityBandKey = "1_to_5y"
	MaturityOverFiveYears MaturityBandKey = "over_5y"
	MaturityUndated       MaturityBandKey = "undated"
)

// MaturityBand ist ein Band der Restlaufzeit mit seinen Posten.
type MaturityBand struct {
	Key    MaturityBandKey `json:"key"`
	Label  string          `json:"label"`
	Amount Cents           `json:"amount"`
	Items  int             `json:"items"`
}

// OpenItemsAgingSide ist die Auswertung einer Seite — Forderungen oder
// Verbindlichkeiten.
type OpenItemsAgingSide struct {
	// Side ist "receivables" oder "payables".
	Side       string         `json:"side"`
	Label      string         `json:"label"`
	Total      Cents          `json:"total"`
	Items      int            `json:"items"`
	Buckets    []AgingBucket  `json:"buckets"`
	Maturities []MaturityBand `json:"maturities"`
}

// OpenItemsAging ist die Altersstruktur und die Restlaufzeitengliederung der
// offenen Posten zu einem Stichtag.
//
// Beide zusammen, weil beide dieselbe Grundlage haben und verschiedene Fragen
// beantworten: die Altersstruktur sagt, wie lange ein Posten schon überfällig
// ist — das ist eine Frage des Mahnwesens und der Wertberichtigung —, die
// Restlaufzeit sagt, wann er fällig wird, und ist die Angabe unter der Bilanz
// (§ 268 Abs. 4 und 5 HGB). Zwei getrennte Abfragen über dieselben Posten
// könnten auseinanderlaufen.
type OpenItemsAging struct {
	Cutoff string               `json:"cutoff"`
	Sides  []OpenItemsAgingSide `json:"sides"`
	// Reference nennt die Norm der Restlaufzeitengliederung.
	Reference string `json:"reference"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (a *OpenItemsAging) EnsureLists() {
	if a.Sides == nil {
		a.Sides = make([]OpenItemsAgingSide, 0)
	}
	for i := range a.Sides {
		if a.Sides[i].Buckets == nil {
			a.Sides[i].Buckets = make([]AgingBucket, 0)
		}
		if a.Sides[i].Maturities == nil {
			a.Sides[i].Maturities = make([]MaturityBand, 0)
		}
	}
}
