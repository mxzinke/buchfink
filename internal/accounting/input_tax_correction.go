package accounting

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Vorsteuerberichtigung nach § 15a UStG.
//
// Der Vorsteuerabzug eines Wirtschaftsguts wird im Jahr der Anschaffung nach der
// beabsichtigten Verwendung gewährt. Ändert sich die Verwendung innerhalb des
// Berichtigungszeitraums, ist der Abzug für jedes betroffene Jahr anteilig zu
// berichtigen. Das ist keine Buchung, die sich aus dem Journal ergibt: sie
// braucht den ursprünglichen Anteil, den heutigen Anteil und den Zeitraum — drei
// Angaben, die nur ein eigenes Verzeichnis über die Jahre trägt.

// TaxKeyInputTaxCorrection ist der Steuerschlüssel der Berichtigung. Er läuft in
// die Kennziffer 64 des Vordrucks USt 1 A und in keine andere: die berichtigte
// Vorsteuer darf nicht in Kz 66 landen, wo sie als Vorsteuer aus Rechnungen
// dieses Zeitraums gelesen würde.
const TaxKeyInputTaxCorrection = "VST15A"

// Die vier Konten, die der SKR04 für § 15a Abs. 1 UStG führt. Er trennt sie nach
// zwei Merkmalen, und beide sind fachlich: die Richtung der Berichtigung
// (nachträglich abziehbar oder zurückzuzahlen) und die Beweglichkeit des
// Wirtschaftsguts, an der auch die Länge des Zeitraums hängt.
const (
	InputTaxCorrectionDeductibleMovable   = "1396"
	InputTaxCorrectionRepayableMovable    = "1397"
	InputTaxCorrectionDeductibleImmovable = "1398"
	InputTaxCorrectionRepayableImmovable  = "1399"
)

// Die beiden Berichtigungszeiträume des § 15a Abs. 1 UStG: fünf Jahre für
// bewegliche Wirtschaftsgüter, zehn für Grundstücke und Gebäude.
const (
	CorrectionPeriodMovableYears   = 5
	CorrectionPeriodImmovableYears = 10
)

// CorrectionPeriodYears liefert den Berichtigungszeitraum eines Wirtschaftsguts.
func CorrectionPeriodYears(immovable bool) int {
	if immovable {
		return CorrectionPeriodImmovableYears
	}
	return CorrectionPeriodMovableYears
}

// InputTaxCorrectionAccount liefert das Konto der Berichtigung.
//
// refund heißt: die Vorsteuer ist zurückzuzahlen, weil die abziehbare Verwendung
// zurückgegangen ist.
func InputTaxCorrectionAccount(immovable, refund bool) string {
	switch {
	case immovable && refund:
		return InputTaxCorrectionRepayableImmovable
	case immovable:
		return InputTaxCorrectionDeductibleImmovable
	case refund:
		return InputTaxCorrectionRepayableMovable
	default:
		return InputTaxCorrectionDeductibleMovable
	}
}

// InputTaxCorrectionRequest ist ein Berichtigungsjahr eines Wirtschaftsguts.
type InputTaxCorrectionRequest struct {
	// InputTaxAmount ist die beim Zugang angefallene Vorsteuer in voller Höhe —
	// also vor dem Abzug des Verwendungsanteils. § 44 Abs. 1 UStDV misst seine
	// Bagatellgrenze an ihr und nicht am tatsächlich gezogenen Betrag.
	InputTaxAmount domain.Cents
	// OriginalPermille ist der Anteil, mit dem die Vorsteuer beim Zugang gezogen
	// wurde, CurrentPermille der Anteil des Berichtigungsjahres — beide in
	// Promille, 1000 = volle Verwendung für abziehbare Umsätze.
	OriginalPermille int64
	CurrentPermille  int64
	// PeriodYears ist der Berichtigungszeitraum in Jahren.
	PeriodYears int
	// Immovable entscheidet über das Konto.
	Immovable bool
}

// InputTaxCorrectionAssessment ist das Ergebnis eines Berichtigungsjahres.
type InputTaxCorrectionAssessment struct {
	// Amount ist der Berichtigungsbetrag mit Vorzeichen: positiv, wo nachträglich
	// Vorsteuer abziehbar wird, negativ, wo sie zurückzuzahlen ist. Das Vorzeichen
	// ist die Aussage — ein Betrag ohne es ließe sich nicht buchen.
	Amount domain.Cents `json:"amount"`
	// Required sagt, ob überhaupt zu berichtigen ist. Ist es null, nennt Reason
	// die Bagatellgrenze, an der es liegt.
	Required bool `json:"required"`
	// DeferToAnnual sagt, dass die Berichtigung nach § 44 Abs. 3 UStDV erst bei
	// der Steuerberechnung für das Kalenderjahr vorzunehmen ist — sie entfällt
	// nicht, sie wandert ans Jahresende.
	DeferToAnnual bool `json:"deferToAnnual"`
	// Account ist das Berichtigungskonto, leer wo nicht zu berichtigen ist.
	Account string `json:"account,omitempty"`
	// Reason ist ein Satz in Klartext: warum berichtigt wird oder warum nicht.
	Reason string `json:"reason"`
}

// AssessInputTaxCorrection rechnet ein Berichtigungsjahr und wendet die
// Bagatellgrenzen des § 44 UStDV an.
//
// Die Reihenfolge der Prüfungen ist die des Gesetzes und nicht beliebig:
// Absatz 1 nimmt das Wirtschaftsgut ganz aus der Berichtigung, Absatz 2 nur das
// einzelne Jahr, und Absatz 3 verschiebt bloß den Zeitpunkt. Wer mit Absatz 2
// begänne, berichtigte ein Wirtschaftsgut, das nach Absatz 1 gar nicht ins
// Verzeichnis gehört.
func AssessInputTaxCorrection(
	req InputTaxCorrectionRequest, params TaxParameters,
) (InputTaxCorrectionAssessment, error) {
	if req.PeriodYears <= 0 {
		return InputTaxCorrectionAssessment{}, fmt.Errorf(
			"ohne Berichtigungszeitraum lässt sich der Anteil eines Jahres nicht bestimmen")
	}
	if req.InputTaxAmount < 0 {
		return InputTaxCorrectionAssessment{}, fmt.Errorf("die Vorsteuer kann nicht negativ sein")
	}
	if err := validatePermille(req.OriginalPermille, "der ursprüngliche Verwendungsanteil"); err != nil {
		return InputTaxCorrectionAssessment{}, err
	}
	if err := validatePermille(req.CurrentPermille, "der Verwendungsanteil des Jahres"); err != nil {
		return InputTaxCorrectionAssessment{}, err
	}

	if req.InputTaxAmount <= params.InputTaxCorrectionFloor {
		return InputTaxCorrectionAssessment{Reason: fmt.Sprintf(
			"Die Vorsteuer beträgt %s € und damit höchstens %s €. Nach § 44 Abs. 1 UStDV entfällt die "+
				"Berichtigung für dieses Wirtschaftsgut ganz.",
			req.InputTaxAmount, params.InputTaxCorrectionFloor)}, nil
	}

	// Der Berichtigungsbetrag: die Vorsteuer eines Jahres ist ihr Anteil am
	// Zeitraum, und berichtigt wird die Differenz der Verwendungsanteile.
	deltaPermille := req.CurrentPermille - req.OriginalPermille
	yearShare := domain.MulRound(req.InputTaxAmount, 1, int64(req.PeriodYears))
	amount := domain.MulRound(yearShare, deltaPermille, 1000)

	if amount == 0 {
		return InputTaxCorrectionAssessment{Reason: fmt.Sprintf(
			"Der Verwendungsanteil ist mit %s unverändert. Es ist nichts zu berichtigen.",
			PermilleLabel(req.CurrentPermille))}, nil
	}

	points := deltaPermille / 10
	if points < 0 {
		points = -points
	}
	magnitude := amount
	if magnitude < 0 {
		magnitude = -magnitude
	}
	// § 44 Abs. 2 UStDV: beide Bedingungen müssen zusammentreffen. Eine große
	// Änderung mit kleinem Betrag wird ebenso berichtigt wie eine kleine
	// Änderung mit großem Betrag.
	if points < params.InputTaxCorrectionMinorPoints && magnitude <= params.InputTaxCorrectionMinorAmount {
		return InputTaxCorrectionAssessment{Reason: fmt.Sprintf(
			"Die Verwendung hat sich um %d Prozentpunkte geändert und der Betrag läge bei %s €. "+
				"Nach § 44 Abs. 2 UStDV entfällt die Berichtigung für dieses Jahr: weniger als %d "+
				"Prozentpunkte und höchstens %s €.",
			points, magnitude, params.InputTaxCorrectionMinorPoints, params.InputTaxCorrectionMinorAmount)}, nil
	}

	out := InputTaxCorrectionAssessment{
		Amount:   amount,
		Required: true,
		Account:  InputTaxCorrectionAccount(req.Immovable, amount < 0),
	}
	direction := "nachträglich abziehbar"
	if amount < 0 {
		direction = "zurückzuzahlen"
	}
	out.Reason = fmt.Sprintf(
		"Der Verwendungsanteil hat sich von %s auf %s geändert. Auf ein Jahr des %d-jährigen "+
			"Berichtigungszeitraums entfallen %s € Vorsteuer; davon sind %s € %s (§ 15a Abs. 1 UStG).",
		PermilleLabel(req.OriginalPermille), PermilleLabel(req.CurrentPermille), req.PeriodYears,
		yearShare, magnitude, direction)

	// § 44 Abs. 3 UStDV: bis 6.000 € wird nicht im Voranmeldungszeitraum
	// berichtigt, sondern erst bei der Steuerberechnung für das Kalenderjahr.
	// Buchfink bucht sie deshalb zum Ende des Wirtschaftsjahres — dort landet sie
	// in der Kennziffer 64 der letzten Voranmeldung des Jahres.
	if magnitude <= params.InputTaxCorrectionAnnualAmount {
		out.DeferToAnnual = true
		out.Reason += fmt.Sprintf(
			" Der Betrag bleibt unter %s €: nach § 44 Abs. 3 UStDV wird erst bei der Steuerberechnung "+
				"für das Kalenderjahr berichtigt, also zum Ende des Wirtschaftsjahres.",
			params.InputTaxCorrectionAnnualAmount)
	}
	return out, nil
}

// validatePermille weist einen Anteil ab, der kein Anteil ist.
func validatePermille(permille int64, what string) error {
	if permille < 0 || permille > 1000 {
		return fmt.Errorf("%s liegt zwischen 0 und 100 %% (angegeben: %s)", what, PermilleLabel(permille))
	}
	return nil
}

// PermilleLabel rendert einen Promilleanteil als Prozentangabe.
func PermilleLabel(permille int64) string {
	return permilleLabel(permille, 1000)
}
