package domain

import (
	"context"
	"fmt"
	"time"
)

// ClosingStepKey benennt einen Baustein des Jahresabschlusses.
//
// Der Abschluss ist kein Knopf, sondern eine Folge von Arbeiten, die
// aufeinander aufbauen: die Abgrenzung verschiebt Aufwand aus dem Jahr heraus,
// die Rückstellung bringt welchen hinein, und erst danach steht das Ergebnis,
// aus dem die Steuerrückstellung gerechnet wird. Ohne festgehaltenen Zustand
// wäre nach einer Unterbrechung nicht mehr erkennbar, was schon erledigt ist —
// und ein übersprungener Schritt ließe sich nicht von einem vergessenen
// unterscheiden.
type ClosingStepKey string

const (
	ClosingStepDepreciation ClosingStepKey = "depreciation" // AfA-Lauf
	// ClosingStepWriteUp ist die jährliche Frage nach der Wertaufholung: ist der
	// Grund einer früheren außerplanmäßigen Abschreibung weggefallen? Sie steht
	// direkt hinter der AfA, weil sie dieselbe Kartei betrifft.
	ClosingStepWriteUp ClosingStepKey = "write_up"
	// ClosingStepCurrencyValuation ist die Stichtagsbewertung der
	// Fremdwährungsposten (§ 256a HGB). Sie steht vor der Abgrenzung, weil sie
	// den Wert der offenen Posten ändert, auf denen alles Weitere aufsetzt.
	ClosingStepCurrencyValuation ClosingStepKey = "currency_valuation"
	ClosingStepAccruals          ClosingStepKey = "accruals"       // Rechnungsabgrenzung
	ClosingStepProvisions        ClosingStepKey = "provisions"     // Rückstellungen
	ClosingStepInventory         ClosingStepKey = "inventory"      // Inventurwert der Vorräte
	ClosingStepVatSettlement     ClosingStepKey = "vat_settlement" // Umsatzsteuer-Verrechnung
	// ClosingStepInputTaxCorrection ist die Vorsteuerberichtigung nach § 15a
	// UStG. Sie steht hinter der Umsatzsteuer-Verrechnung, weil sie eine weitere
	// Zeile in dieselbe Voranmeldung schreibt.
	ClosingStepInputTaxCorrection ClosingStepKey = "input_tax_correction"
	ClosingStepTaxProvision       ClosingStepKey = "tax_provision" // Steuerrückstellung
	ClosingStepCheckRun           ClosingStepKey = "check_run"     // Prüfbericht
	ClosingStepStatement          ClosingStepKey = "statement"     // Bilanz und GuV
	ClosingStepAdoption           ClosingStepKey = "adoption"      // Aufstellen und Feststellen
	ClosingStepDisclosure         ClosingStepKey = "disclosure"    // E-Bilanz und Offenlegung
	ClosingStepAppropriation      ClosingStepKey = "appropriation" // Saldenvortrag und Ergebnisverwendung
)

// ClosingStepState ist der Zustand eines Bausteins.
type ClosingStepState string

const (
	// ClosingStepOpen heißt: noch nicht bearbeitet.
	ClosingStepOpen ClosingStepState = "open"
	// ClosingStepDone heißt: erledigt — die Buchungen des Bausteins stehen oder
	// der Schritt wurde ausdrücklich abgehakt.
	ClosingStepDone ClosingStepState = "done"
	// ClosingStepSkipped heißt: bewusst übergangen. Nur mit Grund, weil ein
	// Abschluss ohne Rückstellungen eine Aussage ist und kein Versehen.
	ClosingStepSkipped ClosingStepState = "skipped"
)

// ClosingStepDefinition beschreibt einen Baustein unabhängig vom Jahr.
type ClosingStepDefinition struct {
	Key   ClosingStepKey `json:"key"`
	Order int            `json:"order"`
	Label string         `json:"label"`
	// Hint ist der Erklärtext der Oberfläche: was der Schritt tut und warum er
	// an dieser Stelle steht.
	Hint string `json:"hint"`
	// Automatic sagt, ob der Zustand aus den Daten folgt (AfA-Lauf, Bilanz,
	// Feststellung) oder ob ihn der Baustein selbst setzt.
	Automatic bool `json:"automatic"`
}

// AllClosingSteps liefert die Bausteine in der Reihenfolge, in der sie
// abgearbeitet werden. Die Reihenfolge ist fachlich und nicht kosmetisch: die
// Steuerrückstellung braucht das Ergebnis, und das Ergebnis steht erst, wenn
// Abgrenzung, Rückstellungen und Vorräte gebucht sind.
func AllClosingSteps() []ClosingStepDefinition {
	return []ClosingStepDefinition{
		{ClosingStepDepreciation, 1, "Abschreibungen",
			"Die planmäßige AfA des Jahres. Sie ist eine Abschlussbuchung und lässt sich später nicht nachholen.", true},
		{ClosingStepWriteUp, 2, "Wertaufholung prüfen",
			"Für jedes Anlagegut mit außerplanmäßiger Abschreibung: ist der Grund weggefallen? Dann ist " +
				"zuzuschreiben — § 253 Abs. 5 Satz 1 HGB macht daraus ein Gebot und kein Wahlrecht.", true},
		{ClosingStepCurrencyValuation, 3, "Fremdwährungsbewertung",
			"Offene Posten in Fremdwährung werden zum Stichtagskurs bewertet (§ 256a HGB). Bei einer " +
				"Restlaufzeit über einem Jahr wird nur der Verlust erfasst.", true},
		{ClosingStepAccruals, 4, "Rechnungsabgrenzung",
			"Ausgaben und Einnahmen, die wirtschaftlich ins nächste Jahr gehören (§ 250 HGB).", false},
		{ClosingStepProvisions, 5, "Rückstellungen",
			"Verpflichtungen, deren Höhe oder Fälligkeit noch offen ist (§ 249 HGB).", false},
		{ClosingStepInventory, 6, "Vorräte",
			"Der Inventurwert zum Stichtag und die Bestandsveränderung, die daraus folgt.", false},
		{ClosingStepVatSettlement, 7, "Umsatzsteuer-Verrechnung",
			"Vorsteuer, Umsatzsteuer und Vorauszahlungen werden zu einem Saldo verrechnet.", false},
		{ClosingStepInputTaxCorrection, 8, "Vorsteuerberichtigung § 15a",
			"Hat sich die Verwendung eines Wirtschaftsguts im Berichtigungszeitraum geändert, ist der " +
				"Vorsteuerabzug anteilig zu berichtigen. Der Betrag geht in die Kennziffer 64.", true},
		{ClosingStepTaxProvision, 9, "Steuerrückstellung",
			"Körperschaftsteuer, Solidaritätszuschlag und Gewerbesteuer auf das Ergebnis des Jahres.", false},
		{ClosingStepCheckRun, 10, "Prüfbericht",
			"Der Prüflauf über Buchungen, Belege und Fristen vor der Festschreibung.", true},
		{ClosingStepStatement, 11, "Bilanz und GuV",
			"Die Gliederung nach §§ 266 und 275 HGB samt Anhang.", true},
		{ClosingStepAdoption, 12, "Aufstellen und Feststellen",
			"Die Aufstellung durch die Geschäftsführung und der Beschluss der Gesellschafter.", true},
		{ClosingStepDisclosure, 13, "E-Bilanz und Offenlegung",
			"Die Übermittlung nach § 5b EStG und die Offenlegung nach § 325 HGB.", true},
		{ClosingStepAppropriation, 14, "Vortrag und Ergebnisverwendung",
			"Der Saldenvortrag ins Folgejahr und der Beschluss über das Ergebnis.", false},
	}
}

// ClosingStepDefinitionFor sucht die Beschreibung zu einem Schlüssel.
func ClosingStepDefinitionFor(key ClosingStepKey) (ClosingStepDefinition, bool) {
	for _, def := range AllClosingSteps() {
		if def.Key == key {
			return def, true
		}
	}
	return ClosingStepDefinition{}, false
}

// ClosingStep ist der gespeicherte Zustand eines Bausteins in einem Jahr.
//
// Eigene Tabelle statt eines JSON-Feldes am Geschäftsjahr: der Zustand trägt
// Grund und Zeitpunkt, und beides wird später ausgewertet — der Prüflauf führt
// die übersprungenen Schritte mit ihrem Grund als Hinweis auf (Regel
// closing_step_skipped), und ein Grund, der nur in einem JSON-Klumpen steht,
// ließe sich nicht abfragen.
type ClosingStep struct {
	Year  int              `gorm:"primaryKey;autoIncrement:false" json:"year"`
	Key   ClosingStepKey   `gorm:"primaryKey;size:30" json:"key"`
	State ClosingStepState `gorm:"size:20;not null;default:'open'" json:"state"`
	// Reason ist bei „übersprungen" Pflicht.
	Reason    string    `gorm:"size:500;serializer:encrypted" json:"reason,omitempty"`
	ChangedOn string    `gorm:"size:10" json:"changedOn,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Validate prüft den Zustand eines Bausteins.
func (s *ClosingStep) Validate() error {
	if s.Year <= 0 {
		return fmt.Errorf("zum Abschlussschritt gehört ein Geschäftsjahr")
	}
	if _, ok := ClosingStepDefinitionFor(s.Key); !ok {
		return fmt.Errorf("unbekannter Abschlussschritt %q", s.Key)
	}
	switch s.State {
	case ClosingStepOpen, ClosingStepDone:
	case ClosingStepSkipped:
		if s.Reason == "" {
			return fmt.Errorf("einen Abschlussschritt zu überspringen verlangt eine Begründung")
		}
	default:
		return fmt.Errorf("unbekannter Zustand %q", s.State)
	}
	return nil
}

// ClosingStepRepository persistiert die Bausteinzustände.
type ClosingStepRepository interface {
	FindByYear(ctx context.Context, year int) ([]ClosingStep, error)
	Save(ctx context.Context, step *ClosingStep) error
}
