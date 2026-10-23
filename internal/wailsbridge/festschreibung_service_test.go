package wailsbridge

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/timestamp"
)

// withTSA setzt eine Zeitstempelstelle ein, die eine feste beglaubigte Zeit
// liefert.
//
// Ohne diesen Austausch ließe sich der Abgleich der Systemzeit mit der
// beglaubigten Zeit nicht prüfen: der Test müsste entweder die Uhr des Rechners
// verstellen oder eine Stelle im Netz erreichen.
func withTSA(genTime time.Time, name string) func(*core.HostOptions) {
	return func(o *core.HostOptions) {
		o.Network.Timestamp = func(ctx context.Context, tsaURL, sha256Hex string) (*timestamp.Result, error) {
			return &timestamp.Result{Token: []byte("test-token"), GenTime: genTime, TSAName: name}, nil
		}
	}
}

// Weicht die beglaubigte Zeit über die Toleranz von der Systemzeit ab, entsteht
// ein eigener Protokolleintrag mit beiden Zeiten, und die Festschreibung hat
// den Hinweis.
//
// Der Hinweis allein genügt nicht: er steht dort, wo ohnehin schon jemand
// hinsieht. Als eigener Vorgang im Protokoll fällt die Abweichung auf.
func TestCommitPeriodLogsTimeDriftBeyondTolerance(t *testing.T) {
	b := wiredBridge(t, withTSA(time.Now().UTC().Add(90*time.Minute), "Test-TSA"))

	postSimpleEntry(t, b, "2026-01-15", "Wareneingang Januar")

	// Neunzig Minuten Rückstand der Systemzeit gegenüber der beglaubigten Zeit:
	// weit über der Toleranz von fünf Minuten.

	rec, err := b.CommitPeriod("month", "Januar 2026", "2026-01-31",
		"Testlauf ohne Belege — die Belegprüfung wird bewusst übergangen")
	if err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	if rec.TimeDriftNote == "" {
		t.Fatal("die Festschreibung hat keinen Hinweis auf die Zeitabweichung")
	}
	if !strings.Contains(rec.TimeDriftNote, "geht nach") {
		t.Errorf("der Hinweis muss die Richtung nennen: %q", rec.TimeDriftNote)
	}

	entries := auditEntries(t, b, domain.AuditFilter{EntityType: "TIME_DRIFT"})
	if len(entries) != 1 {
		t.Fatalf("erwartet ein Protokolleintrag zur Zeitabweichung, gefunden %d", len(entries))
	}
	details := entries[0].Details
	// Beide Zeiten stehen im Eintrag: ohne sie ließe sich die Abweichung später
	// nicht mehr nachrechnen.
	if !strings.Contains(details, "Systemzeit") || !strings.Contains(details, "beglaubigte Zeit") {
		t.Errorf("der Eintrag muss beide Zeiten nennen: %q", details)
	}
	if rec.TSAGenTime == nil ||
		!strings.Contains(details, rec.TSAGenTime.UTC().Format("2006-01-02T15:04")) {
		t.Errorf("die beglaubigte Zeit fehlt im Eintrag: %q", details)
	}
}

// Liegt die beglaubigte Zeit innerhalb der Toleranz, entsteht kein Eintrag. Eine
// Meldung über jede Sekunde Abweichung wäre Rauschen und niemand läse sie.
func TestCommitPeriodStaysSilentWithinTolerance(t *testing.T) {
	b := wiredBridge(t, withTSA(time.Now().UTC().Add(time.Minute), "Test-TSA"))
	postSimpleEntry(t, b, "2026-01-15", "Wareneingang Januar")

	rec, err := b.CommitPeriod("month", "Januar 2026", "2026-01-31", "Testlauf ohne Belege")
	if err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	if rec.TimeDriftNote != "" {
		t.Errorf("eine Minute liegt in der Toleranz, bekommen: %q", rec.TimeDriftNote)
	}
	entries := auditEntries(t, b, domain.AuditFilter{EntityType: "TIME_DRIFT"})
	if len(entries) != 0 {
		t.Errorf("ohne Abweichung darf kein Eintrag entstehen, gefunden %d", len(entries))
	}
}

// Nach der Festschreibung haben alle Buchungen bis zum Stichtag ihren
// Festschreibungszeitpunkt und den Verweis auf die Festschreibung; die
// Buchungen danach nicht.
func TestCommitPeriodStampsEntriesUpToTheCutoff(t *testing.T) {
	b := wiredBridge(t, withTSA(time.Now().UTC(), "Test-TSA"))

	inPeriod := postSimpleEntry(t, b, "2026-01-15", "Wareneingang Januar")
	afterPeriod := postSimpleEntry(t, b, "2026-02-03", "Wareneingang Februar")

	rec, err := b.CommitPeriod("month", "Januar 2026", "2026-01-31", "Testlauf ohne Belege")
	if err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	if rec.EntriesStamped != 1 {
		t.Errorf("gestempelt wurden %d Buchungen, erwartet 1", rec.EntriesStamped)
	}

	var stamped, open *domain.JournalEntry
	testRead(t, b, func(ctx context.Context, s *core.Services) error {
		var err error
		if stamped, err = s.Repos.Journal.FindByID(ctx, inPeriod.ID); err != nil {
			return err
		}
		open, err = s.Repos.Journal.FindByID(ctx, afterPeriod.ID)
		return err
	})
	if stamped.CommittedAt == nil {
		t.Error("die Buchung bis zum Stichtag hat keinen Festschreibungszeitpunkt")
	}
	if stamped.FestschreibungID == nil || *stamped.FestschreibungID != rec.ID {
		t.Errorf("die Buchung verweist nicht auf die Festschreibung %d: %v",
			rec.ID, stamped.FestschreibungID)
	}

	if open.CommittedAt != nil || open.FestschreibungID != nil {
		t.Error("eine Buchung nach dem Stichtag darf nicht festgeschrieben sein")
	}
}
