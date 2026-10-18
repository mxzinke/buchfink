package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Anlagenthemen der Welle 5c: die Sperren am Anlagekonto, die Zuschreibung
// mit Grund, der Wertaufholungsbericht und die Einheitlichkeit des
// Sammelposten-Wahlrechts.

// Ein Gebäude wird nicht degressiv abgeschrieben: § 7 Abs. 2 EStG gibt es nur
// für bewegliche Wirtschaftsgüter. Bis hierher ließ sich der Plan rechnen, und
// er sah plausibel aus.
func TestDegressiveIsRefusedForBuildings(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)

	_, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Bürogebäude", Class: domain.AssetClassTangible, Account: "0240",
		DepreciationAccount: "6221", AcquisitionDate: "2026-07-01",
		AcquisitionCost: 100_000_000, UsefulLifeMonths: 400,
		Method: domain.DepreciationDegressive,
	})
	if err == nil {
		t.Fatal("die degressive Abschreibung gibt es für ein Gebäude nicht")
	}
	if !strings.Contains(err.Error(), "§ 7 Abs. 4") {
		t.Errorf("die Meldung muss auf die festen Sätze verweisen: %v", err)
	}
}

// Dasselbe Gebäude mit den festen Sätzen des § 7 Abs. 4 EStG: 3 % im Jahr, im
// Anschaffungsjahr zeitanteilig.
func TestBuildingUsesTheStatutoryRate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Bürogebäude", Class: domain.AssetClassTangible, Account: "0240",
		DepreciationAccount: "6221", AcquisitionDate: "2026-07-01",
		BuildingReferenceDate: "2005-03-01",
		AcquisitionCost:       100_000_000,
		Method:                domain.DepreciationBuildingLinear,
	})
	if err != nil {
		t.Fatalf("Gebäude: %v", err)
	}
	detail, err := svc.Get(ctx, 2026, asset.ID)
	if err != nil {
		t.Fatalf("Anlagegut lesen: %v", err)
	}
	if len(detail.Schedule) < 2 {
		t.Fatalf("%d Planjahre — erwartet den ganzen Gebäudeplan", len(detail.Schedule))
	}
	if detail.Schedule[0].Amount != 1_500_000 {
		t.Errorf("erstes Jahr %s € — erwartet 15.000,00 € (3 %% für sechs Monate)",
			detail.Schedule[0].Amount)
	}
	if detail.Schedule[1].Amount != 3_000_000 {
		t.Errorf("zweites Jahr %s € — erwartet 30.000,00 €", detail.Schedule[1].Amount)
	}
}

// Ein Elektrofahrzeug im Fenster des § 7 Abs. 2a EStG bekommt die Staffel; ein
// Verbrenner auf demselben Konto nicht.
func TestElectricVehicleNeedsItsFlag(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)

	if _, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Dienstwagen", Class: domain.AssetClassTangible, Account: "0520",
		DepreciationAccount: "6222", AcquisitionDate: "2026-03-01",
		AcquisitionCost: 10_000_000, Method: domain.DepreciationElectricVehicle,
	}); err == nil {
		t.Fatal("ohne das Kennzeichen für ein rein elektrisches Fahrzeug gibt es die Staffel nicht")
	}

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Dienstwagen elektrisch", Class: domain.AssetClassTangible, Account: "0520",
		DepreciationAccount: "6222", AcquisitionDate: "2026-03-01",
		AcquisitionCost: 10_000_000, Method: domain.DepreciationElectricVehicle,
		IsElectric: true,
	})
	if err != nil {
		t.Fatalf("E-Fahrzeug: %v", err)
	}
	detail, err := svc.Get(ctx, 2026, asset.ID)
	if err != nil {
		t.Fatalf("Anlagegut lesen: %v", err)
	}
	if len(detail.Schedule) != 6 {
		t.Fatalf("%d Planjahre — die Staffel läuft über sechs", len(detail.Schedule))
	}
	if detail.Schedule[0].Amount != 7_500_000 {
		t.Errorf("erstes Jahr %s € — erwartet 75.000,00 € (75 %%, ohne Zeitanteil)",
			detail.Schedule[0].Amount)
	}
}

// Eine Nutzungsdauer, die vom Vorschlag des Kontenkatalogs abweicht, braucht
// ihre Begründung. Für EDV-Hardware schlägt das BMF-Schreiben vom 22.02.2022
// ein Jahr vor.
func TestUsefulLifeDeviationNeedsAReason(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)

	if _, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Server", Class: domain.AssetClassTangible, Account: "0690",
		DepreciationAccount: "6220", AcquisitionDate: "2026-02-01",
		AcquisitionCost: 900_000, UsefulLifeMonths: 36,
		Method: domain.DepreciationLinear,
	}); err == nil {
		t.Fatal("eine abweichende Nutzungsdauer ohne Begründung darf nicht durchgehen")
	} else if !strings.Contains(err.Error(), "BMF") {
		t.Errorf("die Meldung muss den Vorschlag und seine Quelle nennen: %v", err)
	}

	if _, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Server", Class: domain.AssetClassTangible, Account: "0690",
		DepreciationAccount: "6220", AcquisitionDate: "2026-02-01",
		AcquisitionCost: 900_000, UsefulLifeMonths: 36,
		Method:           domain.DepreciationLinear,
		UsefulLifeReason: "Server im Dauerbetrieb, Herstellergarantie über drei Jahre",
	}); err != nil {
		t.Fatalf("mit Begründung muss die Abweichung durchgehen: %v", err)
	}

	// Der Vorschlag selbst braucht keine Begründung.
	if _, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Notebook", Class: domain.AssetClassTangible, Account: "0690",
		DepreciationAccount: "6220", AcquisitionDate: "2026-02-01",
		AcquisitionCost: 200_000, UsefulLifeMonths: 12,
		Method: domain.DepreciationLinear,
	}); err != nil {
		t.Fatalf("der Vorschlag selbst braucht keine Begründung: %v", err)
	}
}

// -------------------------------------------------------------------------
// Gebäudestichtag und Nutzungsdauer
// -------------------------------------------------------------------------

// Ohne Stichtag gibt es keinen Gebäudesatz. Der frühere Rückfall auf das
// Anschaffungsdatum lieferte für ein Betriebsgebäude mit altem Bauantrag 3 %
// statt der 2 % des § 7 Abs. 4 Satz 1 Nr. 2 Buchst. b EStG — eine stille
// Überschreibung der AfA um die Hälfte.
func TestBuildingNeedsItsReferenceDate(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)

	_, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Betriebsgebäude", Class: domain.AssetClassTangible,
		Account: "0240", DepreciationAccount: "6221",
		AcquisitionDate: "2026-02-01", AcquisitionCost: 50_000_000,
		UsefulLifeMonths: 400, Method: domain.DepreciationBuildingLinear,
	})
	if err == nil {
		t.Fatal("ohne Stichtag darf kein Gebäude mit festem Satz angelegt werden")
	}
	if !strings.Contains(err.Error(), "Bauantrag") {
		t.Errorf("die Meldung muss den maßgeblichen Stichtag benennen: %v", err)
	}

	// Mit dem Bauantrag von 1980 gilt der Satz von 2 %.
	asset, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Betriebsgebäude", Class: domain.AssetClassTangible,
		Account: "0240", DepreciationAccount: "6221",
		AcquisitionDate: "2026-02-01", AcquisitionCost: 50_000_000,
		UsefulLifeMonths: 400, Method: domain.DepreciationBuildingLinear,
		BuildingReferenceDate: "1980-05-01",
	})
	if err != nil {
		t.Fatalf("Gebäude mit Stichtag: %v", err)
	}
	detail, err := svc.Get(context.Background(), 2026, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	// 2 % von 500.000 € sind 10.000 €; im Anschaffungsjahr elf Zwölftel davon.
	if len(detail.Schedule) == 0 || detail.Schedule[0].Amount != 916_667 {
		t.Errorf("AfA des Anschaffungsjahres %+v — erwartet 2 %% zeitanteilig (9.166,67 €), "+
			"nicht 3 %%", detail.Schedule)
	}
}

// building ist ein Betriebsgebäude über 200.000 € mit Bauantrag nach 1985.
func (e *testEnv) building(t *testing.T, svc *AssetService) *domain.FixedAsset {
	t.Helper()
	asset, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Betriebsgebäude", Class: domain.AssetClassTangible,
		Account: "0240", DepreciationAccount: "6221",
		AcquisitionDate: "2026-01-02", AcquisitionCost: 20_000_000,
		UsefulLifeMonths: 400, Method: domain.DepreciationBuildingLinear,
		BuildingReferenceDate: "2005-03-01",
	})
	if err != nil {
		t.Fatalf("Gebäude: %v", err)
	}
	return asset
}

// Die Begründungspflicht für eine abweichende Nutzungsdauer gilt dem Wahlrecht
// des BMF-Schreibens vom 22.02.2022 und nicht jeder AfA-Tabelle. Für einen Pkw
// steht dort ein Erfahrungswert, und eine Begründung dafür zu verlangen ginge
// über die Vorschrift hinaus — und sperrte bestehende Anlagen beim Speichern.
func TestUsefulLifeReasonOnlyForTheDigitalProposal(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)

	if _, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Firmenwagen", Class: domain.AssetClassTangible,
		Account: "0520", DepreciationAccount: "6222",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 4_000_000,
		UsefulLifeMonths: 84, Method: domain.DepreciationLinear,
	}); err != nil {
		t.Errorf("eine abweichende Nutzungsdauer nach der AfA-Tabelle braucht keine Begründung: %v",
			err)
	}

	// Für die Hardware bleibt sie Pflicht: zwölf Monate sind ein Wahlrecht.
	if _, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Server", Class: domain.AssetClassTangible,
		Account: "0690", DepreciationAccount: "6220",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 900_000,
		UsefulLifeMonths: 60, Method: domain.DepreciationLinear,
	}); err == nil {
		t.Error("die Abweichung vom BMF-Vorschlag ist zu begründen")
	}
}

// -------------------------------------------------------------------------
// Nutzungsdauer: Begründung nur bei der Entscheidung
// -------------------------------------------------------------------------

// Eine Stammdatenänderung an einem bestehenden Anlagegut verlangt keine
// Begründung für die Nutzungsdauer.
//
// Die Pflicht gehört zur Entscheidung über die Nutzungsdauer. Griffe sie bei
// jedem Speichern, ließe sich der vor dieser Welle mit 36 Monaten angelegte
// Server nicht einmal mehr umbenennen — eine Vorschrift rückwirkend auf einen
// abgeschlossenen Vorgang anzuwenden, ist nicht ihr Zweck.
func TestUsefulLifeReasonOnlyWhenTheDurationChanges(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)

	server, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Server", Class: domain.AssetClassTangible,
		Account: "0690", DepreciationAccount: "6220",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 900_000,
		UsefulLifeMonths: 36, Method: domain.DepreciationLinear,
		UsefulLifeReason: "Herstellergarantie über drei Jahre, Nutzung im Rechenzentrum",
	})
	if err != nil {
		t.Fatalf("Zugang mit Begründung: %v", err)
	}

	// Der Bestand aus der Zeit vor dieser Welle: die Begründung fehlt, die
	// Nutzungsdauer bleibt.
	server.Name = "Server (Rack 2)"
	server.UsefulLifeReason = ""
	if _, err := svc.Save(ctx, server); err != nil {
		t.Errorf("die Stammdatenänderung darf nicht an der Begründung hängen: %v", err)
	}

	// Die geänderte Nutzungsdauer dagegen ist die Entscheidung, um die es geht.
	server.UsefulLifeMonths = 48
	_, err = svc.Save(ctx, server)
	if err == nil {
		t.Fatal("eine geänderte Nutzungsdauer ohne Begründung ist abzuweisen")
	}
	if !strings.Contains(err.Error(), "Computerhardware") {
		t.Errorf("die Meldung soll sagen, wofür der Vorschlag gilt: %v", err)
	}
}
