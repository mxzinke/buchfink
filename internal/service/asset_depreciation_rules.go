package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Anlagenthemen der Welle 5c: die Sperren, die sich nach dem Anlagekonto
// richten, der Wertaufholungsbericht und die Einheitlichkeit des
// Sammelposten-Wahlrechts.

// validateMethodForAccount hält die Abschreibungsverfahren an den Merkmalen des
// Anlagekontos fest.
//
// Ob ein Wirtschaftsgut beweglich ist, steht im Kontenkatalog, nicht am
// Anlagegut — und danach richten sich drei Regeln, die bisher nur zum Teil
// durchgesetzt waren: die degressive AfA des § 7 Abs. 2 EStG gilt für bewegliche
// Wirtschaftsgüter, die festen Sätze des § 7 Abs. 4 EStG für Gebäude, und die
// Staffel des § 7 Abs. 2a EStG für neue Elektrofahrzeuge. Ohne diese Prüfung
// ließ sich ein Bürogebäude degressiv abschreiben, und der Plan sah plausibel
// aus.
func validateMethodForAccount(asset *domain.FixedAsset) error {
	entry, known := accounting.LookupAssetAccount(asset.Account)

	switch asset.Method {
	case domain.DepreciationDegressive:
		if known && entry.Immovable {
			return fmt.Errorf(
				"%s (%s) ist ein unbewegliches Wirtschaftsgut. Die degressive Abschreibung des "+
					"§ 7 Abs. 2 EStG gibt es nur für bewegliche Wirtschaftsgüter des Anlagevermögens — "+
					"für Gebäude gelten die festen Sätze des § 7 Abs. 4 EStG",
				asset.Account, entry.Name)
		}
		if asset.Class == domain.AssetClassIntangible {
			return fmt.Errorf(
				"die degressive Abschreibung des § 7 Abs. 2 EStG gilt für bewegliche Wirtschaftsgüter " +
					"des Sachanlagevermögens; ein immaterieller Vermögensgegenstand ist keines")
		}
	case domain.DepreciationBuildingLinear:
		if !known {
			return fmt.Errorf(
				"die festen Sätze des § 7 Abs. 4 EStG gelten für Gebäude. Das Konto %s steht nicht im "+
					"Katalog der Anlagekonten — nötig ist eines der Gebäudekonten", asset.Account)
		}
		if !entry.Immovable {
			return fmt.Errorf(
				"%s (%s) ist kein Gebäude. Die festen Sätze des § 7 Abs. 4 EStG gelten für Gebäude; "+
					"alles andere wird linear über seine betriebsgewöhnliche Nutzungsdauer abgeschrieben",
				asset.Account, entry.Name)
		}
		if _, err := accounting.BuildingRateFor(
			entry.Residential, asset.BuildingReferenceDate); err != nil {
			return err
		}
	case domain.DepreciationElectricVehicle:
		if known && entry.Group != "Fahrzeuge" {
			return fmt.Errorf(
				"%s (%s) ist kein Fahrzeug. § 7 Abs. 2a EStG gilt für neue, rein elektrisch "+
					"betriebene Fahrzeuge — sie gehören auf ein Fahrzeugkonto",
				asset.Account, entry.Name)
		}
		if _, ok := accounting.ElectricVehicleWindowFor(asset.DepreciationStart()); !ok {
			return fmt.Errorf(
				"für eine Anschaffung am %s gibt es die Staffel des § 7 Abs. 2a EStG nicht. Sie gilt "+
					"für Anschaffungen nach dem 30.06.2025 und vor dem 01.01.2028",
				asset.DepreciationStart())
		}
	}
	return nil
}

// checkUsefulLifeReason verlangt eine Begründung, wo die Nutzungsdauer vom
// Vorschlag des BMF-Schreibens vom 22.02.2022 abweicht.
//
// Nur dort und nicht bei jedem Konto mit einem Vorschlag: das Schreiben zur
// einjährigen Nutzungsdauer digitaler Wirtschaftsgüter ist die eine Stelle, an
// der Buchfink einen Wert vorschlägt, der von den AfA-Tabellen erheblich
// abweicht und ein Wahlrecht ist. Für einen Pkw oder Büromöbel steht in den
// Tabellen ein Erfahrungswert, kein Wahlrecht — dort eine Begründung zu
// verlangen ginge über die Vorschrift hinaus und sperrte bestehende Anlagen beim
// nächsten Speichern.
//
// Die AfA-Tabellen und das BMF-Schreiben binden ohnehin die Finanzverwaltung und
// nicht den Steuerpflichtigen. Eine begründete abweichende Nutzungsdauer ist
// zulässig — „begründet" heißt, dass die Begründung existiert. Ohne sie ist die
// Abweichung im Zweifel kein Wahlrecht, sondern ein Tippfehler, und der fällt
// erst in der Betriebsprüfung auf.
func (s *AssetService) checkUsefulLifeReason(
	ctx context.Context, asset *domain.FixedAsset,
) error {
	if asset.Method != domain.DepreciationLinear && asset.Method != domain.DepreciationDegressive {
		return nil
	}
	// Bei einem bestehenden Anlagegut greift die Pflicht nur, wo sich die
	// Nutzungsdauer ändert. Sonst hielte jede Stammdatenänderung — ein
	// berichtigter Name, ein neuer Standort — das Anlagegut an, bis jemand eine
	// Begründung nachträgt. Die Begründung
	// gehört zur Entscheidung über die Nutzungsdauer und nicht zum Speichern.
	if asset.ID != 0 {
		existing, err := s.assetRepo.FindByID(ctx, asset.ID)
		// Der Lesefehler bleibt hier unbeantwortet: die Speicherroutine lädt den
		// Bestand gleich darauf selbst und meldet ihn mit dem Text, der die
		// Kennung nennt. Zwei Meldungen für einen Fehler wären eine zu viel.
		if err == nil && existing != nil && existing.UsefulLifeMonths == asset.UsefulLifeMonths {
			return nil
		}
	}
	entry, known := accounting.LookupAssetAccount(asset.Account)
	if !known || entry.DefaultUsefulLifeMonths <= 0 {
		return nil
	}
	// Dieselbe Kennzeichnung, die der Kontenkatalog an die Maske gibt: sie soll
	// das Feld genau dort anbieten, wo es hier verlangt wird.
	if !entry.UsefulLifeReasonRequired {
		return nil
	}
	if asset.UsefulLifeMonths == entry.DefaultUsefulLifeMonths {
		return nil
	}
	if strings.TrimSpace(asset.UsefulLifeReason) != "" {
		return nil
	}
	return fmt.Errorf(
		"für %s (%s) schlägt Buchfink %d Monate vor (%s). Der Vorschlag gilt der Computerhardware "+
			"und der Software dieses Schreibens; auf dem Konto stehen auch andere Wirtschaftsgüter, "+
			"und das ist der Grund für die Abweichung. Eingetragen sind %d Monate — festzuhalten ist, "+
			"worauf diese Nutzungsdauer beruht. Die Tabellen binden nur die Finanzverwaltung; eine "+
			"Abweichung ohne Begründung hält aber keiner Prüfung stand",
		asset.Account, entry.Name, entry.DefaultUsefulLifeMonths, entry.UsefulLifeSource,
		asset.UsefulLifeMonths)
}
