package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Anlagenthemen der Welle 5c: die Sperren am Anlagekonto, die Zuschreibung
// mit Grund, der Wertaufholungsbericht, die Einheitlichkeit des
// Sammelposten-Wahlrechts und die anschaffungsnahen Herstellungskosten.

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
	detail, err := svc.Get(ctx, asset.ID)
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
	detail, err := svc.Get(ctx, asset.ID)
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

// Eine Zuschreibung ohne Grund ist von einer willkürlichen Erhöhung des
// Buchwerts nicht zu unterscheiden.
func TestWriteUpNeedsAReason(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)
	asset := env.impairedAsset(t, svc)

	if _, err := svc.BookWriteUp(ctx, WriteUpRequest{
		AssetID: asset.ID, Date: "2027-12-31", Amount: 50_000,
	}); err == nil {
		t.Fatal("eine Zuschreibung ohne Grund darf nicht gebucht werden")
	} else if !strings.Contains(err.Error(), "§ 253 Abs. 5") {
		t.Errorf("die Meldung muss die Vorschrift nennen: %v", err)
	}

	if _, err := svc.BookWriteUp(ctx, WriteUpRequest{
		AssetID: asset.ID, Date: "2027-12-31", Amount: 50_000,
		Reason: "Der Wasserschaden ist behoben, der Marktwert liegt wieder über dem Buchwert",
	}); err != nil {
		t.Fatalf("mit Grund muss die Zuschreibung durchgehen: %v", err)
	}
}

// Der Wertaufholungsbericht führt das Anlagegut mit Spielraum auf und nimmt es
// heraus, sobald jemand bestätigt hat, dass der Grund fortbesteht.
func TestWriteUpReportListsAssetsWithRoom(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)
	asset := env.impairedAsset(t, svc)

	report, err := svc.WriteUpReport(ctx, 2027)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	if len(report.Candidates) != 1 {
		t.Fatalf("%d Anlagegüter im Bericht — erwartet eines", len(report.Candidates))
	}
	candidate := report.Candidates[0]
	if candidate.AssetID != asset.ID {
		t.Errorf("Anlagegut %d — erwartet %d", candidate.AssetID, asset.ID)
	}
	if candidate.MaxWriteUp <= 0 {
		t.Errorf("Spielraum %s € — erwartet die Differenz zu den fortgeführten Anschaffungskosten",
			candidate.MaxWriteUp)
	}
	if len(candidate.Impairments) != 1 {
		t.Errorf("%d außerplanmäßige Abschreibungen — erwartet eine", len(candidate.Impairments))
	}
	if report.Open != 1 || candidate.Confirmed {
		t.Errorf("offen=%d bestätigt=%v — vor der Antwort ist der Fall offen",
			report.Open, candidate.Confirmed)
	}

	if _, err := svc.ConfirmImpairmentPersists(ctx, asset.ID, 2027, ""); err == nil {
		t.Error("auch das Unterlassen der Zuschreibung braucht eine Begründung")
	}
	after, err := svc.ConfirmImpairmentPersists(ctx, asset.ID, 2027,
		"Der Wasserschaden besteht fort, die Sanierung beginnt erst 2028")
	if err != nil {
		t.Fatalf("Bestätigung: %v", err)
	}
	if after.Open != 0 || !after.Candidates[0].Confirmed {
		t.Errorf("nach der Bestätigung ist der Fall für 2027 beantwortet: offen=%d", after.Open)
	}

	if after.Candidates[0].ConfirmedNote == "" {
		t.Error("die Begründung gehört zur Bestätigung und wird aufgehoben")
	}

	// Die Bestätigung gilt für ihr Geschäftsjahr und für kein anderes: § 253
	// Abs. 5 Satz 1 HGB stellt die Frage jedes Jahr neu, und eine Antwort von
	// 2027 beantwortet sie für 2026 nicht.
	other, err := svc.WriteUpReport(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht 2026: %v", err)
	}
	if other.Open != 1 || other.Candidates[0].Confirmed {
		t.Errorf("offen=%d bestätigt=%v — die Bestätigung für 2027 gilt nicht für 2026",
			other.Open, other.Candidates[0].Confirmed)
	}
}

// impairedAsset ist eine Maschine mit einer außerplanmäßigen Abschreibung.
func (e *testEnv) impairedAsset(t *testing.T, svc *AssetService) *domain.FixedAsset {
	t.Helper()
	ctx := context.Background()
	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Fräse", Class: domain.AssetClassTangible, Account: "0440",
		DepreciationAccount: "6220", AcquisitionDate: "2026-01-02",
		AcquisitionCost: 1_200_000, UsefulLifeMonths: 120,
		Method: domain.DepreciationLinear,
	})
	if err != nil {
		t.Fatalf("Anlagegut: %v", err)
	}
	if _, err := svc.BookImpairment(ctx, ImpairmentRequest{
		AssetID: asset.ID, Date: "2026-12-31", Amount: 300_000, Permanent: true,
		Reason: "Wasserschaden in der Werkhalle",
	}); err != nil {
		t.Fatalf("außerplanmäßige Abschreibung: %v", err)
	}
	return asset
}

// § 6 Abs. 2a Satz 5 EStG: das Wahlrecht wird für alle Wirtschaftsgüter eines
// Wirtschaftsjahres einheitlich ausgeübt.
func TestPoolAndImmediateCannotShareAFiscalYear(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)

	if _, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Sammelposten 2026", Class: domain.AssetClassTangible, Account: "0675",
		DepreciationAccount: "6264", AcquisitionDate: "2026-02-01",
		AcquisitionCost: 60_000, Method: domain.DepreciationPool, PoolYear: 2026,
	}); err != nil {
		t.Fatalf("Sammelposten: %v", err)
	}

	_, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Bürostuhl", Class: domain.AssetClassTangible, Account: "0670",
		DepreciationAccount: "6260", AcquisitionDate: "2026-06-01",
		AcquisitionCost: 60_000, Method: domain.DepreciationImmediate,
	})
	if err == nil {
		t.Fatal("ein Sofortabzug neben einem Sammelposten desselben Jahres bricht das Wahlrecht")
	}
	if !strings.Contains(err.Error(), "§ 6 Abs. 2a Satz 5") {
		t.Errorf("die Meldung muss die Vorschrift nennen: %v", err)
	}

	// Der Bericht sagt dasselbe, ohne anzuhalten.
	report, err := svc.PoolConsistency(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	if !report.Consistent || len(report.Pooled) != 1 || len(report.Immediate) != 0 {
		t.Errorf("Bericht %+v — nach der Abweisung steht nur der Sammelposten", report)
	}

	// Unterhalb der Sammelposten-Grenze stellt sich die Frage nicht: dort gibt
	// es kein Wahlrecht, das einheitlich auszuüben wäre.
	if _, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Locher", Class: domain.AssetClassTangible, Account: "0670",
		DepreciationAccount: "6260", AcquisitionDate: "2026-06-01",
		AcquisitionCost: 2_000, Method: domain.DepreciationImmediate,
	}); err != nil {
		t.Fatalf("unter 250 € gibt es nur den Sofortabzug: %v", err)
	}
}

// § 6 Abs. 1 Nr. 1a EStG: Instandsetzungs- und Modernisierungsaufwand der ersten
// drei Jahre wird zu Herstellungskosten, sobald er 15 % der
// Gebäude-Anschaffungskosten übersteigt.
func TestNearAcquisitionCostWarnsAtFifteenPercent(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)
	vendor := env.vendor(t, "Bauunternehmen GmbH", "DE", "")

	building, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Lagerhalle", Class: domain.AssetClassTangible, Account: "0250",
		DepreciationAccount: "6221", AcquisitionDate: "2026-01-15",
		BuildingReferenceDate: "2005-01-01",
		AcquisitionCost:       20_000_000, // 200.000 €
		Method:                domain.DepreciationBuildingLinear,
	})
	if err != nil {
		t.Fatalf("Gebäude: %v", err)
	}

	// Der Rahmen liegt bei 15 % von 200.000 €, also 30.000 €.
	check, err := svc.CheckNearAcquisitionCost(ctx, building.ID, "2026-06-01", 2_000_000)
	if err != nil {
		t.Fatalf("Prüfung: %v", err)
	}
	if !check.Applicable || check.Limit != 3_000_000 {
		t.Fatalf("Rahmen %s € — erwartet 30.000,00 €", check.Limit)
	}
	if check.Exceeded {
		t.Errorf("20.000 € bleiben unter dem Rahmen: %s", check.Note)
	}

	if _, err := svc.BookMaintenance(ctx, MaintenanceRequest{
		AssetID: building.ID, Date: "2026-06-01", Amount: 2_000_000,
		TaxTreatment: domain.TaxTreatmentDomestic, TaxRate: domain.TaxRateStandard,
		Settlement: SettlementOpen, ContactID: vendor.ID,
		Note: "Dachsanierung, Wiederherstellung des ursprünglichen Zustands",
	}); err != nil {
		t.Fatalf("Erhaltungsaufwand: %v", err)
	}

	// Mit weiteren 12.000 € sind es 32.000 € und damit mehr als 30.000 €.
	second, err := svc.CheckNearAcquisitionCost(ctx, building.ID, "2027-04-01", 1_200_000)
	if err != nil {
		t.Fatalf("Prüfung: %v", err)
	}
	if second.Spent != 2_000_000 {
		t.Errorf("bisher %s € — erwartet die 20.000,00 € des ersten Vorgangs", second.Spent)
	}
	if !second.Exceeded {
		t.Fatalf("32.000 € übersteigen den Rahmen von 30.000 €: %s", second.Note)
	}
	if !strings.Contains(second.Note, "Herstellungskosten") {
		t.Errorf("die Meldung muss die Folge nennen: %s", second.Note)
	}

	// Nach Ablauf der drei Jahre stellt sich die Frage nicht mehr.
	late, err := svc.CheckNearAcquisitionCost(ctx, building.ID, "2029-06-01", 5_000_000)
	if err != nil {
		t.Fatalf("Prüfung: %v", err)
	}
	if late.Applicable {
		t.Errorf("nach dem Dreijahreszeitraum ist Instandsetzungsaufwand sofort abziehbar: %s",
			late.Note)
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
