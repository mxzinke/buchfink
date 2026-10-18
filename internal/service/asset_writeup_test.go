package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

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
