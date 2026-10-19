package service

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// --- UTC in den Persistenzpfaden -------------------------------------------

// Zeitpunkte, die gespeichert werden, stehen in UTC. Eine Ortszeit ohne Zone ist
// in der Nacht der Zeitumstellung mehrdeutig, und die Reihenfolge der
// Aufzeichnungen richtet sich nach ihr.
func TestPersistedTimestampsAreUTC(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Sicherungslauf.
	backupRepo := repository.NewBackupRunRepository(env.db)
	run := &domain.BackupRun{
		Kind: domain.BackupKindManual, StartedAt: time.Now().UTC(),
		FinishedAt: time.Now().UTC(), Success: true, Target: "/media/sicherung/x.zip",
	}
	if err := backupRepo.Create(ctx, run); err != nil {
		t.Fatalf("Sicherungslauf schreiben: %v", err)
	}
	runs, err := backupRepo.FindRecent(ctx, 1)
	if err != nil || len(runs) == 0 {
		t.Fatalf("Sicherungslauf lesen: %v", err)
	}
	if _, offset := runs[0].StartedAt.Zone(); offset != 0 {
		t.Errorf("der Startzeitpunkt des Sicherungslaufs hat den Zonenversatz %d", offset)
	}

	// Voranmeldung.
	vatRepo := repository.NewVatReturnRepository(env.db)
	rec := &domain.VatReturn{
		FiscalYear: 2026, PeriodType: domain.VatPeriodMonth, PeriodKey: "2026-03",
		PeriodFrom: "2026-03-01", PeriodTo: "2026-03-31", Status: domain.VatReturnDraft,
	}
	if err := vatRepo.Create(ctx, rec); err != nil {
		t.Fatalf("Voranmeldung schreiben: %v", err)
	}
	if _, offset := rec.CreatedAt.Zone(); offset != 0 {
		t.Errorf("der Zeitpunkt der Voranmeldung hat den Zonenversatz %d", offset)
	}

	// Prüflauf.
	checkRepo := repository.NewCheckRunRepository(env.db)
	checkRun := &domain.CheckRun{FiscalYear: 2026, CutoffDate: "2026-03-31"}
	if err := checkRepo.Create(ctx, checkRun); err != nil {
		t.Fatalf("Prüflauf schreiben: %v", err)
	}
	if _, offset := checkRun.CreatedAt.Zone(); offset != 0 {
		t.Errorf("der Zeitpunkt des Prüflaufs hat den Zonenversatz %d", offset)
	}

	// Einstellung.
	settingsRepo := repository.NewSettingsRepository(env.db)
	if err := settingsRepo.Set(ctx, "test_key", "wert"); err != nil {
		t.Fatalf("Einstellung schreiben: %v", err)
	}
	var item domain.SettingItem
	if err := env.db.Where("key = ?", "test_key").Take(&item).Error; err != nil {
		t.Fatalf("Einstellung lesen: %v", err)
	}
	if _, offset := item.UpdatedAt.Zone(); offset != 0 {
		t.Errorf("der Zeitpunkt der Einstellung hat den Zonenversatz %d", offset)
	}

	// Zeitpunkte, die als Zeichenkette gespeichert oder ausgegeben werden,
	// führen ihre Zone mit: eine Ortszeit ohne Zonenangabe ist außerhalb des
	// Rechners, auf dem sie entstand, nicht mehr eindeutig.
	utcString := func(label, value string) {
		t.Helper()
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Errorf("%s: %q ist kein RFC-3339-Zeitpunkt: %v", label, value, err)
			return
		}
		if _, offset := parsed.Zone(); offset != 0 {
			t.Errorf("%s: %q hat den Zonenversatz %d", label, value, offset)
		}
	}

	// Lückenvermerk am Nummernkreis.
	if err := env.invoicesWired(t).RecordNumberGapReason(
		ctx, 2026, 7, domain.NumberGapAborted, "Erfassung abgebrochen"); err != nil {
		t.Fatalf("Lückengrund festhalten: %v", err)
	}
	gaps, err := repository.NewNumberGapRepository(env.db).FindByYear(ctx, domain.NumberRangeInvoice, 2026)
	if err != nil || len(gaps) == 0 {
		t.Fatalf("Lückenvermerk lesen: %v (%d)", err, len(gaps))
	}
	utcString("Lückenvermerk", gaps[0].RecordedAt)

	// Prüfzeitpunkte der beiden Integritätsprüfungen.
	fileCheck, err := env.receipts.VerifyReceiptFiles(ctx)
	if err != nil {
		t.Fatalf("Belegdateien prüfen: %v", err)
	}
	utcString("Belegdateiprüfung", fileCheck.CheckedAt)

	integrity, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Kette prüfen: %v", err)
	}
	utcString("Kettenprüfung", integrity.CheckedAt)
}

func TestCheckAndExportCountGermanCalendarDays(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2026-03-28T23:30:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-03-29T22:30:00Z")
	if days, ok := daysBetweenEntryDates("2026-03-29", from); !ok || days != 0 {
		t.Fatalf("Erfassungstag: %d, %v", days, ok)
	}
	if days, ok := daysBetweenTimes(from, to); !ok || days != 1 {
		t.Fatalf("Festschreibung nach Sommerzeitwechsel: %d, %v", days, ok)
	}
}
