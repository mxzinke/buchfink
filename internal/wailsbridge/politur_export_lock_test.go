package wailsbridge

import (
	"sync"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/receiptstore"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

// Der Export hält die Sperre nur zum Einsammeln der Dienste, nicht über seine
// Laufzeit.
//
// Ein Archiv über ein ganzes Geschäftsjahr läuft Sekunden bis Minuten. Wird die
// Lesesperre so lange gehalten, wartet jede schreibende Bridge-Methode auf sie:
// der Anwender kann während des Exports nicht buchen, und der Mandantenwechsel
// steht ebenfalls.
//
// Gemessen wird deshalb, wann ein Schreiber die Sperre bekommt. Er greift
// bewusst erst zu, wenn der Export schon läuft — sonst wäre nicht zu
// unterscheiden, ob er sie bekam, weil sie frei war, oder weil er schneller
// war.
func TestExportDoesNotHoldTheBridgeLock(t *testing.T) {
	b := testBridge(t)
	b.exportSvc = service.NewExportService(
		b.journalRepo,
		b.accountRepo,
		b.contactRepo,
		repository.NewReceiptRepository(b.db),
		repository.NewAssetRepository(b.db),
		repository.NewPaymentAllocationRepository(b.db),
		b.auditRepo,
		b.settingsRepo,
		repository.NewFestschreibungRepository(b.db),
		repository.NewVatReturnRepository(b.db),
		repository.NewCheckRunRepository(b.db),
		repository.NewFiscalYearRepository(b.db),
		receiptstore.New(b.dataDir),
		b.dataDir,
		2026,
	)

	// Etwas Bestand, damit der Lauf messbar Zeit braucht.
	for i := 0; i < 60; i++ {
		entry := manualEntry()
		if _, err := b.journalSvc.Post(t.Context(), &entry); err != nil {
			t.Fatalf("Buchung %d: %v", i, err)
		}
	}

	// Der Schreiber wartet erst einmal, damit der Export die Sperre vor ihm
	// nimmt; danach misst er, wie lange er auf sie wartet.
	const headStart = 5 * time.Millisecond

	var wg sync.WaitGroup
	var exportDuration, writerWait time.Duration

	wg.Add(2)
	start := time.Now()
	go func() {
		defer wg.Done()
		if _, err := b.ExportZ3(2026, t.TempDir()); err != nil {
			t.Errorf("Export: %v", err)
		}
		exportDuration = time.Since(start)
	}()
	go func() {
		defer wg.Done()
		time.Sleep(headStart)
		waitFrom := time.Now()
		b.mu.Lock()
		writerWait = time.Since(waitFrom)
		b.mu.Unlock()
	}()
	wg.Wait()

	t.Logf("Export %s, Wartezeit des Schreibers %s", exportDuration, writerWait)
	if exportDuration <= 4*headStart {
		t.Skipf("der Export lief nur %s; so kurz lässt sich die Wartezeit nicht beurteilen", exportDuration)
	}
	// Großzügig: der Schreiber darf auf das kurze Einsammeln der Dienste warten,
	// aber nicht auf den Export.
	if writerWait > exportDuration/2 {
		t.Errorf("der Schreiber wartete %s bei einer Exportdauer von %s; "+
			"der Export hält die Sperre über seine Laufzeit", writerWait, exportDuration)
	}

	var count int64
	if err := b.db.Model(&domain.JournalEntry{}).Count(&count).Error; err != nil {
		t.Fatalf("Buchungen zählen: %v", err)
	}
	if count != 60 {
		t.Errorf("es stehen %d Buchungen im Journal, erwartet 60", count)
	}
}
