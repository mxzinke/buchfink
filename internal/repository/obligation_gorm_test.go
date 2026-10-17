package repository

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestObligationRepository(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = CloseDB(db) })
	repo := NewObligationRepository(db)
	ctx := context.Background()

	if rec, err := repo.FindRecord(ctx, "ust-erklaerung.2026"); err != nil || rec != nil {
		t.Fatalf("ohne Eintrag: %+v, %v", rec, err)
	}
	now := time.Now().UTC()
	rec := &domain.ObligationRecord{Key: "ust-erklaerung.2026", Kind: domain.ObligationAnnualVatReturn,
		Status: domain.ObligationDone, EvidenceDate: "2027-05-04", EvidenceKind: domain.EvidenceTransferTicket,
		EvidenceReference: "et-1", DecidedChannel: "ui", DecidedAt: &now}
	if err := repo.SaveRecord(ctx, rec); err != nil {
		t.Fatal(err)
	}
	rec.EvidenceReference = "et-2"
	if err := repo.SaveRecord(ctx, rec); err != nil {
		t.Fatal(err)
	}
	all, err := repo.FindRecords(ctx)
	if err != nil || len(all) != 1 || all[0].EvidenceReference != "et-2" {
		t.Fatalf("Einträge: %+v, %v", all, err)
	}
	if err := repo.DeleteRecord(ctx, rec.Key); err != nil {
		t.Fatal(err)
	}
	if all, _ := repo.FindRecords(ctx); len(all) != 0 {
		t.Errorf("nach dem Löschen: %+v", all)
	}

	for _, date := range []string{"2026-05-04", "2026-03-01"} {
		if err := repo.CreateEvent(ctx, &domain.ObligationEvent{Kind: domain.EventShareholderChange, Date: date}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := repo.FindEvents(ctx)
	if err != nil || len(events) != 2 || events[0].Date != "2026-03-01" || events[0].ID != 2 {
		t.Fatalf("Anlässe: %+v, %v", events, err)
	}
}
