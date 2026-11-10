package wailsbridge

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/core"
)

// Der Export ist ein Lesevorgang und hält die Schreibsperre nicht.
//
// Ein Archiv über ein ganzes Geschäftsjahr läuft Sekunden bis Minuten. Hielte
// der Export die Sperre, wartete jede Buchung auf ihn. Gemessen wird deshalb,
// wie lange eine Buchung braucht, die beginnt, während der Export läuft.
func TestExportDoesNotBlockWriters(t *testing.T) {
	b := testBridge(t)

	// Etwas Bestand, damit der Lauf messbar Zeit braucht.
	for i := 0; i < 60; i++ {
		postSimpleEntry(t, b, "2026-03-01", "Bestand")
	}
	request := manualEntryRequest(t, b)

	const headStart = 5 * time.Millisecond
	var wg sync.WaitGroup
	var exportDuration, writerDuration time.Duration

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
		writeFrom := time.Now()
		if _, err := postManualEntry(b, request); err != nil {
			t.Errorf("Buchung während des Exports: %v", err)
		}
		writerDuration = time.Since(writeFrom)
	}()
	wg.Wait()

	t.Logf("Export %s, Buchung währenddessen %s", exportDuration, writerDuration)
	if exportDuration <= 4*headStart {
		t.Skipf("der Export lief nur %s; so kurz lässt sich die Wartezeit nicht beurteilen", exportDuration)
	}
	if writerDuration > exportDuration/2 && writerDuration > 500*time.Millisecond {
		t.Errorf("die Buchung brauchte %s bei einer Exportdauer von %s; der Export hält die Sperre",
			writerDuration, exportDuration)
	}

	count := 0
	testRead(t, b, func(ctx context.Context, s *core.Services) error {
		entries, err := s.Repos.Journal.FindAll(ctx, 2026)
		count = len(entries)
		return err
	})
	if count != 61 {
		t.Errorf("es stehen %d Buchungen im Journal, erwartet 61", count)
	}
}
