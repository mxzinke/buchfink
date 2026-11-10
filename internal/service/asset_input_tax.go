package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Vorsteuer der Zugangsbuchung.
//
// Das Verzeichnis nach § 15a UStG entsteht bei der Aktivierung, und es braucht
// zwei Zahlen, die das Anlagegut selbst nicht kennt: die Vorsteuer des Zugangs
// und den Anteil, mit dem sie gezogen wurde. Beide stehen in der Buchung, aus
// der das Anlagegut hervorgeht — sie dort zu lassen und den Anwender abtippen zu
// lassen, hieße, das Verzeichnis von seinem Gedächtnis abhängig zu machen. Ein
// Verzeichnis, das nur füllt, wer daran denkt, ist nach § 22 Abs. 4 UStG keines.
//
// Übernommen wird nur, wo die Zuordnung eindeutig ist. Wo sie es nicht ist —
// mehrere Steuersätze auf einem Beleg, ein geteilter Abzug neben anderen
// Positionen —, bleibt das Feld leer und der Anwender trägt die Vorsteuer ein:
// eine geratene Zahl im Verzeichnis wäre über zehn Jahre die schlechtere Lücke.

// acquisitionInputTax liefert die Vorsteuer, die aus einer Zugangsbuchung auf
// eine Anlagekontozeile entfällt, und den Anteil, mit dem sie gezogen wurde.
//
// amount ist der Betrag der gesuchten Zeile; null heißt: die einzige Zeile auf
// diesem Konto. Das dritte Ergebnis sagt, ob die Zuordnung eindeutig war.
func acquisitionInputTax(
	entry *domain.JournalEntry, account string, amount domain.Cents,
) (domain.Cents, int, bool) {
	if entry == nil || account == "" {
		return 0, 0, false
	}

	// 1. Die Zeile des Wirtschaftsguts.
	idx, only, matches := -1, -1, 0
	for i, l := range entry.Lines {
		if l.Side != domain.SideDebit || l.Account != account {
			continue
		}
		matches++
		if only < 0 {
			only = i
		}
		if idx < 0 && amount != 0 && l.Amount == amount {
			idx = i
		}
	}
	if idx < 0 && matches == 1 {
		idx = only
	}
	if idx < 0 {
		return 0, 0, false
	}
	line := entry.Lines[idx]

	permille := 1000
	switch share := line.InputTaxShare; {
	case share == domain.InputTaxExcluded:
		// Kein Abzug gezogen: die Steuer steckt im Aufwand, und wie hoch sie war,
		// sagt die Buchung nicht mehr.
		return 0, 0, false
	case share > 0 && share < 1000:
		permille = share
	}

	// 2. Die Vorsteuerzeile. Mehrere Steuersätze in einer Buchung lassen sich
	// nicht mehr auf die Positionen zurückrechnen.
	taxIdx := -1
	for i, l := range entry.Lines {
		if l.Side != domain.SideDebit || l.TaxBase <= 0 {
			continue
		}
		if !deductibleInputTaxKey(l.TaxKey) && !reverseChargeInputTaxKey(l.TaxKey) {
			continue
		}
		if taxIdx >= 0 {
			return 0, 0, false
		}
		taxIdx = i
	}
	if taxIdx < 0 {
		return 0, 0, false
	}
	tax := entry.Lines[taxIdx]

	// 3. Trägt die Buchung außer dem Wirtschaftsgut keine weitere Position, so
	// gehört ihm die ganze Bemessungsgrundlage — und nur dann lässt sich bei
	// einem geteilten Abzug die volle Vorsteuer hochrechnen.
	sole := true
	for i, l := range entry.Lines {
		if i == idx || i == taxIdx || l.Side != domain.SideDebit || l.TaxKey != "" {
			continue
		}
		sole = false
		break
	}

	switch {
	case permille == 1000:
		if line.Amount > tax.TaxBase {
			return 0, 0, false
		}
		return domain.MulRound(tax.Amount, int64(line.Amount), int64(tax.TaxBase)), permille, true
	case sole:
		// Bei geteiltem Abzug steht in der Steuerzeile nur der gezogene Teil; das
		// Verzeichnis führt die Vorsteuer in voller Höhe und den Anteil daneben.
		return domain.MulRound(tax.Amount, 1000, int64(permille)), permille, true
	default:
		return 0, 0, false
	}
}

// fillInputTaxFromAcquisition übernimmt Vorsteuer und Vorsteueranteil aus der
// Zugangsbuchung, wo der Aufrufer sie nicht gesetzt hat.
//
// Ein Fehlschlag bleibt still: das Anlagegut entsteht auch ohne die Zahl, und
// die Aktivierung an einer Journalabfrage scheitern zu lassen, wäre die
// schlechtere Antwort. Was fehlt, trägt der Anwender im Verzeichnis nach.
func (s *AssetService) fillInputTaxFromAcquisition(ctx context.Context, asset *domain.FixedAsset) {
	if asset == nil || asset.InputTaxAmount != 0 ||
		asset.AcquisitionEntryID == nil || s.journalRepo == nil {
		return
	}
	entry, err := s.journalRepo.FindByID(ctx, *asset.AcquisitionEntryID)
	if err != nil || entry == nil {
		return
	}
	tax, permille, ok := acquisitionInputTax(entry, asset.Account, asset.AcquisitionCost)
	if !ok || tax <= 0 {
		return
	}
	asset.InputTaxAmount = tax
	if asset.InputTaxPermille == 0 {
		asset.InputTaxPermille = permille
	}
}

// statedPermille macht aus dem Anteil eines Anlageguts eine Angabe oder keine.
//
// Am Anlagegut ist die Null der unausgefüllte Wert: die Erfassungsmaske fragt
// den Anteil nicht ab, und ein leeres Feld heißt dort „nicht angegeben" und
// nicht „null Promille". Das Verzeichnis unterscheidet beides, deshalb wird
// hier übersetzt statt durchgereicht.
func statedPermille(permille int) *int {
	if permille <= 0 {
		return nil
	}
	return &permille
}

// yearsAfter liefert den letzten Tag des Zeitraums von n Jahren ab einem Datum.
func yearsAfter(date string, years int) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", fmt.Errorf("%q ist kein Datum im Format JJJJ-MM-TT", date)
	}
	return t.AddDate(years, 0, 0).AddDate(0, 0, -1).Format("2006-01-02"), nil
}

// -------------------------------------------------------------------------
// Aufnahme ins Verzeichnis nach § 15a UStG bei der Aktivierung
// -------------------------------------------------------------------------

// inputTaxRegistrar ist der Ausschnitt des Verzeichnisses nach § 15a UStG, den
// die Anlagenbuchhaltung braucht.
type inputTaxRegistrar interface {
	Register(ctx context.Context, req RegisterInputTaxRequest) (*domain.InputTaxCorrection, error)
}

// SetInputTaxRegister koppelt das Verzeichnis nach § 15a UStG an die
// Anlagenbuchhaltung.
//
// Ohne diese Kopplung landete ein Pkw mit 7.600 € Vorsteuer nur dann im
// Verzeichnis, wenn der Anwender ihn von Hand nachtrug — und ein Verzeichnis,
// das nur füllt, wer daran denkt, ist nach § 22 Abs. 4 UStG keines.
func (s *AssetService) SetInputTaxRegister(r inputTaxRegistrar) { s.inputTax = r }

// registerInputTaxCorrection nimmt ein aktiviertes Wirtschaftsgut ins
// Verzeichnis nach § 15a UStG auf.
//
// Die Schwelle ist die des § 44 Abs. 1 UStDV: ab 1.000 € Vorsteuer je
// Wirtschaftsgut. Grundstücke und Gebäude kommen unabhängig davon hinein — bei
// ihnen läuft der Berichtigungszeitraum zehn Jahre, und wer nach zehn Jahren
// nach dem ursprünglichen Verwendungsanteil gefragt wird, findet ihn nur dort.
//
// Ein Fehler beim Eintrag reißt die Aktivierung nicht mit: das Anlagegut steht
// dann in der Kartei und fehlt im Verzeichnis, und das ist der bessere der
// beiden Zustände — die Meldung sagt, was nachzutragen ist.
func (s *AssetService) registerInputTaxCorrection(ctx context.Context, asset *domain.FixedAsset) error {
	if s.inputTax == nil || asset == nil {
		return nil
	}
	// Ohne gezogene Vorsteuer gibt es nichts zu berichtigen — beim Grundstück so
	// wenig wie beim Schreibtisch. § 15a UStG berichtigt den Abzug, den es
	// gegeben hat, und das Verzeichnis weist einen Eintrag über null Euro
	// Vorsteuer folgerichtig ab. Der Regelfall wäre sonst betroffen: der
	// Grundstückserwerb ist nach § 4 Nr. 9 Buchst. a UStG steuerfrei, ohne Option
	// nach § 9 UStG steht in der Rechnung keine Steuer, und ebenso kommt ein
	// Gebäude aus dem Saldenvortrag ohne Vorsteuer in die Kartei. Die Aktivierung
	// ist zu diesem Zeitpunkt schon gespeichert; ein Fehler hier ließe den
	// Anwender den Vorgang wiederholen und legte das Anlagegut ein zweites Mal
	// an. Die Zehnjahresregel gilt dem Grundstück *mit* Vorsteuer.
	if asset.InputTaxAmount <= 0 {
		return nil
	}
	entry, known := accounting.LookupAssetAccount(asset.Account)
	immovable := known && entry.Immovable
	params, err := accounting.TaxParametersFor(asset.AcquisitionDate)
	if err != nil {
		return err
	}
	if asset.InputTaxAmount <= params.InputTaxCorrectionFloor && !immovable {
		return nil
	}
	label := asset.Name
	if asset.InventoryNumber != "" {
		label = fmt.Sprintf("%s (%s)", asset.Name, asset.InventoryNumber)
	}
	_, err = s.inputTax.Register(ctx, RegisterInputTaxRequest{
		AssetID:          asset.ID,
		Label:            label,
		Account:          asset.Account,
		AcquisitionDate:  asset.AcquisitionDate,
		NetAmount:        asset.AcquisitionCost,
		InputTaxAmount:   asset.InputTaxAmount,
		OriginalPermille: statedPermille(asset.InputTaxPermille),
		Immovable:        immovable,
		Note:             "Bei der Aktivierung aufgenommen",
	})
	if err != nil {
		return fmt.Errorf(
			"%s wurde aktiviert, der Eintrag im Verzeichnis nach § 15a UStG aber nicht angelegt. "+
				"Er ist nachzutragen — ohne ihn fehlt der ursprüngliche Verwendungsanteil, und der ist "+
				"später aus keiner Buchung mehr zu gewinnen: %w", asset.InventoryNumber, err)
	}
	return nil
}
