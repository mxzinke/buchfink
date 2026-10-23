package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

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
