package core

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Der Pflichtenkalender liest im Lesevorgang, ein Agent meldet „vorbereitet",
// erledigt meldet nur der Mensch in der App.
func TestObligationsThroughTheCore(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	agent := agentCtx(t)
	const key = "kst-erklaerung.2026"

	get := func(ctx context.Context) *domain.Obligation {
		t.Helper()
		var o *domain.Obligation
		if err := tenant.Read(ctx, func(ctx context.Context, s *Services) (err error) {
			o, err = s.Obligations.Get(ctx, key, "2027-01-15")
			return err
		}); err != nil {
			t.Fatalf("Pflicht lesen: %v", err)
		}
		return o
	}
	if o := get(agent); o.Status != domain.ObligationOpen || o.DueDate != "2027-08-02" {
		t.Fatalf("Körperschaftsteuererklärung 2026: %s, fällig %s — erwartet offen, 02.08.2027", o.Status, o.DueDate)
	}

	// Die Liste entsteht im lesenden Vorgang: keine Quelle darf dabei schreiben.
	if err := tenant.Read(agent, func(ctx context.Context, s *Services) error {
		_, err := s.Obligations.List(ctx, service.ObligationQuery{From: "2026-01-01", To: "2027-12-31"})
		return err
	}); err != nil {
		t.Fatalf("Pflichten im Lesevorgang: %v", err)
	}

	if err := tenant.Write(agent, func(ctx context.Context, s *Services) error {
		_, err := s.Obligations.MarkPrepared(ctx, key, "Zahlen aus der Überleitung vorbereitet")
		return err
	}); err != nil {
		t.Fatalf("ein Agent meldet vorbereitet: %v", err)
	}
	o := get(agent)
	if o.Status != domain.ObligationPrepared || o.Preparation == nil || o.Preparation.Channel != "agent:claude-desktop" {
		t.Fatalf("nach der Meldung: %s %+v", o.Status, o.Preparation)
	}

	err := tenant.Write(agent, func(ctx context.Context, s *Services) error {
		_, err := s.Obligations.Confirm(ctx, key, service.ConfirmInput{
			Date: "2026-10-01", Kind: domain.EvidenceTransferTicket, Reference: "et-1"})
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "vorbehalten") {
		t.Fatalf("ein Agent bestätigt die Erledigung: %v", err)
	}

	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.Obligations.Confirm(ctx, key, service.ConfirmInput{
			Date: "2026-10-01", Kind: domain.EvidenceTransferTicket, Reference: "et-4711"})
		return err
	}, AppOnly()); err != nil {
		t.Fatalf("die App bestätigt: %v", err)
	}
	o = get(uiCtx)
	if o.Status != domain.ObligationDone || o.Proof == nil || o.Proof.Reference != "et-4711" || o.Proof.Channel != "ui" {
		t.Fatalf("nach der Bestätigung: %s %+v", o.Status, o.Proof)
	}

	var entries []domain.AuditLogEntry
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) (err error) {
		entries, err = s.Repos.Audit.FindFiltered(ctx, 50, domain.AuditFilter{EntityType: "OBLIGATION"})
		return err
	}); err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	channels := map[string]bool{}
	for _, e := range entries {
		if e.EntityID == key {
			channels[e.Channel] = true
		}
	}
	if !channels["ui"] || !channels["agent:claude-desktop"] {
		t.Errorf("Protokoll der Pflicht nach Erfassungsweg: %v", channels)
	}

	// Ein Anlass darf auch ein Agent melden; er fügt nur Pflichten hinzu.
	if err := tenant.Write(agent, func(ctx context.Context, s *Services) error {
		_, err := s.Obligations.RecordEvent(ctx, domain.EventShareholderChange, "2026-10-01", "Anteilskauf")
		return err
	}); err != nil {
		t.Fatalf("ein Agent meldet einen Anlass: %v", err)
	}
	var list []domain.Obligation
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) (err error) {
		list, err = s.Obligations.List(ctx, service.ObligationQuery{
			Kinds: []domain.ObligationKind{domain.ObligationShareholderList}})
		return err
	}); err != nil {
		t.Fatalf("Pflichten lesen: %v", err)
	}
	if len(list) != 1 || list[0].Channel != domain.ChannelNotary || !list[0].Immediate {
		t.Fatalf("Gesellschafterliste nach dem Anlass: %+v", list)
	}
}
