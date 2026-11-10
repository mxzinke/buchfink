package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestTangiblePartialDisposalAndReversal(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	env.journal.AddEntryEffects(svc.EntryEffect())
	ctx := context.Background()
	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Vier Maschinen", Class: domain.AssetClassTangible,
		Account: "0440", DepreciationAccount: "6220", AcquisitionDate: "2026-01-10",
		AcquisitionCost: 1_200_000, UsefulLifeMonths: 48, Method: domain.DepreciationLinear,
		Quantity: 40_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.Dispose(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-06-30", Kind: domain.DisposalSale,
		Quantity: 10_000, Proceeds: 300_000, Settlement: SettlementPaid,
		PaymentAccount: "1800", TaxTreatment: domain.TaxTreatmentDomestic,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Gesamt-AfA 1.500 €, abgegangene AHK 3.000 €, Restbuchwert 2.625 €.
	if result.Asset.Cost != 900_000 || result.Asset.Accumulated != 112_500 || result.Asset.UnitsHeld != 30_000 {
		t.Fatalf("Restbestand: %+v", result.Asset)
	}
	run, err := svc.Run(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Due) != 1 || run.Due[0].Due != 112_500 {
		t.Fatalf("restliche AfA: %+v", run)
	}
	if _, err := env.journal.Reverse(ctx, result.DisposalEntry.ID, "Verkauf berichtigen"); err != nil {
		t.Fatal(err)
	}
	detail, err := svc.Get(ctx, 2026, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Asset.Cost != 1_200_000 || detail.Asset.Accumulated != 150_000 || detail.Asset.UnitsHeld != 40_000 {
		t.Fatalf("nach Storno: %+v", detail.Asset)
	}
}

func TestPoolItemDisposalLeavesPoolAndCanBeReversed(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	pool, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Sammelposten", Class: domain.AssetClassTangible, Account: "0675",
		DepreciationAccount: "6264", AcquisitionDate: "2026-01-01", AcquisitionCost: 60_000,
		Method: domain.DepreciationPool, PoolYear: 2026,
	})
	if err != nil {
		t.Fatal(err)
	}
	item := pool.Movements[0].ID
	result, err := svc.Dispose(ctx, DisposalRequest{
		AssetID: pool.ID, Date: "2026-06-30", Kind: domain.DisposalScrapped, PoolItemMovementID: item,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Asset.Cost != 60_000 || result.Asset.IsDisposed() || result.Asset.DueAmount != 12_000 {
		t.Fatalf("Sammelposten verändert: %+v", result.Asset)
	}
	var disposal uint
	for _, m := range result.Asset.Movements {
		if m.Kind == domain.AssetMovementDisposal {
			disposal = m.ID
		}
	}
	if _, err := svc.ReverseDisposal(ctx, pool.ID, disposal, "Irrtum"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReverseDisposal(ctx, pool.ID, disposal, "noch einmal"); err == nil {
		t.Fatal("doppeltes Storno")
	}
}

func TestTransferReversalRestoresDepreciationStart(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	env.journal.AddEntryEffects(svc.EntryEffect())
	ctx := context.Background()
	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Maschine im Bau", Class: domain.AssetClassTangible, Account: "0700",
		AcquisitionDate: "2026-01-01", AcquisitionCost: 1_200_000, Method: domain.DepreciationNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	completed, err := svc.Transfer(ctx, TransferRequest{
		AssetID: asset.ID, Date: "2026-07-01", Account: "0440", DepreciationAccount: "6220",
		Method: domain.DepreciationLinear, UsefulLifeMonths: 48, Note: "Abnahme",
	})
	if err != nil {
		t.Fatal(err)
	}
	if completed.DueAmount != 150_000 {
		t.Fatalf("AfA: %v", completed.DueAmount)
	}
	var entry uint
	for _, m := range completed.Movements {
		if m.Kind == domain.AssetMovementTransfer {
			entry = *m.JournalEntryID
		}
	}
	if _, err := env.journal.Reverse(ctx, entry, "Abnahme berichtigen"); err != nil {
		t.Fatal(err)
	}
	detail, err := svc.Get(ctx, 2026, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Asset.Account != "0700" || detail.Asset.Method != domain.DepreciationNone ||
		detail.Asset.InServiceDate != "" || detail.Asset.DueAmount != 0 || detail.Asset.Cost != 1_200_000 {
		t.Fatalf("nach Storno: %+v", detail.Asset)
	}
}

func TestImpairmentAfterPartialDisposalAndReversalOrder(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	env.journal.AddEntryEffects(svc.EntryEffect())
	ctx := context.Background()
	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Vier Maschinen", Class: domain.AssetClassTangible,
		Account: "0440", DepreciationAccount: "6220", AcquisitionDate: "2026-01-01",
		AcquisitionCost: 1_200_000, UsefulLifeMonths: 48, Method: domain.DepreciationLinear,
		Quantity: 40_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	disposal, err := svc.Dispose(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-06-30", Kind: domain.DisposalScrapped, Quantity: 10_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	impairment, err := svc.BookImpairment(ctx, ImpairmentRequest{
		AssetID: asset.ID, Date: "2026-09-30", Amount: 300_000, Permanent: true,
		Reason: "Schaden am Restbestand", Evidence: "Gutachten",
	})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := svc.Get(ctx, 2027, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range detail.Schedule {
		if row.FiscalYear == 2027 && row.Amount != 125_000 {
			t.Fatalf("AfA auf 3.750 € Restwert über 36 Monate: %v", row.Amount)
		}
	}
	if _, err := env.journal.Reverse(ctx, disposal.DisposalEntry.ID, "Abgang berichtigen"); err == nil {
		t.Fatal("Abgang trotz abhängiger Wertminderung storniert")
	}
	if _, err := env.journal.Reverse(ctx, impairment.ID, "Wertminderung berichtigen"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.journal.Reverse(ctx, disposal.DisposalEntry.ID, "Abgang berichtigen"); err != nil {
		t.Fatal(err)
	}
	detail, err = svc.Get(ctx, 2026, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Asset.Cost != 1_200_000 || detail.Asset.UnitsHeld != 40_000 || detail.Asset.Accumulated != 150_000 {
		t.Fatalf("Bestand nach beiden Stornos: %+v", detail.Asset)
	}
}

func TestDisposalFromBankTransaction(t *testing.T) {
	env := newTestEnv(t)
	env.paymentsWithBank(t)
	svc := env.assets(t)
	svc.SetBankTransactions(repository.NewBankRepository(env.db))
	env.journal.AddEntryEffects(svc.EntryEffect())
	ctx := context.Background()
	asset := env.machine(t, svc)
	bankID := env.bankLine(t, "2026-06-30", 1_428_000, "Käufer GmbH", "Maschinenkauf")
	req := DisposalRequest{
		AssetID: asset.ID, Date: "2026-06-30", Kind: domain.DisposalSale,
		Proceeds: 1_200_000, BankTxID: bankID, TaxTreatment: domain.TaxTreatmentDomestic,
	}
	wrong := req
	wrong.Date = "2026-05-31"
	if _, err := svc.Dispose(ctx, wrong); err == nil {
		t.Fatal("Abgangsdatum durch Zahlungstag überschrieben")
	}
	wrong = req
	wrong.Proceeds = 1_100_000
	if _, err := svc.Dispose(ctx, wrong); err == nil {
		t.Fatal("abweichenden Bruttoerlös angenommen")
	}
	result, err := svc.Dispose(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, result.DisposalEntry, []bookedLine{
		{domain.SideDebit, "1800", 1_428_000},
		{domain.SideCredit, "4845", 1_200_000},
		{domain.SideCredit, "3806", 228_000},
		{domain.SideDebit, "4855", 1_050_000},
		{domain.SideCredit, "0440", 1_050_000},
	})
	env.assertMatch(t, bankID, domain.MatchStatusMatched)
	statements := env.statements(t)
	statements.SetNotesSources(NotesSources{Assets: svc})
	fs, err := statements.Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatal(err)
	}
	if fs.Notes.Assets.Totals.Disposals != 1_200_000 || fs.Notes.Assets.Totals.DepreciationDisposal != 150_000 {
		t.Fatalf("Abgang im Anhang: %+v", fs.Notes.Assets.Totals)
	}
	csv, err := statements.ExportCSV(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csv, "0440.abgaenge") || !strings.Contains(csv, "12000,00") {
		t.Fatalf("Abgang fehlt im CSV: %s", csv)
	}
	var pdfSource strings.Builder
	writeTypstNotes(&pdfSource, fs)
	if !strings.Contains(pdfSource.String(), "Abgänge zu Anschaffungskosten") ||
		!strings.Contains(pdfSource.String(), "12.000,00") {
		t.Fatalf("Abgang fehlt im PDF-Quelltext: %s", pdfSource.String())
	}

	if _, err := env.journal.Reverse(ctx, result.DisposalEntry.ID, "Verkauf berichtigen"); err != nil {
		t.Fatal(err)
	}
	env.assertMatch(t, bankID, domain.MatchStatusUnmatched)
	fs, err = statements.Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatal(err)
	}
	if fs.Notes.Assets.Totals.Disposals != 0 || fs.Notes.Assets.Totals.BookValueClosing != 1_050_000 {
		t.Fatalf("Anhang nach Storno: %+v", fs.Notes.Assets.Totals)
	}
}

func TestWriteUpRestoresDepreciationBasis(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	asset := env.machine(t, svc)
	ctx := context.Background()
	if _, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BookImpairment(ctx, ImpairmentRequest{
		AssetID: asset.ID, Date: "2026-12-31", Amount: 300_000, Permanent: true,
		Reason: "Schaden", Evidence: "Gutachten",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2027}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BookWriteUp(ctx, WriteUpRequest{
		AssetID: asset.ID, Date: "2027-12-31", Amount: 200_000, Reason: "Erholung", Evidence: "Neues Gutachten",
	}); err != nil {
		t.Fatal(err)
	}
	detail, err := svc.Get(ctx, 2028, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Asset.BookValue != 600_000 || detail.Asset.DueAmount != 300_000 {
		t.Fatalf("Zuschreibung und folgende AfA: %+v", detail.Asset)
	}
}
