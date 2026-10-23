package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// -------------------------------------------------------------------------
// Aufnahme ins Verzeichnis nach § 15a UStG bei der Aktivierung
// -------------------------------------------------------------------------

// Der Pkw aus dem Lehrbuch, diesmal über die Anlagenbuchhaltung: 7.600 €
// Vorsteuer sind mehr als die 1.000 € des § 44 Abs. 1 UStDV, also entsteht der
// Eintrag mit der Aktivierung und nicht erst, wenn jemand daran denkt.
func TestActivationRegistersTheInputTaxCorrection(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	assets := env.assets(t)
	register := env.inputTax(t)
	assets.SetInputTaxRegister(register)

	asset, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Firmenwagen", Class: domain.AssetClassTangible,
		Account: "0520", DepreciationAccount: "6222",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 4_000_000,
		UsefulLifeMonths: 72, Method: domain.DepreciationLinear,
		InputTaxAmount: 760_000, InputTaxPermille: 1000,
	})
	if err != nil {
		t.Fatalf("Aktivierung: %v", err)
	}

	view, err := register.Year(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if len(view.Rows) != 1 {
		t.Fatalf("%d Einträge im Verzeichnis — die Aktivierung muss einen anlegen", len(view.Rows))
	}
	row := view.Rows[0].Correction
	if row.InputTaxAmount != 760_000 || row.OriginalPermille != 1000 {
		t.Errorf("Eintrag %+v — erwartet 7.600,00 € Vorsteuer bei voller Verwendung", row)
	}
	if row.CorrectionPeriodYears != 5 {
		t.Errorf("Zeitraum %d Jahre — ein Pkw ist beweglich", row.CorrectionPeriodYears)
	}
	if row.AssetID == nil || *row.AssetID != asset.ID {
		t.Errorf("der Eintrag muss auf das Anlagegut zeigen: %+v", row.AssetID)
	}
	if !strings.Contains(row.Label, asset.InventoryNumber) {
		t.Errorf("Bezeichnung %q — sie muss die Inventarnummer nennen", row.Label)
	}

	// Ein Kopierer mit 900 € Vorsteuer bleibt draußen: unterhalb der
	// Bagatellgrenze wird nie berichtigt.
	if _, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Kopierer", Class: domain.AssetClassTangible,
		Account: "0650", DepreciationAccount: "6220",
		AcquisitionDate: "2026-03-01", AcquisitionCost: 473_700,
		UsefulLifeMonths: 156, Method: domain.DepreciationLinear,
		InputTaxAmount: 90_000,
	}); err != nil {
		t.Fatalf("zweite Aktivierung: %v", err)
	}
	after, err := register.Year(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if len(after.Rows) != 1 {
		t.Errorf("%d Einträge — ein Wirtschaftsgut unter der Bagatellgrenze des § 44 Abs. 1 UStDV "+
			"wird nicht berichtigt und braucht keinen Eintrag", len(after.Rows))
	}
}

// -------------------------------------------------------------------------
// Die Vorsteuer der Zugangsbuchung im Verzeichnis nach § 15a UStG
// -------------------------------------------------------------------------

// Der Weg vom Beleg ins Verzeichnis: der Pkw wird über den Eingangsbeleg
// gebucht, erscheint als Zugangskandidat und wird daraus aktiviert. Die
// Vorsteuer tippt dabei niemand ab — sie steht in der Buchung.
func TestAcquisitionCandidateCarriesTheInputTaxIntoTheRegister(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	assets := env.assets(t)
	register := env.inputTax(t)
	assets.SetInputTaxRegister(register)
	vendor := env.vendor(t, "Autohaus GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "", 4_000_000, domain.TaxRateStandard,
		domain.TaxTreatmentDomestic)
	req.Positions[0].PostingGroup = ""
	req.Positions[0].Account = "0520"
	req.Positions[0].Text = "Firmenwagen"
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Zugangsbeleg: %v", err)
	}
	if debitOn(entry, domain.AccountVorsteuer19) != 760_000 {
		t.Fatalf("Vorsteuer %s € — erwartet 7.600,00 €", debitOn(entry, domain.AccountVorsteuer19))
	}

	candidates, err := assets.AcquisitionCandidates(ctx, 2026)
	if err != nil {
		t.Fatalf("Zugangskandidaten: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("%d Kandidaten — erwartet die eine Zugangsbuchung: %+v", len(candidates), candidates)
	}
	if candidates[0].InputTaxAmount != 760_000 || candidates[0].InputTaxPermille != 1000 {
		t.Fatalf("Kandidat mit %s € Vorsteuer zu %d ‰ — erwartet 7.600,00 € bei voller Verwendung",
			candidates[0].InputTaxAmount, candidates[0].InputTaxPermille)
	}

	// Aktiviert wird aus dem Kandidaten heraus; die Vorsteuer setzt der Aufrufer
	// nicht.
	entryID := candidates[0].EntryID
	asset, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Firmenwagen", Class: domain.AssetClassTangible,
		Account: "0520", DepreciationAccount: "6222",
		AcquisitionDate: "2026-03-10", AcquisitionCost: 4_000_000,
		UsefulLifeMonths: 72, Method: domain.DepreciationLinear,
		AcquisitionEntryID: &entryID,
	})
	if err != nil {
		t.Fatalf("Aktivierung: %v", err)
	}
	if asset.InputTaxAmount != 760_000 {
		t.Errorf("das Anlagegut hat %s € Vorsteuer — erwartet 7.600,00 € aus der Zugangsbuchung",
			asset.InputTaxAmount)
	}

	view, err := register.Year(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if len(view.Rows) != 1 {
		t.Fatalf("%d Einträge im Verzeichnis — die Aktivierung muss einen anlegen", len(view.Rows))
	}
	row := view.Rows[0].Correction
	if row.InputTaxAmount != 760_000 || row.OriginalPermille != 1000 {
		t.Errorf("Eintrag mit %s € zu %d ‰ — erwartet 7.600,00 € bei voller Verwendung",
			row.InputTaxAmount, row.OriginalPermille)
	}
	if row.AssetID == nil || *row.AssetID != asset.ID {
		t.Errorf("der Eintrag muss auf das Anlagegut zeigen: %+v", row.AssetID)
	}
}

// Ein geteilter Vorsteuerabzug geht mit dem Anteil ins Verzeichnis: 600 ‰
// gezogen, 7.600 € Vorsteuer angefallen. Genau diese beiden Zahlen braucht
// § 15a UStG später.
func TestAcquisitionCandidateCarriesTheInputTaxShare(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	assets := env.assets(t)
	register := env.inputTax(t)
	assets.SetInputTaxRegister(register)
	vendor := env.vendor(t, "Autohaus GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "", 4_000_000, domain.TaxRateStandard,
		domain.TaxTreatmentDomestic)
	req.Positions[0].PostingGroup = ""
	req.Positions[0].Account = "0520"
	req.Positions[0].InputTaxShare = 600
	req.Positions[0].InputTaxShareReason = "Kfz zu 60 % betrieblich genutzt, Fahrtenbuch 2026"
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("Zugangsbeleg: %v", err)
	}

	candidates, err := assets.AcquisitionCandidates(ctx, 2026)
	if err != nil {
		t.Fatalf("Zugangskandidaten: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("%d Kandidaten — erwartet einen: %+v", len(candidates), candidates)
	}
	if candidates[0].InputTaxAmount != 760_000 || candidates[0].InputTaxPermille != 600 {
		t.Fatalf("Kandidat mit %s € zu %d ‰ — erwartet die volle Vorsteuer von 7.600,00 € und den "+
			"gezogenen Anteil von 600 ‰", candidates[0].InputTaxAmount, candidates[0].InputTaxPermille)
	}

	entryID := candidates[0].EntryID
	asset, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Firmenwagen", Class: domain.AssetClassTangible,
		Account: "0520", DepreciationAccount: "6222",
		AcquisitionDate: "2026-03-10", AcquisitionCost: 4_304_000,
		UsefulLifeMonths: 72, Method: domain.DepreciationLinear,
		AcquisitionEntryID: &entryID,
	})
	if err != nil {
		t.Fatalf("Aktivierung: %v", err)
	}
	if asset.InputTaxAmount != 760_000 || asset.InputTaxPermille != 600 {
		t.Fatalf("das Anlagegut hat %s € zu %d ‰", asset.InputTaxAmount, asset.InputTaxPermille)
	}
	view, err := register.Year(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if len(view.Rows) != 1 || view.Rows[0].Correction.OriginalPermille != 600 {
		t.Fatalf("Verzeichnis %+v — erwartet einen Eintrag mit 600 ‰", view.Rows)
	}
}

// -------------------------------------------------------------------------
// Verzeichnis nach § 15a UStG: die Aktivierung ohne Vorsteuer
// -------------------------------------------------------------------------

// Ein Gebäude ohne gezogene Vorsteuer kommt nicht ins Verzeichnis — und die
// Aktivierung geht trotzdem durch.
//
// Der Grundstückserwerb ist nach § 4 Nr. 9 Buchst. a UStG steuerfrei; ohne
// Option nach § 9 UStG steht in der Rechnung keine Steuer. § 15a UStG
// berichtigt aber den Abzug, den es gegeben hat, und ein Eintrag über null Euro
// Vorsteuer ist keiner. Vor der Nachbesserung meldete das Speichern einen
// Fehler, obwohl das Anlagegut längst in der Kartei stand: der Anwender
// wiederholte den Vorgang und legte es ein zweites Mal an.
func TestBuildingWithoutInputTaxIsNotRegistered(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)
	svc.SetInputTaxRegister(env.inputTax(t))

	building := env.building(t, svc)
	if building.InventoryNumber == "" {
		t.Fatal("das Gebäude wurde nicht aktiviert")
	}

	register := repository.NewInputTaxCorrectionRepository(env.db)
	entry, err := register.FindByAsset(ctx, building.ID)
	if err != nil {
		t.Fatalf("Verzeichnis lesen: %v", err)
	}
	if entry != nil {
		t.Errorf("das Verzeichnis führt %s mit %s € Vorsteuer — ohne Vorsteuer gibt es nichts zu "+
			"berichtigen", entry.Label, entry.InputTaxAmount)
	}
}

// Mit Vorsteuer — nach der Option des § 9 UStG — gehört das Gebäude ins
// Verzeichnis, und zwar mit dem zehnjährigen Berichtigungszeitraum des
// § 15a Abs. 1 Satz 2 UStG.
func TestBuildingWithInputTaxIsRegisteredForTenYears(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)
	svc.SetInputTaxRegister(env.inputTax(t))

	building, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Betriebsgebäude", Class: domain.AssetClassTangible,
		Account: "0240", DepreciationAccount: "6221",
		AcquisitionDate: "2026-01-02", AcquisitionCost: 20_000_000,
		UsefulLifeMonths: 400, Method: domain.DepreciationBuildingLinear,
		BuildingReferenceDate: "2005-03-01",
		InputTaxAmount:        3_800_000,
	})
	if err != nil {
		t.Fatalf("Aktivierung mit Vorsteuer: %v", err)
	}

	register := repository.NewInputTaxCorrectionRepository(env.db)
	entry, err := register.FindByAsset(ctx, building.ID)
	if err != nil {
		t.Fatalf("Verzeichnis lesen: %v", err)
	}
	if entry == nil {
		t.Fatal("ein Gebäude mit gezogener Vorsteuer gehört ins Verzeichnis nach § 15a UStG")
	}
	if entry.CorrectionPeriodYears != 10 {
		t.Errorf("Berichtigungszeitraum %d Jahre — für ein Gebäude sind es zehn",
			entry.CorrectionPeriodYears)
	}
	if entry.InputTaxAmount != 3_800_000 {
		t.Errorf("Vorsteuer %s € — erwartet 38.000,00 €", entry.InputTaxAmount)
	}
}
