package wailsbridge

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Bestätigen, Entfallen und Zurücknehmen einer Pflicht sind der App
// vorbehalten; ein Agent meldet höchstens „vorbereitet“.
func TestObligationDecisionsAreAppOnly(t *testing.T) {
	methods := bridgeMethods(t)
	for _, name := range []string{"ConfirmObligation", "WaiveObligation", "ReopenObligation"} {
		m, ok := methods[name]
		if !ok {
			t.Errorf("die Methode %s gibt es nicht", name)
			continue
		}
		if !m.guarded || !m.appOnly {
			t.Errorf("%s.%s schreibt nicht über write mit core.AppOnly()", m.file, name)
		}
	}
}

// Eine Pflicht, die Buchfink nicht aus den Daten erkennt, bestätigt der Mensch
// mit Nachweis. Danach verlangt sie nichts mehr; zurückgenommen steht sie
// wieder offen, als entfallen gekennzeichnet braucht sie einen Grund.
func TestConfirmObligationWithEvidence(t *testing.T) {
	b := testBridge(t)
	next, err := b.GetNextObligations(0)
	if err != nil {
		t.Fatalf("Pflichten lesen: %v", err)
	}
	var key string
	for _, o := range next {
		if !o.Target.Valid() {
			t.Errorf("%s führt an eine Stelle, die die Oberfläche nicht kennt: %+v", o.Key, o.Target)
		}
		if key == "" && o.Evidence != domain.EvidenceFromData {
			key = o.Key
		}
	}
	if key == "" {
		t.Fatalf("keine bestätigbare Pflicht unter %d offenen", len(next))
	}
	today := time.Now().Format(domain.DateLayout)

	if _, err := b.ConfirmObligation(key, service.ConfirmInput{
		Date: today, Kind: domain.EvidenceTransferTicket,
	}); err == nil {
		t.Error("ein Transferticket ohne Nummer darf nicht als Nachweis gelten")
	}

	done, err := b.ConfirmObligation(key, service.ConfirmInput{
		Date: today, Kind: domain.EvidenceTransferTicket, Reference: " et-4711 ",
	})
	if err != nil {
		t.Fatalf("Bestätigen: %v", err)
	}
	if done.Status != domain.ObligationDone || done.Proof == nil || done.Proof.Reference != "et-4711" {
		t.Errorf("nach der Bestätigung: Stand %s, Nachweis %+v", done.Status, done.Proof)
	}
	if done.Proof != nil && done.Proof.Channel != string(actor.ChannelUI) {
		t.Errorf("bestätigt über %q, erwartet ui", done.Proof.Channel)
	}
	if after, err := b.GetNextObligations(0); err != nil {
		t.Fatalf("Pflichten lesen: %v", err)
	} else {
		for _, o := range after {
			if o.Key == key {
				t.Errorf("%s steht nach der Bestätigung weiter unter den offenen Pflichten", key)
			}
		}
	}

	reopened, err := b.ReopenObligation(key, "versehentlich bestätigt")
	if err != nil {
		t.Fatalf("Zurücknehmen: %v", err)
	}
	if reopened.Status.Closed() {
		t.Errorf("nach dem Zurücknehmen steht die Pflicht auf %s", reopened.Status)
	}

	if _, err := b.WaiveObligation(key, "  "); err == nil {
		t.Error("Entfallen ohne Grund darf nicht gelingen")
	}
	waived, err := b.WaiveObligation(key, "Gegenstand rein freiberuflich")
	if err != nil {
		t.Fatalf("Entfallen: %v", err)
	}
	if waived.Status != domain.ObligationWaived || waived.WaivedReason != "Gegenstand rein freiberuflich" {
		t.Errorf("nach dem Entfallen: Stand %s, Grund %q", waived.Status, waived.WaivedReason)
	}
	got, err := b.GetObligation(key)
	if err != nil || got.Status != domain.ObligationWaived {
		t.Errorf("GetObligation nach dem Entfallen: %+v, %v", got, err)
	}
}

// Die Übersicht zeigt, was verbundene Apps seit der letzten Festschreibung
// gebucht haben: nur Buchungen mit Erfassungsweg agent:*, nur nicht
// festgeschriebene, die jüngste zuerst und mit dem Vermerk, ob ein Storno sie
// aufhebt.
func TestAgentBookingsSinceLastCommit(t *testing.T) {
	b := wiredBridge(t, withTSA(time.Now().UTC(), "Test-TSA"))

	postSimpleEntry(t, b, "2026-01-15", "Wareneingang Januar")
	postAgentEntry(t, b, "2026-01-20", "Hosting Januar")
	kept := postAgentEntry(t, b, "2026-02-03", "Hosting Februar")
	wrong := postAgentEntry(t, b, "2026-02-04", "Softwarelizenz")
	postSimpleEntry(t, b, "2026-02-05", "Wareneingang Februar")
	var reversal *domain.JournalEntry
	agentWrite(t, b, func(ctx context.Context, s *core.Services) error {
		var err error
		reversal, err = s.Journal.Reverse(ctx, wrong.ID, "falsches Konto")
		return err
	})

	if _, err := b.CommitPeriod("month", "Januar 2026", "2026-01-31", "Testlauf ohne Belege"); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}

	got, err := b.GetAgentBookings(0)
	if err != nil {
		t.Fatalf("GetAgentBookings: %v", err)
	}
	if got.LastCommit == nil || got.LastCommit.CutoffDate != "2026-01-31" {
		t.Errorf("letzte Festschreibung %+v, erwartet Stichtag 2026-01-31", got.LastCommit)
	}
	want := []uint{reversal.ID, wrong.ID, kept.ID}
	if got.Total != len(want) || len(got.Entries) != len(want) {
		t.Fatalf("%d Buchungen (insgesamt %d), erwartet %d", len(got.Entries), got.Total, len(want))
	}
	for i, id := range want {
		e := got.Entries[i]
		if e.ID != id {
			t.Errorf("Stelle %d: Buchung %d, erwartet %d", i, e.ID, id)
		}
		if !strings.HasPrefix(e.Channel, "agent:") || e.CommittedAt != nil {
			t.Errorf("%s: Erfassungsweg %q, festgeschrieben %v", e.EntryNumber, e.Channel, e.CommittedAt)
		}
		if e.Reversed != (id == wrong.ID) {
			t.Errorf("%s: storniert %v", e.EntryNumber, e.Reversed)
		}
	}

	limited, err := b.GetAgentBookings(1)
	if err != nil {
		t.Fatalf("GetAgentBookings(1): %v", err)
	}
	if limited.Total != len(want) || len(limited.Entries) != 1 {
		t.Errorf("mit Grenze 1: %d Buchungen, insgesamt %d", len(limited.Entries), limited.Total)
	}
}

// agentWrite schreibt im offenen Mandanten, wie es ein verbundener Agent tut.
func agentWrite(t *testing.T, b *BuchfinkBridge, fn func(ctx context.Context, s *core.Services) error) {
	t.Helper()
	channel, err := actor.Agent("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.host.Write(actor.WithChannel(context.Background(), channel), fn); err != nil {
		t.Fatalf("Schreibvorgang des Agenten: %v", err)
	}
}

// postAgentEntry bucht wie postSimpleEntry, aber über den Erfassungsweg eines
// Agenten.
func postAgentEntry(t *testing.T, b *BuchfinkBridge, bookingDate, description string) *domain.JournalEntry {
	t.Helper()
	entry := &domain.JournalEntry{
		FiscalYear: 2026, BookingDate: bookingDate, DocumentDate: bookingDate,
		ServiceDateFrom: bookingDate, ServiceDateTo: bookingDate,
		Description: description, Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
		Kind: domain.EntryKindNormal, Currency: "EUR", ExchangeRateMicros: 1_000_000,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 4200},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 4200},
		},
	}
	var created *domain.JournalEntry
	agentWrite(t, b, func(ctx context.Context, s *core.Services) error {
		var err error
		created, err = s.Journal.Post(ctx, entry)
		return err
	})
	return created
}
