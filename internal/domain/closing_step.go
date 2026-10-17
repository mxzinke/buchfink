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
	// ClosingStepReceivableAllowance ist die Wertberichtigung auf Forderungen
	// (§ 253 Abs. 4 HGB). Sie folgt der Fremdwährungsbewertung, weil sie auf
	// den Eurobeträgen der offenen Posten aufsetzt.
	ClosingStepReceivableAllowance ClosingStepKey = "receivable_allowance"
	// ClosingStepWorkInProgress sind die unfertigen Leistungen: am Stichtag
	// begonnene, nicht abgerechnete Projekte zu Herstellungskosten (§ 255
	// Abs. 2 HGB).
	ClosingStepWorkInProgress ClosingStepKey = "work_in_progress"
	ClosingStepAccruals       ClosingStepKey = "accruals"       // Rechnungsabgrenzung
	ClosingStepProvisions     ClosingStepKey = "provisions"     // Rückstellungen
	ClosingStepVatSettlement  ClosingStepKey = "vat_settlement" // Umsatzsteuer-Verrechnung
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
	// an dieser Stelle steht. Ohne Norm; die steht in Reference.
	Hint string `json:"hint"`
	// Automatic sagt, ob der Zustand aus den Daten folgt (AfA-Lauf, Bilanz,
	// Feststellung) oder ob ihn der Baustein selbst setzt.
	Automatic bool `json:"automatic"`
	// Reference nennt die Normen; die Oberfläche zeigt sie nur im Detaildialog.
	Reference string `json:"reference,omitempty"`
}

// AllClosingSteps liefert die Bausteine in der Reihenfolge, in der sie
// abgearbeitet werden. Die Reihenfolge ist fachlich und nicht kosmetisch: die
// Steuerrückstellung braucht das Ergebnis, und das Ergebnis steht erst, wenn
// Abgrenzung und Rückstellungen gebucht sind.
func AllClosingSteps() []ClosingStepDefinition {
	return []ClosingStepDefinition{
		{ClosingStepDepreciation, 1, "Abschreibungen",
			"Die planmäßige AfA des Jahres. Sie ist eine Abschlussbuchung und lässt sich später nicht nachholen.",
			true, "§ 253 Abs. 3 HGB, § 7 EStG"},
		{ClosingStepWriteUp, 2, "Wertaufholung prüfen",
			"Für jedes Anlagegut mit außerplanmäßiger Abschreibung: ist der Grund weggefallen? Dann ist " +
				"zuzuschreiben. Das ist ein Gebot und kein Wahlrecht.", true, "§ 253 Abs. 5 Satz 1 HGB"},
		{ClosingStepCurrencyValuation, 3, "Fremdwährungsbewertung",
			"Offene Posten in Fremdwährung werden zum Stichtagskurs bewertet. Bei einer Restlaufzeit über " +
				"einem Jahr wird nur der Verlust erfasst.", true, "§ 256a HGB"},
		{ClosingStepReceivableAllowance, 4, "Wertberichtigung auf Forderungen",
			"Forderungen mit Ausfallrisiko werden einzeln, die übrigen pauschal auf den niedrigeren Wert " +
				"berichtigt, netto: die Umsatzsteuer folgt erst der Uneinbringlichkeit.", false,
			"§ 253 Abs. 4 Satz 2 HGB, § 17 Abs. 2 Nr. 1 UStG"},
		{ClosingStepWorkInProgress, 5, "Unfertige Leistungen",
			"Begonnene, am Stichtag nicht abgerechnete Projekte werden zu Herstellungskosten aktiviert und " +
				"im Folgejahr aufgelöst; erhaltene Anzahlungen bleiben unter den Verbindlichkeiten oder werden " +
				"offen abgesetzt.", false, "§ 255 Abs. 2 HGB, § 268 Abs. 5 Satz 2 HGB"},
		{ClosingStepAccruals, 6, "Rechnungsabgrenzung",
			"Ausgaben und Einnahmen, die wirtschaftlich ins nächste Jahr gehören.", false, "§ 250 HGB"},
		{ClosingStepProvisions, 7, "Rückstellungen",
			"Verpflichtungen, deren Höhe oder Fälligkeit noch offen ist.", false, "§ 249 HGB"},
		{ClosingStepVatSettlement, 8, "Umsatzsteuer-Verrechnung",
			"Vorsteuer, Umsatzsteuer und Vorauszahlungen werden zu einem Saldo verrechnet.", false, ""},
		{ClosingStepInputTaxCorrection, 9, "Vorsteuerberichtigung",
			"Hat sich die Verwendung eines Wirtschaftsguts im Berichtigungszeitraum geändert, ist der " +
				"Vorsteuerabzug anteilig zu berichtigen. Der Betrag geht in die Kennziffer 64.", true, "§ 15a UStG"},
		{ClosingStepTaxProvision, 10, "Steuerrückstellung",
			"Körperschaftsteuer, Solidaritätszuschlag und Gewerbesteuer auf das Ergebnis des Jahres, " +
				"abzüglich Vorauszahlungen und anrechenbarer Kapitalertragsteuer; eine Überzahlung ist ein " +
				"Erstattungsanspruch.", false, "§ 249 Abs. 1 HGB"},
		{ClosingStepCheckRun, 11, "Prüfbericht",
			"Der Prüflauf über Buchungen, Belege und Fristen vor der Festschreibung.", true, "GoBD Rz. 100 ff."},
		{ClosingStepStatement, 12, "Bilanz und GuV",
			"Die Gliederung von Bilanz und Gewinn- und Verlustrechnung samt Anhang.", true, "§§ 266, 275 HGB"},
		{ClosingStepAdoption, 13, "Aufstellen und Feststellen",
			"Die Aufstellung durch die Geschäftsführung und der Beschluss der Gesellschafter.", true,
			"§ 264 Abs. 1 HGB, § 46 Nr. 1 GmbHG"},
		{ClosingStepDisclosure, 14, "E-Bilanz und Offenlegung",
			"Die Übermittlung der E-Bilanz an das Finanzamt und die Offenlegung im Unternehmensregister.", true,
			"§ 5b EStG, § 325 HGB"},
		{ClosingStepAppropriation, 15, "Vortrag und Ergebnisverwendung",
			"Der Saldenvortrag ins Folgejahr und der Beschluss über das Ergebnis.", false, "§ 29 GmbHG"},
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
// Eigene Tabelle statt eines JSON-Feldes am Geschäftsjahr: der Zustand hat
// Grund und Zeitpunkt, und beides wird später ausgewertet — der Prüflauf führt
// die übersprungenen Schritte mit ihrem Grund als Hinweis auf (Regel
// closing_step_skipped), und ein Grund, der nur in einem JSON-Klumpen steht,
// ließe sich nicht abfragen.
type ClosingStep struct {
	Year  int              `gorm:"primaryKey;autoIncrement:false" json:"year"`
	Key   ClosingStepKey   `gorm:"primaryKey;size:30" json:"key"`
	State ClosingStepState `gorm:"size:20;not null;default:'open'" json:"state"`
	// Reason ist bei „übersprungen" Pflicht.
	Reason    string    `gorm:"size:500" json:"reason,omitempty"`
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
