package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
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

	pool, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Sammelposten 2026", Class: domain.AssetClassTangible, Account: "0675",
		DepreciationAccount: "6264", AcquisitionDate: "2026-01-01",
		AcquisitionCost: 420_000, Method: domain.DepreciationPool, PoolYear: 2026,
	})
	if err != nil {
		t.Fatalf("Sammelposten: %v", err)
	}
	if _, err := svc.PreviewDisposal(ctx, DisposalRequest{
		AssetID: pool.ID, Date: "2026-08-01", Kind: domain.DisposalScrapped,
	}); err == nil {
		t.Fatal("ein Sammelposten darf nicht abgehen")
	}
}
