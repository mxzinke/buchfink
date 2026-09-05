package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/receiptstore"
	"github.com/buchfink/buchfink/internal/repository"
)

func (e *testEnv) assets(t *testing.T) *AssetService {
	t.Helper()
	return NewAssetService(
		repository.NewAssetRepository(e.db),
		e.journalRepo,
		e.journal,
		e.numberRepo,
		e.contactRepo,
		repository.NewSettingsRepository(e.db),
		repository.NewAuditRepository(e.db),
		e.fiscalYear,
	)
}

// machine is a Sachanlage with a clean four-year plan: 12.000 € über 48 Monate
// sind 3.000 € im Jahr.
func (e *testEnv) machine(t *testing.T, svc *AssetService) *domain.FixedAsset {
	t.Helper()
	asset, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name:                "Fräsmaschine",
		Class:               domain.AssetClassTangible,
		Account:             "0440",
		DepreciationAccount: "6220",
		AcquisitionDate:     "2026-01-10",
		AcquisitionCost:     1_200_000,
		UsefulLifeMonths:    48,
		Method:              domain.DepreciationLinear,
	})
	if err != nil {
		t.Fatalf("Anlagegut konnte nicht angelegt werden: %v", err)
	}
	return asset
}

func TestSaveAssetAllocatesInventoryNumberAndRecordsAcquisition(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	asset := env.machine(t, svc)

	if !strings.HasPrefix(asset.InventoryNumber, "AN-2026-") {
		t.Errorf("Inventarnummer %q — erwartet das Muster AN-2026-nnnn", asset.InventoryNumber)
	}
	if asset.Cost != 1_200_000 || asset.BookValue != 1_200_000 || asset.Accumulated != 0 {
		t.Errorf("nach dem Zugang: AHK %s €, Buchwert %s €, kumulierte AfA %s €",
			asset.Cost, asset.BookValue, asset.Accumulated)
	}
	if asset.Status != domain.AssetStatusDepreciateDue {
		t.Errorf("Status %q — erwartet den Hinweis auf die offene Abschreibung", asset.Status)
	}

	detail, err := svc.Get(context.Background(), asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if len(detail.Schedule) != 4 {
		t.Errorf("AfA-Plan über %d Jahre — erwartet vier (2026 bis 2029)", len(detail.Schedule))
	}
	if len(detail.Notes) == 0 {
		t.Error("zu jedem Anlagegut gehört mindestens eine Erklärung")
	}
}

// Finanzanlagen nutzen sich nicht ab. Ein planmäßiger Plan auf ihnen wäre still
// falsch, also wird er abgelehnt.
func TestFinancialAssetRejectsPlannedDepreciation(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)

	_, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Anteile Musterwerk GmbH", Class: domain.AssetClassFinancial,
		Account: "0850", DepreciationAccount: "6220",
		AcquisitionDate: "2026-02-01", AcquisitionCost: 5_000_000,
		UsefulLifeMonths: 120, Method: domain.DepreciationLinear,
	})
	if err == nil {
		t.Fatal("eine Finanzanlage darf keinen planmäßigen Abschreibungsplan bekommen")
	}
	if !strings.Contains(err.Error(), "253") {
		t.Errorf("die Meldung sollte auf § 253 HGB verweisen, lautet aber: %v", err)
	}
}

func TestDepreciationRunBooksAgainstTheAssetAccount(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	asset := env.machine(t, svc)
	ctx := context.Background()

	run, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("Abschreibungslauf: %v", err)
	}
	if len(run.Due) != 1 || run.Due[0].Due != 300_000 {
		t.Fatalf("fällige AfA %+v — erwartet 3.000,00 € für ein volles Jahr", run.Due)
	}
	if run.BookingDate != "2026-12-31" {
		t.Errorf("Buchungsdatum %q — die AfA ist eine Abschlussbuchung zum Bilanzstichtag", run.BookingDate)
	}

	result, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026})
	if err != nil {
		t.Fatalf("AfA buchen: %v", err)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("erwartet eine Buchung je Anlagegut, bekommen %d", len(result.Entries))
	}
	entry := result.Entries[0]
	if entry.Source != domain.EntrySourceDepreciation {
		t.Errorf("Buchungsherkunft %q — erwartet die Abschreibung", entry.Source)
	}
	if entry.DocumentNumber != asset.InventoryNumber {
		t.Errorf("Belegfeld %q — erwartet die Inventarnummer %s", entry.DocumentNumber, asset.InventoryNumber)
	}
	var debit, credit string
	for _, l := range entry.Lines {
		if l.Side == domain.SideDebit {
			debit = l.Account
		} else {
			credit = l.Account
		}
	}
	if debit != "6220" || credit != "0440" {
		t.Errorf("Buchungssatz %s an %s — erwartet 6220 an 0440", debit, credit)
	}

	// Zweiter Lauf: nichts mehr offen.
	again, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("zweiter Abschreibungslauf: %v", err)
	}
	if len(again.Due) != 0 {
		t.Errorf("nach dem Lauf ist noch etwas offen: %+v", again.Due)
	}
	if _, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026}); err == nil {
		t.Error("ein zweiter Lauf darf die AfA nicht doppelt buchen")
	}

	reloaded, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if reloaded.Asset.BookValue != 900_000 {
		t.Errorf("Buchwert %s € — erwartet 9.000,00 €", reloaded.Asset.BookValue)
	}
	if reloaded.Schedule[0].Status != "gebucht" {
		t.Errorf("Status des ersten Planjahres %q — erwartet gebucht", reloaded.Schedule[0].Status)
	}
	if reloaded.Movements[1].EntryNumber == "" {
		t.Error("die Bewegung muss auf ihre Buchung verweisen")
	}
}

// § 253 Abs. 3 Satz 5 HGB lässt die außerplanmäßige Abschreibung auf Sachanlagen
// nur bei voraussichtlich dauernder Wertminderung zu; Satz 6 gibt die Ausnahme
// allein den Finanzanlagen.
func TestImpairmentRulesDifferBetweenTangibleAndFinancialAssets(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	machine := env.machine(t, svc)

	_, err := svc.BookImpairment(ctx, ImpairmentRequest{
		AssetID: machine.ID, Date: "2026-11-30", Amount: 100_000,
		Permanent: false, Reason: "Marktwert gefallen",
	})
	if err == nil {
		t.Fatal("auf eine Sachanlage darf bei nicht dauernder Wertminderung nicht abgeschrieben werden")
	}

	if _, err := svc.BookImpairment(ctx, ImpairmentRequest{
		AssetID: machine.ID, Date: "2026-11-30", Amount: 100_000, Permanent: true,
	}); err == nil {
		t.Error("eine außerplanmäßige Abschreibung ohne Begründung darf nicht durchgehen")
	}

	entry, err := svc.BookImpairment(ctx, ImpairmentRequest{
		AssetID: machine.ID, Date: "2026-11-30", Amount: 100_000, Permanent: true,
		Reason: "Dauerhafter Schaden am Antrieb",
	})
	if err != nil {
		t.Fatalf("außerplanmäßige Abschreibung: %v", err)
	}
	if entry.Lines[0].Account != "6230" {
		t.Errorf("Aufwandskonto %s — erwartet 6230", entry.Lines[0].Account)
	}

	share, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Anteile Musterwerk GmbH", Class: domain.AssetClassFinancial,
		Account: "0850", AcquisitionDate: "2026-02-01", AcquisitionCost: 5_000_000,
		Method: domain.DepreciationNone,
	})
	if err != nil {
		t.Fatalf("Finanzanlage konnte nicht angelegt werden: %v", err)
	}
	financial, err := svc.BookImpairment(ctx, ImpairmentRequest{
		AssetID: share.ID, Date: "2026-12-01", Amount: 500_000,
		Permanent: false, Reason: "Kurs unter Anschaffungskosten",
	})
	if err != nil {
		t.Fatalf("außerplanmäßige Abschreibung auf die Finanzanlage: %v", err)
	}
	if financial.Lines[0].Account != "7201" {
		t.Errorf("Aufwandskonto %s — erwartet 7201 für die nicht dauernde Wertminderung",
			financial.Lines[0].Account)
	}
}

// Zugeschrieben wird höchstens bis zu den fortgeführten Anschaffungskosten
// (§ 253 Abs. 5 Satz 1 HGB).
func TestWriteUpIsCappedAtContinuedCost(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	share, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Anteile Musterwerk GmbH", Class: domain.AssetClassFinancial,
		Account: "0850", AcquisitionDate: "2026-02-01", AcquisitionCost: 5_000_000,
		Method: domain.DepreciationNone,
	})
	if err != nil {
		t.Fatalf("Finanzanlage: %v", err)
	}
	if _, err := svc.BookImpairment(ctx, ImpairmentRequest{
		AssetID: share.ID, Date: "2026-06-30", Amount: 500_000,
		Permanent: true, Reason: "Verlustjahr der Beteiligung",
	}); err != nil {
		t.Fatalf("außerplanmäßige Abschreibung: %v", err)
	}

	if _, err := svc.BookWriteUp(ctx, WriteUpRequest{
		AssetID: share.ID, Date: "2026-12-31", Amount: 600_000, Reason: "Erholung",
	}); err == nil {
		t.Fatal("über die fortgeführten Anschaffungskosten hinaus darf nicht zugeschrieben werden")
	}

	entry, err := svc.BookWriteUp(ctx, WriteUpRequest{
		AssetID: share.ID, Date: "2026-12-31", Amount: 500_000, Reason: "Grund entfallen",
	})
	if err != nil {
		t.Fatalf("Zuschreibung: %v", err)
	}
	var revenue string
	for _, l := range entry.Lines {
		if l.Side == domain.SideCredit {
			revenue = l.Account
		}
	}
	if revenue != "4912" {
		t.Errorf("Ertragskonto %s — erwartet 4912 für die Zuschreibung auf eine Finanzanlage", revenue)
	}

	detail, err := svc.Get(ctx, share.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if detail.Asset.BookValue != 5_000_000 {
		t.Errorf("Buchwert %s € — erwartet wieder die vollen Anschaffungskosten", detail.Asset.BookValue)
	}
}

// Der SKR04 wählt das Erlöskonto nach dem Ergebnis: derselbe Verkauf läuft bei
// Buchgewinn über 4845/4855 und bei Buchverlust über 6885/6895.
func TestDisposalPicksAccountsByResult(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	customer := env.customer(t, "Käufer GmbH", "DE", "DE999999999")

	gainAsset := env.machine(t, svc)
	preview, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: gainAsset.ID, Date: "2026-06-30", Kind: domain.DisposalSale,
		Proceeds: 1_100_000, TaxTreatment: domain.TaxTreatmentDomestic,
		TaxRate: domain.TaxRateStandard, Settlement: SettlementPaid, PaymentAccount: "1800",
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.CatchUpAmount != 150_000 {
		t.Errorf("nachzuholende AfA %s € — erwartet 1.500,00 € für sechs Monate", preview.CatchUpAmount)
	}
	if preview.BookValue != 1_050_000 {
		t.Errorf("Restbuchwert %s € — erwartet 10.500,00 €", preview.BookValue)
	}
	if !preview.IsGain || preview.Result != 50_000 {
		t.Errorf("Ergebnis %s € (Gewinn: %v) — erwartet einen Buchgewinn von 500,00 €",
			preview.Result, preview.IsGain)
	}
	if preview.Accounts.Revenue != "4845" || preview.Accounts.BookValue != "4855" {
		t.Errorf("Konten %+v — erwartet 4845 und 4855 bei Buchgewinn", preview.Accounts)
	}
	if preview.Tax != 209_000 {
		t.Errorf("Umsatzsteuer %s € — erwartet 2.090,00 €", preview.Tax)
	}

	result, err := svc.Dispose(ctx, DisposalRequest{
		AssetID: gainAsset.ID, Date: "2026-06-30", Kind: domain.DisposalSale,
		Proceeds: 1_100_000, TaxTreatment: domain.TaxTreatmentDomestic,
		TaxRate: domain.TaxRateStandard, Settlement: SettlementPaid, PaymentAccount: "1800",
		Note: "Verkauf an Käufer GmbH",
	})
	if err != nil {
		t.Fatalf("Abgang: %v", err)
	}
	if result.CatchUpEntry == nil {
		t.Error("die AfA bis zum Abgangsmonat muss als eigene Buchung entstehen")
	}
	if result.DisposalEntry == nil || !result.DisposalEntry.IsBalanced() {
		t.Fatalf("die Abgangsbuchung fehlt oder ist nicht ausgeglichen: %+v", result.DisposalEntry)
	}
	if result.Asset.BookValue != 0 || result.Asset.Cost != 0 {
		t.Errorf("nach dem Abgang: AHK %s €, Buchwert %s € — beide müssen die Bücher verlassen",
			result.Asset.Cost, result.Asset.BookValue)
	}
	if result.Asset.Status != domain.AssetStatusDisposed {
		t.Errorf("Status %q — erwartet abgegangen", result.Asset.Status)
	}

	// Derselbe Vorgang mit einem Erlös unter dem Restbuchwert.
	lossAsset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Transporter", Class: domain.AssetClassTangible, Account: "0540",
		DepreciationAccount: "6222", AcquisitionDate: "2026-01-10",
		AcquisitionCost: 1_200_000, UsefulLifeMonths: 48, Method: domain.DepreciationLinear,
		// Kürzer als der Erfahrungswert der AfA-Tabelle. Das ist zulässig — die
		// Tabelle bindet die Finanzverwaltung und nicht den Steuerpflichtigen —
		// und die Begründung steht hier freiwillig. Verlangt wird sie nur dort,
		// wo Buchfink das Wahlrecht des BMF-Schreibens vom 22.02.2022
		// vorschlägt.
		UsefulLifeReason: "Einsatz im Baustellenverkehr, Laufleistung 90.000 km im Jahr",
	})
	if err != nil {
		t.Fatalf("zweites Anlagegut: %v", err)
	}
	lossPreview, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: lossAsset.ID, Date: "2026-06-30", Kind: domain.DisposalSale,
		Proceeds: 900_000, TaxTreatment: domain.TaxTreatmentDomestic,
		TaxRate: domain.TaxRateStandard, Settlement: SettlementOpen, ContactID: customer.ID,
	})
	if err != nil {
		t.Fatalf("Vorschau Buchverlust: %v", err)
	}
	if lossPreview.IsGain || lossPreview.Result != -150_000 {
		t.Errorf("Ergebnis %s € — erwartet einen Buchverlust von 1.500,00 €", lossPreview.Result)
	}
	if lossPreview.Accounts.Revenue != "6885" || lossPreview.Accounts.BookValue != "6895" {
		t.Errorf("Konten %+v — erwartet 6885 und 6895 bei Buchverlust", lossPreview.Accounts)
	}
	if _, err := svc.Dispose(ctx, DisposalRequest{
		AssetID: lossAsset.ID, Date: "2026-06-30", Kind: domain.DisposalSale,
		Proceeds: 900_000, TaxTreatment: domain.TaxTreatmentDomestic,
		TaxRate: domain.TaxRateStandard, Settlement: SettlementOpen, ContactID: customer.ID,
	}); err != nil {
		t.Fatalf("Abgang mit Buchverlust: %v", err)
	}
}

// Eine Verschrottung hat keinen Erlös; der Restbuchwert geht in den Aufwand.
func TestScrappingBooksOnlyTheRemainingBookValue(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	asset := env.machine(t, svc)

	if _, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-09-30", Kind: domain.DisposalScrapped, Proceeds: 100,
	}); err == nil {
		t.Fatal("eine Verschrottung mit Erlös ist ein Widerspruch und muss abgelehnt werden")
	}

	result, err := svc.Dispose(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-09-30", Kind: domain.DisposalScrapped,
		Note: "Totalschaden",
	})
	if err != nil {
		t.Fatalf("Verschrottung: %v", err)
	}
	if result.DisposalEntry == nil {
		t.Fatal("der Restbuchwert muss ausgebucht werden")
	}
	var expense string
	for _, l := range result.DisposalEntry.Lines {
		if l.Side == domain.SideDebit {
			expense = l.Account
		}
	}
	if expense != "6895" {
		t.Errorf("Aufwandskonto %s — erwartet 6895 (Anlagenabgang bei Buchverlust)", expense)
	}
}

func TestAnlagenspiegelShowsTheDevelopmentOfEachPosition(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	env.machine(t, svc)

	if _, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026}); err != nil {
		t.Fatalf("AfA buchen: %v", err)
	}

	spiegel, err := svc.Anlagenspiegel(ctx)
	if err != nil {
		t.Fatalf("Anlagenspiegel: %v", err)
	}
	if len(spiegel.Rows) != 1 {
		t.Fatalf("erwartet eine Zeile je Anlagekonto, bekommen %d", len(spiegel.Rows))
	}
	row := spiegel.Rows[0]
	if row.CostOpening != 0 || row.Additions != 1_200_000 || row.CostClosing != 1_200_000 {
		t.Errorf("Anschaffungskosten: Anfang %s €, Zugänge %s €, Ende %s €",
			row.CostOpening, row.Additions, row.CostClosing)
	}
	if row.DepreciationYear != 300_000 || row.DepreciationClosing != 300_000 {
		t.Errorf("Abschreibungen: Jahr %s €, kumuliert %s €", row.DepreciationYear, row.DepreciationClosing)
	}
	if row.BookValueClosing != 900_000 {
		t.Errorf("Buchwert am Ende %s € — erwartet 9.000,00 €", row.BookValueClosing)
	}
	if spiegel.Totals.BookValueClosing != 900_000 || len(spiegel.ClassTotals) != 1 {
		t.Errorf("Summenzeile %+v", spiegel.Totals)
	}
}

// Der Zugang wird über den Beleg gebucht. Was dabei auf einem Anlagekonto
// landet, gehört anschließend ins Verzeichnis — und genau das zeigt die Liste
// der Zugangskandidaten.
func TestAcquisitionCandidatesFindUnregisteredBookings(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, simpleEntry("0650", "1800", 250_000)); err != nil {
		t.Fatalf("Zugangsbuchung: %v", err)
	}

	candidates, err := svc.AcquisitionCandidates(ctx)
	if err != nil {
		t.Fatalf("Zugangskandidaten: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Account != "0650" || candidates[0].Amount != 250_000 {
		t.Fatalf("erwartet die eine Buchung auf 0650, bekommen %+v", candidates)
	}

	entryID := candidates[0].EntryID
	if _, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Schreibtisch", Class: domain.AssetClassTangible, Account: "0650",
		DepreciationAccount: "6220", AcquisitionDate: "2026-03-01", AcquisitionCost: 250_000,
		UsefulLifeMonths: 156, Method: domain.DepreciationLinear, AcquisitionEntryID: &entryID,
	}); err != nil {
		t.Fatalf("Anlagegut zur Buchung: %v", err)
	}

	after, err := svc.AcquisitionCandidates(ctx)
	if err != nil {
		t.Fatalf("Zugangskandidaten nach der Erfassung: %v", err)
	}
	if len(after) != 0 {
		t.Errorf("die erfasste Buchung darf nicht mehr als offen gelten: %+v", after)
	}
}

// Nachträgliche Anschaffungskosten erhöhen die Bemessungsgrundlage, ein Skonto
// mindert sie (§ 255 Abs. 1 HGB) — beides ohne eigene Buchung, weil der Beleg
// bzw. der Zahlungsflow sie bereits gebucht hat.
func TestCostAdjustmentsChangeTheDepreciationBase(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	asset := env.machine(t, svc)

	if _, err := svc.RecordCostAdjustment(ctx, CostAdjustmentRequest{
		AssetID: asset.ID, Date: "2026-02-01", Amount: 120_000, Note: "Fracht und Montage",
	}); err != nil {
		t.Fatalf("nachträgliche Anschaffungskosten: %v", err)
	}
	updated, err := svc.RecordCostAdjustment(ctx, CostAdjustmentRequest{
		AssetID: asset.ID, Date: "2026-02-15", Amount: 20_000, Reduction: true, Note: "Skonto",
	})
	if err != nil {
		t.Fatalf("Anschaffungskostenminderung: %v", err)
	}
	if updated.Cost != 1_300_000 {
		t.Fatalf("Anschaffungskosten %s € — erwartet 13.000,00 €", updated.Cost)
	}

	run, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("Abschreibungslauf: %v", err)
	}
	if len(run.Due) != 1 || run.Due[0].Due != 325_000 {
		t.Errorf("fällige AfA %+v — erwartet 3.250,00 € aus der erhöhten Bemessungsgrundlage", run.Due)
	}
}

// Ein Anlagegut mit Buchung verlässt das Verzeichnis nicht mehr über das
// Löschen, sondern nur über einen Abgang.
func TestDeleteRefusesBookedAssets(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	asset := env.machine(t, svc)

	if err := svc.Delete(ctx, asset.ID); err != nil {
		t.Fatalf("ein Anlagegut ohne Buchung muss löschbar sein: %v", err)
	}

	second := env.machine(t, svc)
	if _, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026}); err != nil {
		t.Fatalf("AfA buchen: %v", err)
	}
	if err := svc.Delete(ctx, second.ID); err == nil {
		t.Error("ein gebuchtes Anlagegut darf nicht gelöscht werden")
	}
}

// Die Festschreibung des Jahres fragt vorher, ob die AfA vollständig gebucht
// ist. Ohne diese Prüfung schlösse das Jahr mit fehlender Abschlussbuchung.
func TestPendingDepreciationReportsWhatTheYearIsMissing(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	env.machine(t, svc)

	pending, err := svc.PendingDepreciation(ctx, 2026)
	if err != nil {
		t.Fatalf("Prüfung: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("erwartet ein offenes Anlagegut, bekommen %d", len(pending))
	}

	if _, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026}); err != nil {
		t.Fatalf("AfA buchen: %v", err)
	}
	pending, err = svc.PendingDepreciation(ctx, 2026)
	if err != nil {
		t.Fatalf("Prüfung nach dem Lauf: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("nach dem Lauf darf nichts mehr offen sein: %+v", pending)
	}
}

// Ein geringwertiges Wirtschaftsgut mit Sofortabzug läuft nicht über den
// Abschreibungslauf: sein Aufwand entsteht mit der Belegbuchung. Im Verzeichnis
// steht es trotzdem — ab 250 € verlangt § 6 Abs. 2 Satz 4 EStG das.
func TestImmediateWriteOffStaysOutOfTheDepreciationRun(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	gwg, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Notebook", Class: domain.AssetClassTangible, Account: "0670",
		DepreciationAccount: "6260", AcquisitionDate: "2026-04-20",
		AcquisitionCost: 78_000, Method: domain.DepreciationImmediate,
	})
	if err != nil {
		t.Fatalf("GWG konnte nicht erfasst werden: %v", err)
	}
	if gwg.BookValue != 0 || gwg.Accumulated != 78_000 {
		t.Errorf("Buchwert %s €, kumulierte AfA %s € — der Sofortabzug ist mit dem Zugang erledigt",
			gwg.BookValue, gwg.Accumulated)
	}

	run, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("Abschreibungslauf: %v", err)
	}
	for _, due := range run.Due {
		if due.AssetID == gwg.ID {
			t.Errorf("das GWG darf im Abschreibungslauf nicht auftauchen: %+v", due)
		}
	}
}

// Ein Wechsel der Methode zieht den Sofortabzug mit: sonst bliebe ein GWG mit
// einem Buchwert stehen, den es nicht hat, oder umgekehrt.
func TestChangingTheMethodKeepsTheImmediateWriteOffInStep(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	gwg, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Werkzeugkoffer", Class: domain.AssetClassTangible, Account: "0670",
		DepreciationAccount: "6260", AcquisitionDate: "2026-03-01",
		AcquisitionCost: 60_000, Method: domain.DepreciationImmediate,
	})
	if err != nil {
		t.Fatalf("GWG: %v", err)
	}

	// Betrag korrigiert: der Sofortabzug muss mitwandern.
	gwg.AcquisitionCost = 70_000
	updated, err := svc.Save(ctx, gwg)
	if err != nil {
		t.Fatalf("Betrag ändern: %v", err)
	}
	if updated.Cost != 70_000 || updated.BookValue != 0 {
		t.Errorf("AHK %s €, Buchwert %s € — erwartet 700,00 € und null", updated.Cost, updated.BookValue)
	}

	// Doch kein GWG: aktiviert und über die Nutzungsdauer abgeschrieben.
	updated.Method = domain.DepreciationLinear
	updated.Account = "0630"
	updated.DepreciationAccount = "6220"
	updated.UsefulLifeMonths = 60
	activated, err := svc.Save(ctx, updated)
	if err != nil {
		t.Fatalf("Methode ändern: %v", err)
	}
	if activated.Accumulated != 0 || activated.BookValue != 70_000 {
		t.Errorf("kumulierte AfA %s €, Buchwert %s € — der Sofortabzug muss verschwunden sein",
			activated.Accumulated, activated.BookValue)
	}
}

// Der Abgang eines einzelnen Wirtschaftsguts mindert den Sammelposten nicht
// (§ 6 Abs. 2a Satz 4 EStG). Ein Abgang des Postens selbst wäre deshalb falsch.
func TestPoolCannotBeDisposed(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	// Der Posten entsteht aus einzelnen Wirtschaftsgütern, von denen jedes für sich
	// die Wertgrenze einhält — die Summe darf darüber liegen.
	pool, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Sammelposten 2026", Class: domain.AssetClassTangible, Account: "0675",
		DepreciationAccount: "6264", AcquisitionDate: "2026-01-01",
		AcquisitionCost: 60_000, Method: domain.DepreciationPool, PoolYear: 2026,
	})
	if err != nil {
		t.Fatalf("Sammelposten: %v", err)
	}
	if _, err := svc.RecordCostAdjustment(ctx, CostAdjustmentRequest{
		AssetID: pool.ID, Date: "2026-04-01", Amount: 90_000, Note: "Regal",
	}); err != nil {
		t.Fatalf("Zugang zum Sammelposten: %v", err)
	}
	if _, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: pool.ID, Date: "2026-08-01", Kind: domain.DisposalScrapped,
	}); err == nil {
		t.Fatal("ein Sammelposten darf nicht abgehen")
	}
}

// Die Wertgrenzen des § 6 Abs. 2 und 2a EStG sind keine Empfehlung. Ohne diese
// Prüfung nähme das Verzeichnis einen Sofortabzug für eine 5.000-€-Maschine an.
func TestValueLimitsBindTheTreatment(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	_, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Fräsmaschine", Class: domain.AssetClassTangible, Account: "0670",
		DepreciationAccount: "6260", AcquisitionDate: "2026-03-01",
		AcquisitionCost: 500_000, Method: domain.DepreciationImmediate,
	})
	if err == nil {
		t.Fatal("ein Sofortabzug über der GWG-Grenze darf nicht durchgehen")
	}
	if !strings.Contains(err.Error(), "§ 6 Abs. 2 Satz 1 EStG") {
		t.Errorf("die Meldung sollte die Fundstelle nennen, lautet aber: %v", err)
	}

	if _, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Bürostuhl", Class: domain.AssetClassTangible, Account: "0675",
		DepreciationAccount: "6264", AcquisitionDate: "2026-03-01",
		AcquisitionCost: 20_000, Method: domain.DepreciationPool, PoolYear: 2026,
	}); err == nil {
		t.Error("unter der Sammelposten-Untergrenze gehört das Gut in den Sofortabzug")
	}
	if _, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Werkbank", Class: domain.AssetClassTangible, Account: "0675",
		DepreciationAccount: "6264", AcquisitionDate: "2026-03-01",
		AcquisitionCost: 150_000, Method: domain.DepreciationPool, PoolYear: 2026,
	}); err == nil {
		t.Error("über der Sammelposten-Obergrenze wird aktiviert")
	}
}

// § 6 Abs. 2a EStG kennt genau einen Sammelposten je Wirtschaftsjahr. Weitere
// Güter des Jahres kommen als Zugang hinein, nicht als zweiter Posten.
func TestOnlyOnePoolPerFiscalYear(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	pool := &domain.FixedAsset{
		Name: "Sammelposten 2026", Class: domain.AssetClassTangible, Account: "0675",
		DepreciationAccount: "6264", AcquisitionDate: "2026-02-01",
		AcquisitionCost: 60_000, Method: domain.DepreciationPool, PoolYear: 2026,
	}
	first, err := svc.Save(ctx, pool)
	if err != nil {
		t.Fatalf("erster Sammelposten: %v", err)
	}

	_, err = svc.Save(ctx, &domain.FixedAsset{
		Name: "Sammelposten 2026 (zweiter)", Class: domain.AssetClassTangible, Account: "0675",
		DepreciationAccount: "6264", AcquisitionDate: "2026-06-01",
		AcquisitionCost: 40_000, Method: domain.DepreciationPool, PoolYear: 2026,
	})
	if err == nil {
		t.Fatal("ein zweiter Sammelposten desselben Jahres darf nicht entstehen")
	}
	if !strings.Contains(err.Error(), first.InventoryNumber) {
		t.Errorf("die Meldung sollte den bestehenden Posten nennen, lautet aber: %v", err)
	}

	// Der richtige Weg: als Zugang in den bestehenden Posten.
	updated, err := svc.RecordCostAdjustment(ctx, CostAdjustmentRequest{
		AssetID: first.ID, Date: "2026-06-01", Amount: 40_000, Note: "Regal",
	})
	if err != nil {
		t.Fatalf("Zugang zum Sammelposten: %v", err)
	}
	if updated.Cost != 100_000 {
		t.Errorf("Sammelposten %s € — erwartet 1.000,00 €", updated.Cost)
	}

	// Auch der Zugang hält die Wertgrenze je Wirtschaftsgut ein.
	if _, err := svc.RecordCostAdjustment(ctx, CostAdjustmentRequest{
		AssetID: first.ID, Date: "2026-07-01", Amount: 150_000, Note: "Werkbank",
	}); err == nil {
		t.Error("ein Gut über 1.000 € gehört nicht in den Sammelposten")
	}
}

// Die Maske rechnet den Plan nicht selbst — sie fragt dieselbe Rechnung, die
// später auch bucht.
func TestPreviewPlanAnswersBeforeTheAssetExists(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)

	rows, err := svc.PreviewPlan(context.Background(), PlanRequest{
		AcquisitionDate:  "2026-09-15",
		Cost:             1_200_000,
		UsefulLifeMonths: 48,
		Method:           domain.DepreciationLinear,
	})
	if err != nil {
		t.Fatalf("Planvorschau: %v", err)
	}
	if len(rows) != 5 || rows[0].Amount != 100_000 {
		t.Fatalf("Vorschau %+v — erwartet fünf Jahre, im ersten 1.000,00 €", rows)
	}
}

// Die Obergrenze der Zuschreibung steht in der Detailansicht, bevor jemand mehr
// eingibt.
func TestDetailCarriesTheWriteUpCeiling(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	share, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Anteile Musterwerk GmbH", Class: domain.AssetClassFinancial,
		Account: "0850", AcquisitionDate: "2026-02-01", AcquisitionCost: 5_000_000,
		Method: domain.DepreciationNone,
	})
	if err != nil {
		t.Fatalf("Finanzanlage: %v", err)
	}
	if _, err := svc.BookImpairment(ctx, ImpairmentRequest{
		AssetID: share.ID, Date: "2026-06-30", Amount: 500_000,
		Permanent: true, Reason: "Verlustjahr",
	}); err != nil {
		t.Fatalf("außerplanmäßige Abschreibung: %v", err)
	}

	detail, err := svc.Get(ctx, share.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if detail.WriteUpCeiling != 500_000 {
		t.Errorf("Obergrenze %s € — erwartet 5.000,00 €", detail.WriteUpCeiling)
	}
}

// Eine Erweiterung ändert die Vergangenheit nicht.
//
// Die Maschine von 2024 ist zwei Jahre abgeschrieben, als 2026 ein Zusatzmodul
// dazukommt: Restbuchwert 6.000 € plus 2.400 € verteilen sich auf die 24
// Restmonate. Den Plan von vorn zu rechnen — der naheliegende Fehler — würde
// behaupten, 2024 und 2025 sei zu wenig abgeschrieben worden.
func TestSubsequentCostSpreadsOverTheRemainingLife(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	repo := repository.NewAssetRepository(env.db)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Fräsmaschine", Class: domain.AssetClassTangible, Account: "0440",
		DepreciationAccount: "6220", AcquisitionDate: "2024-01-01",
		AcquisitionCost: 1_200_000, UsefulLifeMonths: 48, Method: domain.DepreciationLinear,
	})
	if err != nil {
		t.Fatalf("Anlagegut: %v", err)
	}
	for _, year := range []int{2024, 2025} {
		if err := repo.AddMovement(ctx, &domain.AssetMovement{
			AssetID: asset.ID, Kind: domain.AssetMovementDepreciation,
			Date: "2024-12-31", FiscalYear: year, DepreciationAmount: 300_000,
		}); err != nil {
			t.Fatalf("AfA %d: %v", year, err)
		}
	}

	if _, err := svc.RecordCostAdjustment(ctx, CostAdjustmentRequest{
		AssetID: asset.ID, Date: "2026-03-01", Amount: 240_000, Note: "Zusatzmodul",
	}); err != nil {
		t.Fatalf("Erweiterung: %v", err)
	}

	run, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("Abschreibungslauf: %v", err)
	}
	if len(run.MissingPriorYears) > 0 {
		t.Errorf("für %v soll nichts fehlen — die Erweiterung wirkt ab ihrem eigenen Jahr",
			run.MissingPriorYears)
	}
	if len(run.Due) != 1 || run.Due[0].Due != 420_000 {
		t.Fatalf("fällige AfA %+v — erwartet 4.200,00 € (6.000 + 2.400 auf 24 Monate)", run.Due)
	}

	detail, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	var total domain.Cents
	for _, row := range detail.Schedule {
		total += row.Amount
		if row.FiscalYear < 2026 && row.Amount != 300_000 {
			t.Errorf("%d: %s € — die gebuchten Jahre bleiben, wie sie sind", row.FiscalYear, row.Amount)
		}
	}
	if total != 1_440_000 {
		t.Errorf("Summe des Plans %s € — erwartet die vollen 14.400,00 €", total)
	}
	if detail.Schedule[len(detail.Schedule)-1].FiscalYear != 2027 {
		t.Errorf("letztes Jahr %d — ohne verlängerte Nutzungsdauer bleibt es bei 2027",
			detail.Schedule[len(detail.Schedule)-1].FiscalYear)
	}
}

// Verlängert die Erweiterung die Nutzungsdauer, wirkt auch das nach vorn.
func TestExtensionCanProlongTheRemainingLife(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	asset := env.machine(t, svc) // 12.000 €, 48 Monate ab 2026-01-10

	if _, err := svc.RecordCostAdjustment(ctx, CostAdjustmentRequest{
		AssetID: asset.ID, Date: "2026-07-01", Amount: 600_000,
		ExtendLifeMonths: 24, Note: "Generalüberholung",
	}); err != nil {
		t.Fatalf("Erweiterung: %v", err)
	}

	detail, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	last := detail.Schedule[len(detail.Schedule)-1]
	if last.FiscalYear != 2031 {
		t.Errorf("letztes Jahr %d — 48 Monate ab 2026 plus 24 enden 2031", last.FiscalYear)
	}
	var total domain.Cents
	for _, row := range detail.Schedule {
		total += row.Amount
	}
	if total != 1_800_000 {
		t.Errorf("Summe %s € — erwartet 18.000,00 €", total)
	}
	if last.ClosingBookValue != 0 {
		t.Errorf("Restbuchwert am Ende %s € — erwartet null", last.ClosingBookValue)
	}
}

// In den Sammelposten kommen nur Güter seines eigenen Wirtschaftsjahres, und ein
// GWG bleibt auch mit Erweiterung eines.
func TestAdditionsRespectPoolYearAndGWGLimit(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	pool, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Sammelposten 2026", Class: domain.AssetClassTangible, Account: "0675",
		DepreciationAccount: "6264", AcquisitionDate: "2026-02-01",
		AcquisitionCost: 60_000, Method: domain.DepreciationPool, PoolYear: 2026,
	})
	if err != nil {
		t.Fatalf("Sammelposten: %v", err)
	}
	if _, err := svc.RecordCostAdjustment(ctx, CostAdjustmentRequest{
		AssetID: pool.ID, Date: "2027-02-01", Amount: 50_000, Note: "Regal",
	}); err == nil {
		t.Error("ein Gut aus 2027 gehört nicht in den Sammelposten 2026")
	}

	// Das Notebook liegt bewusst in einem anderen Wirtschaftsjahr als der
	// Sammelposten: § 6 Abs. 2a Satz 5 EStG verlangt die einheitliche Ausübung
	// des Wahlrechts innerhalb eines Wirtschaftsjahres, und ein Sofortabzug
	// neben einem Sammelposten desselben Jahres wird deshalb zurückgewiesen.
	// Geprüft wird hier die Wertgrenze und nicht die Einheitlichkeit.
	gwg, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Notebook", Class: domain.AssetClassTangible, Account: "0670",
		DepreciationAccount: "6260", AcquisitionDate: "2027-04-20",
		AcquisitionCost: 78_000, Method: domain.DepreciationImmediate,
	})
	if err != nil {
		t.Fatalf("GWG: %v", err)
	}
	if _, err := svc.RecordCostAdjustment(ctx, CostAdjustmentRequest{
		AssetID: gwg.ID, Date: "2027-05-02", Amount: 10_000, Note: "Dockingstation",
	}); err == nil {
		t.Error("zusammen über der GWG-Grenze war der Sofortabzug nie zulässig")
	}
}

// Die Fertigstellung bucht um und lässt erst dann die Abschreibung beginnen.
func TestTransferCompletesAnAssetUnderConstruction(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	aib, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Fertigungslinie", Class: domain.AssetClassTangible, Account: "0700",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 6_000_000,
		Method: domain.DepreciationNone,
	})
	if err != nil {
		t.Fatalf("Anlage im Bau: %v", err)
	}
	if _, err := svc.RecordCostAdjustment(ctx, CostAdjustmentRequest{
		AssetID: aib.ID, Date: "2026-04-01", Amount: 2_000_000, Note: "zweite Rate",
	}); err != nil {
		t.Fatalf("weitere Anzahlung: %v", err)
	}

	// Solange sie nicht fertig ist, wird nichts abgeschrieben.
	run, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("Abschreibungslauf: %v", err)
	}
	if len(run.Due) != 0 {
		t.Errorf("eine Anlage im Bau wird nicht abgeschrieben: %+v", run.Due)
	}

	done, err := svc.Transfer(ctx, TransferRequest{
		AssetID: aib.ID, Date: "2026-07-01", Account: "0440",
		DepreciationAccount: "6220", Method: domain.DepreciationLinear,
		UsefulLifeMonths: 96, Note: "Abnahme erfolgt",
	})
	if err != nil {
		t.Fatalf("Fertigstellung: %v", err)
	}
	if done.Account != "0440" || done.InServiceDate != "2026-07-01" {
		t.Errorf("nach der Umbuchung: Konto %s, betriebsbereit am %s", done.Account, done.InServiceDate)
	}
	if done.Cost != 8_000_000 || done.BookValue != 8_000_000 {
		t.Errorf("AHK %s €, Buchwert %s € — die Umbuchung ändert den Wert nicht",
			done.Cost, done.BookValue)
	}

	// Die AfA läuft ab der Betriebsbereitschaft, nicht ab der ersten Anzahlung:
	// sechs Monate von 96, also die Hälfte eines Jahresbetrags von 1.000.000.
	run, err = svc.Run(ctx)
	if err != nil {
		t.Fatalf("Abschreibungslauf nach der Fertigstellung: %v", err)
	}
	if len(run.Due) != 1 || run.Due[0].Due != 500_000 || run.Due[0].Months != 6 {
		t.Fatalf("fällige AfA %+v — erwartet 5.000,00 € für sechs Monate", run.Due)
	}

	// Der Anlagenspiegel zeigt beide Positionen: 0700 gibt ab, 0440 nimmt auf.
	spiegel, err := svc.Anlagenspiegel(ctx)
	if err != nil {
		t.Fatalf("Anlagenspiegel: %v", err)
	}
	var out, in *domain.AnlagenspiegelRow
	for i := range spiegel.Rows {
		switch spiegel.Rows[i].Account {
		case "0700":
			out = &spiegel.Rows[i]
		case "0440":
			in = &spiegel.Rows[i]
		}
	}
	if out == nil || in == nil {
		t.Fatalf("erwartet je eine Zeile für 0700 und 0440, bekommen %+v", spiegel.Rows)
	}
	if out.Additions != 8_000_000 || out.Transfers != -8_000_000 || out.CostClosing != 0 {
		t.Errorf("Anlage im Bau: Zugänge %s €, Umbuchungen %s €, Ende %s €",
			out.Additions, out.Transfers, out.CostClosing)
	}
	if in.Transfers != 8_000_000 || in.CostClosing != 8_000_000 {
		t.Errorf("Maschinen: Umbuchungen %s €, Ende %s €", in.Transfers, in.CostClosing)
	}
	if spiegel.Totals.Transfers != 0 {
		t.Errorf("über alle Positionen summieren sich Umbuchungen zu null, hier %s €",
			spiegel.Totals.Transfers)
	}
}

// Umgebucht wird von den Konten der Anlagen im Bau — ein Kontowechsel an einer
// laufenden Anlage ist eine Korrektur und keine Fertigstellung.
func TestTransferOnlyFromConstructionAccounts(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	machine := env.machine(t, svc)

	if _, err := svc.Transfer(ctx, TransferRequest{
		AssetID: machine.ID, Date: "2026-07-01", Account: "0630",
		DepreciationAccount: "6220", Method: domain.DepreciationLinear, UsefulLifeMonths: 48,
	}); err == nil {
		t.Fatal("von einem laufenden Anlagekonto wird nicht umgebucht")
	}
}

// Der Teilabgang ist bei Finanzanlagen der Normalfall: eine Tranche wird
// verkauft, der Rest bleibt im Bestand.
func TestPartialDisposalOfAFinancialAsset(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	share, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Anleihe 2031", Class: domain.AssetClassFinancial, Account: "0920",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 2_500_000,
		Method: domain.DepreciationNone, Identifier: "DE000A2LQ5H0",
	})
	if err != nil {
		t.Fatalf("Finanzanlage: %v", err)
	}

	preview, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: share.ID, Date: "2026-09-30", Kind: domain.DisposalSale,
		Proceeds: 1_100_000, CostShare: 1_000_000,
		TaxTreatment: domain.TaxTreatmentExempt,
		Settlement:   SettlementPaid, PaymentAccount: "1800",
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if !preview.Partial || preview.BookValue != 1_000_000 || preview.Result != 100_000 {
		t.Fatalf("Vorschau %+v — erwartet Teilabgang, Restbuchwert 10.000 €, Gewinn 1.000 €", preview)
	}
	if preview.Accounts.Revenue != "4851" || preview.Accounts.BookValue != "4857" {
		t.Errorf("Konten %+v — erwartet 4851 und 4857 bei Buchgewinn einer Finanzanlage",
			preview.Accounts)
	}

	result, err := svc.Dispose(ctx, DisposalRequest{
		AssetID: share.ID, Date: "2026-09-30", Kind: domain.DisposalSale,
		Proceeds: 1_100_000, CostShare: 1_000_000,
		TaxTreatment: domain.TaxTreatmentExempt,
		Settlement:   SettlementPaid, PaymentAccount: "1800", Note: "40 von 100 Stück",
	})
	if err != nil {
		t.Fatalf("Teilabgang: %v", err)
	}
	if result.Asset.IsDisposed() {
		t.Error("nach einem Teilabgang bleibt das Anlagegut im Bestand")
	}
	if result.Asset.Cost != 1_500_000 || result.Asset.BookValue != 1_500_000 {
		t.Errorf("Rest: AHK %s €, Buchwert %s € — erwartet je 15.000,00 €",
			result.Asset.Cost, result.Asset.BookValue)
	}

	// Ein zweiter Teilabgang über den Rest hinaus geht nicht.
	if _, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: share.ID, Date: "2026-10-30", Kind: domain.DisposalSale,
		Proceeds: 100, CostShare: 2_000_000, TaxTreatment: domain.TaxTreatmentExempt,
		Settlement: SettlementPaid, PaymentAccount: "1800",
	}); err == nil {
		t.Error("mehr als der Rest kann nicht abgehen")
	}
}

// Eine außerplanmäßige Abschreibung wandert beim Teilabgang anteilig mit hinaus.
func TestPartialDisposalCarriesItsShareOfTheImpairment(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	share, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Beteiligung Süd GmbH", Class: domain.AssetClassFinancial, Account: "0850",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 4_000_000,
		Method: domain.DepreciationNone,
	})
	if err != nil {
		t.Fatalf("Finanzanlage: %v", err)
	}
	if _, err := svc.BookImpairment(ctx, ImpairmentRequest{
		AssetID: share.ID, Date: "2026-06-30", Amount: 800_000,
		Permanent: true, Reason: "Dauerhafter Wertverlust",
	}); err != nil {
		t.Fatalf("außerplanmäßige Abschreibung: %v", err)
	}

	// Ein Viertel geht ab: 10.000 € AHK und 2.000 € der Abschreibung.
	result, err := svc.Dispose(ctx, DisposalRequest{
		AssetID: share.ID, Date: "2026-11-30", Kind: domain.DisposalSale,
		Proceeds: 900_000, CostShare: 1_000_000, TaxTreatment: domain.TaxTreatmentExempt,
		Settlement: SettlementPaid, PaymentAccount: "1800",
	})
	if err != nil {
		t.Fatalf("Teilabgang: %v", err)
	}
	if result.Asset.Cost != 3_000_000 || result.Asset.Accumulated != 600_000 {
		t.Errorf("Rest: AHK %s €, kumulierte Abschreibung %s € — erwartet 30.000 € und 6.000 €",
			result.Asset.Cost, result.Asset.Accumulated)
	}
	if result.Asset.BookValue != 2_400_000 {
		t.Errorf("Restbuchwert %s € — erwartet 24.000,00 €", result.Asset.BookValue)
	}
}

// Bei Sachanlagen gibt es keinen Teilabgang: ein halber Pkw geht nicht ab.
func TestPartialDisposalIsRefusedForTangibleAssets(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	machine := env.machine(t, svc)

	if _, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: machine.ID, Date: "2026-06-30", Kind: domain.DisposalSale,
		Proceeds: 100_000, CostShare: 200_000, TaxTreatment: domain.TaxTreatmentDomestic,
		TaxRate: domain.TaxRateStandard, Settlement: SettlementPaid, PaymentAccount: "1800",
	}); err == nil {
		t.Fatal("ein Teilabgang einer Sachanlage muss abgelehnt werden")
	}
}

// Die Sonderabschreibung des § 7g Abs. 5 EStG wird nicht mehr gebucht.
//
// Seit dem BilMoG ist § 254 HGB entfallen, und § 253 HGB regelt die
// handelsrechtliche Bewertung abschließend: eine Abschreibung, die allein
// steuerlich begründet ist, hat in der Handelsbilanz nichts verloren. Sie wird
// deshalb als steuerlicher Wert an der Bewegung geführt — der Buchwert bleibt
// der handelsrechtliche, und die Differenz erscheint im Verzeichnis nach
// § 5 Abs. 1 Satz 2 EStG.
func TestSpecialDepreciationIsTaxOnlyAndNotBooked(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name:                "Fertigungsroboter",
		Class:               domain.AssetClassTangible,
		Account:             "0440",
		DepreciationAccount: "6220",
		AcquisitionDate:     "2026-01-05",
		AcquisitionCost:     10_000_000, // 100.000,00 €
		UsefulLifeMonths:    120,
		Method:              domain.DepreciationLinear,
		SpecialPermille:     400,
		SpecialYears:        5,
		SpecialReason:       "Gewinn 2025: 140.000 €; ausschließlich betriebliche Nutzung",
	})
	if err != nil {
		t.Fatalf("Anlagegut mit Sonderabschreibung: %v", err)
	}
	if asset.SpecialAccount != "6241" {
		t.Errorf("Konto der Sonderabschreibung %q — erwartet 6241", asset.SpecialAccount)
	}

	run, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("Abschreibungslauf: %v", err)
	}
	if len(run.Due) != 1 {
		t.Fatalf("erwartet eine fällige Zeile, bekommen %d", len(run.Due))
	}
	due := run.Due[0]
	if due.Due != 1_000_000 || due.SpecialDue != 800_000 {
		t.Fatalf("fällig: planmäßig %s €, Sonderabschreibung %s € — erwartet 10.000,00 € und 8.000,00 €",
			due.Due, due.SpecialDue)
	}

	result, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026})
	if err != nil {
		t.Fatalf("AfA buchen: %v", err)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("erwartet eine Buchung, bekommen %d", len(result.Entries))
	}
	lines := map[string]domain.Cents{}
	for _, l := range result.Entries[0].Lines {
		if l.Side == domain.SideDebit {
			lines[l.Account] += l.Amount
		} else {
			lines["haben:"+l.Account] += l.Amount
		}
	}
	if lines["6220"] != 1_000_000 || lines["haben:0440"] != 1_000_000 {
		t.Errorf("Buchung %+v — erwartet 6220 an 0440 über 10.000,00 €", lines)
	}
	if lines["6241"] != 0 {
		t.Errorf("die Sonderabschreibung steht mit %s € im Journal; sie darf handelsrechtlich "+
			"nicht gebucht werden", lines["6241"])
	}

	reloaded, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if reloaded.Asset.BookValue != 9_000_000 {
		t.Errorf("Buchwert %s € — erwartet 90.000,00 €: der handelsrechtliche Buchwert kennt nur "+
			"die planmäßige AfA", reloaded.Asset.BookValue)
	}
	if reloaded.Schedule[0].SpecialBooked != 800_000 || reloaded.Schedule[0].SpecialDue != 0 {
		t.Errorf("2026 im Plan: %s € gebucht, %s € offen — erwartet 8.000,00 € und null",
			reloaded.Schedule[0].SpecialBooked, reloaded.Schedule[0].SpecialDue)
	}
}

// Der handelsrechtliche Plan läuft neben der Sonderabschreibung unverändert
// weiter: über die Nutzungsdauer summiert sich die planmäßige AfA auf die vollen
// Anschaffungskosten, und der Buchwert endet auf null (§ 253 HGB).
func TestCommercialPlanIsUntouchedBySpecialDepreciation(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name:                "Fertigungsroboter",
		Class:               domain.AssetClassTangible,
		Account:             "0440",
		DepreciationAccount: "6220",
		AcquisitionDate:     "2026-01-05",
		AcquisitionCost:     10_000_000, // 100.000,00 €
		UsefulLifeMonths:    120,
		Method:              domain.DepreciationLinear,
		SpecialPermille:     400,
		SpecialYears:        5,
		SpecialReason:       "Gewinn 2025: 140.000 €; ausschließlich betriebliche Nutzung",
	})
	if err != nil {
		t.Fatalf("Anlagegut: %v", err)
	}
	detail, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if len(detail.Schedule) != 10 {
		t.Fatalf("erwartet zehn Planjahre, bekommen %d", len(detail.Schedule))
	}
	var planned domain.Cents
	for _, row := range detail.Schedule {
		planned += row.Amount
	}
	if planned != 10_000_000 {
		t.Errorf("Summe der planmäßigen AfA %s € — erwartet die vollen Anschaffungskosten", planned)
	}
	last := detail.Schedule[len(detail.Schedule)-1]
	if last.ClosingBookValue != 0 {
		t.Errorf("Restbuchwert nach der Nutzungsdauer %s € — erwartet null", last.ClosingBookValue)
	}
	if last.FiscalYear != 2035 {
		t.Errorf("letztes Planjahr %d — erwartet 2035", last.FiscalYear)
	}
}

// Ist nur noch die Sonderabschreibung offen, entsteht keine Buchung — sie ist
// ein reiner Steuerwert. Der Lauf darf daran nicht scheitern.
func TestDepreciationRunHandlesTheTaxOnlyCase(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name:                "Fertigungsroboter",
		Class:               domain.AssetClassTangible,
		Account:             "0440",
		DepreciationAccount: "6220",
		AcquisitionDate:     "2026-01-05",
		AcquisitionCost:     10_000_000,
		UsefulLifeMonths:    120,
		Method:              domain.DepreciationLinear,
		SpecialPermille:     400,
		SpecialYears:        5,
		SpecialReason:       "Gewinn 2025: 140.000 €; ausschließlich betriebliche Nutzung",
	})
	if err != nil {
		t.Fatalf("Anlagegut: %v", err)
	}
	// Die planmäßige AfA des Jahres steht bereits in der Kartei; offen ist nur
	// noch die Sonderabschreibung.
	if err := repository.NewAssetRepository(env.db).AddMovement(ctx, &domain.AssetMovement{
		AssetID: asset.ID, Kind: domain.AssetMovementDepreciation, Account: asset.Account,
		Date: "2026-12-31", FiscalYear: 2026, DepreciationAmount: 1_000_000,
	}); err != nil {
		t.Fatalf("Bewegung: %v", err)
	}

	result, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026})
	if err != nil {
		t.Fatalf("Abschreibungslauf: %v", err)
	}
	if len(result.Entries) != 0 {
		t.Errorf("erwartet keine Buchung, bekommen %d", len(result.Entries))
	}
	if len(result.TaxOnly) != 1 {
		t.Fatalf("der Lauf nennt die nur steuerliche Sonderabschreibung nicht: %+v", result)
	}
	detail, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if detail.Schedule[0].SpecialBooked != 800_000 {
		t.Errorf("erfasste Sonderabschreibung %s € — erwartet 8.000,00 €",
			detail.Schedule[0].SpecialBooked)
	}
	if detail.Asset.Vorabpauschalen != 0 {
		t.Errorf("die Sonderabschreibung erscheint mit %s € als Vorabpauschale",
			detail.Asset.Vorabpauschalen)
	}
}

// Beim Abgang wird die Sonderabschreibung des Jahres als steuerlicher Wert
// festgehalten, auch wenn handelsrechtlich nichts mehr nachzuholen ist. Ginge
// sie hier verloren, fehlte sie im Verzeichnis nach § 5 Abs. 1 Satz 2 EStG.
func TestDisposalKeepsTheSpecialDepreciationAsTaxValue(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name:                "Fertigungsroboter",
		Class:               domain.AssetClassTangible,
		Account:             "0440",
		DepreciationAccount: "6220",
		AcquisitionDate:     "2026-01-05",
		AcquisitionCost:     10_000_000,
		UsefulLifeMonths:    120,
		Method:              domain.DepreciationLinear,
		SpecialPermille:     400,
		SpecialYears:        5,
		SpecialReason:       "Gewinn 2025: 140.000 €; ausschließlich betriebliche Nutzung",
	})
	if err != nil {
		t.Fatalf("Anlagegut: %v", err)
	}
	// Die planmäßige AfA bis zum Abgangsmonat steht schon in der Kartei; offen
	// bleibt allein die Sonderabschreibung.
	if err := repository.NewAssetRepository(env.db).AddMovement(ctx, &domain.AssetMovement{
		AssetID: asset.ID, Kind: domain.AssetMovementDepreciation, Account: asset.Account,
		Date: "2026-06-30", FiscalYear: 2026, DepreciationAmount: 500_000,
	}); err != nil {
		t.Fatalf("Bewegung: %v", err)
	}

	preview, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-06-30", Kind: domain.DisposalScrapped,
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.CatchUpAmount != 0 || preview.SpecialCatchUp != 800_000 {
		t.Fatalf("nachzuholen: planmäßig %s €, Sonderabschreibung %s € — erwartet null und 8.000,00 €",
			preview.CatchUpAmount, preview.SpecialCatchUp)
	}

	if _, err := svc.Dispose(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-06-30", Kind: domain.DisposalScrapped,
	}); err != nil {
		t.Fatalf("Abgang: %v", err)
	}

	stored, err := repository.NewAssetRepository(env.db).FindByID(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Anlagegut laden: %v", err)
	}
	var special domain.Cents
	for _, m := range stored.Movements {
		if m.Kind == domain.AssetMovementSpecialDepreciation {
			special += m.TaxAmount
			if m.DepreciationAmount != 0 {
				t.Errorf("die Sonderabschreibung steht mit %s € als handelsrechtliche Abschreibung",
					m.DepreciationAmount)
			}
		}
	}
	if special != 800_000 {
		t.Errorf("festgehaltene Sonderabschreibung %s € — erwartet 8.000,00 €", special)
	}
}

// Der Migrationshinweis: Sonderabschreibungen, die vor dieser Fassung gebucht
// wurden, bleiben stehen — die Anlagenseite nennt sie.
func TestLegacySpecialDepreciationsAreListed(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name:                "Fertigungsroboter",
		Class:               domain.AssetClassTangible,
		Account:             "0440",
		DepreciationAccount: "6220",
		AcquisitionDate:     "2026-01-05",
		AcquisitionCost:     10_000_000,
		UsefulLifeMonths:    120,
		Method:              domain.DepreciationLinear,
		SpecialPermille:     400,
		SpecialYears:        5,
		SpecialReason:       "Gewinn 2025: 140.000 €",
	})
	if err != nil {
		t.Fatalf("Anlagegut: %v", err)
	}

	notice, err := svc.LegacySpecialDepreciations(ctx)
	if err != nil {
		t.Fatalf("Migrationshinweis: %v", err)
	}
	if len(notice.Rows) != 0 {
		t.Errorf("ohne alte Buchung darf der Hinweis leer sein, hat aber %d Zeilen", len(notice.Rows))
	}

	// Eine Bewegung aus der Zeit, in der die Sonderabschreibung noch gebucht
	// wurde: Betrag als Abschreibung, mit Buchungsverweis.
	entry, err := env.journal.Post(ctx, &domain.JournalEntry{
		BookingDate: "2025-12-31", DocumentDate: "2025-12-31",
		ServiceDateFrom: "2025-12-31", ServiceDateTo: "2025-12-31",
		Description: "Sonderabschreibung 2025", Source: domain.EntrySourceDepreciation,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6241", Amount: 800_000},
			{Side: domain.SideCredit, Account: "0440", Amount: 800_000},
		},
	})
	if err != nil {
		t.Fatalf("alte Buchung: %v", err)
	}
	if err := repository.NewAssetRepository(env.db).AddMovement(ctx, &domain.AssetMovement{
		AssetID: asset.ID, Kind: domain.AssetMovementSpecialDepreciation, Account: asset.Account,
		Date: "2025-12-31", FiscalYear: 2025, DepreciationAmount: 800_000,
		JournalEntryID: &entry.ID,
	}); err != nil {
		t.Fatalf("Bewegung: %v", err)
	}

	notice, err = svc.LegacySpecialDepreciations(ctx)
	if err != nil {
		t.Fatalf("Migrationshinweis: %v", err)
	}
	if len(notice.Rows) != 1 || notice.Total != 800_000 {
		t.Fatalf("Hinweis %+v — erwartet eine Zeile über 8.000,00 €", notice)
	}
	if notice.Rows[0].EntryNumber != entry.EntryNumber {
		t.Errorf("die Zeile nennt die Buchung %q — erwartet %q",
			notice.Rows[0].EntryNumber, entry.EntryNumber)
	}
	if !strings.Contains(notice.Note, "BilMoG") {
		t.Errorf("der Hinweis erklärt nicht, warum die Buchung nicht mehr entsteht: %q", notice.Note)
	}
}

// § 7g Abs. 6 EStG hängt an Sachverhalten, die keine Software kennt. Ohne
// festgehaltene Begründung entsteht die Sonderabschreibung nicht — wie bei der
// außerplanmäßigen Abschreibung ist der Grund Pflicht.
func TestSpecialDepreciationNeedsItsGrounds(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)

	_, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name:                "Fertigungsroboter",
		Class:               domain.AssetClassTangible,
		Account:             "0440",
		DepreciationAccount: "6220",
		AcquisitionDate:     "2026-01-05",
		AcquisitionCost:     10_000_000,
		UsefulLifeMonths:    120,
		Method:              domain.DepreciationLinear,
		SpecialPermille:     400,
		SpecialYears:        5,
	})
	if err == nil {
		t.Fatal("ohne Begründung darf die Sonderabschreibung nicht gespeichert werden")
	}
	if !strings.Contains(err.Error(), "7g Abs. 6") {
		t.Errorf("die Meldung nennt die Voraussetzung nicht: %v", err)
	}
}

// Ein Gebäude ist eine Sachanlage, aber unbeweglich. § 7g Abs. 5 EStG gilt für
// es nicht — und das muss beim Speichern auffallen, nicht bei der Betriebsprüfung.
func TestSpecialDepreciationIsRefusedForBuildings(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)

	_, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name:                "Lagerhalle",
		Class:               domain.AssetClassTangible,
		Account:             "0250",
		DepreciationAccount: "6221",
		AcquisitionDate:     "2026-01-05",
		AcquisitionCost:     50_000_000,
		UsefulLifeMonths:    396,
		Method:              domain.DepreciationLinear,
		SpecialPermille:     400,
		SpecialYears:        5,
		SpecialReason:       "Gewinn 2025 unter der Grenze",
	})
	if err == nil {
		t.Fatal("für ein Gebäude darf es keine Sonderabschreibung nach § 7g Abs. 5 EStG geben")
	}
	if !strings.Contains(err.Error(), "beweglich") {
		t.Errorf("die Meldung nennt den Grund nicht: %v", err)
	}
}

// Ist die Sonderabschreibung einmal gebucht, lässt sie sich nicht mehr
// umverteilen: das änderte den Plan eines abgeschlossenen Jahres.
func TestBookedSpecialDepreciationCannotBeRedistributed(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name:                "Fertigungsroboter",
		Class:               domain.AssetClassTangible,
		Account:             "0440",
		DepreciationAccount: "6220",
		AcquisitionDate:     "2026-01-05",
		AcquisitionCost:     10_000_000,
		UsefulLifeMonths:    120,
		Method:              domain.DepreciationLinear,
		SpecialPermille:     400,
		SpecialYears:        5,
		SpecialReason:       "Gewinn 2025: 140.000 €",
	})
	if err != nil {
		t.Fatalf("Anlagegut: %v", err)
	}
	if _, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026}); err != nil {
		t.Fatalf("AfA buchen: %v", err)
	}

	changed := *asset
	changed.SpecialYears = 2
	if _, err := svc.Save(ctx, &changed); err == nil {
		t.Fatal("eine gebuchte Sonderabschreibung darf nicht nachträglich umverteilt werden")
	}
}

// Erhaltungsaufwand gehört zum Anlagegut, ändert seinen Wert aber nicht. Genau
// das unterscheidet ihn von den nachträglichen Herstellungskosten.
func TestMaintenanceIsLinkedButDoesNotChangeTheBookValue(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	asset := env.machine(t, svc)
	before := asset.BookValue

	result, err := svc.BookMaintenance(ctx, MaintenanceRequest{
		AssetID: asset.ID, Date: "2026-05-04", Amount: 90_000,
		TaxRate: domain.TaxRateStandard, Settlement: SettlementPaid, PaymentAccount: "1800",
		Note: "Lagerschaden behoben, kein Mehrwert gegenüber dem ursprünglichen Zustand",
	})
	if err != nil {
		t.Fatalf("Erhaltungsaufwand: %v", err)
	}
	// Eine Maschine ist kein Gebäude: die Prüfung des 15-%-Rahmens ist nicht
	// einschlägig, und das Ergebnis sagt es statt zu schweigen.
	if result.NearAcquisition != nil && result.NearAcquisition.Applicable {
		t.Errorf("§ 6 Abs. 1 Nr. 1a EStG gilt für Gebäude: %+v", result.NearAcquisition)
	}
	lines := map[string]domain.Cents{}
	for _, l := range result.Entry.Lines {
		lines[string(l.Side)+":"+l.Account] += l.Amount
	}
	// Eine Maschine: Reparaturen und Instandhaltung von technischen Anlagen.
	if lines["S:6460"] != 90_000 {
		t.Errorf("Aufwandskonto %+v — erwartet 6460 mit 900,00 €", lines)
	}
	if lines["H:1800"] != 107_100 {
		t.Errorf("Zahlung %+v — erwartet 1.071,00 € brutto", lines)
	}

	detail, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if detail.Asset.BookValue != before {
		t.Errorf("Buchwert %s € — Erhaltungsaufwand darf ihn nicht ändern (vorher %s €)",
			detail.Asset.BookValue, before)
	}
	var found bool
	for _, m := range detail.Movements {
		if m.Kind == domain.AssetMovementMaintenance {
			found = true
			if m.CostAmount != 0 || m.DepreciationAmount != 0 {
				t.Errorf("die Bewegung trägt %s €/%s € — erwartet null in beiden Spalten",
					m.CostAmount, m.DepreciationAmount)
			}
			if m.JournalEntryID == nil {
				t.Error("die Bewegung ist mit keiner Buchung verknüpft")
			}
		}
	}
	if !found {
		t.Error("der Erhaltungsaufwand taucht beim Anlagegut nicht auf")
	}

	spiegel, err := svc.Anlagenspiegel(ctx)
	if err != nil {
		t.Fatalf("Anlagenspiegel: %v", err)
	}
	if spiegel.Totals.Additions != 1_200_000 {
		t.Errorf("Zugänge im Anlagenspiegel %s € — erwartet allein den Zugang von 12.000,00 €; "+
			"Erhaltungsaufwand gehört dort nicht hinein", spiegel.Totals.Additions)
	}
}

// Ohne festgehaltene Abgrenzung entsteht kein Erhaltungsaufwand: die
// Unterscheidung zur Erweiterung ist eine Einschätzung, keine Rechnung.
func TestMaintenanceNeedsItsReasoning(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	asset := env.machine(t, svc)

	_, err := svc.BookMaintenance(context.Background(), MaintenanceRequest{
		AssetID: asset.ID, Date: "2026-05-04", Amount: 90_000,
		Settlement: SettlementPaid, PaymentAccount: "1800",
	})
	if err == nil {
		t.Fatal("ohne Begründung darf der Erhaltungsaufwand nicht gebucht werden")
	}
}

// Eine Dividende ist Ertrag des Jahres und kein Rückfluss der
// Anschaffungskosten: der Buchwert der Beteiligung bleibt unberührt. Die
// einbehaltene Kapitalertragsteuer mindert den Zufluss, nicht den Ertrag.
func TestAssetIncomeKeepsTheBookValueAndSplitsTheWithholdingTax(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name:            "Beteiligung Musterwerk GmbH",
		Class:           domain.AssetClassFinancial,
		Account:         "0820",
		AcquisitionDate: "2026-02-01",
		AcquisitionCost: 5_000_000,
		Method:          domain.DepreciationNone,
		HoldingPermille: 300,
	})
	if err != nil {
		t.Fatalf("Beteiligung: %v", err)
	}

	entry, err := svc.BookAssetIncome(ctx, AssetIncomeRequest{
		AssetID: asset.ID, Date: "2026-07-01", Amount: 400_000, WithholdingTax: 100_000,
		Settlement: SettlementPaid, PaymentAccount: "1800", Note: "Gewinnausschüttung 2025",
	})
	if err != nil {
		t.Fatalf("Ertrag: %v", err)
	}
	lines := map[string]domain.Cents{}
	for _, l := range entry.Lines {
		lines[string(l.Side)+":"+l.Account] += l.Amount
	}
	if lines["H:7000"] != 400_000 {
		t.Errorf("Ertragskonto %+v — erwartet 7000 mit 4.000,00 €", lines)
	}
	if lines["S:7630"] != 100_000 {
		t.Errorf("Kapitalertragsteuer %+v — erwartet 7630 mit 1.000,00 €", lines)
	}
	if lines["S:1800"] != 300_000 {
		t.Errorf("Zufluss %+v — erwartet 3.000,00 € auf der Bank", lines)
	}

	detail, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if detail.Asset.BookValue != 5_000_000 {
		t.Errorf("Buchwert %s € — ein Ertrag mindert die Anschaffungskosten nicht", detail.Asset.BookValue)
	}
}

// Laufende Erträge werden für Sachanlagen nicht mit dem Anlagegut verknüpft —
// dort gibt es sie in diesem Sinn nicht.
func TestAssetIncomeIsOnlyForFinancialAssets(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	asset := env.machine(t, svc)

	_, err := svc.BookAssetIncome(context.Background(), AssetIncomeRequest{
		AssetID: asset.ID, Date: "2026-07-01", Amount: 10_000,
		Settlement: SettlementPaid, PaymentAccount: "1800",
	})
	if err == nil {
		t.Fatal("für eine Maschine darf kein laufender Ertrag verknüpft werden")
	}
}

// Wer eine Tranche verkauft, nennt Stücke und keinen Betrag. Buchfink rechnet
// daraus den Anteil der Anschaffungskosten — und der Rest bleibt mit seiner
// Stückzahl im Bestand.
func TestPartialDisposalByQuantity(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name:            "Anleihe Musterbank 2035",
		Class:           domain.AssetClassFinancial,
		Account:         "0920",
		AcquisitionDate: "2026-01-15",
		AcquisitionCost: 10_000_00, // 10.000,00 €
		Method:          domain.DepreciationNone,
		Identifier:      "DE000A1B2C3",
		Quantity:        100 * domain.UnitScale,
	})
	if err != nil {
		t.Fatalf("Wertpapier: %v", err)
	}
	if asset.UnitsHeld != 100*domain.UnitScale {
		t.Fatalf("Bestand %s Stück — erwartet 100", asset.UnitsHeld)
	}

	req := DisposalRequest{
		AssetID: asset.ID, Date: "2026-09-30", Kind: domain.DisposalSale,
		Quantity: 40 * domain.UnitScale, Proceeds: 4_500_00,
		Settlement: SettlementPaid, PaymentAccount: "1800",
	}
	preview, err := svc.PreviewDisposal(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if !preview.Partial {
		t.Fatal("40 von 100 Stück sind ein Teilabgang")
	}
	if preview.CostShare != 4_000_00 {
		t.Errorf("Anteil der Anschaffungskosten %s € — erwartet 4.000,00 €", preview.CostShare)
	}
	if preview.UnitsRemaining != 60*domain.UnitScale {
		t.Errorf("Restbestand %s Stück — erwartet 60", preview.UnitsRemaining)
	}

	if _, err := svc.Dispose(ctx, req); err != nil {
		t.Fatalf("Teilabgang: %v", err)
	}
	detail, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if detail.Asset.IsDisposed() {
		t.Error("nach einem Teilabgang bleibt das Wertpapier im Bestand")
	}
	if detail.Asset.UnitsHeld != 60*domain.UnitScale {
		t.Errorf("Bestand %s Stück — erwartet 60", detail.Asset.UnitsHeld)
	}
	if detail.Asset.Cost != 6_000_00 {
		t.Errorf("verbliebene Anschaffungskosten %s € — erwartet 6.000,00 €", detail.Asset.Cost)
	}
}

// Mehr Stücke als im Bestand können nicht abgehen.
func TestDisposalRefusesMoreUnitsThanHeld(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Anleihe", Class: domain.AssetClassFinancial, Account: "0920",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 10_000_00,
		Method: domain.DepreciationNone, Quantity: 100 * domain.UnitScale,
	})
	if err != nil {
		t.Fatalf("Wertpapier: %v", err)
	}
	_, err = svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-09-30", Kind: domain.DisposalSale,
		Quantity: 140 * domain.UnitScale, Proceeds: 1, Settlement: SettlementPaid, PaymentAccount: "1800",
	})
	if err == nil {
		t.Fatal("140 von 100 Stück dürfen nicht abgehen")
	}
}

// § 256a HGB rechnet zum Devisenkassamittelkurs des Stichtags um — nach oben
// aber nur bis zu den Anschaffungskosten (§ 253 Abs. 1 Satz 1 HGB).
func TestCurrencyValuationProposesButNeverExceedsCost(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	// 12.000 USD zu 1,20 USD/EUR sind 10.000,00 €.
	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "US-Staatsanleihe", Class: domain.AssetClassFinancial, Account: "0920",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 10_000_00,
		Method: domain.DepreciationNone, Currency: "USD", ForeignCost: 12_000_00,
	})
	if err != nil {
		t.Fatalf("Wertpapier: %v", err)
	}

	// Der Dollar fällt: 1,50 USD/EUR machen aus 12.000 USD nur noch 8.000,00 €.
	down, err := svc.ValuateCurrency(ctx, CurrencyValuationRequest{
		AssetID: asset.ID, Date: "2026-12-31", RatePerEuro: 1_500_000,
	})
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if down.Proposal != "impairment" || down.ProposedAmount != 2_000_00 {
		t.Errorf("Vorschlag %q über %s € — erwartet eine Abschreibung von 2.000,00 €",
			down.Proposal, down.ProposedAmount)
	}
	if down.AcquisitionRate != 1_200_000 {
		t.Errorf("Anschaffungskurs %d — erwartet 1,20 USD/EUR", down.AcquisitionRate)
	}

	// Der Dollar steigt über den Anschaffungskurs: zuzuschreiben ist trotzdem
	// nichts, solange nie abgeschrieben wurde.
	up, err := svc.ValuateCurrency(ctx, CurrencyValuationRequest{
		AssetID: asset.ID, Date: "2026-12-31", RatePerEuro: 1_000_000,
	})
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if up.Proposal != "none" || up.ProposedAmount != 0 {
		t.Errorf("Vorschlag %q über %s € — über die Anschaffungskosten hinaus darf nicht bewertet werden",
			up.Proposal, up.ProposedAmount)
	}

	// Nach einer außerplanmäßigen Abschreibung ist wieder Luft nach oben.
	if _, err := svc.BookImpairment(ctx, ImpairmentRequest{
		AssetID: asset.ID, Date: "2026-06-30", Amount: 2_000_00,
		Reason: "Kursverfall des US-Dollars",
	}); err != nil {
		t.Fatalf("außerplanmäßige Abschreibung: %v", err)
	}
	again, err := svc.ValuateCurrency(ctx, CurrencyValuationRequest{
		AssetID: asset.ID, Date: "2026-12-31", RatePerEuro: 1_000_000,
	})
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if again.Proposal != "write_up" || again.ProposedAmount != 2_000_00 {
		t.Errorf("Vorschlag %q über %s € — erwartet eine Zuschreibung von 2.000,00 € bis zu den "+
			"Anschaffungskosten", again.Proposal, again.ProposedAmount)
	}
}

// Eine Fremdwährung ohne den Betrag in dieser Währung ergibt keinen Kurs — und
// ohne Kurs ist am Stichtag nichts zu rechnen.
func TestForeignCurrencyNeedsItsAmount(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)

	_, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "US-Staatsanleihe", Class: domain.AssetClassFinancial, Account: "0920",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 10_000_00,
		Method: domain.DepreciationNone, Currency: "USD",
	})
	if err == nil {
		t.Fatal("ohne Fremdwährungsbetrag darf keine Währung gespeichert werden")
	}
}

// Ein Skonto auf eine Anlagenrechnung mindert die Anschaffungskosten und nicht
// den Aufwand (§ 255 Abs. 1 Satz 3 HGB). Auf 5736 gebucht wäre es ein Ertrag des
// Zahlungsjahres, und die AfA liefe weiter von einem Wert, den die Maschine nie
// gekostet hat. Die Steuerkorrektur nach § 17 Abs. 1 UStG bleibt davon unberührt.
func TestSkontoOnAnAssetInvoiceLowersTheAcquisitionCost(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	payments := env.payments(t)
	payments.SetAssetRegister(svc)
	ctx := context.Background()

	vendor := env.vendor(t, "Maschinenbau Muster", "DE", "")
	invoice, err := env.journal.Post(ctx, &domain.JournalEntry{
		BookingDate: "2026-03-01", DocumentDate: "2026-03-01",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01",
		Description: "Fräsmaschine", Source: domain.EntrySourceManual,
		DocumentNumber: "ER-2026-0001", TaxTreatment: domain.TaxTreatmentDomestic,
		ContactID: &vendor.ID,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "0440", Amount: 1_000_000},
			{Side: domain.SideDebit, Account: domain.AccountVorsteuer19, Amount: 190_000,
				TaxKey: "VST19", TaxBase: 1_000_000},
			{Side: domain.SideCredit, Account: vendor.LedgerAccount, ContactID: &vendor.ID, Amount: 1_190_000},
		},
	})
	if err != nil {
		t.Fatalf("Eingangsrechnung: %v", err)
	}

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Fräsmaschine", Class: domain.AssetClassTangible,
		Account: "0440", DepreciationAccount: "6220",
		AcquisitionDate: "2026-03-01", AcquisitionCost: 1_000_000,
		UsefulLifeMonths: 48, Method: domain.DepreciationLinear,
		AcquisitionEntryID: &invoice.ID,
	})
	if err != nil {
		t.Fatalf("Anlagegut: %v", err)
	}

	// 2 % Skonto auf 11.900,00 € brutto sind 238,00 € (200,00 netto + 38,00 Steuer).
	entry, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-10",
		Allocations: []AllocationRequest{{
			OpenItemEntryID:  invoice.ID,
			SettledAmount:    1_190_000,
			DifferenceKind:   domain.DifferenceSkonto,
			DifferenceAmount: 23_800,
		}},
	})
	if err != nil {
		t.Fatalf("Zahlung mit Skonto: %v", err)
	}

	lines := map[string]domain.Cents{}
	for _, l := range entry.Lines {
		lines[string(l.Side)+":"+l.Account] += l.Amount
	}
	if lines["H:0440"] != 20_000 {
		t.Errorf("Buchung %+v — das Skonto gehört mit 200,00 € im Haben auf das Anlagekonto 0440", lines)
	}
	if lines["H:5736"] != 0 {
		t.Errorf("auf 5736 stehen %s € — auf einer Anlagenrechnung gehört das Skonto dort nicht hin",
			lines["H:5736"])
	}
	if lines["H:"+domain.AccountVorsteuer19] != 3_800 {
		t.Errorf("Vorsteuerkorrektur %+v — § 17 Abs. 1 UStG verlangt sie unverändert", lines)
	}

	detail, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if detail.Asset.Cost != 980_000 {
		t.Errorf("Anschaffungskosten %s € — erwartet 9.800,00 € nach dem Skonto", detail.Asset.Cost)
	}
	var linked bool
	for _, m := range detail.Movements {
		if m.Kind == domain.AssetMovementCostReduction {
			linked = m.JournalEntryID != nil && *m.JournalEntryID == entry.ID
		}
	}
	if !linked {
		t.Error("die Minderung ist nicht mit der Zahlungsbuchung verknüpft")
	}

	// Und der Plan rechnet ab jetzt von der geminderten Bemessungsgrundlage.
	if detail.Schedule[0].Amount != 204_167 {
		t.Errorf("AfA 2026 %s € — erwartet 2.041,67 € (9.800,00 € über 48 Monate, zehn Monate)",
			detail.Schedule[0].Amount)
	}
}

// Auf jeder anderen Rechnung bleibt das Skonto, was es war: ein Ertrag auf 5736
// mit Vorsteuerkorrektur.
func TestSkontoOnAnOrdinaryInvoiceIsUnchanged(t *testing.T) {
	env := newTestEnv(t)
	payments := env.payments(t)
	payments.SetAssetRegister(env.assets(t))
	ctx := context.Background()

	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100_000, domain.TaxRateStandard)

	entry, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-05",
		Allocations: []AllocationRequest{{
			OpenItemEntryID:  invoice.ID,
			SettledAmount:    119_000,
			DifferenceKind:   domain.DifferenceSkonto,
			DifferenceAmount: 2_380,
		}},
	})
	if err != nil {
		t.Fatalf("Zahlung mit Skonto: %v", err)
	}
	var found bool
	for _, l := range entry.Lines {
		if l.Account == "5736" && l.Side == domain.SideCredit && l.Amount == 2_000 {
			found = true
		}
	}
	if !found {
		t.Errorf("ohne Anlagenbezug gehört das Skonto weiter auf 5736: %+v", entry.Lines)
	}
}

// Ein Vertrag zum Anlagegut ist kein Beleg: er trägt keine Belegnummer, gehört
// zu keinem Geschäftsjahr und wird nicht gebucht. Abgelegt wird er trotzdem auf
// demselben Weg — inhaltsadressiert und dedupliziert.
func TestAssetDocumentsAreKeptAlongsideTheAsset(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	svc.SetDocumentStore(receiptstore.New(t.TempDir()))
	ctx := context.Background()
	asset := env.machine(t, svc)

	path := filepath.Join(t.TempDir(), "kaufvertrag.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4 Kaufvertrag Fräsmaschine"), 0o600); err != nil {
		t.Fatalf("Testdatei: %v", err)
	}

	updated, err := svc.AttachDocument(ctx, AttachDocumentRequest{
		AssetID: asset.ID, Kind: domain.AssetDocContract, Path: path,
		Title: "Kaufvertrag Maschinenbau Muster", DocumentDate: "2026-01-08",
	})
	if err != nil {
		t.Fatalf("Dokument ablegen: %v", err)
	}
	if len(updated.Documents) != 1 {
		t.Fatalf("erwartet ein Dokument, bekommen %d", len(updated.Documents))
	}
	document := updated.Documents[0]
	if document.Kind != domain.AssetDocContract || document.MimeType != "application/pdf" {
		t.Errorf("Dokument %+v — erwartet einen Vertrag als PDF", document)
	}
	if document.DisplayTitle() != "Kaufvertrag Maschinenbau Muster" {
		t.Errorf("Titel %q", document.DisplayTitle())
	}

	content, err := svc.DocumentContent(ctx, document.ID)
	if err != nil {
		t.Fatalf("Inhalt lesen: %v", err)
	}
	if !content.Intact {
		t.Error("die abgelegte Datei stimmt nicht mehr mit ihrer Prüfsumme überein")
	}
	if string(content.Data) != "%PDF-1.4 Kaufvertrag Fräsmaschine" {
		t.Errorf("Inhalt %q — erwartet die abgelegte Datei", string(content.Data))
	}

	after, err := svc.RemoveDocument(ctx, asset.ID, document.ID)
	if err != nil {
		t.Fatalf("Dokument entfernen: %v", err)
	}
	if len(after.Documents) != 0 {
		t.Errorf("nach dem Entfernen bleiben %d Dokumente", len(after.Documents))
	}
	if _, err := svc.DocumentContent(ctx, document.ID); err == nil {
		t.Error("ein entferntes Dokument darf nicht mehr lesbar sein")
	}
}

// Ein Ablaufdatum, das niemand wieder liest, ist keine Angabe. Die Kartei
// beantwortet deshalb, was bis zu einem Stichtag ausläuft.
func TestExpiringDocumentsAreFound(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	svc.SetDocumentStore(receiptstore.New(t.TempDir()))
	ctx := context.Background()
	asset := env.machine(t, svc)

	dir := t.TempDir()
	for i, c := range []struct {
		name       string
		validUntil string
	}{
		{"police.pdf", "2026-12-31"},
		{"wartung.pdf", "2028-06-30"},
	} {
		path := filepath.Join(dir, c.name)
		if err := os.WriteFile(path, []byte(fmt.Sprintf("%%PDF-1.4 Dokument %d", i)), 0o600); err != nil {
			t.Fatalf("Testdatei: %v", err)
		}
		if _, err := svc.AttachDocument(ctx, AttachDocumentRequest{
			AssetID: asset.ID, Kind: domain.AssetDocInsurance, Path: path, ValidUntil: c.validUntil,
		}); err != nil {
			t.Fatalf("Dokument ablegen: %v", err)
		}
	}

	due, err := svc.ExpiringDocuments(ctx, "2026-12-31")
	if err != nil {
		t.Fatalf("Fristen: %v", err)
	}
	if len(due) != 1 || due[0].ValidUntil != "2026-12-31" {
		t.Fatalf("erwartet genau die zum 31.12.2026 auslaufende Police, bekommen %+v", due)
	}
	if due[0].InventoryNumber != asset.InventoryNumber {
		t.Errorf("die Frist zeigt auf %q statt auf %q", due[0].InventoryNumber, asset.InventoryNumber)
	}
}

// Eine Tilgung ist kein Verkauf. Zum Buchwert zurückgezahlt entsteht weder
// Erlös noch Buchgewinn — nur Geld gegen Ausleihung. Ein Erlöskonto stellte
// hier einen Umsatz in die GuV, den es nie gab.
func TestRepaymentBooksWithoutRevenue(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Darlehen an Werftgrund GmbH", Class: domain.AssetClassFinancial,
		Account: "0940", AcquisitionDate: "2024-03-01", AcquisitionCost: 5_000_000,
		Method: domain.DepreciationNone, MaturityDate: "2027-03-01",
	})
	if err != nil {
		t.Fatalf("Darlehen: %v", err)
	}

	// Teiltilgung über 20.000,00 € zum Buchwert.
	req := DisposalRequest{
		AssetID: asset.ID, Date: "2026-06-30", Kind: domain.DisposalRepayment,
		CostShare: 2_000_000, Proceeds: 2_000_000,
		Settlement: SettlementPaid, PaymentAccount: "1800",
	}
	preview, err := svc.PreviewDisposal(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Result != 0 || preview.Accounts.Revenue != "" {
		t.Errorf("zum Buchwert getilgt: Ergebnis %s €, Erlöskonto %q — erwartet null und keines",
			preview.Result, preview.Accounts.Revenue)
	}
	if preview.Tax != 0 {
		t.Errorf("eine Tilgung trägt keine Umsatzsteuer, hier %s €", preview.Tax)
	}

	result, err := svc.Dispose(ctx, req)
	if err != nil {
		t.Fatalf("Tilgung: %v", err)
	}
	lines := map[string]domain.Cents{}
	for _, l := range result.DisposalEntry.Lines {
		lines[string(l.Side)+":"+l.Account] += l.Amount
	}
	if len(lines) != 2 || lines["S:1800"] != 2_000_000 || lines["H:0940"] != 2_000_000 {
		t.Errorf("Buchung %+v — erwartet allein 1800 an 0940 über 20.000,00 €", lines)
	}
	if result.DisposalEntry.TaxTreatment != domain.TaxTreatmentNotTaxable {
		t.Errorf("Steuerfall %q — eine Rückzahlung ist kein Leistungsaustausch",
			result.DisposalEntry.TaxTreatment)
	}
	if result.Asset.IsDisposed() {
		t.Error("nach einer Teiltilgung bleibt das Darlehen im Bestand")
	}
	if result.Asset.Cost != 3_000_000 {
		t.Errorf("Restvaluta %s € — erwartet 30.000,00 €", result.Asset.Cost)
	}
}

// Getilgt wird eine Ausleihung. Eine Beteiligung wird verkauft.
func TestRepaymentIsOnlyForLoans(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Beteiligung", Class: domain.AssetClassFinancial, Account: "0820",
		AcquisitionDate: "2024-03-01", AcquisitionCost: 5_000_000, Method: domain.DepreciationNone,
	})
	if err != nil {
		t.Fatalf("Beteiligung: %v", err)
	}
	_, err = svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-06-30", Kind: domain.DisposalRepayment,
		Proceeds: 5_000_000, Settlement: SettlementPaid, PaymentAccount: "1800",
	})
	if err == nil {
		t.Fatal("eine Beteiligung wird nicht getilgt")
	}
}

// § 256a Satz 2 HGB nimmt Posten mit einer Restlaufzeit von höchstens einem
// Jahr vom Anschaffungskostenprinzip aus. Ein Darlehen, das in Monaten
// zurückfließt, weist den Kursgewinn deshalb voll aus — eine Beteiligung ohne
// Fälligkeit nicht.
func TestShortMaturityLiftsTheAcquisitionCostCeiling(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	// 12.000 USD zu 1,20 USD/EUR sind 10.000,00 €.
	loan, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Darlehen in USD", Class: domain.AssetClassFinancial, Account: "0940",
		AcquisitionDate: "2025-01-15", AcquisitionCost: 10_000_00,
		Method: domain.DepreciationNone, Currency: "USD", ForeignCost: 12_000_00,
		MaturityDate: "2027-03-31",
	})
	if err != nil {
		t.Fatalf("Darlehen: %v", err)
	}

	// Stichtag 31.12.2026: die Restlaufzeit beträgt drei Monate. Kurs 1,00
	// macht aus 12.000 USD 12.000,00 € — 2.000,00 € über den Anschaffungskosten.
	up, err := svc.ValuateCurrency(ctx, CurrencyValuationRequest{
		AssetID: loan.ID, Date: "2026-12-31", RatePerEuro: 1_000_000,
	})
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if !up.ShortTerm {
		t.Fatal("zum 31.12.2026 ist ein am 31.03.2027 fälliges Darlehen kurzfristig")
	}
	if up.Proposal != "write_up" || up.ProposedAmount != 2_000_00 {
		t.Errorf("Vorschlag %q über %s € — § 256a Satz 2 HGB hebt hier die Obergrenze auf",
			up.Proposal, up.ProposedAmount)
	}

	// Zwei Jahre vorher ist die Restlaufzeit länger als ein Jahr: dann greift
	// das Anschaffungskostenprinzip wieder.
	early, err := svc.ValuateCurrency(ctx, CurrencyValuationRequest{
		AssetID: loan.ID, Date: "2025-12-31", RatePerEuro: 1_000_000,
	})
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if early.ShortTerm || early.Proposal != "none" {
		t.Errorf("zum 31.12.2025 ist die Restlaufzeit über ein Jahr: Vorschlag %q über %s €",
			early.Proposal, early.ProposedAmount)
	}
}

// Der Kursverlust läuft über die Konten der Währungsumrechnung, nicht über die
// der außerplanmäßigen Abschreibung: sonst sähe er aus wie eine Wertminderung
// des Papiers selbst.
func TestCurrencyValuationBooksOnItsOwnAccounts(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "US-Anleihe", Class: domain.AssetClassFinancial, Account: "0920",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 10_000_00,
		Method: domain.DepreciationNone, Currency: "USD", ForeignCost: 12_000_00,
	})
	if err != nil {
		t.Fatalf("Anleihe: %v", err)
	}

	entry, err := svc.BookCurrencyValuation(ctx, CurrencyValuationRequest{
		AssetID: asset.ID, Date: "2026-12-31", RatePerEuro: 1_500_000,
	})
	if err != nil {
		t.Fatalf("Währungsumrechnung: %v", err)
	}
	lines := map[string]domain.Cents{}
	for _, l := range entry.Lines {
		lines[string(l.Side)+":"+l.Account] += l.Amount
	}
	if lines["S:6880"] != 2_000_00 || lines["H:0920"] != 2_000_00 {
		t.Errorf("Buchung %+v — erwartet 6880 an 0920 über 2.000,00 €", lines)
	}

	detail, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if detail.Asset.BookValue != 8_000_00 {
		t.Errorf("Buchwert %s € — erwartet 8.000,00 €", detail.Asset.BookValue)
	}
}

// Ein ETF steht in der Bilanz wie jedes andere Wertpapier. Steuerlich legt das
// InvStG zwei Rechnungen daneben: die Vorabpauschale wird über die Jahre
// angesetzt, ohne dass etwas gebucht wird, und beim Abgang zusammen mit der
// Teilfreistellung wieder abgezogen.
func TestFundDisposalShowsTheInvestmentTaxNote(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Stammdaten: %v", err)
	}
	// Die Anlegerstellung folgt sonst aus der Rechtsform; hier wird sie
	// ausdrücklich gesetzt, damit der Test nicht am Katalog hängt.
	cfg.InvestorOverride = domain.InvestorCorporate
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Stammdaten speichern: %v", err)
	}

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "MSCI-World-ETF", Class: domain.AssetClassFinancial, Account: "0900",
		AcquisitionDate: "2025-01-02", AcquisitionCost: 10_000_000, Method: domain.DepreciationNone,
		Identifier: "IE00B4L5Y983", FundClass: string(accounting.FundEquity),
		Quantity: 1000 * domain.UnitScale,
	})
	if err != nil {
		t.Fatalf("ETF: %v", err)
	}

	// Thesaurierend: 2025 entsteht eine Vorabpauschale.
	result, err := svc.Vorabpauschale(ctx, VorabpauschaleRequest{
		AssetID: asset.ID, Year: 2025, OpeningPrice: 10_000_000, ClosingPrice: 11_000_000,
		BasisPoints: 253, Record: true,
	})
	if err != nil {
		t.Fatalf("Vorabpauschale: %v", err)
	}
	// Erwerb im Januar: keine Kürzung.
	if result.Amount != 177_100 {
		t.Errorf("Vorabpauschale %s € — erwartet 1.771,00 €", result.Amount)
	}

	detail, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if detail.Asset.BookValue != 10_000_000 {
		t.Errorf("Buchwert %s € — die Vorabpauschale wird nicht gebucht", detail.Asset.BookValue)
	}
	if detail.Asset.Vorabpauschalen != 177_100 {
		t.Errorf("angesetzte Vorabpauschalen %s €", detail.Asset.Vorabpauschalen)
	}

	// Für dasselbe Jahr ein zweites Mal geht nicht.
	if _, err := svc.Vorabpauschale(ctx, VorabpauschaleRequest{
		AssetID: asset.ID, Year: 2025, OpeningPrice: 10_000_000, ClosingPrice: 11_000_000,
		BasisPoints: 253, Record: true,
	}); err == nil {
		t.Error("eine Vorabpauschale je Kalenderjahr, nicht zwei")
	}

	// Verkauf mit 20.000,00 € Buchgewinn.
	preview, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-09-30", Kind: domain.DisposalSale,
		Proceeds: 12_000_000, Settlement: SettlementPaid, PaymentAccount: "1800",
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Investment == nil {
		t.Fatal("zu einem Investmentanteil gehört die steuerliche Nebenrechnung")
	}
	note := preview.Investment
	if note.GrossAmount != 2_000_000 || note.Vorabpauschalen != 177_100 {
		t.Fatalf("Buchgewinn %s €, Vorabpauschalen %s €", note.GrossAmount, note.Vorabpauschalen)
	}
	// 20.000,00 − 1.771,00 = 18.229,00 €; davon 80 % steuerfrei.
	if note.Exemption.Permille != 800 {
		t.Errorf("Teilfreistellung %d Promille — bei einem KSt-Anleger sind es 80 %%",
			note.Exemption.Permille)
	}
	if note.ExemptAmount != 1_458_320 || note.TaxableAmount != 364_580 {
		t.Errorf("steuerfrei %s €, steuerpflichtig %s € — erwartet 14.583,20 € und 3.645,80 €",
			note.ExemptAmount, note.TaxableAmount)
	}
}

// Bei einer Personengesellschaft lässt die Rechtsform die Anlegerstellung
// offen: § 20 Abs. 3a InvStG stellt auf den Gesellschafter ab. Dann rechnet
// Buchfink keine Teilfreistellung, sondern sagt, was zu entscheiden ist.
func TestFundWithoutInvestorTypeStaysUnreduced(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Stammdaten: %v", err)
	}
	cfg.LegalForm = "GmbH & Co. KG"
	cfg.InvestorOverride = ""
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Stammdaten speichern: %v", err)
	}

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "MSCI-World-ETF", Class: domain.AssetClassFinancial, Account: "0900",
		AcquisitionDate: "2025-01-02", AcquisitionCost: 10_000_000, Method: domain.DepreciationNone,
		FundClass: string(accounting.FundEquity),
	})
	if err != nil {
		t.Fatalf("ETF: %v", err)
	}
	preview, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-09-30", Kind: domain.DisposalSale,
		Proceeds: 12_000_000, Settlement: SettlementPaid, PaymentAccount: "1800",
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Investment == nil || preview.Investment.ExemptionError == "" {
		t.Fatal("ohne Anlegerstellung muss die Nebenrechnung sagen, was fehlt")
	}
	if preview.Investment.ExemptAmount != 0 || preview.Investment.TaxableAmount != 2_000_000 {
		t.Errorf("steuerfrei %s €, steuerpflichtig %s € — erwartet null und den vollen Buchgewinn",
			preview.Investment.ExemptAmount, preview.Investment.TaxableAmount)
	}
}

// Für einen Einzeltitel gibt es weder Vorabpauschale noch Teilfreistellung.
func TestSingleSecurityHasNoInvestmentNote(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Anleihe", Class: domain.AssetClassFinancial, Account: "0920",
		AcquisitionDate: "2025-01-02", AcquisitionCost: 10_000_00, Method: domain.DepreciationNone,
	})
	if err != nil {
		t.Fatalf("Anleihe: %v", err)
	}
	if _, err := svc.Vorabpauschale(ctx, VorabpauschaleRequest{
		AssetID: asset.ID, Year: 2025, OpeningPrice: 10_000_00, ClosingPrice: 11_000_00,
		BasisPoints: 253,
	}); err == nil {
		t.Error("eine Einzelanleihe trägt keine Vorabpauschale")
	}
	preview, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-09-30", Kind: domain.DisposalSale,
		Proceeds: 11_000_00, Settlement: SettlementPaid, PaymentAccount: "1800",
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Investment != nil {
		t.Error("zu einem Einzeltitel gehört keine investmentsteuerliche Nebenrechnung")
	}
}

// Der Regelfall braucht keine zweite Angabe: aus der Rechtsform folgt die
// Anlegerstellung, und damit die Teilfreistellung.
func TestPartialExemptionFollowsFromTheLegalFormAlone(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()

	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Stammdaten: %v", err)
	}
	cfg.LegalForm = "Einzelunternehmen"
	cfg.InvestorOverride = ""
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Stammdaten speichern: %v", err)
	}

	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "MSCI-World-ETF", Class: domain.AssetClassFinancial, Account: "0900",
		AcquisitionDate: "2025-01-02", AcquisitionCost: 10_000_000, Method: domain.DepreciationNone,
		FundClass: string(accounting.FundEquity),
	})
	if err != nil {
		t.Fatalf("ETF: %v", err)
	}
	preview, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-09-30", Kind: domain.DisposalSale,
		Proceeds: 12_000_000, Settlement: SettlementPaid, PaymentAccount: "1800",
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Investment == nil || preview.Investment.ExemptionError != "" {
		t.Fatalf("aus der Rechtsform Einzelunternehmen folgt die Anlegerstellung: %+v", preview.Investment)
	}
	if preview.Investment.Exemption.Permille != 600 {
		t.Errorf("Teilfreistellung %d Promille — eine natürliche Person im Betriebsvermögen trägt 60 %%",
			preview.Investment.Exemption.Permille)
	}
}
